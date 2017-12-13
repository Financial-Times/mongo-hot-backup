package main

// import (
// 	"io"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// func TestBackup_Ok(t *testing.T) {
// 	mockedMongoService := new(mockMongoService)
// 	// stringWriter := bytes.NewBufferString("")
// 	mockedMongoService.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)

// 	backupService := newMongoBackupService(mockedMongoService, "127.0.0.1:27010,127.0.0.2:27010", "s3bucket", "s3dir", "s3domain", "accessKey", "secretKey")

// 	err := backupService.Backup("s3dir", "database1", "collection1")

// 	assert.NoError(t, err, "Error wasn't expected during backup.")
// 	// assert.Equal(t, "datadatadata", stringWriter.String())
// }

// type mockMongoService struct {
// 	mock.Mock
// }

// func (m *mockMongoService) DumpCollectionTo(connStr, database, collection string, writer io.Writer) error {
// 	args := m.Called(connStr, database, collection, writer)
// 	return args.Error(0)
// }

// func (m *mockMongoService) RestoreCollectionFrom(connStr, database, collection string, reader io.Reader) error {
// 	args := m.Called(connStr, database, collection, reader)
// 	return args.Error(0)
// }
