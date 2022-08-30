package main

import (
	"fmt"
	"time"
)

type backupService interface {
	Backup(collections []dbColl) error
	Restore(dateDir string, collections []dbColl) error
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

func (m *mongoBackupService) Backup(collections []dbColl) error {
	date := formattedNow()
	for _, coll := range collections {
		if err := m.backup(date, coll); err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoBackupService) backup(date string, coll dbColl) error {
	w := m.storageService.Writer(date, coll.database, coll.collection)
	defer w.Close()

	if err := m.dbService.DumpCollectionTo(coll.database, coll.collection, w); err != nil {
		result := backupResult{
			Timestamp:  time.Now().UTC(),
			Collection: coll,
		}
		_ = m.statusKeeper.Save(result)

		return fmt.Errorf("dumping failed for %s/%s: %v", coll.database, coll.collection, err)
	}

	result := backupResult{
		Success:    true,
		Timestamp:  time.Now().UTC(),
		Collection: coll,
	}
	return m.statusKeeper.Save(result)
}

func (m *mongoBackupService) Restore(date string, collections []dbColl) error {
	for _, coll := range collections {
		if err := m.restore(date, coll); err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoBackupService) restore(date string, coll dbColl) error {
	r := m.storageService.Reader(date, coll.database, coll.collection)
	defer r.Close()

	return m.dbService.RestoreCollectionFrom(coll.database, coll.collection, r)
}

func formattedNow() string {
	return time.Now().UTC().Format(dateFormat)
}
