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
	mongoService := newMongoService(mockedMongoLib, nil)
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

	err := mongoService.DumpCollectionTo("127.0.0.1:27010,127.0.0.2:27010", "database1", "collection1", stringWriter)

	assert.NoError(t, err, "Error wasn't expected during dump.")
	assert.Equal(t, "datadatadata", stringWriter.String())
}

func TestDumpCollectionTo_SessionErr(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mongoService := newMongoService(mockedMongoLib, nil)
	stringWriter := bytes.NewBufferString("")
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(&labixSession{}, fmt.Errorf("oops"))

	err := mongoService.DumpCollectionTo("127.0.0.1:27010,127.0.0.2:27010", "database1", "collection1", stringWriter)

	assert.Error(t, err, "Error was expected during dial.")
	assert.Equal(t, "oops", err.Error())
}

func TestDumpCollectionTo_WriterErr(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mongoService := newMongoService(mockedMongoLib, nil)
	cappedStringWriter := newCappedBuffer(make([]byte, 0, 4), 11)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("DialWithTimeout", "127.0.0.1:27010,127.0.0.2:27010", 0*time.Millisecond).Return(mockedMongoSession, nil)
	mockedMongoSession.On("SetPrefetch", 1.0).Return()
	mockedMongoSession.On("Close").Return()
	mockedMongoIter := new(mockMongoIter)
	mockedMongoSession.On("SnapshotIter", "database1", "collection1", nil).Return(mockedMongoIter)
	mockedMongoIter.On("Next").Return([]byte("data"), true)
	mockedMongoIter.On("Err").Return(nil)

	err := mongoService.DumpCollectionTo("127.0.0.1:27010,127.0.0.2:27010", "database1", "collection1", cappedStringWriter)

	assert.Error(t, err, "Error expected during write.")
	assert.Equal(t, "buffer overflow", err.Error())
}

func TestDumpCollectionTo_IterationErr(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mongoService := newMongoService(mockedMongoLib, nil)
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

	err := mongoService.DumpCollectionTo("127.0.0.1:27010,127.0.0.2:27010", "database1", "collection1", stringWriter)

	assert.Error(t, err, "Error expected for iterator.")
	assert.Equal(t, "datadatadata", stringWriter.String())
	assert.Equal(t, "iteration error", err.Error())
}

func TestRestoreCollectionFrom_Ok(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mockedBsonService := new(mockBsonService)
	mongoService := newMongoService(mockedMongoLib, mockedBsonService)
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

	err := mongoService.RestoreCollectionFrom("127.0.0.1:27010,127.0.0.2:27010", "database1", "collection1", strings.NewReader("nothing"))

	assert.NoError(t, err, "Error wasn't expected during restore.")
	assert.Equal(t, []byte("bsonbsonbson"), insertedData)
}

type mockMongoLib struct {
	mock.Mock
}

func (m *mockMongoLib) DialWithTimeout(url string, timeout time.Duration) (mongoSession, error) {
	args := m.Called(url, timeout)
	return args.Get(0).(mongoSession), args.Error(1)
}

type mockMongoSession struct {
	mock.Mock
}

func (m *mockMongoSession) SnapshotIter(database, collection string, findQuery interface{}) mongoIter {
	args := m.Called(database, collection, findQuery)
	return args.Get(0).(mongoIter)
}

func (m *mockMongoSession) SetPrefetch(p float64) {
	m.Called(p)
}

func (m *mockMongoSession) Close() {
	m.Called()
}

func (m *mockMongoSession) Bulk(database, collection string) mongoBulk {
	args := m.Called(database, collection)
	return args.Get(0).(mongoBulk)
}

func (m *mockMongoSession) RemoveAll(database, collection string, removeQuery interface{}) error {
	args := m.Called(database, collection, removeQuery)
	return args.Error(0)
}

type mockMongoIter struct {
	mock.Mock
}

func (m *mockMongoIter) Next() ([]byte, bool) {
	args := m.Called()
	return args.Get(0).([]byte), args.Bool(1)
}

func (m *mockMongoIter) Err() error {
	args := m.Called()
	return args.Error(0)
}

type mockMongoBulk struct {
	mock.Mock
}

func (m *mockMongoBulk) Run() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockMongoBulk) Insert(data []byte) {
	m.Called(data)
}

type cappedBuffer struct {
	cap   int
	mybuf *bytes.Buffer
}

func (b *cappedBuffer) Write(p []byte) (n int, err error) {
	if len(p)+b.mybuf.Len() > b.cap {
		fmt.Printf(b.mybuf.String())
		return len(p), fmt.Errorf("buffer overflow")
	}
	b.mybuf.Write(p)
	return len(p), nil
}

func newCappedBuffer(buf []byte, cap int) *cappedBuffer {
	return &cappedBuffer{mybuf: bytes.NewBuffer(buf), cap: cap}
}
