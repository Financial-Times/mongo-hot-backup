package main

import (
	"io"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/klauspost/compress/snappy"
	log "github.com/sirupsen/logrus"
)

const extension = ".bson.snappy"

type storageService interface {
	Writer(date, database, collection string) io.WriteCloser
	Reader(date, database, collection string) io.ReadCloser
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

func (s *s3StorageService) Writer(date, database, collection string) io.WriteCloser {
	path := filepath.Join(s.dir, date, database, collection+extension)

	logEntry := log.WithField("path", path)
	logEntry.Info("Uploading file...")

	reader, writer := io.Pipe()

	go func() {
		uploader := s3manager.NewUploader(s.session)

		_, err := uploader.Upload(&s3manager.UploadInput{
			Key:                  aws.String(path),
			Bucket:               aws.String(s.bucket),
			Body:                 reader,
			ServerSideEncryption: aws.String("AES256"),
		})
		if err != nil {
			logEntry.WithError(err).Error("Failed to upload file")

			_ = writer.Close()
			_ = reader.Close()
		}
	}()

	return newSnappyWriteCloser(writer)
}

func (s *s3StorageService) Reader(date, database, collection string) io.ReadCloser {
	path := filepath.Join(s.dir, date, database, collection+extension)

	logEntry := log.WithField("path", path)
	logEntry.Info("Downloading file...")

	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		downloader := s3manager.NewDownloader(s.session, func(d *s3manager.Downloader) {
			d.Concurrency = 1
		})

		_, err := downloader.Download(pipeWriterAt{writer}, &s3.GetObjectInput{
			Key:    aws.String(path),
			Bucket: aws.String(s.bucket),
		})
		if err != nil {
			logEntry.WithError(err).Error("Failed to download file")
		}
	}()

	return newSnappyReadCloser(reader)
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
