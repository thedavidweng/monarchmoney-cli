# monarchmoney-cli

A local, agent-friendly CLI for Monarch Money.

## Features
- **Agent-first:** Stable JSON output, predictable exit codes.
- **Safety:** `--read-only`, `--dry-run`, and `--confirm` guards.
- **Auditable:** Daily mutation logs in `~/.monarchmoney-cli/audit/`.
- **Full Coverage:** Support for accounts, transactions, budgets, cashflow, and more.

## Installation
```bash
# Clone and build
git clone https://github.com/monarchmoney-cli/monarch
cd monarch
make build
./dist/monarch --help
```

## Usage
### Authentication
```bash
monarch auth login --email user@example.com
monarch auth status
```

### Reading Data
```bash
monarch accounts list
monarch transactions list --limit 10
monarch budgets list --month 2026-05
```

### Mutations (Safe)
```bash
# Dry-run first
monarch transactions update <tx-id> --notes "New note" --dry-run

# Execute with confirmation
monarch transactions update <tx-id> --notes "New note" --confirm
```

## Configuration
Config is stored in `~/.monarchmoney-cli/config.yaml`.
Session is stored in `~/.monarchmoney-cli/session.json`.

<p align="center">
  <a href="https://github.com/<owner>/monarchmoney-cli/actions/workflows/test.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/<owner>/monarchmoney-cli/test.yml?branch=main&label=ci"></a>
  <a href="https://github.com/<owner>/monarchmoney-cli/releases"><img alt="Release" src="https://img.shields.io/github/v/release/<owner>/monarchmoney-cli"></a>
  <a href="https://github.com/<owner>/monarchmoney-cli/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/<owner>/monarchmoney-cli"></a>
  <a href="https://goreportcard.com/report/github.com/<owner>/monarchmoney-cli"><img alt="Go Report" src="https://goreportcard.com/badge/github.com/<owner>/monarchmoney-cli"></a>
  <a href="https://github.com/<owner>/homebrew-monarch"><img alt="Homebrew" src="https://img.shields.io/badge/homebrew-supported-orange"></a>
</p>

---

## What is `monarchmoney-cli`?

`monarchmoney-cli` is a single-binary command line tool for working with Monarch Money data from your terminal, scripts, and local agents.

It is designed for people who want to query, export, analyze, and carefully modify Monarch Money data without opening the web app. The CLI exposes Monarch account, transaction, budget, cashflow, category, tag, recurring transaction, subscription, credit, attachment, and manual account workflows through a stable command contract.

The command is named `monarch`:

```bash
monarch accounts list --json
monarch transactions search "amazon" --from 2026-01-01 --json
monarch budgets list --month 2026-05 --json
monarch cashflow summary --from 2026-01 --to 2026-05 --json
```

The project is built for agent use from the beginning:

- clean JSON on stdout
- diagnostics on stderr
- stable exit codes
- `--read-only` mode
- `--dry-run` plans
- explicit `--confirm` gates for remote writes
- audit logs for executed mutations
- OpenClaw and Hermes integration guides
- optional MCP server

---

## Important disclaimer

`monarchmoney-cli` is an independent, community-maintained project.

This project is not affiliated with, sponsored by, endorsed by, or supported by Monarch Money, Inc. Monarch Money, Monarch, and related marks are trademarks or registered trademarks of their respective owners.

This project uses unofficial Monarch Money interfaces. Those interfaces may change without notice. Commands that depend on remote Monarch behavior can break if Monarch changes authentication, GraphQL operations, response shapes, endpoint behavior, or account security policies.

You are responsible for complying with Monarch Money's terms and policies. Use this tool only with your own Monarch account or an account you are authorized to access.

The images in this README are referenced from public Monarch website assets for contextual illustration. All rights to those images remain with Monarch Money, Inc. Replace them with original project screenshots or assets before publishing if your distribution policy requires fully owned imagery.

---

## Project status

`monarchmoney-cli` is designed as a production-quality open source CLI. Early releases may still be marked `0.x` while command names, output schemas, and API coverage stabilize.

