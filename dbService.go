package main

import (
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type dbService interface {
	DumpCollectionTo(database, collection string, writer io.Writer) error
	RestoreCollectionFrom(database, collection string, reader io.Reader) error
}

type mongoService struct {
	connectionString string
	mgoLib           mongoLib
	bsonService      bsonService
	mongoTimeout     time.Duration
	rateLimit        time.Duration
	batchLimit       int
}

func newMongoService(connectionString string, mgoLib mongoLib, bsonService bsonService, mongoTimeout time.Duration, rateLimit time.Duration, batchLimit int) *mongoService {
	return &mongoService{
		connectionString: connectionString,
		mgoLib:           mgoLib,
		bsonService:      bsonService,
		mongoTimeout:     mongoTimeout,
		rateLimit:        rateLimit,
		batchLimit:       batchLimit,
	}
}

func (m *mongoService) DumpCollectionTo(database, collection string, writer io.Writer) error {
	session, err := m.mgoLib.DialWithTimeout(m.connectionString, m.mongoTimeout)
	if err != nil {
		return fmt.Errorf("Coulnd't dial mongo session: %v", err)
	}

	defer session.Close()

	start := time.Now()
	log.Infof("backing up %s/%s", database, collection)

	iter := session.SnapshotIter(database, collection, nil)
	err = iter.Err()
	if err != nil {
		return fmt.Errorf("Couldn't obtain iterator over collection=%v/%v: %v", database, collection, err)
	}
	for {
		result, hasNext := iter.Next()
		if !hasNext {
			break
		}
		_, err := writer.Write(result)
		if err != nil {
			return err
		}
	}

	log.Infof("backing up finished for %s/%s. duration=%v", database, collection, time.Now().Sub(start))
	err = iter.Err()
	if err != nil {
		return fmt.Errorf("Error while iterating over collection=%v/%v noticed only at the end: %v", database, collection, err)
	}
	return nil
}

func (m *mongoService) RestoreCollectionFrom(database, collection string, reader io.Reader) error {
	session, err := m.mgoLib.DialWithTimeout(m.connectionString, 0)
	if err != nil {
		return fmt.Errorf("error while dialing mongo session: %v", err)
	}
	defer session.Close()

	err = m.clearCollection(session, database, collection)
	if err != nil {
		return fmt.Errorf("error while clearing collection=%v/%v: %v", database, collection, err)
	}

	start := time.Now()
	log.Infof("starting restore of %s/%s", database, collection)

	bulk := session.Bulk(database, collection)

	var batchBytes int
	batchStart := time.Now()

	limiter := rate.NewLimiter(rate.Every(m.rateLimit), 1)

	for {
		next, err := m.bsonService.ReadNextBSON(reader)
		if err != nil {
			return fmt.Errorf("error while reading bson: %v", err)
		}
		if next == nil {
			break
		}

		// If we have something to write and the next doc would push the batch over
		// the limit, write the batch out now. 15000000 is intended to be within the
		// expected 16MB limit
		if batchBytes > 0 && batchBytes+len(next) > m.batchLimit {
			err = bulk.Run()
			if err != nil {
				return fmt.Errorf("error while writing bulk: %v", err)
			}

			var duration = time.Since(batchStart)
			log.Infof("Written bulk restore batch for %s/%s. Took %v", database, collection, duration)

			// rate limit between writes to prevent overloading MongoDB
			limiter.Wait(context.Background())

			bulk = session.Bulk(database, collection)
			batchBytes = 0
			batchStart = time.Now()
		}

		bulk.Insert(next)

		batchBytes += len(next)
	}
	err = bulk.Run()
	if err != nil {
		return fmt.Errorf("error while writing bulk: %v", err)
	}
	log.Infof("finished restore of %s/%s. Duration: %v", database, collection, time.Since(start))
	return nil
}

func (m *mongoService) clearCollection(session mongoSession, database, collection string) error {
	start := time.Now()
	log.Infof("clearing collection %s/%s", database, collection)
	err := session.RemoveAll(database, collection, nil)
	log.Infof("finished clearing collection %s/%s. Duration : %v", database, collection, time.Now().Sub(start))

	return err
}
