FROM golang:1.9.2-alpine3.7
ADD . /source/
RUN apk add --update bash \
  && apk --update add git go libc-dev ca-certificates \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/mongo-hot-backup/" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /source/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get ./... \
  && go install ${REPO_PATH} \
  && mv ${GOPATH}/bin/mongo-hot-backup / \
  && apk del go git libc-dev \
  && rm -rf $GOPATH /var/cache/apk/*

EXPOSE 8080

ENTRYPOINT ["/mongo-hot-backup"]
CMD ["scheduled-backup"]
