#
#   Author: Rohith
#   Date: 2015-08-05 01:03:05 +0100 (Wed, 05 Aug 2015)
#
#  vim:ts=2:sw=2:et
#
NAME=node-register
AUTHOR="gambol99"
HARDWARE=$(shell uname -m)
SHA=$(shell git log --pretty=format:'%h' -n 1)
VERSION=$(shell awk '/VERSION/ { print $$3 }' version.go | sed 's/"//g')

.PHONY: build docker clean release

build:
	mkdir -p ./bin
	mkdir -p ./release
	sed -i "s/^const GitSha.*/const GitSha = \"${SHA}\"/" version.go
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o bin/node-register

docker: build
	sudo docker build -t ${AUTHOR}/${NAME} .

clean:
	rm -rf ./bin
	rm -rf ./release

release:
	mkdir -p release
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o release/${NAME}
	gzip -c release/${NAME} > release/${NAME}_${VERSION}_linux_${HARDWARE}.gz
	rm -f release/${NAME}
	# github-release release -r node-register -n "node-register, version: ${VERSION}" -u gambol99 --pre-release -t v${VERSION}
    # github-release upload --name node-register_${VERSION}_linux_x86_64.gz -f node-register_0.0.2_linux_x86_64.gz -r node-register -u gambol99 -t v${VERSION}

    