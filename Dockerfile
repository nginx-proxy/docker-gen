FROM alpine:latest
LABEL maintainer="Jason Wilder <mail@jasonwilder.com>"

RUN apk --update --no-cache add curl gzip tar

ENV VERSION 0.7.3
ENV DOWNLOAD_URL https://github.com/jwilder/docker-gen/releases/download/$VERSION/docker-gen-alpine-linux-amd64-$VERSION.tar.gz
ENV DOCKER_HOST unix:///tmp/docker.sock

RUN curl -fsSL $DOWNLOAD_URL | tar -zxv -C /usr/local/bin

ENTRYPOINT ["/usr/local/bin/docker-gen"]
