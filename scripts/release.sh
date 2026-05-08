#!/bin/bash
set -e

VERSION=$(grep "Version =" internal/version/version.go | cut -d'"' -f2)
echo "Building monarch $VERSION..."

mkdir -p dist

# Build for macOS arm64
GOOS=darwin GOARCH=arm64 go build -o dist/monarch-darwin-arm64 ./cmd/monarch

# Build for Linux amd64
GOOS=linux GOARCH=amd64 go build -o dist/monarch-linux-amd64 ./cmd/monarch

echo "Build complete."
ls -lh dist/
