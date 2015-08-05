#
#   Author: Rohith
#   Date: 2015-08-05 01:03:05 +0100 (Wed, 05 Aug 2015)
#
#  vim:ts=2:sw=2:et
#
NAME=node-register
AUTHOR="gambol99"
HARDWARE=$(shell uname -m)
VERSION=$(shell awk '/VERSION/ { print $$3 }' version.go | sed 's/"//g')

.PHONY: build docker clean release

build:
	mkdir -p ./bin
	mkdir -p ./release
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o bin/node-register

docker: build
	sudo docker build -t ${AUTHOR}/${NAME} .

clean:
	rm -rf ./bin
	rm -rf ./release

release:
	mkdir -p release
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o release/${NAME}
	cd release && gzip -c ${NAME} > ${NAME}_${VERSION}_linux_${HARDWARE}.gz
	rm -f release/${NAME}
