# Mongolizer

This tool can back up or restore MongoDB collections while DB is running to/from AWS S3.

You can deploy a docker container that will run backups on schedule (default at 10:30am every day). Or you can just run the container to make a single backup, or restore from a given point of time.

For the schedule option, the state of backups is kept in a boltdb file at `/var/data/mongolizer/state.db`

Health endpoint is available at `0.0.0.0:8080/__health` and will report healthy if there was a successful backup for each configured collection in the last 26 hours.

An initial backup will be ran upon startup if there's no backup found in the last 26 hours. Can be disabled.

## Installation

```
go get -u github.com/Financial-Times/mongolizer
```

## Building

```
docker build -t coco/mongolizer .
```

## Usage

### Creating backups on a schedule

example:

```
export MONGODB=ip-172-24-11-64.eu-west-1.compute.internal:27018,ip-172-24-186-252.eu-west-1.compute.internal:27020,ip-172-24-74-51.eu-west-1.compute.internal:27019
export S3_BUCKET=com.ft.coco-mongo-backup.prod;
export S3_DOMAIN=s3-eu-west-1.amazonaws.com
export S3_DIR=pre-prod-uk
export AWS_ACCESS_KEY_ID=123
export AWS_SECRET_ACCESS_KEY=456
export CRON="1 0 * * *"
export RUN=false

docker run --rm --env "MONGODB==$MONGODB" --env "S3_DOMAIN=$S3_DOMAIN" --env "S3_BUCKET=$S3_BUCKET" --env "S3_DIR=$S3_DIR" --env "AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID" --env "AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY" --env "CRON=$CRON" --env "RUN=false" coco/mongolizer:2.0.0-productionize-rc7 backup --collections="upp-store/lists,upp-store/list-notifications"
```

The help `docker run --rm coco/mongolizer scheduled-backup --help` could supply you a bit more information about how arguments should be received.

### Creating a single backup

```
export MONGODB=ip-172-24-11-64.eu-west-1.compute.internal:27018,ip-172-24-186-252.eu-west-1.compute.internal:27020,ip-172-24-74-51.eu-west-1.compute.internal:27019
export S3_BUCKET=com.ft.coco-mongo-backup.prod;
export S3_DOMAIN=s3-eu-west-1.amazonaws.com
export S3_DIR=pre-prod-uk
export AWS_ACCESS_KEY_ID=123
export AWS_SECRET_ACCESS_KEY=456

docker run --rm --env "MONGODB==$MONGODB" --env "S3_DOMAIN=$S3_DOMAIN" --env "S3_BUCKET=$S3_BUCKET" --env "S3_DIR=$S3_DIR" --env "AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID" --env "AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY" coco/mongolizer:2.0.0-productionize-rc7 backup --collections="upp-store/lists,upp-store/list-notifications"
```

You can also try `docker run --rm coco/mongolizer backup --help`

### Restoring

```
export MONGODB=ip-172-24-11-64.eu-west-1.compute.internal:27018,ip-172-24-186-252.eu-west-1.compute.internal:27020,ip-172-24-74-51.eu-west-1.compute.internal:27019
export S3_BUCKET=com.ft.coco-mongo-backup.prod;
export S3_DOMAIN=s3-eu-west-1.amazonaws.com
export S3_DIR=pre-prod-uk
export AWS_ACCESS_KEY_ID=123
export AWS_SECRET_ACCESS_KEY=456

docker run --rm --env "MONGODB==$MONGODB" --env "S3_DOMAIN=$S3_DOMAIN" --env "S3_BUCKET=$S3_BUCKET" --env "S3_DIR=$S3_DIR" --env "AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID" --env "AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY" coco/mongolizer:2.0.0-productionize-rc7 restore --collections="upp-store/lists,upp-store/list-notifications" --date="2017-09-04T12-40-36"
```

You can also try `docker run --rm coco/mongolizer restore --help`

## Links

* [mongodb backup/restore documentation on FT Technology's Google Sites](https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/mongo-db-run-book/mongo-db-backup-restore)
