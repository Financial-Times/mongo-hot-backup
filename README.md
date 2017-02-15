# MongoDB Hot Backup

This tool can back up or restore MongoDB collections while DB is running to/from AWS S3.

# Usage
## Build/install
### Go app
```
go get -u github.com/Financial-Times/up-mongolizer
```
### Docker app
```
docker build -t coco/up-mongolizer   .
```

## Run
### Backup
```
docker run \
        -e MONGODB=<MONGODB_ADDRESSES> \
        -e S3_DOMAIN=<S3_DOMAIN> \
        -e S3_BUCKET=<S3_BUCKET> \
        -e S3_DIR=<ENVIRONMENT_TAG> \
        -e AWS_ACCESS_KEY_ID=<AWS_ACCESS_KEY> \
        -e AWS_SECRET_ACCESS_KEY=<AWS_SECRET_KEY> \
        coco/up-mongolizer<:app_version> /up-mongolizer backup <db>/<coll_1>,<db>/<coll_2>,<db>/<coll_3>
```

* <MONGODB_ADDRESSES> - The address to connect to MongoDB cluster
* <S3_DOMAIN> - The domain name of S3 location where the backup should go
* <S3_BUCKET> - The S3 bucket name
* <ENVIRONMENT_TAG> - The S3 folder name, which should represent the environment tag
* <AWS_ACCESS_KEY> - The AWS access key
* <AWS_SECRET_KEY> - The AWS secret key
* <app_version> - The Docker image version of the app. Latest if omitted
* <db> - The DB under the collections are
* <coll_nr> - The collection to be backed up

Example:
```
docker run \
          -e MONGODB=$(for x in $(etcdctl ls /ft/config/mongodb);do echo -n $(etcdctl get $x/host):$(etcdctl get $x/port)"," ; done | sed s/.$//) \
          -e S3_DOMAIN=s3-eu-west-1.amazonaws.com \
          -e S3_BUCKET=com.ft.coco-mongo-backup.prod \
          -e S3_DIR=$(/usr/bin/etcdctl get /ft/config/environment_tag) \
          -e AWS_ACCESS_KEY_ID=$(/usr/bin/etcdctl get /ft/_credentials/aws/aws_access_key_id) \
          -e AWS_SECRET_ACCESS_KEY=$(/usr/bin/etcdctl get /ft/_credentials/aws/aws_secret_access_key) \
          coco/up-mongolizer:v0.2.0 /up-mongolizer backup upp-store/content,upp-store/lists,upp-store/notifications
```


### Restore
```
 docker run \
           -e MONGODB=<MONGODB_ADDRESSES> \
           -e S3_DOMAIN=<S3_DOMAIN> \
           -e S3_BUCKET=<S3_BUCKET> \
           -e S3_DIR=<ENVIRONMENT_TAG> \
           -e AWS_ACCESS_KEY_ID=<AWS_ACCESS_KEY> \
           -e AWS_SECRET_ACCESS_KEY=<AWS_SECRET_KEY> \
           coco/up-mongolizer<:app_version> /up-mongolizer restore <db>/<coll_1>,<db>/<coll_2>,<db>/<coll_3> <timestamp>
```

* <MONGODB_ADDRESSES> - The address to connect to MongoDB cluster
* <S3_DOMAIN> - The domain name of S3 location where the restore should go
* <S3_BUCKET> - The S3 bucket name
* <ENVIRONMENT_TAG> - The S3 folder name, which should represent the environment tag
* <AWS_ACCESS_KEY> - The AWS access key
* <AWS_SECRET_KEY> - The AWS secret key
* <app_version> - The Docker image version of the app. Latest if omitted
* <db> - The DB under the collections are
* <coll_nr> - The collection to be restored
* <timestamp> - The timestamp of the backup date

Example:
```
docker run \
          -e MONGODB=$(for x in $(etcdctl ls /ft/config/mongodb);do echo -n $(etcdctl get $x/host):$(etcdctl get $x/port)"," ; done | sed s/.$//) \
          -e S3_DOMAIN=s3-eu-west-1.amazonaws.com \
          -e S3_BUCKET=com.ft.coco-mongo-backup.prod \
          -e S3_DIR=/pre-prod-uk/ \
          -e AWS_ACCESS_KEY_ID=$(/usr/bin/etcdctl get /ft/_credentials/aws/aws_access_key_id) \
          -e AWS_SECRET_ACCESS_KEY=$(/usr/bin/etcdctl get /ft/_credentials/aws/aws_secret_access_key) \
          coco/up-mongolizer:v0.2.0 /up-mongolizer restore upp-store/content,upp-store/lists,upp-store/notifications 2017-02-14T08-25-36
```