| Area | Status |
|---|---|
| CLI foundation | Planned / in development |
| Auth and session persistence | Planned / in development |
| Read-only Monarch commands | Planned / in development |
| Safe mutations | Planned |
| Homebrew distribution | Planned |
| OpenClaw integration | Planned |
| Hermes integration | Planned |
| SQLite cache | Planned |
| MCP server | Planned |

The public stability target is `v1.0`, where the command tree and JSON schemas become stable.

---

## Why this exists

Existing community libraries proved that Monarch Money data can be accessed programmatically. `monarchmoney-cli` turns that capability into a polished local tool for automation.

The goal is not to recreate the Monarch web app. The goal is to provide a reliable local command layer that agents and scripts can call safely.

`monarchmoney-cli` focuses on:

- terminal-first workflows
- agent-first JSON contracts
- safe financial mutations
- one-command installation
- transparent authentication and session handling
- auditable local operations
- durable integrations with OpenClaw, Hermes, shell scripts, cron jobs, and MCP clients

---

## Features

### Agent-first command contract

Every command that returns structured data supports `--json`.

```bash
monarch transactions list --limit 25 --json
```

Output is wrapped in a predictable envelope:

```json
{
  "ok": true,
  "data": [],
  "meta": {
    "command": "transactions.list",
    "profile": "default",
    "duration_ms": 81,
    "schema_version": "2026-05-08",
    "request_id": "local-uuid"
  }
}
```

Errors use the same contract:

```json
{
  "ok": false,
  "error": {
    "code": "AUTH_SESSION_EXPIRED",
    "message": "Session expired. Run `monarch auth login`.",
    "category": "auth",
    "retryable": true
  },
  "meta": {
    "command": "auth.status",
    "profile": "default",
    "duration_ms": 23,
    "schema_version": "2026-05-08"
  }
}
```

### Safe by default

Financial data changes require explicit intent.

```bash
# Preview a change. Makes no remote mutation.
monarch transactions update tx_123 --category cat_restaurants --dry-run --json

# Execute after review.
monarch transactions update tx_123 --category cat_restaurants --confirm --json
```

Agent integrations should default to read-only mode:

```bash
MONARCH_READONLY=1 monarch transactions search "uber" --from 2026-05-01 --json
```

### Local authentication and session storage

```bash
monarch auth login
monarch auth status --json
monarch auth logout
```

Sessions are stored locally with restrictive permissions. Credentials and session tokens are redacted from logs, diagnostics, and audit records.

### Doctor command

```bash
monarch doctor
monarch doctor --json
monarch doctor --connect --json
```

`doctor` checks local configuration, session state, file permissions, API connectivity, output mode, read-only mode, and common agent integration problems.

### Homebrew install

```bash
brew tap <owner>/monarch
brew install monarchmoney-cli
```

The installed binary is named `monarch`.

### OpenClaw and Hermes ready

This project includes integration files and examples for:

- OpenClaw skill usage
- Hermes terminal backend usage
- Hermes Docker backend usage
- optional Hermes MCP server usage

### Optional local cache

A later release includes local SQLite caching for large transaction histories and low-latency agent queries.

```bash
monarch cache sync transactions --from 2024-01-01 --events
monarch cache search "amazon" --json
monarch cache stats --json
```

---

## Installation

### Homebrew

```bash
brew tap <owner>/monarch
brew install monarchmoney-cli
```

Verify the installation:

```bash
monarch --version
monarch doctor
```

### GitHub Releases

Download the latest release for your platform from:

```text
https://github.com/<owner>/monarchmoney-cli/releases
```

Supported release targets:

```text
macOS arm64
macOS x86_64
Linux arm64
Linux x86_64
```

Example manual install:

```bash
curl -L -o monarch.tar.gz \
  https://github.com/<owner>/monarchmoney-cli/releases/download/v0.1.0/monarch_Darwin_arm64.tar.gz

tar -xzf monarch.tar.gz
chmod +x monarch
sudo mv monarch /usr/local/bin/monarch
```

### Build from source

Requirements:

- Go 1.22 or newer
- Git
- Make

