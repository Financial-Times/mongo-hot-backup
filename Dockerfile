FROM alpine:3.5
ADD . /source/
RUN apk add --update bash \
  && apk --update add git go libc-dev ca-certificates \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/mongodb-hot-backup/" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /source/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get ./... \
  && go install ${REPO_PATH} \
  && mv ${GOPATH}/bin/mongodb-hot-backup / \
  && apk del go git libc-dev \
  && rm -rf $GOPATH /var/cache/apk/*

EXPOSE 8080

CMD [ "/mongodb-hot-backup", "backup"  ]
