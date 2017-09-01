# Mongolizer

This tool can back up or restore MongoDB collections while DB is running to/from AWS S3.

## Installation

```
go get -u github.com/Financial-Times/mongolizer
```

## Building

```
docker build -t coco/mongodb-hot-backup .
```

## Running

```
mongolizer --help
```

## Running backups on schedule

You can deploy a docker container that will run backups on schedule (default at 10:30am every day)

The state of backups is kept in a boltdb file at `/var/data/mongolizer/state.db`

Health endpoint is available at `0.0.0.0:8080/__/health` and will report healthy if there was a successful backup in the last 13h.

Instrumentation endpoint is available at `0.0.0.0:8080/__/metrics`, a prom gauge is exposed where the value of `mongolizer_status` gauge is either 1 or 0 depending on result of the previous backup. Gauge is labbeled with `database` and collection `labels`.

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

### Restoring

```
# docker run --rm mongolizer /mongolizer restore --help

Usage: mongolizer restore [OPTIONS]

restore a set of mongodb collections

Options:
  --collections="foo/content,foo/bar"   Collections to process (comma separated <database>/<collection>) ($MONGODB_COLLECTIONS)
  --date="2006-01-02T15-04-05"          Date to restore backup from
```
