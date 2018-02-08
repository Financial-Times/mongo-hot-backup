FROM alpine:3.7

ENV GOPATH=/go

WORKDIR /go/src/app
ADD . /go/src/app/

EXPOSE 8080

RUN apk --no-cache add git go ca-certificates musl-dev && \
  go get ./... && \
  go test -v && \
  CGO_ENABLED=0 go build -ldflags '-s -extldflags "-static"' -o /mongolizer . && \
  apk del go git musl-dev && \
  rm -rf ${GOPATH}

ENTRYPOINT ["/mongolizer"]
CMD ["scheduled-backup"]
