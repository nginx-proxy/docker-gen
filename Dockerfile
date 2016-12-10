FROM alpine:latest
MAINTAINER Grant Millar <grant@cylo.io>

RUN apk -U add openssl

COPY dist/alpine-linux/amd64/docker-gen /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/docker-gen"]
