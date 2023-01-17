package main

import (
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/time/rate"
)

type dbService interface {
	SaveCollection(ctx context.Context, database, collection string, writer io.Writer) error
	RestoreCollection(ctx context.Context, database, collection string, reader io.Reader) error
}

type mongoService struct {
	session     mongoSession
	bsonService bsonService
	rateLimit   time.Duration
	batchLimit  int
}

func newMongoService(mongoClient mongoSession, bsonService bsonService, rateLimit time.Duration, batchLimit int) *mongoService {
	return &mongoService{
		session:     mongoClient,
		bsonService: bsonService,
		rateLimit:   rateLimit,
		batchLimit:  batchLimit,
	}
}

func (m *mongoService) SaveCollection(ctx context.Context, database, collection string, writer io.Writer) error {

	return fmt.Errorf("artificial save collection error")

	cur, err := m.session.FindAll(ctx, database, collection)
	if err != nil {
		return fmt.Errorf("couldn't obtain iterator over collection=%v/%v: %v", database, collection, err)
	}

	defer func() {
		_ = cur.Close(context.Background())
	}()

	for cur.Next(ctx) {
		if _, err = writer.Write(cur.Current()); err != nil {
			return err
		}
	}

	if err = cur.Err(); err != nil {
		return fmt.Errorf("error while iterating over collection=%v/%v noticed only at the end: %v", database, collection, err)
	}
	return nil
}

func (m *mongoService) RestoreCollection(ctx context.Context, database, collection string, reader io.Reader) error {
	err := m.session.RemoveAll(ctx, database, collection)
	if err != nil {
		return fmt.Errorf("error while clearing collection=%v/%v: %v", database, collection, err)
	}

	var batchBytes int
	batchStart := time.Now().UTC()

	limiter := rate.NewLimiter(rate.Every(m.rateLimit), 1)

	var models []mongo.WriteModel

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
			if err = m.session.BulkWrite(ctx, database, collection, models); err != nil {
				return fmt.Errorf("error while writing bulk: %w", err)
			}

			var duration = time.Since(batchStart)
			log.Infof("Written bulk restore batch for %s/%s. Took %v", database, collection, duration)

			// rate limit between writes to prevent overloading MongoDB
			_ = limiter.Wait(context.Background())

			models = nil
			batchBytes = 0
			batchStart = time.Now().UTC()
		}

		document := bson.Raw(next)
		models = append(models, mongo.NewInsertOneModel().SetDocument(document))

		batchBytes += len(next)
	}

	if err = m.session.BulkWrite(ctx, database, collection, models); err != nil {
		return fmt.Errorf("error while writing bulk: %w", err)
	}

	return nil
}
