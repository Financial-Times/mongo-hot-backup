package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/jawher/mow.cli"
	"github.com/klauspost/compress/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rlmcpherson/s3gof3r"
	"github.com/utilitywarehouse/go-operational/op"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	cron "gopkg.in/robfig/cron.v2"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const extension = ".bson.snappy"

func main() {

	s3gof3r.DefaultConfig.Md5Check = false

	app := cli.App("mongolizer", "Backup and restore mongodb collections to/from s3\nBackups are put in a directory structure /<base-dir>/<date>/database/collection")

	connStr := app.String(cli.StringOpt{
		Name:   "mongodb",
		Desc:   "mongodb connection string",
		EnvVar: "MONGODB",
		Value:  "localhost:27017",
	})

	s3domain := app.String(cli.StringOpt{
		Name:   "s3domain",
		Desc:   "s3 domain",
		EnvVar: "S3_DOMAIN",
		Value:  "s3-eu-west-1.amazonaws.com",
	})

	s3bucket := app.String(cli.StringOpt{
		Name:   "bucket",
		Desc:   "s3 bucket name",
		EnvVar: "S3_BUCKET",
		Value:  "com.ft.coco-mongo-backup.prod",
	})

	s3dir := app.String(cli.StringOpt{
		Name:   "base-dir",
		Desc:   "s3 base directory name",
		EnvVar: "S3_DIR",
		Value:  "/backups/",
	})

	accessKey := app.String(cli.StringOpt{
		Name:   "aws_access_key_id",
		Desc:   "AWS Access key id",
		EnvVar: "AWS_ACCESS_KEY_ID",
	})

	secretKey := app.String(cli.StringOpt{
		Name:      "aws_secret_access_key",
		Desc:      "AWS secret access key",
		EnvVar:    "AWS_SECRET_ACCESS_KEY",
		HideValue: true,
	})

	app.Command("scheduled-backup", "backup a set of mongodb collections", func(cmd *cli.Cmd) {
		colls := cmd.String(cli.StringOpt{
			Name:   "collections",
			Desc:   "Collections to process (comma separated <database>/<collection>)",
			EnvVar: "MONGODB_COLLECTIONS",
			Value:  "foo/content,foo/bar",
		})

		cronExpr := cmd.String(cli.StringOpt{
			Name:   "cron",
			Desc:   "Cron expression for when to run",
			EnvVar: "CRON",
			Value:  "30 10 * * *",
			//Value:  "@every 30s",
		})

		dbPath := cmd.String(cli.StringOpt{
			Name:   "dbPath",
			Desc:   "Path to store boltdb file",
			EnvVar: "DBPATH",
			Value:  "/var/data/mongolizer/state.db",
		})

		run := cmd.Bool(cli.BoolOpt{
			Name:   "run",
			Desc:   "Run backups on startup?",
			EnvVar: "RUN",
			Value:  true,
		})

		cmd.Action = func() {
			m := newMongolizer(*connStr, *s3bucket, *s3dir, *s3domain, *accessKey, *secretKey)
			if err := m.backupScheduled(*colls, *cronExpr, *dbPath, *run); err != nil {
				log.Fatalf("backup failed : %v\n", err)
			}
		}
	})

	app.Command("backup", "backup a set of mongodb collections", func(cmd *cli.Cmd) {
		colls := cmd.String(cli.StringOpt{
			Name:   "collections",
			Desc:   "Collections to process (comma separated <database>/<collection>)",
			EnvVar: "MONGODB_COLLECTIONS",
			Value:  "foo/content,foo/bar",
		})
		cmd.Action = func() {
			m := newMongolizer(*connStr, *s3bucket, *s3dir, *s3domain, *accessKey, *secretKey)
			if err := m.backupAll(*colls); err != nil {
				log.Fatalf("backup failed : %v\n", err)
			}
		}
	})
	app.Command("restore", "restore a set of mongodb collections", func(cmd *cli.Cmd) {
		colls := cmd.String(cli.StringOpt{
			Name:   "collections",
			Desc:   "Collections to process (comma separated <database>/<collection>)",
			EnvVar: "MONGODB_COLLECTIONS",
			Value:  "foo/content,foo/bar",
		})
		dateDir := cmd.String(cli.StringOpt{
			Name: "date",
			Desc: "Date to restore backup from",
		})
		cmd.Action = func() {
			m := newMongolizer(*connStr, *s3bucket, *s3dir, *s3domain, *accessKey, *secretKey)
			if err := m.restoreAll(*dateDir, *colls); err != nil {
				log.Fatalf("restore failed : %v\n", err)
			}
		}
	})

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

type mongolizer struct {
	connectionString string
	s3bucket         string
	s3dir            string
	s3               *s3gof3r.S3
}

func newMongolizer(connectionString, s3bucket, s3dir, s3domain, accessKey, secretKey string) *mongolizer {
	return &mongolizer{
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

func (m *mongolizer) backupAll(colls string) error {
	parsed, err := parseCollections(colls)
	if err != nil {
		return err
	}
	dateDir := formattedNow()
	for _, coll := range parsed {
		dir := filepath.Join(m.s3dir, dateDir)
		err := m.backup(dir, coll.database, coll.collection)

		if err != nil {
			return err
		}
	}
	return nil
}

type scheduledJob struct {
	eId  cron.EntryID
	coll collName
}

type scheduledJobResult struct {
	Success   bool
	Timestamp time.Time
}

func (m *mongolizer) backupScheduled(colls string, cronExpr string, dbPath string, run bool) error {

	err := os.MkdirAll(filepath.Dir(dbPath), 0600)

	if err != nil {
		return err
	}

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Results"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})

	parsed, err := parseCollections(colls)
	if err != nil {
		return err
	}

	c := cron.New()

	metric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mongolizer_status",
		Help: "Captures whether last backup was ok or not",
	}, []string{"database", "collection"})

	opHandler := op.NewStatus("Mongolizer", "backs up mongo on schedule").
		ReadyAlways().
		AddMetrics(metric)

	var ids []scheduledJob

	for _, collection := range parsed {

		coll := collection

		cronFunc := func() {
			dateDir := formattedNow()
			dir := filepath.Join(m.s3dir, dateDir)
			err := m.backup(dir, coll.database, coll.collection)

			result := scheduledJobResult{true, time.Now()}

			if err != nil {
				metric.With(prometheus.Labels{"database": coll.database, "collection": coll.collection}).Set(0)
				result.Success = false
			} else {
				metric.With(prometheus.Labels{"database": coll.database, "collection": coll.collection}).Set(1)
			}

			r, _ := json.Marshal(result)
			db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("Results"))
				err := b.Put([]byte(fmt.Sprintf("%s/%s", coll.database, coll.collection)), r)
				return err
			})
		}

		// on startup, we are registering status metrics to notify prom of the last backup status immediately
		// we are also checking how long has it been since last backup, and if more than 13h we will trigger backup immediately
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("Results"))
			v := b.Get([]byte(fmt.Sprintf("%s/%s", coll.database, coll.collection)))

			result := scheduledJobResult{}

			json.Unmarshal(v, &result)

			if !result.Success {
				metric.With(prometheus.Labels{"database": coll.database, "collection": coll.collection}).Set(0)
			} else {
				metric.With(prometheus.Labels{"database": coll.database, "collection": coll.collection}).Set(1)
			}

			if time.Since(result.Timestamp).Hours() > 13 && run {
				go cronFunc()
			}

			return nil
		})

		//each time health endpoint is called we will pull the latest backup state from DB and report based on that
		opHandler.AddChecker(fmt.Sprintf("%s/%s", coll.database, coll.collection), func(cr *op.CheckResponse) {
			db.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("Results"))
				v := b.Get([]byte(fmt.Sprintf("%s/%s", coll.database, coll.collection)))

				result := scheduledJobResult{}

				json.Unmarshal(v, &result)

				if time.Since(result.Timestamp).Hours() > 13 {
					cr.Unhealthy("Last backup more than 13h ago", "Check backup was taken", "Stale backup data")
					return nil
				}

				if !result.Success {
					cr.Unhealthy("Backup failed", "Check backup was taken", "Stale backup data")
					return nil
				}

				cr.Healthy(fmt.Sprintf("Backed up %.0f hours ago", time.Since(result.Timestamp).Hours()))
				return nil
			})
		})

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

	http.Handle("/__/", op.NewHandler(opHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))

	return nil
}

