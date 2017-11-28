package main

import (
	"testing"
	"bytes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/assert"
)

func TestDumpCollectionTo_Ok(t *testing.T) {
	mockedMongoLib := new(mockMongoLib)
	mongoService := newMongoService(mockedMongoLib)
	stringWriter := bytes.NewBufferString("")
	mockedMongoSession := new(mockMongoSession)
	mockedMongoLib.On("Dial", "127.0.0.1:27010,127.0.0.2:27010").Return(mockedMongoSession, nil)
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

type mockMongoLib struct {
	mock.Mock
}

func (m *mockMongoLib) Dial(url string) (mongoSession, error) {
	args := m.Called(url)
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
