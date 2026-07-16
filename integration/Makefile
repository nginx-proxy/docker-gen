.SILENT :
.PHONY : test

GIT_TAG:=`git describe --tags`
IMAGE_REPO:=nginxproxy/docker-gen
ALPINE_IMAGE:=$(IMAGE_REPO):test-alpine
DEBIAN_IMAGE:=$(IMAGE_REPO):test-debian

build-docker-gen-test-images: build-docker-gen-alpine-test-image build-docker-gen-debian-test-image

cleanup-docker-gen-test-images:
	docker rmi $(ALPINE_IMAGE) || true
	docker rmi $(DEBIAN_IMAGE) || true

build-docker-gen-alpine-test-image:
	docker build --build-arg DOCKER_GEN_VERSION=$(GIT_TAG) -t $(ALPINE_IMAGE) -f ../Dockerfile.alpine ../

build-docker-gen-debian-test-image:
	docker build --build-arg DOCKER_GEN_VERSION=$(GIT_TAG) -t $(DEBIAN_IMAGE) -f ../Dockerfile.debian ../

get-deps:
	go mod download

check-gofmt:
	if [ -n "$(shell go fmt ./...)" ]; then \
		echo 1>&2 'The following files need to be formatted:'; \
		gofmt -l ./...; \
		exit 1; \
	fi

test:
	go test -v ./...
