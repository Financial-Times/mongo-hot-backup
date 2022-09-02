package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
)

type mockMongoSession struct {
	mock.Mock
}

func (m *mockMongoSession) FindAll(ctx context.Context, database, collection string) (mongoCursor, error) {
	args := m.Called(ctx, database, collection)
	return args.Get(0).(mongoCursor), args.Error(1)
}

func (m *mockMongoSession) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockMongoSession) BulkWrite(ctx context.Context, database, collection string, models []mongo.WriteModel) error {
	args := m.Called(ctx, database, collection, models)
	return args.Error(0)
}

func (m *mockMongoSession) RemoveAll(ctx context.Context, database, collection string) error {
	args := m.Called(ctx, database, collection)
	return args.Error(0)
}

type mockMongoCur struct {
	mock.Mock
}

func (m *mockMongoCur) Next(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *mockMongoCur) Current() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *mockMongoCur) Err() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockMongoCur) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type cappedBuffer struct {
	cap int
	buf *bytes.Buffer
}

func (b *cappedBuffer) Write(p []byte) (n int, err error) {
	if len(p)+b.buf.Len() > b.cap {
		return len(p), fmt.Errorf("buffer overflow")
	}
	b.buf.Write(p)
	return len(p), nil
}

func newCappedBuffer(buf []byte, cap int) *cappedBuffer {
	return &cappedBuffer{buf: bytes.NewBuffer(buf), cap: cap}
}

type mockBsonService struct {
	mock.Mock
}

func (m *mockBsonService) ReadNextBSON(reader io.Reader) ([]byte, error) {
	args := m.Called(reader)
	return args.Get(0).([]byte), args.Error(1)
}

type mockMongoService struct {
	mock.Mock
}

func (m *mockMongoService) SaveCollection(ctx context.Context, database, collection string, writer io.Writer) error {
	args := m.Called(ctx, database, collection, writer)
	return args.Error(0)
}

func (m *mockMongoService) RestoreCollection(ctx context.Context, database, collection string, reader io.Reader) error {
	args := m.Called(ctx, database, collection, reader)
	return args.Error(0)
}

type mockStorageService struct {
	mock.Mock
}

func (m *mockStorageService) Upload(date, database, collection string, reader io.Reader) error {
	args := m.Called(date, database, collection, reader)
	return args.Error(0)
}

func (m *mockStorageService) Download(date, database, collection string, writer io.Writer) error {
	args := m.Called(date, database, collection, writer)
	return args.Error(0)
}

type mockStatusKeeper struct {
	mock.Mock
}

func (m *mockStatusKeeper) Save(result backupResult) error {
	args := m.Called(result)
	return args.Error(0)
}

func (m *mockStatusKeeper) Get(coll dbColl) (backupResult, error) {
	args := m.Called(coll)
	return args.Get(0).(backupResult), args.Error(1)
}

func (m *mockStatusKeeper) Close() error {
	args := m.Called()
	return args.Error(0)
}
