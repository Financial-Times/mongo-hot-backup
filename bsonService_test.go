package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRead_OK(t *testing.T) {
	bsonService := defaultBsonService{}

	bson := "\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00"

	result, err := bsonService.ReadNextBSON(strings.NewReader(bson))

	assert.NoError(t, err, "Error wasn't expected during read.")
	assert.Equal(t, []byte(bson), result)
}

func TestRead_ErrorTooShort(t *testing.T) {
	bsonService := defaultBsonService{}

	_, err := bsonService.ReadNextBSON(strings.NewReader("\x04"))

	assert.Error(t, err, "Error was expected for being to short.")
}

func TestRead_ErrorUnexpectedEOF(t *testing.T) {
	bsonService := defaultBsonService{}

	_, err := bsonService.ReadNextBSON(strings.NewReader("\x16\x00\x00\x00\x02hel-\x00\x00"))

	assert.Error(t, err, "Error was expected for broken doc.")
	assert.Equal(t, "error reading (partial) from buffer: unexpected EOF", err.Error())
}
