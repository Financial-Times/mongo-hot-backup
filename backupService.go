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
		f := func(errs chan error) {
			errs <- m.storageService.Upload(ctx, date, coll.database, coll.collection, reader)
		}
		return createFunctionWithContext(ctx, f)
	})
	g.Go(func() error {
		defer func() {
			_ = writer.Close()
		}()

		f := func(errs chan error) {
			errs <- m.dbService.SaveCollection(ctx, coll.database, coll.collection, writer)
		}
		return createFunctionWithContext(ctx, f)
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
	defer func() {
		_ = reader.Close()
	}()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer func() {
			_ = writer.Close()
		}()

		f := func(errs chan error) {
			errs <- m.storageService.Download(ctx, date, coll.database, coll.collection, writer)
		}
		return createFunctionWithContext(ctx, f)
	})
	g.Go(func() error {
		f := func(errs chan error) {
			errs <- m.dbService.RestoreCollection(ctx, coll.database, coll.collection, reader)
		}
		return createFunctionWithContext(ctx, f)
	})

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

func createFunctionWithContext(ctx context.Context, f func(errs chan error)) error {
	errs := make(chan error, 1)
	go f(errs)

	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	case err := <-errs:
		return err
	}
}
