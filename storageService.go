package main

import (
	"io"
	"net/http"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/klauspost/compress/snappy"
	"github.com/rlmcpherson/s3gof3r"
)

const extension = ".bson.snappy"

type storageService interface {
	Writer(date, database, collection string) (io.WriteCloser, error)
	Reader(date, database, collection string) (io.ReadCloser, error)
}

type s3StorageService struct {
	s3bucket string
	s3dir    string
	s3       *s3gof3r.S3
}

func newS3StorageService(s3bucket, s3dir, s3domain, accessKey, secretKey string) *s3StorageService {
	return &s3StorageService{
		s3bucket,
		s3dir,
		s3gof3r.New(
			s3domain,
			s3gof3r.Keys{
				AccessKey: accessKey,
				SecretKey: secretKey,
			},
		),
	}
}

func (s *s3StorageService) Writer(date, database, collection string) (io.WriteCloser, error) {
	path := filepath.Join(s.s3dir, date, database, collection+extension)
	log.Infof("saving to path=%s bucket=%s", path, s.s3bucket)
	b := s.s3.Bucket(s.s3bucket)
	w, err := b.PutWriter(path, http.Header{"x-amz-server-side-encryption": []string{"AES256"}}, nil)
	if err != nil {
		return nil, err
	}
	return snappy.NewBufferedWriter(w), nil
}

func (s *s3StorageService) Reader(date, database, collection string) (io.ReadCloser, error) {
	path := filepath.Join(s.s3dir, date, database, collection+extension)

	rc, _, err := s.s3.Bucket(s.s3bucket).GetReader(path, nil)
	if err != nil {
		return nil, err
	}

	return newSnappyReadCloser(snappy.NewReader(rc), rc), nil
}

type snappyReadCloser struct {
	snappyReader *snappy.Reader
	readCloser   io.ReadCloser
}

func newSnappyReadCloser(snappyReader *snappy.Reader, readCloser io.ReadCloser) *snappyReadCloser {
	return &snappyReadCloser{
		snappyReader,
		readCloser,
	}
}

func (src *snappyReadCloser) Read(p []byte) (int, error) {
	return src.snappyReader.Read(p)
}

func (src *snappyReadCloser) Close() error {
	return src.readCloser.Close()
}
