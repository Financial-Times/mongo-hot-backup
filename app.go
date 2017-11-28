package main

import (
	"fmt"
	"os"
	"strings"
	"github.com/jawher/mow.cli"
	"github.com/rlmcpherson/s3gof3r"
	log "github.com/Sirupsen/logrus"
)

const (
	extension = ".bson.snappy"
	dateFormat = "2006-01-02T15-04-05"
)

func main() {
	s3gof3r.DefaultConfig.Md5Check = false

	app := cli.App("mongobackup", "Backup and restore mongodb collections to/from s3\nBackups are put in a directory structure /<base-dir>/<date>/database/collection")

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
	colls := app.String(cli.StringOpt{
		Name:   "collections",
		Desc:   "Collections to process (comma separated <database>/<collection>)",
		EnvVar: "MONGODB_COLLECTIONS",
		Value:  "foo/content,foo/bar",
	})

	app.Command("scheduled-backup", "backup a set of mongodb collections", func(cmd *cli.Cmd) {
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
			Value:  "/var/data/mongobackup/state.db",
		})
		run := cmd.Bool(cli.BoolOpt{
			Name:   "run",
			Desc:   "Run backups on startup?",
			EnvVar: "RUN",
			Value:  true,
		})

		cmd.Action = func() {
			parsedColls, err := parseCollections(*colls)
			if err != nil {
				log.Fatalf("error parsing collections parameter: %v\n", err)
			}
			mongoService := newMongoService(&labixMongo{})
			backupService := newBackupService(mongoService, *connStr, *s3bucket, *s3dir, *s3domain, *accessKey, *secretKey)
			if err := backupService.backupScheduled(parsedColls, *cronExpr, *dbPath, *run); err != nil {
				log.Fatalf("backup failed : %v\n", err)
			}
		}
	})

	app.Command("backup", "backup a set of mongodb collections", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			parsedColls, err := parseCollections(*colls)
			if err != nil {
				log.Fatalf("error parsing collections parameter: %v\n", err)
			}
			mongoService := newMongoService(&labixMongo{})
			backupService := newBackupService(mongoService, *connStr, *s3bucket, *s3dir, *s3domain, *accessKey, *secretKey)
			if err := backupService.backupAll(parsedColls); err != nil {
				log.Fatalf("backup failed : %v\n", err)
			}
		}
	})
	app.Command("restore", "restore a set of mongodb collections", func(cmd *cli.Cmd) {
		dateDir := cmd.String(cli.StringOpt{
			Name:   "date",
			Desc:   "Date to restore backup from",
			EnvVar: "DATE",
			Value:  dateFormat,
		})
		cmd.Action = func() {
			parsedColls, err := parseCollections(*colls)
			if err != nil {
				log.Fatalf("error parsing collections parameter: %v\n", err)
			}
			mongoService := newMongoService(&labixMongo{})
			backupService := newBackupService(mongoService, *connStr, *s3bucket, *s3dir, *s3domain, *accessKey, *secretKey)
			if err := backupService.restoreAll(*dateDir, parsedColls); err != nil {
				log.Fatalf("restore failed : %v\n", err)
			}
		}
	})

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

type fullColl struct {
	database   string
	collection string
}

func parseCollections(colls string) ([]fullColl, error) {
	var cn []fullColl
	for _, coll := range strings.Split(colls, ",") {
		c := strings.Split(coll, "/")
		if len(c) != 2 {
			return nil, fmt.Errorf("failed to parse connections string : %s\n", colls)
		}
		cn = append(cn, fullColl{c[0], c[1]})
	}

	return cn, nil
}
