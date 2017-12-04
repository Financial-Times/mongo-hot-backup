package main

import (
	"io"

	"github.com/stretchr/testify/mock"
)

type mockBsonService struct {
	mock.Mock
}

func (m *mockBsonService) ReadNextBSON(reader io.Reader) ([]byte, error) {
	args := m.Called(reader)
	return args.Get(0).([]byte), args.Error(1)
}
