package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jawher/mow.cli"
	"github.com/klauspost/compress/snappy"
	"github.com/rlmcpherson/s3gof3r"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func main() {
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

	app.Command("backup", "backup a set of mongodb collections", func(cmd *cli.Cmd) {
		colls := cmd.String(cli.StringArg{
			Name:   "COLLECTIONS",
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
		colls := cmd.String(cli.StringArg{
			Name:   "COLLECTIONS",
			Desc:   "Collections to process (comma separated <database>/<collection>)",
			EnvVar: "MONGODB_COLLECTIONS",
			Value:  "foo/content,foo/bar",
		})
		dateDir := cmd.String(cli.StringArg{
			Name: "DATE",
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
func (m *mongolizer) backup(dir, database, collection string) error {

	start := time.Now()
	log.Printf("backing up %s/%s to %s in %s", database, collection, dir, m.s3bucket)

	path := filepath.Join(dir, database, collection)

	b := m.s3.Bucket(m.s3bucket)
	w, err := b.PutWriter(path, nil, nil)
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
	log.Printf("backed up %s/%s to %s in %s. Duration : %v", database, collection, dir, m.s3bucket, time.Now().Sub(start))
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

	path := filepath.Join(dir, database, collection)

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
	session, err := mgo.Dial(connStr)
	if err != nil {
		return err
	}
	defer session.Close()

	err = dropCollectionIfExists(session, database, collection)
	if err != nil {
		return err
	}

	err = createCollection(session, database, collection)
	if err != nil {
		return err
	}

	bulk := session.DB(database).C(collection).Bulk()

	var count int
	for {
		next, err := readNextBSON(reader)
		if err != nil {
			return err
		}
		if next == nil {
			break
		}

		bulk.Insert(bson.Raw{Data: next})
		if count != 0 && count%1024 == 0 {
			_, err = bulk.Run()
			if err != nil {
				return err
			}
			bulk = session.DB(database).C(collection).Bulk()
		}
		count++
	}
	_, err = bulk.Run()
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

func dropCollectionIfExists(session *mgo.Session, database, collection string) error {
	err := session.DB(database).C(collection).DropCollection()
	if err != nil && err.Error() == "ns not found" {
		err = nil
	}
	return err
}

func createCollection(session *mgo.Session, database, collection string) error {
	command := bson.D{{"create", collection}}

	res := bson.M{}
	err := session.DB(database).Run(command, &res)
	if err != nil {
		return err
	}
	result := res["ok"]
	if resf64, ok := result.(float64); ok && resf64 == 1 {
		return nil

	}
	log.Printf("DEBUG result is %v %T\n", result, result)
	return fmt.Errorf("failed to create collection: %v", res["errmsg"])
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
