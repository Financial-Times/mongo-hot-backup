package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
