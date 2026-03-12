.PHONY: build test run clean lint

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

run:
	go run ./cmd/server

clean:
	rm -rf bin/

lint:
	go vet ./...
