.PHONY: docker fmt build lint 

all: docker

docker: fmt build
	docker build -t kvstore .

fmt:
	gofmt -l -w -s internal/**/*.go cmd/**/*.go

build:
	go build ./...

lint:
	golangci-lint run 