# mongo-hot-backup

[![Circle CI](https://circleci.com/gh/Financial-Times/mongo-hot-backup/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/mongo-hot-backup/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/mongo-hot-backup)](https://goreportcard.com/report/github.com/Financial-Times/mongo-hot-backup)
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/mongo-hot-backup/badge.svg)](https://coveralls.io/github/Financial-Times/mongo-hot-backup)

This tool can back up or restore MongoDB collections while DB is running to/from AWS S3.

It is configured to run scheduled backups by default.
The state of backups is kept in a boltdb file at `/var/data/mongo-hot-backup/state.db`.
An initial backup to be run upon startup can be enabled.

The options to create a single backup or restore from a given point of time are described below.

## Installation and Building

```shell
  go get github.com/Financial-Times/mongo-hot-backup
  cd $GOPATH/src/github.com/Financial-Times/mongo-hot-backup
  go build .
```

## Tests

````shell
  go test -mod=readonly -race ./...
````

## Usage

You need to be authenticated in the proper EKS cluster before executing the commands below. Proceed with caution!

### Creating a single backup

```shell
  kubectl run mongo-hot-backup-manual-$(date +%s) \
    --image=nexus.in.ft.com:5000/coco/mongo-hot-backup:v3.2.0 \
    --restart="Never" \
    --overrides='{"apiVersion": "v1", "spec": {"imagePullSecrets": [{"name": "nexusregistry"}], "serviceAccountName": "eksctl-mongo-hot-backup-serviceaccount"}}' \
    --env "MONGODB=mongodb-0.default.svc.cluster.local:27017,mongodb-1.default.svc.cluster.local:27017,mongodb-2.default.svc.cluster.local:27017" \
    --env "S3_BUCKET=com.ft.upp.mongo-backup-dev" \
    --env "S3_DIR=upp-k8s-dev-delivery-eu" \
    --env "MONGODB_COLLECTIONS=upp-store/pages" \
    backup
```

### Restoring

```shell
  kubectl run mongo-hot-backup-manual-$(date +%s) \
    --image=nexus.in.ft.com:5000/coco/mongo-hot-backup:v3.2.0 \
    --restart="Never" \
    --overrides='{"apiVersion": "v1", "spec": {"imagePullSecrets": [{"name": "nexusregistry"}], "serviceAccountName": "eksctl-mongo-hot-backup-serviceaccount"}}' \
    --env "MONGODB=mongodb-0.default.svc.cluster.local:27017,mongodb-1.default.svc.cluster.local:27017,mongodb-2.default.svc.cluster.local:27017" \
    --env "S3_BUCKET=com.ft.upp.mongo-backup-dev" \
    --env "S3_DIR=upp-k8s-dev-delivery-eu" \
    --env "RATE_LIMIT=1000" \
    --env "BATCH_LIMIT=8000000" \
    --env "MONGODB_COLLECTIONS=upp-store/pages" \
    -- restore --date="2022-08-31T15-00-00"
```

## Admin endpoints

The admin endpoints are:

```text
    /__gtg
    /__health (reports whether there was a successful backup for each configured collection in the last X hours)
    /__build-info
```