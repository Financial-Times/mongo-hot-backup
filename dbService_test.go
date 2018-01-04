package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDumpCollectionTo_Ok(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, nil, 0, 250*time.Millisecond)
	stringWriter := bytes.NewBufferString("")
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("SetPrefetch", 1.0).Return()
	mockedMongoSession.On("Close").Return()
	mockedMongoIter := new(mockMongoIter)
	mockedMongoSession.On("SnapshotIter", "database1", "collection1", nil).Return(mockedMongoIter)
	mockedMongoIter.On("Next").Times(3).Return([]byte("data"), true)
	mockedMongoIter.On("Next").Return([]byte{}, false)
	mockedMongoIter.On("Err").Return(nil)

	err := mongoService.DumpCollectionTo("database1", "collection1", stringWriter)

	assert.NoError(t, err, "Error wasn't expected during dump.")
	assert.Equal(t, "datadatadata", stringWriter.String())
}

func TestDumpCollectionTo_SessionErr(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, nil, 0, 250*time.Millisecond)
	stringWriter := bytes.NewBufferString("")
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(&labixSession{}, fmt.Errorf("oops"))

	err := mongoService.DumpCollectionTo("database1", "collection1", stringWriter)

	assert.Error(t, err, "Error was expected during dial.")
	assert.Equal(t, "oops", err.Error())
}

func TestDumpCollectionTo_WriterErr(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, nil, 0, 250*time.Millisecond)
	cappedStringWriter := newCappedBuffer(make([]byte, 0, 4), 11)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("SetPrefetch", 1.0).Return()
	mockedMongoSession.On("Close").Return()
	mockedMongoIter := new(mockMongoIter)
	mockedMongoSession.On("SnapshotIter", "database1", "collection1", nil).Return(mockedMongoIter)
	mockedMongoIter.On("Next").Return([]byte("data"), true)
	mockedMongoIter.On("Err").Return(nil)

	err := mongoService.DumpCollectionTo("database1", "collection1", cappedStringWriter)

	assert.Error(t, err, "Error expected during write.")
	assert.Equal(t, "buffer overflow", err.Error())
}

func TestDumpCollectionTo_IterationErr(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, nil, 0, 250*time.Millisecond)
	stringWriter := bytes.NewBufferString("")
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("SetPrefetch", 1.0).Return()
	mockedMongoSession.On("Close").Return()
	mockedMongoIter := new(mockMongoIter)
	mockedMongoSession.On("SnapshotIter", "database1", "collection1", nil).Return(mockedMongoIter)
	mockedMongoIter.On("Next").Times(3).Return([]byte("data"), true)
	mockedMongoIter.On("Next").Return([]byte{}, false)
	mockedMongoIter.On("Err").Return(fmt.Errorf("iteration error"))

	err := mongoService.DumpCollectionTo("database1", "collection1", stringWriter)

	assert.Error(t, err, "Error expected for iterator.")
	assert.Equal(t, "datadatadata", stringWriter.String())
	assert.Equal(t, "iteration error", err.Error())
}

func TestRestoreCollectionFrom_Ok(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mockedBsonService := new(mockBsonService)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, mockedBsonService, 0, 250*time.Millisecond)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("RemoveAll", "database1", "collection1", nil).Return(nil)
	mockedMongoSession.On("Close").Return()
	mockedMongoBulk := new(mockMongoBulk)
	mockedMongoSession.On("Bulk", "database1", "collection1").Return(mockedMongoBulk)
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(3).Return([]byte("bson"), nil)
	var end []byte
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Return(end, nil)
	insertedData := make([]byte, 0, 8)
	mockedMongoBulk.On("Insert", []byte("bson")).Return().Run(func(args mock.Arguments) {
		insertedData = append(insertedData, args.Get(0).([]byte)...)
	})
	mockedMongoBulk.On("Run").Return(nil)

	err := mongoService.RestoreCollectionFrom("database1", "collection1", strings.NewReader("nothing"))

	assert.NoError(t, err, "Error wasn't expected during restore.")
	assert.Equal(t, []byte("bsonbsonbson"), insertedData)
}

