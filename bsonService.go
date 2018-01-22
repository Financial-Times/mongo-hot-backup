package main

import (
	"encoding/binary"
	"fmt"
	"io"
)

type bsonService interface {
	ReadNextBSON(reader io.Reader) ([]byte, error)
}

type defaultBsonService struct {
}

func (m *defaultBsonService) ReadNextBSON(reader io.Reader) ([]byte, error) {
	var lenBytes [4]byte

	_, err := io.ReadFull(reader, lenBytes[:])
	if err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("error reading (full) from buffer: %v", io.ErrUnexpectedEOF)
		}
		return nil, nil
	}

	docLen := int32(binary.LittleEndian.Uint32(lenBytes[:]))

	if docLen < 5 {
		return nil, fmt.Errorf("invalid document size: %v bytes", docLen)
	}

	buf := make([]byte, docLen)
	copy(buf, lenBytes[:])

	_, err = io.ReadAtLeast(reader, buf[4:], int(docLen-4))
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("error, this is a broken document: %v", io.ErrUnexpectedEOF)
		}
		return nil, fmt.Errorf("error reading (partial) from buffer: %v", io.ErrUnexpectedEOF)
	}
	return buf, nil
}
