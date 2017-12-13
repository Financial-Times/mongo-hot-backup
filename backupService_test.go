package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBackup_Ok(t *testing.T) {
	mockedStorageService := new(mockStorageServie)
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

	err := backupService.Backup([]dbColl{dbColl{"database1", "collection1"}})

	assert.NoError(t, err, "Error wasn't expected during backup.")
}