```bash
git clone https://github.com/<owner>/monarchmoney-cli.git
cd monarchmoney-cli
make build
./dist/monarch --version
```

---

## Quick start

### 1. Install

```bash
brew tap <owner>/monarch
brew install monarchmoney-cli
```

### 2. Check your environment

```bash
monarch doctor
```

### 3. Log in

```bash
monarch auth login
```

For non-interactive usage:

```bash
export MONARCH_EMAIL="you@example.com"
export MONARCH_PASSWORD="your-password"
export MONARCH_MFA_SECRET="your-totp-secret"

monarch auth login --email "$MONARCH_EMAIL" --password "$MONARCH_PASSWORD" --mfa-secret "$MONARCH_MFA_SECRET"
```

### 4. Query your data

```bash
monarch accounts list --json
monarch transactions list --limit 20 --json
monarch budgets list --month 2026-05 --json
monarch cashflow summary --from 2026-01 --to 2026-05 --json
```

### 5. Use read-only mode for agents

```bash
export MONARCH_READONLY=1
monarch transactions search "grocery" --from 2026-05-01 --json
```

---

## Common workflows

### List accounts

```bash
monarch accounts list --json
```

### Show one account

```bash
monarch accounts show acc_123 --json
```

### Search transactions

```bash
monarch transactions search "starbucks" --from 2026-01-01 --to 2026-05-08 --json
```

### Get transactions that need review

```bash
monarch transactions list --needs-review --limit 100 --json
```

### Export transactions

```bash
monarch transactions export \
  --from 2026-01-01 \
  --to 2026-05-08 \
  --format csv \
  --output transactions.csv
```

### Show monthly budgets

```bash
monarch budgets list --month 2026-05 --json
```

### Summarize cashflow

```bash
monarch cashflow summary --from 2026-01 --to 2026-05 --json
```

### List recurring transactions

```bash
monarch recurring list --json
```

### Preview a transaction update

```bash
monarch transactions update tx_123 \
  --category cat_restaurants \
  --notes "Reviewed by automation" \
  --dry-run \
  --json
```

### Execute a confirmed transaction update

```bash
monarch transactions update tx_123 \
  --category cat_restaurants \
  --notes "Reviewed by automation" \
  --confirm \
  --json
```

---

## Command overview

```text
monarch
├── auth
│   ├── login
│   ├── status
│   ├── logout
│   └── session path
├── doctor
├── accounts
│   ├── list
│   ├── show
│   ├── history
│   ├── holdings
│   ├── types
│   ├── refresh
│   ├── refresh-status
│   ├── create-manual
│   ├── update
│   ├── delete
│   └── upload-history
├── institutions
│   └── list
├── transactions
│   ├── list
│   ├── search
│   ├── show
│   ├── summary
│   ├── duplicates
│   ├── splits
│   ├── create
│   ├── update
│   ├── delete
│   ├── split
│   ├── export
│   └── attachments
│       ├── list
│       ├── upload
│       └── download
├── categories
│   ├── list
│   ├── groups
│   ├── create
│   ├── delete
│   └── delete-many
├── tags
│   ├── list
│   └── create
├── budgets
│   ├── list
│   ├── show
│   ├── set
│   ├── reset
│   ├── flexible set
│   ├── flex-rollover set
│   └── export
├── cashflow
│   ├── list
│   ├── summary
│   ├── categories
│   └── merchants
├── recurring
│   ├── list
│   └── update
├── credit
│   └── history
├── subscription
│   └── show
├── cache
│   ├── sync
│   ├── stats
│   ├── search
│   └── cleanup
└── mcp
    └── serve
```

See [`COMMANDS.md`](COMMANDS.md) for full command documentation.

---

## Capability coverage

`monarchmoney-cli` is designed to cover the capabilities exposed by the current Monarch Money community Python ecosystem.

### Read commands

- accounts
- account holdings
- account type options
- account balance history
- institutions
- budgets
- credit history
- subscription details
- recurring transactions
- transaction summaries
- transaction list/search
- duplicate transactions
- transaction categories
- category groups
- transaction details
- transaction splits
- transaction tags
- cashflow
- cashflow summary
- account refresh status

