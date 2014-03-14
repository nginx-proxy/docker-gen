.SILENT :
.PHONY : docker-gen clean fmt

all: docker-gen

docker-gen:
	echo "Building docker-gen"
	go build

