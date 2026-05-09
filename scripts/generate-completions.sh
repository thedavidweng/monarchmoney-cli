#!/bin/bash
set -e

mkdir -p completions

# Assume monarch is built and in dist/monarch or just run go run
go run ./cmd/monarch completion bash > completions/monarch.bash
go run ./cmd/monarch completion zsh > completions/monarch.zsh
go run ./cmd/monarch completion fish > completions/monarch.fish

echo "Completions generated."
