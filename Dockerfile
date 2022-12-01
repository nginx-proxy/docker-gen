ARG DOCKER_GEN_VERSION=main

# Build docker-gen from scratch
FROM golang:1.19.3-alpine as go-builder

ARG DOCKER_GEN_VERSION
WORKDIR /build

# Install the dependencies
COPY . .
RUN go mod download

# Build the docker-gen executable
RUN GOOS=linux CGO_ENABLED=0 go build -ldflags "-X main.buildVersion=${DOCKER_GEN_VERSION}" -o docker-gen ./cmd/docker-gen

FROM alpine:3.15.4

ARG DOCKER_GEN_VERSION
ENV DOCKER_GEN_VERSION=${DOCKER_GEN_VERSION} \
    DOCKER_HOST=unix:///tmp/docker.sock

# Install packages required by the image
RUN apk add --no-cache --virtual .bin-deps openssl

# Install docker-gen from build stage
COPY --from=go-builder /build/docker-gen /usr/local/bin/docker-gen

# Copy the license
COPY LICENSE /usr/local/share/doc/docker-gen/

ENTRYPOINT ["/usr/local/bin/docker-gen"]