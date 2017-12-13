package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/stretchr/testify/mock"
)

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
		return len(p), fmt.Errorf("buffer overflow")
	}
	b.mybuf.Write(p)
	return len(p), nil
}

func newCappedBuffer(buf []byte, cap int) *cappedBuffer {
	return &cappedBuffer{mybuf: bytes.NewBuffer(buf), cap: cap}
}

type mockBsonService struct {
	mock.Mock
}

func (m *mockBsonService) ReadNextBSON(reader io.Reader) ([]byte, error) {
	args := m.Called(reader)
	return args.Get(0).([]byte), args.Error(1)
}
