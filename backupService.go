package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	health "github.com/Financial-Times/go-fthealth/v1_1"
	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"github.com/klauspost/compress/snappy"
	"github.com/rlmcpherson/s3gof3r"
	"gopkg.in/robfig/cron.v2"
)

type backupService interface {
	Backup(dir, database, collection string) error
	BackupAll(colls []fullColl) error
	BackupScheduled(colls []fullColl, cronExpr string, dbPath string, run bool) error
	RestoreAll(dateDir string, colls []fullColl) error
	Restore(dir, database, collection string) error
}

type mongoBackupService struct {
	dbService        dbService
	connectionString string
	s3bucket         string
	s3dir            string
	s3               *s3gof3r.S3
}

type scheduledJob struct {
	eId  cron.EntryID
	coll fullColl
}

type scheduledJobResult struct {
	Success   bool
	Timestamp time.Time
}

func newMongoBackupService(dbService dbService, connectionString, s3bucket, s3dir, s3domain, accessKey, secretKey string) *mongoBackupService {
	return &mongoBackupService{
		dbService,
		connectionString,
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

func (m *mongoBackupService) BackupAll(colls []fullColl) error {
	dateDir := formattedNow()
	for _, coll := range colls {
		dir := filepath.Join(m.s3dir, dateDir)
		err := m.Backup(dir, coll.database, coll.collection)

		if err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoBackupService) BackupScheduled(colls []fullColl, cronExpr string, dbPath string, run bool) error {
	err := os.MkdirAll(filepath.Dir(dbPath), 0600)

	if err != nil {
		return err
	}
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Results"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	c := cron.New()

	var ids []scheduledJob

	for _, collection := range colls {

		coll := collection

		cronFunc := func() {
			dateDir := formattedNow()
			dir := filepath.Join(m.s3dir, dateDir)
			err := m.Backup(dir, coll.database, coll.collection)

			result := scheduledJobResult{true, time.Now()}

			if err != nil {
				log.Errorf("Error backing up %s/%s: %v\n", coll.database, coll.collection, err)
				result.Success = false
			}

			r, _ := json.Marshal(result)
			db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("Results"))
				err := b.Put([]byte(fmt.Sprintf("%s/%s", coll.database, coll.collection)), r)
				return err
			})
		}

		if run {
			go cronFunc()
		}

		eId, _ := c.AddFunc(cronExpr, func() { //now we add the cron methods

			cronFunc()

			for _, job := range ids {
				if job.coll.database == coll.database && job.coll.collection == coll.collection {
					// we find the current job on the list and report next scheduled run
					log.Printf("Next scheduled run for '%s/%s': %v\n", job.coll.database, job.coll.collection, c.Entry(job.eId).Next)
				}
			}
		})

		ids = append(ids, scheduledJob{eId, coll})
	}

	c.Start()

	for _, job := range ids {
		// on startup we report when the next run is expected
		log.Printf("Next scheduled run for '%s/%s': %v\n", job.coll.database, job.coll.collection, c.Entry(job.eId).Next)
	}

	healthService := newHealthService(db, colls, healthConfig{
		appSystemCode: "up-mgz",
		appName:       "mongobackup",
	})
	hc := health.HealthCheck{
		SystemCode:  healthService.config.appSystemCode,
		Name:        healthService.config.appName,
		Description: "Creates periodic backups of mongodb.",
		Checks:      healthService.checks,
	}
	r := mux.NewRouter()
	r.HandleFunc("/__health", http.HandlerFunc(health.Handler(hc)))
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8080", nil))

	return nil
}

func (m *mongoBackupService) Backup(dir, database, collection string) error {
	start := time.Now()
	log.Printf("backing up %s/%s to %s in %s\n", database, collection, dir, m.s3bucket)

	path := filepath.Join(dir, database, collection+extension)

	b := m.s3.Bucket(m.s3bucket)
	w, err := b.PutWriter(path, http.Header{"x-amz-server-side-encryption": []string{"AES256"}}, nil)
	if err != nil {
		return err
	}
	defer w.Close()

	sw := snappy.NewBufferedWriter(w)

	if err := m.dbService.DumpCollectionTo(m.connectionString, database, collection, sw); err != nil {
		return err
	}

	if err := sw.Close(); err != nil {
		return err
	}

	err = w.Close()
	log.Printf("backed up %s/%s to %s in %s. Duration : %v\n", database, collection, dir, m.s3bucket, time.Now().Sub(start))
	return err
}

func (m *mongoBackupService) RestoreAll(dateDir string, colls []fullColl) error {
	for _, coll := range colls {
		dir := filepath.Join(m.s3dir, dateDir)
		err := m.Restore(dir, coll.database, coll.collection)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoBackupService) Restore(dir, database, collection string) error {

	path := filepath.Join(dir, database, collection+extension)

	rc, _, err := m.s3.Bucket(m.s3bucket).GetReader(path, nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	sr := snappy.NewReader(rc)

	if err := m.dbService.RestoreCollectionFrom(m.connectionString, database, collection, sr); err != nil {
		return err
	}
	return nil
}

func formattedNow() string {
	return time.Now().UTC().Format(dateFormat)
}
