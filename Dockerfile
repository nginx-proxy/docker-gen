FROM alpine:latest
MAINTAINER Grant Millar <grant@cylo.io>

RUN apk -U add openssl

ENV VERSION 0.7.4_1
ENV DOWNLOAD_URL https://github.com/rid/docker-gen/releases/download/$VERSION/docker-gen-alpine-linux-amd64-$VERSION.tar.gz
ENV DOCKER_HOST unix:///tmp/docker.sock

RUN wget -qO- $DOWNLOAD_URL | tar xvz -C /usr/local/bin

ENTRYPOINT ["/usr/local/bin/docker-gen"]
