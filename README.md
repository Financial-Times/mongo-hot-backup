# MongoDB Hot Backup

This tool can back up or restore MongoDB collections while DB is running to/from AWS S3.

## Installation
```
go get -u github.com/utilitywarehouse/mongolizer
```
## Running
```
mongolizer --help
```

## Running backups on schedule

You can deploy a docker container that will run backups on schedule (default at 10:30am every day)

The state of backups is kept in a boldb file at `/var/data/mongolizer/state.db`

Health endpoint is available at `0.0.0.0:8080/__/health` and will report healthy if there was a successful backup in the last 13h.

Instrumentation endpoint is available at `0.0.0.0:8080/__/metrics`, a prom gauge is exposed where the value of `mongolizer_status` gaige is either 1 or 0 depending on result of the previous backup.

An initial backup will be ran if there's no backup found in the last 13h.

### Usage

```
# docker run --rm mongolizer /mongolizer scheduled-backup --help

Usage: mongolizer scheduled-backup [OPTIONS] COLLECTIONS

backup a set of mongodb collections

Arguments:
  COLLECTIONS="foo/content,foo/bar"   Collections to process (comma separated <database>/<collection>) ($MONGODB_COLLECTIONS)

Options:
  --cron="30 10 * * *"                       Cron expression for when to run ($CRON)
  --dbPath="/var/data/mongolizer/state.db"   Path to store boltdb file ($DBPATH)
  --run=true                                 Run backups on startup? ($RUN)
```

### Manifest example

```
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: mongolizer
spec:
  replicas: 1
  template:
    metadata:
      name: mongolizer
      labels:
        app: mongolizer
    spec:
      containers:
      - name: mongolizer
        image: mongolizer
        ports:
        - containerPort: 8080
        env:
        - name: MONGODB_COLLECTIONS
          value: "db/collection1,db/collection2"
        - name: MONGODB
          value: "mongo:27017"
        - name: AWS_ACCESS_KEY_ID
          value: "KEY"
        - name: AWS_SECRET_ACCESS_KEY
          value: "SECRET"
        - name: S3_BUCKET
          value: "my-data-bucket"
        - name: S3_DIR
          value: "/backups"
        - name: CRON
          value: "30 10 * * *"
---
```