### Mutation and remote-action commands

- account refresh
- manual account creation
- account update
- account deletion
- account balance history upload
- transaction creation
- transaction update
- transaction deletion
- transaction split update
- category creation
- category deletion
- tag creation
- transaction tag setting
- budget amount updates
- flexible budget updates
- rollover settings updates
- budget reset
- recurring transaction updates
- attachment upload

All mutation commands are routed through the safety model.

---

## Global flags

Most commands support these global flags:

```text
--json                 emit machine-readable JSON
--pretty               pretty-print JSON output
--compact              return compact output fields
--full                 return full normalized output fields
--events               emit NDJSON progress events for long-running commands
--read-only            block remote writes
--dry-run              preview a remote write without executing it
--confirm              explicitly execute a remote write
--timeout 30s          set command timeout
--profile default      use a named profile
--config <path>        use a specific config file
--no-color             disable colored output
--verbose              print more diagnostics to stderr
--debug                print debug diagnostics to stderr with secrets redacted
```

---

## Environment variables

```bash
MONARCH_EMAIL="you@example.com"
MONARCH_PASSWORD="your-password"
MONARCH_MFA_SECRET="your-totp-secret"
MONARCH_MFA_CODE="123456"
MONARCH_SESSION_PATH="$HOME/.monarchmoney-cli/session.json"
MONARCH_CONFIG="$HOME/.monarchmoney-cli/config.yaml"
MONARCH_PROFILE="default"
MONARCH_READONLY="1"
MONARCH_OUTPUT="json"
MONARCH_TIMEOUT="30s"
MONARCH_API_ENDPOINT="https://api.monarch.com/graphql"
```

Precedence:

```text
CLI flags > environment variables > config file > defaults
```

---

## Configuration

Default config path:

```text
~/.monarchmoney-cli/config.yaml
```

Example:

```yaml
profile: default
api_endpoint: https://api.monarch.com/graphql
output: json
timeout: 30s
read_only: false
session_path: ~/.monarchmoney-cli/session.json
audit_log: true
cache_path: ~/.monarchmoney-cli/cache/monarch.sqlite
```

Default local paths:

```text
~/.monarchmoney-cli/config.yaml
~/.monarchmoney-cli/session.json
~/.monarchmoney-cli/audit/
~/.monarchmoney-cli/cache/
```

Recommended permissions:

```text
~/.monarchmoney-cli              0700
config.yaml                      0600
session.json                     0600
audit/*.jsonl                    0600
cache/*.sqlite                   0600
```

Run `monarch doctor` to check permissions.

---

## Safety model

`monarchmoney-cli` treats financial mutations as privileged operations.

| Operation type | Examples | Default behavior |
|---|---|---|
| Read | list, show, search, summary, export | Allowed |
| Remote action | account refresh | Blocked by read-only mode |
| Non-destructive mutation | update transaction notes, set category, update budget | Requires `--confirm` to execute |
| Destructive mutation | delete transaction, delete account, reset budget | Requires `--confirm` and explicit resource ID |
| Bulk mutation | batch category cleanup, bulk tag update | Requires dry-run plan and confirmation |

### Read-only mode

```bash
MONARCH_READONLY=1 monarch transactions list --json
MONARCH_READONLY=1 monarch transactions update tx_123 --category cat_456 --json
```

The second command fails before making a remote write.

### Dry-run

```bash
monarch budgets set --category cat_food --month 2026-05 --amount 500 --dry-run --json
```

Dry-run output contains a planned mutation diff and performs no remote mutation.

### Confirm

```bash
monarch budgets set --category cat_food --month 2026-05 --amount 500 --confirm --json
```

`--confirm` is required for remote writes.

### Audit log

Executed mutations are written to:

```text
~/.monarchmoney-cli/audit/YYYY-MM-DD.jsonl
```

Example record:

```json
{
  "timestamp": "2026-05-08T22:12:00Z",
  "command": "transactions.update",
  "resource_id": "tx_123",
  "dry_run": false,
  "confirmed": true,
  "profile": "default",
  "args_hash": "sha256:...",
  "result": "success"
}
```

