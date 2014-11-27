FROM debian:wheezy
MAINTAINER Jason Wilder <jwilder@litl.com>

ENV VERSION 0.3.6
ENV DOCKER_HOST unix:///tmp/docker.sock

RUN apt-get update && apt-get install -y curl && curl -o docker-gen-linux-amd64-$VERSION.tar.gz -L https://github.com/jwilder/docker-gen/releases/download/$VERSION/docker-gen-linux-amd64-$VERSION.tar.gz && apt-get remove -y curl && apt-get -y clean
RUN tar -C /usr/local/bin -xvzf docker-gen-linux-amd64-$VERSION.tar.gz && rm docker-gen-linux-amd64-$VERSION.tar.gz

ENTRYPOINT ["/usr/local/bin/docker-gen"]
