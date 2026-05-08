BINARY_NAME=monarch
DIST_DIR=dist

.PHONY: all build test clean lint

all: lint test build

build:
	go build -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/monarch

test:
	go test -v ./...

clean:
	rm -rf $(DIST_DIR)

lint:
	go vet ./...