func (m *mongolizer) backup(dir, database, collection string) error {

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

	if err := dumpCollectionTo(m.connectionString, database, collection, sw); err != nil {
		return err
	}

	if err := sw.Close(); err != nil {
		return err
	}

	err = w.Close()
	log.Printf("backed up %s/%s to %s in %s. Duration : %v\n", database, collection, dir, m.s3bucket, time.Now().Sub(start))
	return err
}

func (m *mongolizer) restoreAll(dateDir string, colls string) error {
	parsed, err := parseCollections(colls)
	if err != nil {
		return err
	}
	for _, coll := range parsed {
		dir := filepath.Join(m.s3dir, dateDir)
		err := m.restore(dir, coll.database, coll.collection)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *mongolizer) restore(dir, database, collection string) error {

	path := filepath.Join(dir, database, collection+extension)

	rc, _, err := m.s3.Bucket(m.s3bucket).GetReader(path, nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	sr := snappy.NewReader(rc)

	if err := restoreCollectionFrom(m.connectionString, database, collection, sr); err != nil {
		return err
	}
	return nil
}

func dumpCollectionTo(connStr string, database, collection string, writer io.Writer) error {
	session, err := mgo.Dial(connStr)
	if err != nil {
		return err
	}
	session.SetPrefetch(1.0)
	defer session.Close()

	q := session.DB(database).C(collection).Find(nil).Snapshot()
	iter := q.Iter()

	for {
		raw := &bson.Raw{}
		next := iter.Next(raw)
		if !next {
			break
		}
		_, err := writer.Write(raw.Data)
		if err != nil {
			return err
		}
	}

	return iter.Err()
}

func restoreCollectionFrom(connStr, database, collection string, reader io.Reader) error {
	session, err := mgo.DialWithTimeout(connStr, 5*time.Minute)
	if err != nil {
		return err
	}
	defer session.Close()

	err = clearCollection(session, database, collection)
	if err != nil {
		return err
	}

	start := time.Now()
	log.Printf("starting restore of %s/%s\n", database, collection)

	bulk := session.DB(database).C(collection).Bulk()

	var batchBytes int
	for {
		next, err := readNextBSON(reader)
		if err != nil {
			return err
		}
		if next == nil {
			break
		}

		// If we have something to write and the next doc would push the batch over
		// the limit, write the batch out now. 15000000 is intended to be within the
		// expected 16MB limit
		if batchBytes > 0 && batchBytes+len(next) > 15000000 {
			_, err = bulk.Run()
			if err != nil {
				return err
			}
			bulk = session.DB(database).C(collection).Bulk()
			batchBytes = 0
		}

		bulk.Insert(bson.Raw{Data: next})

		batchBytes += len(next)
	}
	_, err = bulk.Run()
	log.Printf("finished restore of %s/%s. Duration: %v\n", database, collection, time.Now().Sub(start))
	return err

}

func readNextBSON(reader io.Reader) ([]byte, error) {
	var lenBytes [4]byte

	_, err := io.ReadFull(reader, lenBytes[:])
	if err != nil {
		if err != io.EOF {
			return nil, err
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
			// this is a broken document.
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return buf, nil
}

func clearCollection(session *mgo.Session, database, collection string) error {
	start := time.Now()
	log.Printf("clearing collection %s/%s\n", database, collection)
	_, err := session.DB(database).C(collection).RemoveAll(nil)
	log.Printf("finished clearing collection %s/%s. Duration : %v\n", database, collection, time.Now().Sub(start))

	return err
}

type collName struct {
	database   string
	collection string
}

func parseCollections(colls string) ([]collName, error) {
	var cn []collName
	for _, coll := range strings.Split(colls, ",") {
		c := strings.Split(coll, "/")
		if len(c) != 2 {
			return nil, fmt.Errorf("failed to parse connections string : %s\n", colls)
		}
		cn = append(cn, collName{c[0], c[1]})
	}

	return cn, nil
}

func formattedNow() string {
	return time.Now().UTC().Format("2006-01-02T15-04-05")
}
