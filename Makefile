#
#   Author: Rohith
#   Date: 2015-08-05 01:03:05 +0100 (Wed, 05 Aug 2015)
#
#  vim:ts=2:sw=2:et
#
NAME="node-register"
AUTHOR="gambol99"

.PHONY: build docker clean

build:
	mkdir -p ./bin
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o bin/node-register
	
docker: build
	sudo docker build -t ${AUTHOR}/${NAME} .

clean:
	rm -rf ./bin
