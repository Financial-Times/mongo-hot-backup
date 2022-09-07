package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBackup_Ok(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Upload", mock.MatchedBy(isTestContext), mock.AnythingOfType("string"), "database1", "collection1", mock.AnythingOfType("*io.PipeReader")).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("SaveCollection", mock.MatchedBy(isTestContext), "database1", "collection1", mock.AnythingOfType("*main.snappyWriteCloser")).Return(nil)
	mockedStatusKeeper := new(mockStatusKeeper)
	mockedStatusKeeper.On("Save",
		mock.MatchedBy(func(result backupResult) bool {
			return result.Success &&
				result.Collection.collection == "collection1" &&
				result.Collection.database == "database1"
		})).Return(nil)

	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, mockedStatusKeeper)
	err := backupService.Backup(ctx, []dbColl{{"database1", "collection1"}})

	assert.NoError(t, err, "Error wasn't expected during backup.")
}

func TestBackup_ErrorOnDump(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Upload", mock.MatchedBy(isTestContext), mock.AnythingOfType("string"), "database1", "collection1", mock.AnythingOfType("*io.PipeReader")).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("SaveCollection", mock.MatchedBy(isTestContext), "database1", "collection1", mock.AnythingOfType("*main.snappyWriteCloser")).Return(fmt.Errorf("couldn't dump db"))
	mockedStatusKeeper := new(mockStatusKeeper)
	mockedStatusKeeper.On("Save",
		mock.MatchedBy(func(result backupResult) bool {
			return !result.Success &&
				result.Collection.collection == "collection1" &&
				result.Collection.database == "database1"
		})).Return(nil)

	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, mockedStatusKeeper)
	err := backupService.Backup(ctx, []dbColl{{"database1", "collection1"}})

	assert.Error(t, err, "Error was expected during backup.")
	assert.EqualError(t, err, "dumping failed for database1/collection1: couldn't dump db")
}

func TestBackup_ErrorOnSavingStatus(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Upload", mock.MatchedBy(isTestContext), mock.AnythingOfType("string"), "database1", "collection1", mock.AnythingOfType("*io.PipeReader")).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("SaveCollection", mock.MatchedBy(isTestContext), "database1", "collection1", mock.AnythingOfType("*main.snappyWriteCloser")).Return(nil)
	mockedStatusKeeper := new(mockStatusKeeper)
	mockedStatusKeeper.On("Save",
		mock.MatchedBy(func(result backupResult) bool {
			return result.Success &&
				result.Collection.collection == "collection1" &&
				result.Collection.database == "database1"
		})).Return(fmt.Errorf("couldn't save status of backup"))

	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, mockedStatusKeeper)
	err := backupService.Backup(ctx, []dbColl{{"database1", "collection1"}})

	assert.Error(t, err, "Error was expected during backup.")
	assert.EqualError(t, err, "couldn't save status of backup")
}

func TestRestore_OK(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Download", mock.MatchedBy(isTestContext), "2017-09-04T12-40-36", "database1", "collection1", mock.AnythingOfType("*io.PipeWriter")).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("RestoreCollection", mock.MatchedBy(isTestContext), "database1", "collection1", mock.AnythingOfType("*main.snappyReadCloser")).Return(nil)

	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)
	err := backupService.Restore(ctx, "2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})

	assert.NoError(t, err, "Error wasn't expected during backup.")
}

func TestRestore_ErrorOnRestore(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Download", mock.MatchedBy(isTestContext), "2017-09-04T12-40-36", "database1", "collection1", mock.AnythingOfType("*io.PipeWriter")).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("RestoreCollection", mock.MatchedBy(isTestContext), "database1", "collection1", mock.AnythingOfType("*main.snappyReadCloser")).Return(fmt.Errorf("error while restoring"))

	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)
	err := backupService.Restore(ctx, "2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})

	assert.Error(t, err)
	assert.EqualError(t, err, "error while restoring")
}

func isTestContext(ctx context.Context) bool {
	if value := ctx.Value("source"); value == "test" {
		return true
	}

	return false
}