Audit records do not include passwords, MFA secrets, session tokens, cookies, or authorization headers.

---
### Rules for agents

Agents should follow these rules:

1. Always pass `--json` when parsing output.
2. Read stdout only for data.
3. Treat stderr as diagnostics.
4. Use `MONARCH_READONLY=1` by default.
5. Run `monarch doctor --json` before first use.
6. Use `--dry-run` before any mutation.
7. Request explicit user approval before executing commands with `--confirm`.
8. Avoid destructive commands unless the user explicitly asks for the exact operation.

## JSON and exit-code contract

### stdout/stderr

| Stream | Contents |
|---|---|
| stdout | command result only |
| stderr | warnings, progress, diagnostics, debug logs |

### Exit codes

| Code | Meaning |
|---:|---|
| 0 | Success |
| 1 | General error |
| 2 | Invalid command or arguments |
| 3 | Authentication error |
| 4 | Authorization or read-only violation |
| 5 | Network error |
| 6 | Monarch API error |
| 7 | Validation error |
| 8 | Timeout |
| 9 | Partial success |
| 10 | Confirmation required |

### Error codes

Common error codes:

```text
AUTH_REQUIRED
AUTH_SESSION_EXPIRED
AUTH_MFA_REQUIRED
AUTH_MFA_INVALID
NETWORK_UNREACHABLE
NETWORK_TIMEOUT
API_ERROR
API_SCHEMA_CHANGED
VALIDATION_FAILED
READ_ONLY_VIOLATION
CONFIRMATION_REQUIRED
DRY_RUN_REQUIRED
RESOURCE_NOT_FOUND
PARTIAL_SUCCESS
CONFIG_INVALID
SESSION_PERMISSION_INSECURE
```

See [`docs/json-schema.md`](docs/json-schema.md).

---

## Development

### Requirements

- Go 1.22+
- Make
- Git

### Clone

```bash
git clone https://github.com/<owner>/monarchmoney-cli.git
cd monarchmoney-cli
```

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint
```

### Run locally

```bash
go run ./cmd/monarch --version
go run ./cmd/monarch doctor --json
```

### Live integration tests

Live tests require a Monarch account and are gated behind environment variables.

```bash
export MONARCH_TEST_EMAIL="you@example.com"
export MONARCH_TEST_PASSWORD="your-password"
export MONARCH_TEST_MFA_SECRET="your-totp-secret"

make test-integration
```

Mutation integration tests should run only against a dedicated test account.

---

## Repository layout

```text
monarchmoney-cli/
├── cmd/
│   └── monarch/
│       └── main.go
├── internal/
│   ├── app/
│   ├── auth/
│   ├── audit/
│   ├── cache/
│   ├── cli/
│   ├── config/
│   ├── doctor/
│   ├── errors/
│   ├── graphql/
│   ├── monarch/
│   ├── output/
│   ├── safety/
│   └── version/
├── queries/
├── docs/
├── integrations/
│   ├── openclaw/
│   └── hermes/
├── completions/
├── scripts/
├── testdata/
│   ├── golden/
│   ├── fixtures/
│   └── snapshots/
├── Formula/
├── .github/
│   └── workflows/
├── .goreleaser.yaml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Release process

This project uses GitHub Releases, GoReleaser, and a Homebrew tap.

### Create a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

GoReleaser builds:

```text
monarch_Darwin_arm64.tar.gz
monarch_Darwin_x86_64.tar.gz
monarch_Linux_arm64.tar.gz
monarch_Linux_x86_64.tar.gz
checksums.txt
sbom.json
```

### Homebrew

The Homebrew formula installs the `monarch` binary and runs:

```bash
monarch --version
```

as its formula test.

---

## Roadmap

### v0.1 — Foundation and read-only MVP

- CLI skeleton
- config/env/flags
- output envelope
- error taxonomy
- doctor
- auth/session
- accounts list/show
- transactions list/search/show/summary
- categories, tags, budgets, cashflow summary
- Homebrew development formula
- basic OpenClaw/Hermes docs

