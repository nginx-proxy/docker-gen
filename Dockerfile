from golang as gobuild
RUN go get github.com/jwilder/docker-gen/cmd/docker-gen

FROM alpine:latest
LABEL maintainer="Jason Wilder <mail@jasonwilder.com>"

RUN apk -U add openssl libc6-compat

ENV VERSION 0.7.3
ENV DOCKER_HOST unix:///tmp/docker.sock

COPY --from=gobuild /go/bin/docker-gen /usr/local/bin/docker-gen

ENTRYPOINT ["/usr/local/bin/docker-gen"]
