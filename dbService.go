package main

import (
	"context"
	"io"
	"time"

	"golang.org/x/time/rate"

	log "github.com/Sirupsen/logrus"
)

type dbService interface {
	DumpCollectionTo(database, collection string, writer io.Writer) error
	RestoreCollectionFrom(database, collection string, reader io.Reader) error
}

type mongoService struct {
	connectionString string
	mgoLib           mongoLib
	bsonService      bsonService
}

func newMongoService(connectionString string, mgoLib mongoLib, bsonService bsonService) *mongoService {
	return &mongoService{connectionString: connectionString, mgoLib: mgoLib, bsonService: bsonService}
}

func (m *mongoService) DumpCollectionTo(database, collection string, writer io.Writer) error {
	session, err := m.mgoLib.DialWithTimeout(m.connectionString, 0)
	if err != nil {
		return err
	}
	session.SetPrefetch(1.0)
	defer session.Close()

	start := time.Now()
	log.Infof("backing up %s/%s", database, collection)

	iter := session.SnapshotIter(database, collection, nil)
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

	log.Infof("backing up finished for %s/%s. duration=%v", database, collection, time.Now().Sub(start).Truncate(1*time.Second))
	return iter.Err()
}

func (m *mongoService) RestoreCollectionFrom(database, collection string, reader io.Reader) error {
	session, err := m.mgoLib.DialWithTimeout(m.connectionString, 0)
	if err != nil {
		return err
	}
	defer session.Close()

	err = m.clearCollection(session, database, collection)
	if err != nil {
		return err
	}

	start := time.Now()
	log.Infof("starting restore of %s/%s", database, collection)

	bulk := session.Bulk(database, collection)

	var batchBytes int
	batchStart := time.Now()

	// set rate limit to 250ms
	limiter := rate.NewLimiter(rate.Every(250*time.Millisecond), 1)

	for {

		next, err := m.bsonService.ReadNextBSON(reader)
		if err != nil {
			return err
		}
		if next == nil {
			break
		}

		// If we have something to write and the next doc would push the batch over
		// the limit, write the batch out now. 15000000 is intended to be within the
		// expected 16MB limit
		if batchBytes > 0 && batchBytes+len(next) > 15000000 {
			err = bulk.Run()
			if err != nil {
				return err
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
	log.Infof("finished restore of %s/%s. Duration: %v", database, collection, time.Since(start))
	return err
}

func (m *mongoService) clearCollection(session mongoSession, database, collection string) error {
	start := time.Now()
	log.Infof("clearing collection %s/%s", database, collection)
	err := session.RemoveAll(database, collection, nil)
	log.Infof("finished clearing collection %s/%s. Duration : %v", database, collection, time.Now().Sub(start))

	return err
}
