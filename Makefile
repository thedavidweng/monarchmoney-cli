BINARY_NAME=monarch
DIST_DIR=dist

.PHONY: all build test clean lint fmt run-doctor snapshot-test

all: lint test build

build:
	go build -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/monarch

test:
	go test -v ./...

fmt:
	go fmt ./...

lint:
	go vet ./...

clean:
	rm -rf $(DIST_DIR)

run-doctor: build
	./$(DIST_DIR)/$(BINARY_NAME) doctor

snapshot-test:
	@echo "Snapshot testing not yet implemented"
