package main

import (
	"fmt"
	"time"
)

type backupService interface {
	Backup(colls []fullColl) error
	Restore(dateDir string, colls []fullColl) error
}

type mongoBackupService struct {
	dbService      dbService
	storageService storageService
	statusKeeper   statusKeeper
}

func newMongoBackupService(dbService dbService, storageService storageService, statusKeeper statusKeeper) *mongoBackupService {
	return &mongoBackupService{
		dbService,
		storageService,
		statusKeeper,
	}
}

type backupResult struct {
	Success    bool
	Timestamp  time.Time
	Collection fullColl
}

func (m *mongoBackupService) Backup(colls []fullColl) error {
	date := formattedNow()
	for _, coll := range colls {
		err := m.backup(date, coll)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoBackupService) backup(date string, coll fullColl) error {
	w, err := m.storageService.Writer(date, coll.database, coll.collection)
	if err != nil {
		return err
	}
	defer w.Close()

	if err := m.dbService.DumpCollectionTo(coll.database, coll.collection, w); err != nil {
		result := backupResult{false, time.Now(), coll}
		m.statusKeeper.Save(result)
		return fmt.Errorf("dumping failed for %s/%s %v", coll.database, coll.collection, err)
	}

	if err := w.Close(); err != nil {
		return err
	}

	result := backupResult{true, time.Now(), coll}
	m.statusKeeper.Save(result)
	return nil
}

func (m *mongoBackupService) Restore(date string, colls []fullColl) error {
	for _, coll := range colls {
		err := m.restore(date, coll)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoBackupService) restore(date string, coll fullColl) error {
	r, err := m.storageService.Reader(date, coll.database, coll.collection)
	defer r.Close()
	if err != nil {
		return err
	}
	if err := m.dbService.RestoreCollectionFrom(coll.database, coll.collection, r); err != nil {
		return err
	}
	return nil
}

func formattedNow() string {
	return time.Now().UTC().Format(dateFormat)
}
