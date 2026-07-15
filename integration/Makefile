.SILENT :
.PHONY : test

TAG:=`git describe --tags`

build-docker-gen-images: build-docker-gen-alpine-image build-docker-gen-debian-image

cleanup-docker-gen-images:
	docker rmi nginxproxy/docker-gen:test-alpine || true
	docker rmi nginxproxy/docker-gen:test-debian || true

build-docker-gen-alpine-image:
	docker build --build-arg DOCKER_GEN_VERSION=$(TAG) -t nginxproxy/docker-gen:test-alpine -f ../Dockerfile.alpine ../

build-docker-gen-debian-image:
	docker build --build-arg DOCKER_GEN_VERSION=$(TAG) -t nginxproxy/docker-gen:test-debian -f ../Dockerfile.debian ../

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
