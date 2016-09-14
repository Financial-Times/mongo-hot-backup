FROM alpine:3.4
ADD . /source/
RUN apk add --update bash \
  && apk --update add git go ca-certificates \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/up-mongolizer/" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv /source/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get ./... \
  && go install ${REPO_PATH} \
  && mv ${GOPATH}/bin/up-mongolizer / \
  && apk del go git \
  && rm -rf $GOPATH /var/cache/apk/*

EXPOSE 8080

CMD [ "/up-mongolizer", "backup"  ]
