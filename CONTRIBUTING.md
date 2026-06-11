# Contributing to monarchmoney-cli

We love contributions! Whether you're fixing a bug, adding a new command, or improving documentation, your help is appreciated.

## Development Setup

### Prerequisites
- [Go](https://golang.org/doc/install) 1.26.3 or later
- [Make](https://www.gnu.org/software/make/)

### Build
```bash
make build
```

### Run Tests
```bash
make test
```

## Adding a New Command

1.  **GraphQL Query**: Add your `.graphql` file to `internal/monarch/queries/`.
2.  **Service Layer**: Add the corresponding method to the `Service` in `internal/monarch/`.
3.  **CLI Layer**: Implement the Cobra command in `internal/cli/`. Ensure you handle the `--json` flag and use the standard `output.Renderer`.
4.  **Safety Layer**: If the command is a mutation, ensure you call `safety.Check` before execution.

## Pull Request Process

1.  Fork the repository and create your branch from `main`.
2.  Ensure your code follows idiomatic Go patterns.
3.  Run `make fmt` (or `gofmt -s -w .`) before committing — CI rejects unformatted code.
4.  Run `go vet ./...` to catch common issues.
5.  Include tests for any new functionality.
6.  Update the documentation if you've added or changed a command.
7.  Open a PR with a clear description of your changes.

## Code of Conduct

Please be respectful and professional in all interactions within this project.

---

Thank you for helping make `monarchmoney-cli` better!
