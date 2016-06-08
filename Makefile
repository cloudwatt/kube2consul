export GO15VENDOREXPERIMENT=1

build:
	go build -v -i --ldflags '-s -extldflags "-static"'
