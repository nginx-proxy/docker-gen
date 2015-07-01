FROM debian:wheezy
MAINTAINER Jason Wilder <jason@influxdb.com>

ENV VERSION 0.4.0
ENV DOWNLOAD_URL https://github.com/jwilder/docker-gen/releases/download/$VERSION/docker-gen-linux-amd64-$VERSION.tar.gz
ENV DOCKER_HOST unix:///var/run/docker.sock

RUN deps=' \
		curl ca-certificates \
	'; \
	set -x; \
	apt-get update \
	&& apt-get install -y --no-install-recommends $deps \
	&& curl -o docker-gen.tar.gz -L $DOWNLOAD_URL \
	&& tar -C /usr/local/bin -xvzf docker-gen.tar.gz \
	&& rm docker-gen.tar.gz \
	&& apt-get purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false -o APT::AutoRemove::SuggestsImportant=false $deps \
	&& apt-get clean -y \
	&& rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/usr/local/bin/docker-gen"]
