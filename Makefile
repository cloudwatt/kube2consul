VERSION=$(shell git describe --tags --always)

build:
	go build -v -i --ldflags '-s -extldflags "-static" -X main.kube2consulVersion=${VERSION}'