func TestRestoreCollectionFrom_DialErr(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mockedBsonService := new(mockBsonService)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, mockedBsonService, 0, 250*time.Millisecond)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, fmt.Errorf("couldn't dial"))

	err := mongoService.RestoreCollectionFrom("database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err, "Error was expected during restore.")
	assert.Equal(t, "couldn't dial", err.Error())
}

func TestRestoreCollectionFrom_ErrOnClean(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mockedBsonService := new(mockBsonService)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, mockedBsonService, 0, 250*time.Millisecond)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("RemoveAll", "database1", "collection1", nil).Return(fmt.Errorf("couldn't clean"))
	mockedMongoSession.On("Close").Return()

	err := mongoService.RestoreCollectionFrom("database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err, "Error was expected during restore.")
	assert.Equal(t, "couldn't clean", err.Error())
}

func TestRestoreCollectionFrom_ErrOnRead(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mockedBsonService := new(mockBsonService)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, mockedBsonService, 0, 250*time.Millisecond)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("RemoveAll", "database1", "collection1", nil).Return(nil)
	mockedMongoSession.On("Close").Return()
	mockedMongoBulk := new(mockMongoBulk)
	mockedMongoSession.On("Bulk", "database1", "collection1").Return(mockedMongoBulk)
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(3).Return([]byte("bson"), nil)
	var end []byte
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Return(end, fmt.Errorf("error on read from unit test"))
	mockedMongoBulk.On("Insert", []byte("bson")).Return()
	mockedMongoBulk.On("Run").Return(nil)

	err := mongoService.RestoreCollectionFrom("database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err, "Error was expected during restore.")
	assert.Equal(t, "error on read from unit test", err.Error())
}

func TestRestoreCollectionFrom_ErrorOnWrite(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mockedBsonService := new(mockBsonService)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, mockedBsonService, 0, 250*time.Millisecond)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("RemoveAll", "database1", "collection1", nil).Return(nil)
	mockedMongoSession.On("Close").Return()
	mockedMongoBulk := new(mockMongoBulk)
	mockedMongoSession.On("Bulk", "database1", "collection1").Return(mockedMongoBulk)
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(3).Return([]byte("bson"), nil)
	var end []byte
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Return(end, nil)
	mockedMongoBulk.On("Insert", []byte("bson")).Return()
	mockedMongoBulk.On("Run").Return(fmt.Errorf("error writing to db from test"))

	err := mongoService.RestoreCollectionFrom("database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err, "error writing to db from test")
}

func TestRestoreCollectionFrom_ErrorAfterOneBulkBatching(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mockedBsonService := new(mockBsonService)
	mongoService := newMongoService("127.0.0.1:27010,127.0.0.2:27010", mockedMongoLib, mockedBsonService, 0, 250*time.Millisecond)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("RemoveAll", "database1", "collection1", nil).Return(nil)
	mockedMongoSession.On("Close").Return()
	mockedMongoBulk := new(mockMongoBulk)
	mockedMongoSession.On("Bulk", "database1", "collection1").Return(mockedMongoBulk)
	b := make([]byte, 0, 10000)
	for i := 0; i < 10000; i++ {
		b = append(b, 0)
	}
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(1500).Return(b, nil)
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(1).Return([]byte{1}, nil)
	var end []byte
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Return(end, nil)
	mockedMongoBulk.On("Insert", b).Return()
	mockedMongoBulk.On("Insert", []byte{1}).Return()
	mockedMongoBulk.On("Run").Times(1).Return(nil)
	mockedMongoBulk.On("Run").Return(fmt.Errorf("error writing to db from test"))

	err := mongoService.RestoreCollectionFrom("database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err, "error writing to db from test")
}
