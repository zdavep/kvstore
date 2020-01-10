.PHONY: build
build: fmt compile lint 
	docker build -t kvstore .

.PHONY: fmt
fmt:
	gofmt -l -w internal/**/*.go cmd/**/*.go

.PHONY: compile
compile:
	go build ./...

.PHONY: lint 
lint:
	golangci-lint run 