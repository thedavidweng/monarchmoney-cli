BINARY_NAME=monarch
DIST_DIR=dist

.PHONY: all build test clean lint fmt run-doctor snapshot-test

all: lint test build

build:
	go build -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/monarch

test:
	go test -v ./...

fmt:
	gofmt -s -w $$(git ls-files '*.go')

lint:
	@test -z "$$(gofmt -s -l $$(git ls-files '*.go'))" || (echo "gofmt: files need formatting:" && gofmt -s -l $$(git ls-files '*.go') && exit 1)
	go vet ./...

clean:
	rm -rf $(DIST_DIR)

run-doctor: build
	./$(DIST_DIR)/$(BINARY_NAME) doctor

snapshot-test:
	@echo "Snapshot testing not yet implemented"


changelog-preview:
	@echo "Changelog is managed by release-please. See CHANGELOG.md or the Release PR."
