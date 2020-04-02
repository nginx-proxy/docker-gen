FROM alpine:latest
LABEL maintainer="Jason Wilder <mail@jasonwilder.com>"

ARG VERSION=0.7.4
ARG ARCHITECTURE=amd64
ARG DOWNLOAD_URL=https://github.com/jwilder/docker-gen/releases/download/$VERSION/docker-gen-alpine-linux-$ARCHITECTURE-$VERSION.tar.gz
ENV DOCKER_HOST unix:///tmp/docker.sock

RUN apk -U add openssl
RUN wget -qO- $DOWNLOAD_URL | tar xvz -C /usr/local/bin

ENTRYPOINT ["/usr/local/bin/docker-gen"]
