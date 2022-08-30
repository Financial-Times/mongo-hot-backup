package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBackup_Ok(t *testing.T) {
	mockedStorageService := new(mockStorageService)
	mockedWriter := new(mockWriteCloser)
	mockedWriter.On("Close").Return(nil)
	mockedStorageService.On("Writer", mock.MatchedBy(func(date string) bool { return true }), "database1", "collection1").Return(mockedWriter, nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("DumpCollectionTo", "database1", "collection1", mockedWriter).Return(nil)
	mockedStatusKeeper := new(mockStatusKeeper)
	mockedStatusKeeper.On("Save",
		mock.MatchedBy(func(result backupResult) bool {
			return result.Success &&
				result.Collection.collection == "collection1" &&
				result.Collection.database == "database1"
		})).Return(nil)
	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, mockedStatusKeeper)

	err := backupService.Backup([]dbColl{{"database1", "collection1"}})

	assert.NoError(t, err, "Error wasn't expected during backup.")
}

func TestBackup_ErrorOnDump(t *testing.T) {
	mockedStorageService := new(mockStorageService)
	mockedWriter := new(mockWriteCloser)
	mockedWriter.On("Close").Return(nil)
	mockedStorageService.On("Writer", mock.MatchedBy(func(date string) bool { return true }), "database1", "collection1").Return(mockedWriter, nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("DumpCollectionTo", "database1", "collection1", mockedWriter).Return(fmt.Errorf("Couldn't dump db"))
	mockedStatusKeeper := new(mockStatusKeeper)
	mockedStatusKeeper.On("Save",
		mock.MatchedBy(func(result backupResult) bool {
			return !result.Success &&
				result.Collection.collection == "collection1" &&
				result.Collection.database == "database1"
		})).Return(nil)
	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, mockedStatusKeeper)

	err := backupService.Backup([]dbColl{{"database1", "collection1"}})

	assert.Error(t, err, "Error was expected during backup.")
	assert.Equal(t, "dumping failed for database1/collection1: Couldn't dump db", err.Error())
}

func TestBackup_ErrorOnSavingStatus(t *testing.T) {
	mockedStorageService := new(mockStorageService)
	mockedWriter := new(mockWriteCloser)
	mockedWriter.On("Close").Return(nil)
	mockedStorageService.On("Writer", mock.MatchedBy(func(date string) bool { return true }), "database1", "collection1").Return(mockedWriter, nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("DumpCollectionTo", "database1", "collection1", mockedWriter).Return(nil)
	mockedStatusKeeper := new(mockStatusKeeper)
	mockedStatusKeeper.On("Save",
		mock.MatchedBy(func(result backupResult) bool {
			return result.Success &&
				result.Collection.collection == "collection1" &&
				result.Collection.database == "database1"
		})).Return(fmt.Errorf("Coulnd't save status of backup"))
	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, mockedStatusKeeper)

	err := backupService.Backup([]dbColl{{"database1", "collection1"}})

	assert.Error(t, err, "Error was expected during backup.")
	assert.Equal(t, "Coulnd't save status of backup", err.Error())
}

func TestRestore_OK(t *testing.T) {
	mockedStorageService := new(mockStorageService)
	mockedReadCloser := new(mockReadCloser)
	mockedReadCloser.On("Close").Return(nil)
	mockedStorageService.On("Reader", "2017-09-04T12-40-36", "database1", "collection1").Return(mockedReadCloser, nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("RestoreCollectionFrom", "database1", "collection1", mockedReadCloser).Return(nil)
	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)

	err := backupService.Restore("2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})

	assert.NoError(t, err, "Error wasn't expected during backup.")
}

func TestRestore_ErrorOnRestore(t *testing.T) {
	mockedStorageService := new(mockStorageService)
	mockedReadCloser := new(mockReadCloser)
	mockedReadCloser.On("Close").Return(nil)
	mockedStorageService.On("Reader", "2017-09-04T12-40-36", "database1", "collection1").Return(mockedReadCloser, nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("RestoreCollectionFrom", "database1", "collection1", mockedReadCloser).Return(fmt.Errorf("Error while restoring. Test"))
	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)

	err := backupService.Restore("2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})

	assert.Error(t, err, "Error while restoring. Test")
}
