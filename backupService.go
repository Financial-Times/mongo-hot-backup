package main

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type backupService interface {
	Backup(ctx context.Context, collections []dbColl) error
	Restore(ctx context.Context, dateDir string, collections []dbColl) error
}

type dbColl struct {
	database   string
	collection string
}

type mongoBackupService struct {
	dbService      dbService
	storageService storageService
	statusKeeper   statusKeeper
}

func newMongoBackupService(dbService dbService, storageService storageService, statusKeeper statusKeeper) *mongoBackupService {
	return &mongoBackupService{
		dbService:      dbService,
		storageService: storageService,
		statusKeeper:   statusKeeper,
	}
}

type backupResult struct {
	Success    bool
	Timestamp  time.Time
	Collection dbColl
}

func (m *mongoBackupService) Backup(ctx context.Context, collections []dbColl) error {
	date := formattedNow()
	for _, coll := range collections {
		if err := m.backup(ctx, date, coll); err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoBackupService) backup(ctx context.Context, date string, coll dbColl) error {
	start := time.Now().UTC()

	logEntry := log.
		WithField("database", coll.database).
		WithField("collection", coll.collection)

	logEntry.Info("Saving collection...")

	reader, writer := newPipe(uploadOperation)
	defer func() {
		_ = reader.Close()
	}()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		//f := func(errs chan error) {
		//	errs <- m.storageService.Upload(ctx, date, coll.database, coll.collection, reader)
		//}
		//return createFunctionWithContext(ctx, f)
		return m.storageService.Upload(ctx, date, coll.database, coll.collection, reader)
	})
	g.Go(func() error {
		defer func() {
			_ = writer.Close()
		}()

		//f := func(errs chan error) {
		//	errs <- m.dbService.SaveCollection(ctx, coll.database, coll.collection, writer)
		//}
		//return createFunctionWithContext(ctx, f)
		return m.dbService.SaveCollection(ctx, coll.database, coll.collection, writer)
	})

	if err := g.Wait(); err != nil {
		logEntry.WithError(err).Error("Saving collection failed")

		result := backupResult{
			Timestamp:  time.Now().UTC(),
			Collection: coll,
		}
		_ = m.statusKeeper.Save(result)

		return fmt.Errorf("dumping failed for %s/%s: %v", coll.database, coll.collection, err)
	}

	logEntry.Infof("Collection successfully saved. Duration: %v", time.Since(start))

	result := backupResult{
		Success:    true,
		Timestamp:  time.Now().UTC(),
		Collection: coll,
	}
	return m.statusKeeper.Save(result)
}

func (m *mongoBackupService) Restore(ctx context.Context, date string, collections []dbColl) error {
	for _, coll := range collections {
		if err := m.restore(ctx, date, coll); err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoBackupService) restore(ctx context.Context, date string, coll dbColl) error {
	start := time.Now().UTC()

	logEntry := log.
		WithField("database", coll.database).
		WithField("collection", coll.collection)

	logEntry.Info("Restoring collection...")

	reader, writer := newPipe(downloadOperation)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer func() {
			logEntry.Info("G1 writer close start")
			if err := writer.Close(); err != nil {
				logEntry.WithError(err).Error("Closing writer failed")
			}
			logEntry.Info("G1 writer close end")
		}()
		//f := func(errs chan error) {
		//	errs <- m.storageService.Download(cctx, date, coll.database, coll.collection, writer)
		//}
		//return createFunctionWithContext(cctx, f)
		logEntry.Info("G1 start")
		err := m.storageService.Download(gCtx, date, coll.database, coll.collection, writer)
		logEntry.Infof("G1 end, err: %v", err)
		return err
	})
	g.Go(func() error {
		defer func() {
			logEntry.Info("G2 reader close start")
			if err := reader.Close(); err != nil {
				logEntry.WithError(err).Error(" G2 Closing reader failed")
			}
			logEntry.Info("G2 reader close end")
		}()
		//f := func(errs chan error) {
		//errs <- m.dbService.RestoreCollection(gCtx, coll.database, coll.collection, reader)
		//}
		//return createFunctionWithContext(ctx, f)
		logEntry.Info("G2 start")
		err := m.dbService.RestoreCollection(gCtx, coll.database, coll.collection, reader)
		logEntry.Infof("G2 end, err: %v", err)
		return err
	})

	logEntry.Info("Weittting, s 'e'")
	if err := g.Wait(); err != nil {
		logEntry.WithError(err).Error("Restoring collection failed")

		return err
	}

	logEntry.Infof("Finished restoration. Duration: %v", time.Since(start))

	return nil
}

func formattedNow() string {
	return time.Now().UTC().Format(dateFormat)
}
