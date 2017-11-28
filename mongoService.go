package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
	"context"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"golang.org/x/time/rate"

	log "github.com/Sirupsen/logrus"
)

type dbService interface {
	DumpCollectionTo(connStr, database, collection string, writer io.Writer) error
	RestoreCollectionFrom(connStr, database, collection string, reader io.Reader) error
}

type mongoService struct {
	mgoLib mongoLib
}

func newMongoService(mgoLib mongoLib) *mongoService {
	return &mongoService{mgoLib: mgoLib}
}

func (m *mongoService) DumpCollectionTo(connStr, database, collection string, writer io.Writer) error {
	session, err := m.mgoLib.Dial(connStr)
	if err != nil {
		return err
	}
	session.SetPrefetch(1.0)
	defer session.Close()

	iter := session.SnapshotIter(database, collection, nil)

	for {
		raw := &bson.Raw{}
		next := iter.Next(raw)
		if !next {
			break
		}
		_, err := writer.Write(raw.Data)
		if err != nil {
			return err
		}
	}

	return iter.Err()
}

func (m *mongoService) RestoreCollectionFrom(connStr, database, collection string, reader io.Reader) error {
	session, err := mgo.DialWithTimeout(connStr, 0)
	if err != nil {
		return err
	}
	defer session.Close()

	err = m.clearCollection(session, database, collection)
	if err != nil {
		return err
	}

	start := time.Now()
	log.Printf("starting restore of %s/%s\n", database, collection)

	bulk := session.DB(database).C(collection).Bulk()

	var batchBytes int
	batchStart := time.Now()

	// set rate limit to 250ms
	limiter := rate.NewLimiter(rate.Every(250 * time.Millisecond), 1)

	for {

		next, err := m.readNextBSON(reader)
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
			_, err = bulk.Run()
			if err != nil {
				return err
			}

			var duration = time.Since(batchStart)
			log.Infof("Written bulk restore batch for %s/%s. Took %v", database, collection, duration)

			// rate limit between writes to prevent overloading MongoDB
			limiter.Wait(context.Background())

			bulk = session.DB(database).C(collection).Bulk()
			batchBytes = 0
			batchStart = time.Now()
		}

		bulk.Insert(bson.Raw{Data: next})

		batchBytes += len(next)
	}
	_, err = bulk.Run()
	log.Printf("finished restore of %s/%s. Duration: %v\n", database, collection, time.Since(start))
	return err
}

func (m *mongoService) readNextBSON(reader io.Reader) ([]byte, error) {
	var lenBytes [4]byte

	_, err := io.ReadFull(reader, lenBytes[:])
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
		return nil, nil
	}

	docLen := int32(binary.LittleEndian.Uint32(lenBytes[:]))

	if docLen < 5 {
		return nil, fmt.Errorf("invalid document size: %v bytes", docLen)
	}

	buf := make([]byte, docLen)
	copy(buf, lenBytes[:])

	_, err = io.ReadAtLeast(reader, buf[4:], int(docLen-4))
	if err != nil {
		if err == io.EOF {
			// this is a broken document.
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return buf, nil
}

func (m *mongoService) clearCollection(session *mgo.Session, database, collection string) error {
	start := time.Now()
	log.Printf("clearing collection %s/%s\n", database, collection)
	_, err := session.DB(database).C(collection).RemoveAll(nil)
	log.Printf("finished clearing collection %s/%s. Duration : %v\n", database, collection, time.Now().Sub(start))

	return err
}
