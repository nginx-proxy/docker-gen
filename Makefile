.SILENT :
.PHONY : docker-gen clean fmt

TAG:=`git describe --abbrev=0 --tags`
LDFLAGS:=-X main.buildVersion $(TAG)

all: docker-gen

docker-gen:
	echo "Building docker-gen"
	go build -ldflags "$(LDFLAGS)"

dist-clean:
	rm -rf dist
	rm -f docker-gen-linux-*.tar.gz

dist: dist-clean
	mkdir -p dist/linux/amd64 && GOOS=linux GOARCH=amd64 go build -o dist/linux/amd64/docker-gen
	mkdir -p dist/linux/i386  && GOOS=linux GOARCH=386 go build -o dist/linux/i386/docker-gen

release: dist
	godep restore
	tar -cvzf docker-gen-linux-amd64-$(TAG).tar.gz -C dist/linux/amd64 docker-gen
	tar -cvzf docker-gen-linux-i386-$(TAG).tar.gz -C dist/linux/i386 docker-gen