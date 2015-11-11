#
#   Author: Rohith
#   Date: 2015-08-05 01:03:05 +0100 (Wed, 05 Aug 2015)
#
#

NAME=node-register
AUTHOR=gambol99
HARDWARE=$(shell uname -m)
SHA=$(shell git log --pretty=format:'%h' -n 1)
VERSION=$(shell awk '/Version =/ { print $$3 }' version.go | sed 's/"//g')
DEPS=$(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
PACKAGES=$(shell go list ./...)
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

.PHONY: build docker docker-release clean release deps cover vet lint format test

default: build

build:
	@echo "--> Performing a build"
	@mkdir -p ./bin
	@mkdir -p ./release
	@sed -i "s/^const GitSha.*/const GitSha = \"${SHA}\"/" version.go
	CGO_ENABLED=0 GOOS=linux godep go build -a -tags netgo -ldflags '-w' -o bin/node-register

docker: build
	@echo "--> Performing a docker build"
	sudo docker build -t ${AUTHOR}/${NAME} .

clean:
	@echo "--> Performing a clean"
	@rm -rf ./bin
	@rm -rf ./release

release:
	@echo "--> Building a release"
	@mkdir -p release
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o release/${NAME}
	@gzip -c release/${NAME} > release/${NAME}_${VERSION}_linux_${HARDWARE}.gz
	@rm -f release/${NAME}

docker-release: docker
	docker tag -f gambol99/node-register:latest docker.io/gambol99/node-register:latest
	docker push docker.io/gambol99/node-register:latest

deps:
	@echo "--> Installing build dependencies"
	@go get -d -v ./... $(DEPS)

lint:
	@echo "--> Running golint"
	@which golint 2>/dev/null ; if [ $$? -eq 1 ]; then \
		go get -u github.com/golang/lint/golint; \
	fi
	@golint .

vet:
	@echo "--> Running go tool vet $(VETARGS) ."
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@go tool vet $(VETARGS) .

cover:
	@echo "--> Running go test --cover"
	@go test --cover

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

test:
	@echo "--> Running go tests"
	@go test -v
	@$(MAKE) vet
	@$(MAKE) cover
