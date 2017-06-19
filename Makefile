NAME=kube2consul
VERSION=$(shell git describe --tags --always)

build:
	mkdir -p bin
	go build -v -i --ldflags '-s -extldflags "-static" -X main.kube2consulVersion=${VERSION}' -o bin/${NAME} .
