# Contributing to monarchmoney-cli

We love contributions! Whether you're fixing a bug, adding a new command, or improving documentation, your help is appreciated.

## Development Setup

### Prerequisites
- [Go](https://golang.org/doc/install) 1.26.4 or later
- [mise](https://mise.jdx.dev/) (auto-installs the toolchain via `mise install`)

### Build
```bash
mise run build
```

### Run Tests
```bash
mise run test
```

## Adding a New Command

1.  **GraphQL Query**: Add your `.graphql` file to `queries/`.
2.  **Service Layer**: Add the corresponding method to the `Service` in `internal/monarch/`.
3.  **CLI Layer**: Implement the Cobra command in `internal/cli/`. Ensure you handle the `--json` flag and use the standard `output.Renderer`.
4.  **Safety Layer**: If the command is a mutation, ensure you call `safety.Check` before execution.

## Pull Request Process

1.  Fork the repository and create your branch from `main`.
2.  Ensure your code follows idiomatic Go patterns.
3.  Run `mise run fmt` before committing — CI rejects unformatted code.
4.  Run `mise run check` to run fmt, build, test, lint, and conventions.
5.  Include tests for any new functionality.
6.  Update the documentation if you've added or changed a command.
7.  Open a PR with a clear description of your changes.
8.  PRs merge with squash only: merge commits duplicate entries in the release notes because release-please counts both the merge message and the branch commits.

## Code of Conduct

Please be respectful and professional in all interactions within this project.

---

Thank you for helping make `monarchmoney-cli` better!
