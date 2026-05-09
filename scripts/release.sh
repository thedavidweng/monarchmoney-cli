#!/bin/bash
set -euo pipefail

VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || grep 'Version =' internal/version/version.go | cut -d'"' -f2)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-s -w -X github.com/thedavidweng/monarchmoney-cli/internal/version.Version=${VERSION} -X github.com/thedavidweng/monarchmoney-cli/internal/version.Commit=${COMMIT} -X github.com/thedavidweng/monarchmoney-cli/internal/version.Date=${DATE} -X github.com/thedavidweng/monarchmoney-cli/internal/version.BuiltBy=release.sh"

echo "Building monarch ${VERSION}..."
mkdir -p dist

GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "${LDFLAGS}" -o dist/monarch-darwin-arm64 ./cmd/monarch
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "${LDFLAGS}" -o dist/monarch-linux-amd64 ./cmd/monarch

echo "Build complete."
ls -lh dist/