### v0.2 — Full read coverage

- holdings
- account history
- account types
- institutions
- recurring transactions
- credit history
- subscription details
- transaction duplicates
- transaction splits
- export JSON/CSV

### v0.3 — Safe mutations

- safety layer
- read-only enforcement
- dry-run plans
- confirmation gates
- audit logs
- transaction/category/tag/budget/account mutations
- attachment upload

### v0.4 — Distribution and agent polish

- GoReleaser
- Homebrew tap
- shell completions
- OpenClaw skill
- Hermes terminal examples
- integration test scenarios

### v0.5 — Cache and MCP preview

- SQLite cache
- cache sync/search/stats
- MCP server
- read-only MCP tools
- Hermes MCP config

### v1.0 — Stable command and JSON contract

- full capability coverage
- stable command tree
- stable JSON schemas
- complete docs
- live smoke tests
- security review
- polished release process

---

## Security

Please do not open public issues containing secrets, session files, request headers, account data, or transaction data.

Report security issues through the process described in [`SECURITY.md`](SECURITY.md).

Security design goals:

- no credentials in logs
- no tokens in audit records
- restrictive session file permissions
- explicit read-only mode
- explicit mutation confirmation
- local-first session and cache storage
- MCP read-only by default

---

## Troubleshooting

Start with:

```bash
monarch doctor --json
```

Common issues:

| Symptom | Recommended action |
|---|---|
| Login fails | Check email/password and MFA configuration |
| MFA fails | Verify TOTP secret or use `--mfa-code` |
| Session expired | Run `monarch auth login` again |
| JSON parser fails | Ensure the command uses `--json` and parse stdout only |
| Agent cannot write | Check `MONARCH_READONLY` and `--confirm` |
| Permission warning | Run `chmod 700 ~/.monarchmoney-cli` and `chmod 600 ~/.monarchmoney-cli/session.json` |
| API response changed | Run `monarch doctor --connect --json` and check project issues |

See [`docs/troubleshooting.md`](docs/troubleshooting.md).

---

## Contributing

Contributions are welcome.

Good first contribution areas:

- command examples
- docs improvements
- golden tests
- mock API fixtures
- shell completions
- Homebrew formula testing
- OpenClaw/Hermes examples
- GraphQL response normalization

### Development flow

1. Fork the repository.
2. Create a feature branch.
3. Add or update tests.
4. Run `make test`.
5. Update docs when command behavior changes.
6. Open a pull request.

### Pull request expectations

A PR should include:

- clear description
- linked issue or motivation
- tests for behavior changes
- docs updates for user-facing changes
- no secrets or real financial data in fixtures

### Code style

```bash
make fmt
make lint
make test
```

---

## License

This project is licensed under the MIT License.

See [`LICENSE`](LICENSE).

---

## Acknowledgements

This project builds on work and ideas from the Monarch Money community.

Special thanks to:

- [`hammem/monarchmoney`](https://github.com/hammem/monarchmoney) — the original Python API project for accessing Monarch Money data.
- [`bradleyseanf/monarchmoneycommunity`](https://github.com/bradleyseanf/monarchmoneycommunity) — the maintained community fork that documents and extends the current unofficial Monarch Money API surface.

`monarchmoney-cli` is a separate implementation. It may reference the behavior, public documentation, and capability surface of those projects, while maintaining its own codebase, CLI contract, safety model, release process, and integration design.

---

## Unofficial project notice

`monarchmoney-cli` is not affiliated with Monarch Money, Inc.

This repository does not contain Monarch Money source code. It is an independently developed tool intended to help authorized users interact with their own Monarch Money data from a local command line environment.

Use at your own risk. Review commands before running them, use `--read-only` for agent workflows, and prefer `--dry-run` before executing remote mutations.

<p align="center">
  <img src="https://cdn.sanity.io/images/mdewiujj/production/7763fb002fce170db0249278bdb8d5e6bfca14c9-3200x2192.png?auto=format&fit=max&q=90&w=1600" alt="Monarch Money reports preview" width="760">
</p>
