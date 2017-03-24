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

### Kubernetes manifest example

Full example of rolling mongo with persistent volume + mongolizer with metrics scraping

```
# A headless service to create DNS records
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.alpha.kubernetes.io/tolerate-unready-endpoints: "true"
    prometheus.io/scrape: 'true'
    prometheus.io/path:   /__/metrics
    prometheus.io/port:   '8080'
  name: mongo
  labels:
    app: mongo
spec:
  ports:
  - port: 8080
    targetPort: 8080
    name: mongolizer
  - port: 27017
    targetPort: 27017
    name: client
  clusterIP: None
  selector:
    app: mongo
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: mongo
spec:
  replicas: 1
  template:
    metadata:
      name: mongo
      labels:
        app: mongo
    spec:
      imagePullSecrets:
      - name: dockerhub-key
      containers:
      - name: mongolizer
        image: registry.uw.systems/system/mongolizer:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: data
          mountPath: "/var/data/mongolizer/"
          subPath: "mongolizer"
        env:
        - name: MONGODB_COLLECTIONS
          value: "db/collection,db/collection2"
        - name: MONGODB
          value: "mongo:27017"
        - name: AWS_ACCESS_KEY_ID
          value: ""
        - name: AWS_SECRET_ACCESS_KEY
          value: ""
        - name: S3_BUCKET
          value: "backup-bucket"
        - name: S3_DIR
          value: "/"
      - name: mongo
        image: mongo
        ports:
        - containerPort: 27017
        volumeMounts:
        - name: data
          mountPath: /data/db
          subPath: "mongodb"
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: mongo-ebs-pvc
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: mongo-ebs-pvc
  annotations:
    volume.beta.kubernetes.io/storage-class: ebs-gp2
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi

```

### Prom and alerts

To get metrics you can use query similar to

```
1 - avg(mongolizer_status{kubernetes_namespace="default"}) by (app, database, collection) < bool 1
```

Example alert

```
```