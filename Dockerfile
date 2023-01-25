FROM golang:1

ENV PROJECT=mongo-hot-backup

ENV ORG_PATH="github.com/Financial-Times"
ENV SRC_FOLDER="${GOPATH}/src/${ORG_PATH}/${PROJECT}"
ENV BUILDINFO_PACKAGE="${ORG_PATH}/service-status-go/buildinfo."

COPY . ${SRC_FOLDER}
WORKDIR ${SRC_FOLDER}

# Install dependancies and build app
RUN VERSION="version=$(git describe --tag --always 2> /dev/null)" \
    && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
    && REPOSITORY="repository=$(git config --get remote.origin.url)" \
    && REVISION="revision=$(git rev-parse HEAD)" \
    && BUILDER="builder=$(go version)" \
    && LDFLAGS="-X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
    && CGO_ENABLED=0 go build -mod=readonly -a -o /artifacts/${PROJECT} -ldflags="${LDFLAGS}" \
    && echo "Build flags: ${LDFLAGS}"

# Download required Amazon certificate to authenticate to the Document DB cluster
RUN mkdir -p /tmp/amazonaws
WORKDIR /tmp/amazonaws
RUN apt-get update && apt-get install -y wget && wget https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem

# Multi-stage build - copy only the certs and the binary into the image
FROM scratch
WORKDIR /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/* /
COPY --from=0 /tmp/amazonaws/* /

ENTRYPOINT ["/mongo-hot-backup"]
CMD ["scheduled-backup"]
