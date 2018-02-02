# mongo-hot-backup

[![Circle CI](https://circleci.com/gh/Financial-Times/mongo-hot-backup/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/mongo-hot-backup/tree/master)[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/mongo-hot-backup)](https://goreportcard.com/report/github.com/Financial-Times/mongo-hot-backup) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/mongo-hot-backup/badge.svg)](https://coveralls.io/github/Financial-Times/mongo-hot-backup)

This tool can back up or restore MongoDB collections while DB is running to/from AWS S3.

You can deploy a docker container that will run backups on schedule. Or you can just run the container to make a single backup, or restore from a given point of time.

For the schedule option, the state of backups is kept in a boltdb file. (at `/var/data/mongo-hot-backup/state.db` or where you set it)

Health endpoint is available at `0.0.0.0:8080/__health` and will report healthy if there was a successful backup for each configured collection in the last X hours, also configurable. Good-to-go `/__gtg` endpoint available as well, and `/build-info`.

An initial backup to be ran upon startup can be enabled.

## Installation and Building

```
go get -u github.com/kardianos/govendor
go get -u github.com/Financial-Times/mongo-hot-backup
cd $GOPATH/src/github.com/Financial-Times/methode-article-image-set-mapper
govendor sync
docker build -t coco/mongo-hot-backup .
```

## Usage

### Creating backups on a schedule

example:

```
docker run --rm \
  --env "MONGODB=ip-172-24-11-64.eu-west-1.compute.internal:27018,ip-172-24-186-252.eu-west-1.compute.internal:27020,ip-172-24-74-51.eu-west-1.compute.internal:27019" \
  --env "S3_DOMAIN=s3-eu-west-1.amazonaws.com" \
  --env "S3_BUCKET=com.ft.upp.mongo-backup" \
  --env "S3_DIR=upp-staging-delivery-eu" \
  --env "AWS_ACCESS_KEY_ID=123" \
  --env "AWS_SECRET_ACCESS_KEY=456" \
  --env "CRON=0 0 * * *" \
  --env "RUN=false" \
  --env "HEALTH_HOURS=26" \
  --env "MONGODB_COLLECTIONS="upp-store/lists,upp-store/list-notifications"
  nexus.in.ft.com:5000/coco/mongo-hot-backup:2.0.0 scheduled-backup
```

The help `docker run --rm coco/mongo-hot-backup scheduled-backup --help` could supply you a bit more information about how arguments should be received.

### Creating a single backup

```
docker run --rm \
  --env "MONGODB=ip-172-24-11-64.eu-west-1.compute.internal:27018,ip-172-24-186-252.eu-west-1.compute.internal:27020,ip-172-24-74-51.eu-west-1.compute.internal:27019" \
  --env "S3_DOMAIN=s3-eu-west-1.amazonaws.com" \
  --env "S3_BUCKET=com.ft.upp.mongo-backup" \
  --env "S3_DIR=upp-staging-delivery-eu" \
  --env "AWS_ACCESS_KEY_ID=123" \
  --env "AWS_SECRET_ACCESS_KEY=456" \
  --env "MONGODB_COLLECTIONS="upp-store/lists,upp-store/list-notifications"
  nexus.in.ft.com:5000/coco/mongo-hot-backup:2.0.0 backup
```

You can also try `docker run --rm coco/mongo-hot-backup backup --help`

### Restoring

```
docker run --rm \
  --env "MONGODB=ip-172-24-11-64.eu-west-1.compute.internal:27018,ip-172-24-186-252.eu-west-1.compute.internal:27020,ip-172-24-74-51.eu-west-1.compute.internal:27019" \
  --env "S3_DOMAIN=s3-eu-west-1.amazonaws.com" \
  --env "S3_BUCKET=com.ft.upp.mongo-backup" \
  --env "S3_DIR=upp-staging-delivery-eu" \
  --env "AWS_ACCESS_KEY_ID=123" \
  --env "AWS_SECRET_ACCESS_KEY=456" \
  --env "RATE_LIMIT=1250" \
  --env "BATCH_LIMIT=8000000" \
  --env "MONGODB_COLLECTIONS="upp-store/lists,upp-store/list-notifications"
  nexus.in.ft.com:5000/coco/mongo-hot-backup:2.0.0 restore --date="2017-11-23T14-53-20"
```

You can also try `docker run --rm coco/mongo-hot-backup restore --help`

## Links

* [mongodb backup/restore documentation](https://docs.google.com/document/d/1f3-1JHWrXy2mQrBfqs4jRuPNhO5jThKdnh8J7uyoJBU/edit#)
