.PHONY: build run test vet

build:
	go build -o bin/webserver ./cmd/webserver

run:
	go run ./cmd/webserver

test:
	go test -race ./...

vet:
	go vet ./...
