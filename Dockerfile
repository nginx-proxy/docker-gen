# Build docker-gen from scratch
FROM golang:1.17.2-alpine as go-builder

ARG VERSION=main

WORKDIR /build

# Install the dependencies
COPY . .
RUN go mod download -json

# Build the docker-gen executable
RUN CGO_ENABLED=0 go build -ldflags "-X main.buildVersion=${VERSION}" -o docker-gen ./cmd/docker-gen

FROM alpine:3.13

LABEL maintainer="Jason Wilder <mail@jasonwilder.com>"

ENV DOCKER_HOST unix:///tmp/docker.sock

# Install packages required by the image
RUN apk add --no-cache --virtual .bin-deps openssl

# Install docker-gen from build stage
COPY --from=go-builder /build/docker-gen /usr/local/bin/docker-gen

ENTRYPOINT ["/usr/local/bin/docker-gen"]