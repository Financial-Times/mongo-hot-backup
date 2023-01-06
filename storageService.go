package main

import (
	"context"
	"io"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/klauspost/compress/snappy"
	log "github.com/sirupsen/logrus"
)

type operation int

const (
	uploadOperation operation = iota
	downloadOperation
)

type storageService interface {
	Upload(ctx context.Context, date, database, collection string, reader io.Reader) error
	Download(ctx context.Context, date, database, collection string, writer io.Writer) error
}

type s3StorageService struct {
	bucket  string
	dir     string
	session *session.Session
}

func newS3StorageService(bucket, dir string, session *session.Session) *s3StorageService {
	return &s3StorageService{
		bucket:  bucket,
		dir:     dir,
		session: session,
	}
}

func (s *s3StorageService) getFilePath(date, database, collection string) string {
	const extension = ".bson.snappy"

	return filepath.Join(s.dir, date, database, collection+extension)
}

func (s *s3StorageService) Upload(ctx context.Context, date, database, collection string, reader io.Reader) error {
	path := s.getFilePath(date, database, collection)

	uploader := s3manager.NewUploader(s.session)

	_, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Key:                  aws.String(path),
		Bucket:               aws.String(s.bucket),
		Body:                 reader,
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

func (s *s3StorageService) Download(ctx context.Context, date, database, collection string, writer io.Writer) error {
	path := s.getFilePath(date, database, collection)

	downloader := s3manager.NewDownloader(s.session, func(d *s3manager.Downloader) {
		d.Concurrency = 1
	})

	log.Infof("Starting download: %s", collection)
	_, err := downloader.DownloadWithContext(ctx, pipeWriterAt{writer}, &s3.GetObjectInput{
		Key:    aws.String(path),
		Bucket: aws.String(s.bucket),
	})
	log.Infof("Ending download: %s", collection)

	return err
}

func newPipe(op operation) (io.ReadCloser, io.WriteCloser) {
	reader, writer := io.Pipe()

	if op == uploadOperation {
		return reader, newSnappyWriteCloser(writer)
	}

	return newSnappyReadCloser(reader), writer
}

type snappyWriteCloser struct {
	snappyWriter *snappy.Writer
	writeCloser  io.WriteCloser
}

func newSnappyWriteCloser(writeCloser io.WriteCloser) *snappyWriteCloser {
	return &snappyWriteCloser{
		snappyWriter: snappy.NewBufferedWriter(writeCloser),
		writeCloser:  writeCloser,
	}
}

func (swc *snappyWriteCloser) Write(p []byte) (nRet int, errRet error) {
	return swc.snappyWriter.Write(p)
}

func (swc *snappyWriteCloser) Close() error {
	if err := swc.snappyWriter.Close(); err != nil {
		return err
	}
	return swc.writeCloser.Close()
}

type snappyReadCloser struct {
	snappyReader *snappy.Reader
	readCloser   io.ReadCloser
}

func newSnappyReadCloser(readCloser io.ReadCloser) *snappyReadCloser {
	return &snappyReadCloser{
		snappyReader: snappy.NewReader(readCloser),
		readCloser:   readCloser,
	}
}

func (src *snappyReadCloser) Read(p []byte) (int, error) {
	return src.snappyReader.Read(p)
}

func (src *snappyReadCloser) Close() error {
	return src.readCloser.Close()
}

type pipeWriterAt struct {
	w io.Writer
}

func (pw pipeWriterAt) WriteAt(p []byte, offset int64) (n int, err error) {
	return pw.w.Write(p)
}
