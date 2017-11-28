package main

import "testing"

func TestBackup_Ok(t *testing.T) {

}

type mockMessageToNativeMapper struct {
	mock.Mock
}

func (m *mockMessageToNativeMapper) Map(source []byte) (NativeContent, error) {
	args := m.Called(source)
	return args.Get(0).(NativeContent), args.Error(1)
}
