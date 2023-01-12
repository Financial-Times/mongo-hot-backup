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
	mockedStorageService.On("Upload",
		mock.MatchedBy(isTestContext),
		mock.AnythingOfType("string"),
		"database1",
		"collection1",
		mock.AnythingOfType("*io.PipeReader"),
	).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("SaveCollection",
		mock.MatchedBy(isTestContext),
		"database1",
		"collection1",
		mock.AnythingOfType("*main.snappyWriteCloser"),
	).Return(nil)
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

func TestBackup_ErrorOnSavingCollection(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Upload",
		mock.MatchedBy(isTestContext),
		mock.AnythingOfType("string"),
		"database1",
		"collection1",
		mock.AnythingOfType("*io.PipeReader")).
		Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("SaveCollection",
		mock.MatchedBy(isTestContext),
		"database1",
		"collection1",
		mock.AnythingOfType("*main.snappyWriteCloser")).
		Return(fmt.Errorf("error saving collection"))
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
	assert.EqualError(t, err, "dumping failed for database1/collection1: error saving collection")
}

func TestBackup_ErrorOnUploadingCollection(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Upload",
		mock.MatchedBy(isTestContext),
		mock.AnythingOfType("string"),
		"database1",
		"collection1",
		mock.AnythingOfType("*io.PipeReader")).
		Return(fmt.Errorf("error uploading collection"))
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("SaveCollection",
		mock.MatchedBy(isTestContext),
		"database1",
		"collection1",
		mock.AnythingOfType("*main.snappyWriteCloser")).
		Return(nil)
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
	assert.EqualError(t, err, "dumping failed for database1/collection1: error uploading collection")
}

func TestBackup_ErrorOnSavingStatus(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Upload",
		mock.MatchedBy(isTestContext),
		mock.AnythingOfType("string"),
		"database1",
		"collection1",
		mock.AnythingOfType("*io.PipeReader"),
	).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("SaveCollection",
		mock.MatchedBy(isTestContext),
		"database1",
		"collection1",
		mock.AnythingOfType("*main.snappyWriteCloser"),
	).Return(nil)
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

//func TestBackup_ErrorOnlyInUpload(t *testing.T) {
//	start := time.Now()
//	ctx := context.WithValue(context.Background(), "source", "test")
//	mockedStorageService := new(mockStorageService)
//	mockedStorageService.On("Upload",
//		mock.MatchedBy(isTestContext),
//		mock.AnythingOfType("string"),
//		"database1",
//		"collection1",
//		mock.AnythingOfType("*io.PipeReader")).
//		Return(fmt.Errorf("error uploading collection"))
//	mockedMongoService := new(mockMongoService)
//	mockedMongoService.On("SaveCollection",
//		mock.MatchedBy(isTestContext),
//		"database1",
//		"collection1",
//		mock.AnythingOfType("*main.snappyWriteCloser")).
//		After(time.Second * 10).
//		Return(nil)
//	mockedStatusKeeper := new(mockStatusKeeper)
//	mockedStatusKeeper.On("Save",
//		mock.MatchedBy(func(result backupResult) bool {
//			return !result.Success &&
//				result.Collection.collection == "collection1" &&
//				result.Collection.database == "database1"
//		})).Return(nil)
//
//	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, mockedStatusKeeper)
//	err := backupService.Backup(ctx, []dbColl{{"database1", "collection1"}})
//
//	assert.True(t, time.Since(start) < time.Second*10, "all processes should end when error occurs")
//	assert.Error(t, err, "Error was expected during backup.")
//	assert.EqualError(t, err, "dumping failed for database1/collection1: error uploading collection")
//}

//func TestBackup_ErrorOnlyInSave(t *testing.T) {
//	start := time.Now()
//	ctx := context.WithValue(context.Background(), "source", "test")
//	mockedStorageService := new(mockStorageService)
//	mockedStorageService.On("Upload",
//		mock.MatchedBy(isTestContext),
//		mock.AnythingOfType("string"),
//		"database1",
//		"collection1",
//		mock.AnythingOfType("*io.PipeReader")).
//		Return(nil)
//	mockedMongoService := new(mockMongoService)
//	mockedMongoService.On("SaveCollection",
//		mock.MatchedBy(isTestContext),
//		"database1",
//		"collection1",
//		mock.AnythingOfType("*main.snappyWriteCloser")).
//		Return(fmt.Errorf("error saving collection"))
//	mockedStatusKeeper := new(mockStatusKeeper)
//	mockedStatusKeeper.On("Save",
//		mock.MatchedBy(func(result backupResult) bool {
//			return !result.Success &&
//				result.Collection.collection == "collection1" &&
//				result.Collection.database == "database1"
//		})).Return(nil)
//
//	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, mockedStatusKeeper)
//	err := backupService.Backup(ctx, []dbColl{{"database1", "collection1"}})
//
//	assert.True(t, time.Since(start) < time.Second*10, "all processes should end when error occurs")
//	assert.Error(t, err, "Error was expected during backup.")
//	assert.EqualError(t, err, "dumping failed for database1/collection1: error saving collection")
//}

