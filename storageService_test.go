package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/errgroup"
)

func Test_errgroup(t *testing.T) {
	type Result string
	type Search func(ctx context.Context, query string) (Result, error)

	fakeSearch := func(kind string) Search {
		return func(ctx context.Context, query string) (Result, error) {
			timer := time.NewTimer(time.Second * 10)
			select {
			case <-ctx.Done():
				log.Printf("Canceled")
				return "", fmt.Errorf("canceled")
			case <-timer.C:
				return Result(fmt.Sprintf("%s result for %q", kind, query)), nil
			}
		}
	}

	failSearch := func(_ string) Search {
		return func(_ context.Context, query string) (Result, error) {
			log.Println("Fail")
			return "", fmt.Errorf("fail")
		}
	}

	Web := fakeSearch("web")
	Image := failSearch("image")
	Video := fakeSearch("video")

	Google := func(ctx context.Context, query string) ([]Result, error) {
		g, ctx := errgroup.WithContext(ctx)

		searches := []Search{Web, Image, Video}
		results := make([]Result, len(searches))
		for i, search := range searches {
			i, search := i, search // https://golang.org/doc/faq#closures_and_goroutines
			g.Go(func() error {
				result, err := search(ctx, query)
				if err == nil {
					results[i] = result
				}
				log.Printf("received error: %v", err)
				return err
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
		return results, nil
	}

	results, err := Google(context.Background(), "golang")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, result := range results {
		fmt.Println(result)
	}

}

func TestDownload_CanceledContext(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var b bytes.Buffer

	sess, err := session.NewSession(aws.NewConfig().WithRegion("test"))
	if err != nil {
		t.Fatalf("Creating AWS session failed with %e", err)
	}
	storage := newS3StorageService("test", "test", sess)
	if err = storage.Download(ctx, "", "", "", &b); err == nil {
		t.Fatalf("Expected error but got none")
	}
	log.Println(err)
}

func TestRestore1_ErrorDuringRestorationInRestore(t *testing.T) {
	start := time.Now()
	//nolint: staticcheck
	ctx := context.WithValue(context.Background(), "source", "test")

	sess, err := session.NewSession(aws.NewConfig().WithRegion("test"))
	if err != nil {
		t.Fatalf("Creating AWS session failed with %e", err)
	}
	storage := newS3StorageService("test", "test", sess)
	mockedMongoService := new(mockMongoService)
	mockedMongoService.On("RestoreCollection",
		mock.MatchedBy(isTestContext),
		"database1",
		"collection1",
		mock.AnythingOfType("*main.snappyReadCloser"),
	).Return(fmt.Errorf("error restoring collection"))

	backupService := newMongoBackupService(mockedMongoService, storage, nil)
	err = backupService.Restore(ctx, "2017-09-04T12-40-36", []dbColl{{"database1", "collection1"}})

	assert.True(t, time.Since(start) < time.Second*10, "all processes should end when error occurs")
	assert.Error(t, err)
	assert.EqualError(t, err, "error restoring collection")
}

//func TestDownload_PipeWithErrOnReadAndFullWrite(t *testing.T) {
//	r, w := newPipe(downloadOperation)
//
//	defer func() {
//		err := r.Close()
//		assert.NoError(t, err, "unexpected error while closing reader")
//	}()
//
//	g, ctx := errgroup.WithContext(context.Background())
//
//	g.Go(func() error {
//		defer func() {
//			err := w.Close()
//			assert.NoError(t, err, "unexpected error while closing writer")
//		}()
//
//		log.Infof("Timer started")
//		timer := time.NewTimer(time.Second * 10)
//
//		log.Infof("Creating empty byte array")
//		b := make([]byte, 0, 10000)
//		for i := 0; i < 10000; i++ {
//			b = append(b, 0)
//		}
//
//		log.Infof("Writing byte array in pipe")
//		_, err := w.Write(b)
//		assert.NoError(t, err, "unexpected error while writing")
//
//		log.Infof("Write finished")
//
//		select {
//		case <-ctx.Done():
//			log.Infof("context canceled")
//			return fmt.Errorf("context canceled")
//		case <-timer.C:
//			log.Infof("Time ran out")
//			return fmt.Errorf("time ran out")
//		}
//	})
//
//	g.Go(func() error {
//		log.Infof("Read failed simulated")
//		return fmt.Errorf("simple error for canceling context")
//	})
//
//	assert.Error(t, g.Wait(), "simple error for canceling context")
//}

func TestDownload_PipeWithErrOnReadAndFullWrite1(t *testing.T) {
	r, w := newPipe(downloadOperation)

	defer func() {
		err := r.Close()
		assert.NoError(t, err, "unexpected error while closing reader")
	}()

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		defer func() {
			err := w.Close()
			assert.NoError(t, err, "unexpected error while closing writer")
		}()

		log.Infof("Timer started")
		timer := time.NewTimer(time.Second * 10)

		log.Infof("Creating empty byte array")
		b := make([]byte, 0, 10000)
		for i := 0; i < 10000; i++ {
			b = append(b, 0)
		}

		log.Infof("Writing byte array in pipe")
		_, _ = w.Write(b)
		//assert.NoError(t, err, "unexpected error while writing")

		log.Infof("Write finished")

		select {
		case <-ctx.Done():
			log.Infof("context canceled")
			return fmt.Errorf("context canceled")
		case <-timer.C:
			log.Infof("Time ran out")
			return fmt.Errorf("time ran out")
		}
	})

	g.Go(func() error {
		log.Infof("Read fail simulated")
		return fmt.Errorf("simple error for canceling context")
	})

	g.Go(func() error {
		select {
		case <-ctx.Done():
			log.Infof("Error encountered, starting drain")
			r.Close()
			return nil
		}
	})

	assert.Error(t, g.Wait(), "simple error for canceling context")
}
