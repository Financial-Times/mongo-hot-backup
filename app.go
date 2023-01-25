package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cli "github.com/jawher/mow.cli"
	log "github.com/sirupsen/logrus"
)

const (
	dateFormat = "2006-01-02T15-04-05"
	systemCode = "mongo-hot-backup"
)

func main() {
	app := cli.App("mongobackup", "Backup and restore mongodb collections to/from s3\nBackups are put in a directory structure /<base-dir>/<date>/database/collection")

	connStr := app.String(cli.StringOpt{
		Name:   "docdb-cluster-address",
		Desc:   "Address of the Document DB host",
		EnvVar: "DOCDB_CLUSTER_ADDRESS",
		Value:  "",
	})
	docDBUsername := app.String(cli.StringOpt{
		Name:   "docdb-username",
		Desc:   "Username for establishing a connection to Document DB",
		EnvVar: "DOCDB_USERNAME",
		Value:  "",
	})
	docDBPassword := app.String(cli.StringOpt{
		Name:   "docdb-password",
		Desc:   "Password for establishing a connection to Document DB",
		EnvVar: "DOCDB_PASSWORD",
		Value:  "",
	})
	s3bucket := app.String(cli.StringOpt{
		Name:   "bucket",
		Desc:   "s3 bucket name",
		EnvVar: "S3_BUCKET",
		Value:  "com.ft.coco-mongo-backup.prod",
	})
	s3BucketRegion := app.String(cli.StringOpt{
		Name:   "s3-bucket-region",
		Desc:   "s3 bucket region",
		EnvVar: "S3_BUCKET_REGION",
		Value:  "eu-west-1",
	})
	s3dir := app.String(cli.StringOpt{
		Name:   "base-dir",
		Desc:   "s3 base directory name",
		EnvVar: "S3_DIR",
		Value:  "/backups/",
	})
	colls := app.String(cli.StringOpt{
		Name:   "collections",
		Desc:   "Collections to process (comma separated <database>/<collection>)",
		EnvVar: "MONGODB_COLLECTIONS",
		Value:  "foo/content,foo/bar",
	})
	mongoTimeout := app.Int(cli.IntOpt{
		Name:   "mongoTimeout",
		Desc:   "Mongo session connection timeout in seconds. (e.g. 60)",
		EnvVar: "MONGO_TIMEOUT",
		Value:  60,
	})
	rateLimit := app.Int(cli.IntOpt{
		Name:   "rateLimit",
		Desc:   "Rate limit mongo operations in milliseconds. (e.g. 250)",
		EnvVar: "RATE_LIMIT",
		Value:  250,
	})
	batchLimit := app.Int(cli.IntOpt{
		Name:   "batchLimit",
		Desc:   "The size of data in bytes, that a bulk write is writing into mongodb at once. Not recommended to use more than 16MB (e.g. 15000000)",
		EnvVar: "BATCH_LIMIT",
		Value:  15000000,
	})

	app.Command("scheduled-backup", "backup a set of mongodb collections", func(cmd *cli.Cmd) {
		cronExpr := cmd.String(cli.StringOpt{
			Name:   "cron",
			Desc:   "Cron expression for when to run",
			EnvVar: "CRON",
			Value:  "30 10 * * *",
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
		healthHours := cmd.Int(cli.IntOpt{
			Name:   "health-hours",
			Desc:   "Number of hours back in time in which healthy backup needs to exist of each named collection for the app to be healthy. (e.g. 24)",
			EnvVar: "HEALTH_HOURS",
			Value:  24,
		})

		cmd.Action = func() {
			parsedColls, err := parseCollections(*colls)
			if err != nil {
				log.Fatalf("error parsing collections parameter: %v", err)
			}

			timeout := time.Duration(*mongoTimeout) * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			mongoClient, err := newMongoClient(ctx, *connStr, *docDBUsername, *docDBPassword, timeout)
			if err != nil {
				log.WithError(err).Fatal("Error establishing mongo connection")
			}

			dbService := newMongoService(mongoClient, &defaultBsonService{}, time.Duration(*rateLimit)*time.Millisecond, *batchLimit)
			statusKeeper, err := newBoltStatusKeeper(*dbPath)
			if err != nil {
				log.Fatalf("failed setting up to read or write scheduled backup status results: %v", err)
			}
			defer statusKeeper.Close()

			sess, err := session.NewSession(aws.NewConfig().WithRegion(*s3BucketRegion))
			if err != nil {
				log.WithError(err).Fatal("Creating AWS session failed")
			}

			storageService := newS3StorageService(*s3bucket, *s3dir, sess)
			backupService := newMongoBackupService(dbService, storageService, statusKeeper)
			scheduler := newCronScheduler(backupService, statusKeeper)
			healthService := newHealthService(*healthHours, statusKeeper, parsedColls, healthConfig{
				appSystemCode: systemCode,
				appName:       "mongobackup",
			})
			httpService := newScheduleHTTPService(scheduler, healthService)
			httpService.ScheduleAndServe(parsedColls, *cronExpr, *run)
		}
	})

	app.Command("backup", "backup a set of mongodb collections", func(cmd *cli.Cmd) {
		dbPath := cmd.String(cli.StringOpt{
			Name:   "dbPath",
			Desc:   "Path to store boltdb file",
			EnvVar: "DBPATH",
			Value:  "/var/data/mongobackup/state.db",
		})

		cmd.Action = func() {
			parsedColls, err := parseCollections(*colls)
			if err != nil {
				log.Fatalf("error parsing collections parameter: %v", err)
			}

			timeout := time.Duration(*mongoTimeout) * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			mongoClient, err := newMongoClient(ctx, *connStr, *docDBUsername, *docDBPassword, timeout)
			if err != nil {
				log.WithError(err).Fatal("Error establishing mongo connection")
			}

			dbService := newMongoService(mongoClient, &defaultBsonService{}, time.Duration(*rateLimit)*time.Millisecond, *batchLimit)
			statusKeeper, err := newBoltStatusKeeper(*dbPath)
			if err != nil {
				log.Fatalf("failed setting up to read or write scheduled backup status results: %v", err)
			}
			defer statusKeeper.Close()

			sess, err := session.NewSession(aws.NewConfig().WithRegion(*s3BucketRegion))
			if err != nil {
				log.WithError(err).Fatal("Creating AWS session failed")
			}

			storageService := newS3StorageService(*s3bucket, *s3dir, sess)
			backupService := newMongoBackupService(dbService, storageService, statusKeeper)
			if err := backupService.Backup(context.Background(), parsedColls); err != nil {
				log.Fatalf("backup failed : %v", err)
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
				log.Fatalf("error parsing collections parameter: %v", err)
			}

			timeout := time.Duration(*mongoTimeout) * time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			mongoClient, err := newMongoClient(ctx, *connStr, *docDBUsername, *docDBPassword, timeout)
			if err != nil {
				log.WithError(err).Fatal("Error establishing mongo connection")
			}

			dbService := newMongoService(mongoClient, &defaultBsonService{}, time.Duration(*rateLimit)*time.Millisecond, *batchLimit)

			sess, err := session.NewSession(aws.NewConfig().WithRegion(*s3BucketRegion))
			if err != nil {
				log.WithError(err).Fatal("Creating AWS session failed")
			}

			storageService := newS3StorageService(*s3bucket, *s3dir, sess)
			backupService := newMongoBackupService(dbService, storageService, &boltStatusKeeper{})
			if err := backupService.Restore(context.Background(), *dateDir, parsedColls); err != nil {
				log.Fatalf("restore failed : %v", err)
			}
		}
	})

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func parseCollections(colls string) ([]dbColl, error) {
	var cn []dbColl
	for _, coll := range strings.Split(colls, ",") {
		c := strings.Split(coll, "/")
		if len(c) != 2 {
			return nil, fmt.Errorf("failed to parse connections string: %s", colls)
		}
		cn = append(cn, dbColl{c[0], c[1]})
	}

	return cn, nil
}