func TestRestore_OK(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Download",
		mock.MatchedBy(isTestContext),
		"2017-09-04T12-40-36",
		"database1",
		"collection1",
		mock.AnythingOfType("*io.PipeWriter"),
	).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("RestoreCollection",
		mock.MatchedBy(isTestContext),
		"database1",
		"collection1",
		mock.AnythingOfType("*main.snappyReadCloser"),
	).Return(nil)

	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)
	err := backupService.Restore(ctx, "2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})

	assert.NoError(t, err, "Error wasn't expected during backup.")
}

func TestRestore_ErrorOnRestoringCollection(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Download",
		mock.MatchedBy(isTestContext),
		"2017-09-04T12-40-36",
		"database1",
		"collection1",
		mock.AnythingOfType("*io.PipeWriter"),
	).Return(nil)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("RestoreCollection",
		mock.MatchedBy(isTestContext),
		"database1",
		"collection1",
		mock.AnythingOfType("*main.snappyReadCloser"),
	).Return(fmt.Errorf("error restoring collection"))

	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)
	err := backupService.Restore(ctx, "2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})

	assert.Error(t, err)
	assert.EqualError(t, err, "error restoring collection")
}

//func TestRestore_ErrorDuringRestorationInRestore(t *testing.T) {
//	start := time.Now()
//	ctx := context.WithValue(context.Background(), "source", "test")
//	mockedStorageService := new(mockStorageService)
//	mockedStorageService.On("Download",
//		mock.MatchedBy(isTestContext),
//		"2017-09-04T12-40-36",
//		"database1",
//		"collection1",
//		mock.AnythingOfType("*io.PipeWriter"),
//	).After(time.Second * 10).Return(nil)
//	mockedMongoService := new(mockMongoService)
//	mockedMongoService.On("RestoreCollection",
//		mock.MatchedBy(isTestContext),
//		"database1",
//		"collection1",
//		mock.AnythingOfType("*main.snappyReadCloser"),
//	).Return(fmt.Errorf("error restoring collection"))
//
//	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)
//	err := backupService.Restore(ctx, "2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})
//
//	assert.True(t, time.Since(start) < time.Second*10, "all processes should end when error occurs")
//	assert.Error(t, err)
//	assert.EqualError(t, err, "error restoring collection")
//}

//func TestRestore_ErrorDuringRestorationInDownload(t *testing.T) {
//	start := time.Now()
//	ctx := context.WithValue(context.Background(), "source", "test")
//	mockedStorageService := new(mockStorageService)
//	mockedStorageService.On("Download",
//		mock.MatchedBy(isTestContext),
//		"2017-09-04T12-40-36",
//		"database1",
//		"collection1",
//		mock.AnythingOfType("*io.PipeWriter"),
//	).Return(fmt.Errorf("error downloading backup"))
//	mockedMongoService := new(mockMongoService)
//	mockedMongoService.On("RestoreCollection",
//		mock.MatchedBy(isTestContext),
//		"database1",
//		"collection1",
//		mock.AnythingOfType("*main.snappyReadCloser"),
//	).After(time.Second * 10).Return(nil)
//
//	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)
//	err := backupService.Restore(ctx, "2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})
//
//	assert.True(t, time.Since(start) < time.Second*10, "all processes should end when error occurs")
//	assert.Error(t, err)
//	assert.EqualError(t, err, "error downloading backup")
//}

func TestRestore_ErrorOnDownloadingCollection(t *testing.T) {
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")
	mockedStorageService := new(mockStorageService)
	mockedStorageService.On("Download",
		mock.MatchedBy(isTestContext),
		"2017-09-04T12-40-36",
		"database1",
		"collection1",
		mock.AnythingOfType("*io.PipeWriter"),
	).Return(fmt.Errorf("error downloading collection"))
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("RestoreCollection",
		mock.MatchedBy(isTestContext),
		"database1",
		"collection1",
		mock.AnythingOfType("*main.snappyReadCloser"),
	).Return(nil)

	backupService := newMongoBackupService(mockedMongoService, mockedStorageService, nil)
	err := backupService.Restore(ctx, "2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})

	assert.Error(t, err)
	assert.EqualError(t, err, "error downloading collection")
}

func isTestContext(ctx context.Context) bool {
	if value := ctx.Value("source"); value == "test" {
		return true
	}

	return false
}
