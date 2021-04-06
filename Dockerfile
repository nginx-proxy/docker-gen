# Build docker-gen from scratch
FROM golang:1.14-alpine as dockergen
RUN apk add --no-cache git

# Download the sources for the given version
ENV VERSION 0.7.5
ADD https://github.com/jwilder/docker-gen/archive/${VERSION}.tar.gz sources.tar.gz

# Move the sources into the right directory
RUN tar -xzf sources.tar.gz && \
   mkdir -p /go/src/github.com/jwilder/ && \
   mv docker-gen-* /go/src/github.com/jwilder/docker-gen

# Install the dependencies and make the docker-gen executable
WORKDIR /go/src/github.com/jwilder/docker-gen
RUN CGO_ENABLED=0 go build -ldflags "-X main.buildVersion=${VERSION}" ./cmd/docker-gen

FROM alpine:latest
LABEL maintainer="Jason Wilder <mail@jasonwilder.com>"

RUN apk -U add openssl

ENV VERSION 0.7.5
COPY --from=dockergen /go/src/github.com/jwilder/docker-gen/docker-gen /usr/local/bin/docker-gen
ENV DOCKER_HOST unix:///tmp/docker.sock


ENTRYPOINT ["/usr/local/bin/docker-gen"]
