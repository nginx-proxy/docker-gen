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
	rm -f docker-gen-darwin-*.tar.gz

dist: dist-clean
	mkdir -p dist/linux/amd64 && GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/linux/amd64/docker-gen
	mkdir -p dist/linux/i386  && GOOS=linux GOARCH=386 go build -ldflags "$(LDFLAGS)" -o dist/linux/i386/docker-gen
	mkdir -p dist/linux/armel  && GOOS=linux GOARCH=arm GOARM=5 go build -ldflags "$(LDFLAGS)" -o dist/linux/armel/docker-gen
	mkdir -p dist/linux/armhf  && GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "$(LDFLAGS)" -o dist/linux/armhf/docker-gen
	mkdir -p dist/darwin/amd64 && GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/darwin/amd64/docker-gen
	mkdir -p dist/darwin/i386  && GOOS=darwin GOARCH=386 go build -ldflags "$(LDFLAGS)" -o dist/darwin/i386/docker-gen


release: dist
	glock sync github.com/jwilder/docker-gen
	tar -cvzf docker-gen-linux-amd64-$(TAG).tar.gz -C dist/linux/amd64 docker-gen
	tar -cvzf docker-gen-linux-i386-$(TAG).tar.gz -C dist/linux/i386 docker-gen
	tar -cvzf docker-gen-linux-armel-$(TAG).tar.gz -C dist/linux/armel docker-gen
	tar -cvzf docker-gen-linux-armhf-$(TAG).tar.gz -C dist/linux/armhf docker-gen
	tar -cvzf docker-gen-darwin-amd64-$(TAG).tar.gz -C dist/darwin/amd64 docker-gen
	tar -cvzf docker-gen-darwin-i386-$(TAG).tar.gz -C dist/darwin/i386 docker-gen
