package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestSaveCollection_Ok(t *testing.T) {
	ctx := context.Background()
	stringWriter := bytes.NewBufferString("")
	mockedMongoSession := new(mockMongoSession)
	mockedMongoIter := new(mockMongoCur)
	mockedMongoSession.On("FindAll", ctx, "database1", "collection1").Return(mockedMongoIter, nil)
	mockedMongoIter.On("Next", ctx).Times(3).Return(true)
	mockedMongoIter.On("Current").Times(3).Return([]byte("data"))
	mockedMongoIter.On("Next", ctx).Return(false)
	mockedMongoIter.On("Err").Return(nil)
	mockedMongoIter.On("Close", ctx).Return(nil)

	mongoService := newMongoService(mockedMongoSession, nil, 250*time.Millisecond, 15000000)
	err := mongoService.SaveCollection(ctx, "database1", "collection1", stringWriter)

	assert.NoError(t, err, "Error wasn't expected during dump.")
	assert.Equal(t, "datadatadata", stringWriter.String())
}

func TestSaveCollection_WriterErr(t *testing.T) {
	ctx := context.Background()
	cappedStringWriter := newCappedBuffer(make([]byte, 0, 4), 11)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoIter := new(mockMongoCur)
	mockedMongoSession.On("FindAll", ctx, "database1", "collection1").Return(mockedMongoIter, nil)
	mockedMongoIter.On("Next", ctx).Return(true)
	mockedMongoIter.On("Current").Return([]byte("data"))
	mockedMongoIter.On("Err").Return(nil)
	mockedMongoIter.On("Close", ctx).Return(nil)

	mongoService := newMongoService(mockedMongoSession, nil, 250*time.Millisecond, 15000000)
	err := mongoService.SaveCollection(ctx, "database1", "collection1", cappedStringWriter)

	assert.Error(t, err, "Error expected during write.")
	assert.EqualError(t, err, "buffer overflow")
}

func TestSaveCollection_IterationErr(t *testing.T) {
	ctx := context.Background()
	stringWriter := bytes.NewBufferString("")
	mockedMongoSession := new(mockMongoSession)
	mockedMongoIter := new(mockMongoCur)
	mockedMongoSession.On("FindAll", ctx, "database1", "collection1").Return(mockedMongoIter, nil)
	mockedMongoIter.On("Next", ctx).Times(3).Return(true)
	mockedMongoIter.On("Current").Times(3).Return([]byte("data"))
	mockedMongoIter.On("Next", ctx).Return(false)
	mockedMongoIter.On("Err").Return(fmt.Errorf("iteration error"))
	mockedMongoIter.On("Close", ctx).Return(nil)

	mongoService := newMongoService(mockedMongoSession, nil, 250*time.Millisecond, 15000000)
	err := mongoService.SaveCollection(ctx, "database1", "collection1", stringWriter)

	assert.Error(t, err, "Error expected for iterator.")
	assert.Equal(t, "datadatadata", stringWriter.String())
	assert.EqualError(t, err, "error while iterating over collection=database1/collection1 noticed only at the end: iteration error")
}

func TestRestoreCollection_Ok(t *testing.T) {
	ctx := context.Background()
	mockedBsonService := new(mockBsonService)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoSession.On("RemoveAll", ctx, "database1", "collection1").Return(nil)
	model := mongo.NewInsertOneModel().SetDocument(bson.Raw("bson"))
	mockedMongoSession.On("BulkWrite", ctx, "database1", "collection1", []mongo.WriteModel{model, model, model}).Return(nil)
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(3).Return([]byte("bson"), nil)
	var end []byte
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Return(end, nil)

	mongoService := newMongoService(mockedMongoSession, mockedBsonService, 250*time.Millisecond, 15000000)
	err := mongoService.RestoreCollection(ctx, "database1", "collection1", strings.NewReader("nothing"))

	assert.NoError(t, err, "Error wasn't expected during restore.")
}

func TestRestoreCollection_ErrOnClean(t *testing.T) {
	ctx := context.Background()
	mockedBsonService := new(mockBsonService)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoSession.On("RemoveAll", ctx, "database1", "collection1").Return(fmt.Errorf("couldn't clean"))

	mongoService := newMongoService(mockedMongoSession, mockedBsonService, 250*time.Millisecond, 15000000)
	err := mongoService.RestoreCollection(ctx, "database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err, "Error was expected during restore.")
	assert.EqualError(t, err, "error while clearing collection=database1/collection1: couldn't clean")
}

func TestRestoreCollection_ErrOnRead(t *testing.T) {
	ctx := context.Background()
	mockedBsonService := new(mockBsonService)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoSession.On("RemoveAll", ctx, "database1", "collection1").Return(nil)
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(3).Return([]byte("bson"), nil)
	var end []byte
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Return(end, fmt.Errorf("error on read from unit test"))
	mongoService := newMongoService(mockedMongoSession, mockedBsonService, 250*time.Millisecond, 15000000)
	err := mongoService.RestoreCollection(ctx, "database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err, "Error was expected during restore.")
	assert.EqualError(t, err, "error while reading bson: error on read from unit test")
}

func TestRestoreCollection_ErrorOnWrite(t *testing.T) {
	ctx := context.Background()
	mockedBsonService := new(mockBsonService)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoSession.On("RemoveAll", ctx, "database1", "collection1").Return(nil)
	model := mongo.NewInsertOneModel().SetDocument(bson.Raw("bson"))
	mockedMongoSession.On("BulkWrite", ctx, "database1", "collection1", []mongo.WriteModel{model, model, model}).Return(fmt.Errorf("error writing to db from test"))
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(3).Return([]byte("bson"), nil)
	var end []byte
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Return(end, nil)

	mongoService := newMongoService(mockedMongoSession, mockedBsonService, 250*time.Millisecond, 15000000)
	err := mongoService.RestoreCollection(ctx, "database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err)
	assert.EqualError(t, err, "error while writing bulk: error writing to db from test")
}

func TestRestoreCollection_ErrorAfterOneBulkBatching(t *testing.T) {
	ctx := context.Background()
	mockedBsonService := new(mockBsonService)
	mockedMongoSession := new(mockMongoSession)
	mockedMongoSession.On("RemoveAll", ctx, "database1", "collection1").Return(nil)
	b := make([]byte, 0, 10000)
	for i := 0; i < 10000; i++ {
		b = append(b, 0)
	}
	mockedMongoSession.On("BulkWrite", ctx, "database1", "collection1", mock.Anything).Times(1).Return(nil)
	mockedMongoSession.On("BulkWrite", ctx, "database1", "collection1", mock.Anything).Times(1).Return(fmt.Errorf("error writing to db from test"))
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(1500).Return(b, nil)
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Times(1).Return([]byte{1}, nil)
	var end []byte
	mockedBsonService.On("ReadNextBSON", mock.MatchedBy(func(reader io.Reader) bool { return true })).Return(end, nil)

	mongoService := newMongoService(mockedMongoSession, mockedBsonService, 250*time.Millisecond, 15000000)
	err := mongoService.RestoreCollection(ctx, "database1", "collection1", strings.NewReader("nothing"))

	assert.Error(t, err)
	assert.EqualError(t, err, "error while writing bulk: error writing to db from test")
}
