<p align="center">
  <img src="public/Monarch-Money-Press-Kit/logo-light.png" alt="Monarch Money Logo" width="400">
</p>

<h1 align="center">monarchmoney-cli</h1>

<p align="center">
  Local, agent-friendly command-line tool for <a href="https://monarch.com/referral/w4n85kvije">Monarch Money</a>.
</p>

<p align="center">
  <a href="https://github.com/thedavidweng/monarchmoney-cli/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/thedavidweng/monarchmoney-cli/ci.yml?branch=main&style=flat-square&label=ci" alt="CI"></a>
  <a href="https://github.com/thedavidweng/monarchmoney-cli/releases"><img src="https://img.shields.io/github/v/release/thedavidweng/monarchmoney-cli?style=flat-square" alt="Release"></a>
  <a href="https://github.com/thedavidweng/monarchmoney-cli/blob/main/LICENSE"><img src="https://img.shields.io/github/license/thedavidweng/monarchmoney-cli?style=flat-square" alt="License"></a>
  <img src="https://img.shields.io/badge/go-%3E%3D1.26-blue?style=flat-square" alt="Go">
</p>

`monarchmoney-cli` lets you query, manage, and automate Monarch Money data from the terminal or via AI agents, with stable JSON output and explicit safety gates around financial mutations.

## Highlights

- Agent-first: stable JSON output, distinct stdout/stderr, and predictable exit codes
- Safety-first: `--read-only`, `--dry-run`, and `--confirm` gates for financial data mutations
- Auditable: local JSONL audit logs for remote mutations
- Fast single binary with optional SQLite caching
- Broad Monarch surface: accounts, transactions, budgets, cashflow, rules, splits, recurring items, investments, and more

## Why

Monarch Money is powerful in the browser, but automations and agents need a deterministic command surface. `monarchmoney-cli` exposes account, transaction, budget, rule, and cashflow workflows as auditable commands rather than fragile browser steps.

## Quickstart

### Install

Run the following on macOS or Linux:

```shell
curl -fsSL https://raw.githubusercontent.com/thedavidweng/monarchmoney-cli/main/install.sh | sh
```

Run the following on Windows:

```shell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/thedavidweng/monarchmoney-cli/main/install.ps1 | iex"
```

The installer detects Homebrew automatically and uses it when available (recommended for easy upgrades). Otherwise it downloads the binary to `~/.local/bin`.

<details>
<summary>Other installation methods</summary>

**Homebrew Cask (macOS/Linux):**

```shell
brew install --cask thedavidweng/tap/monarchmoney-cli
```

If you installed the old Homebrew formula, migrate to the cask:

```shell
brew update
brew uninstall --formula thedavidweng/tap/monarchmoney-cli
brew install --cask thedavidweng/tap/monarchmoney-cli
monarch version
```

**Go:**

```shell
go install github.com/thedavidweng/monarchmoney-cli/cmd/monarch@latest
```

**Manual download:** grab the archive for your platform from the [latest GitHub Release](https://github.com/thedavidweng/monarchmoney-cli/releases/latest), extract it, and place the `monarch` binary on your `PATH`.

</details>

### Set up

```shell
monarch doctor
monarch auth login
```

Then try it:

```shell
monarch accounts list --json
monarch transactions search "Amazon" --from 2024-01-01 --json
monarch cashflow spending --from 2024-01-01 --to 2024-01-31 --json
```

### Uninstall

```shell
# Homebrew Cask
brew uninstall --cask thedavidweng/tap/monarchmoney-cli

# install.sh
curl -fsSL https://raw.githubusercontent.com/thedavidweng/monarchmoney-cli/main/install.sh | sh -s uninstall

# Go
rm "$(go env GOPATH)/bin/monarch"
```

Remove config if desired: `rm -rf ~/.config/monarchmoney-cli`

## Documentation

- [Capabilities](docs/capabilities.md) — full list of supported commands and features
- [Authentication](docs/auth.md) — MFA support and session management
- [Safety Model](docs/safety.md) — how we protect your financial data
- [JSON Schema](docs/json-schema.md) — stable output contract details
- [Agent Guide](docs/agent-guide.md) — guide for connecting with AI agents
- [Contributing](CONTRIBUTING.md) — development setup and contribution guidelines

## Related

<p align="center">
  <a href="https://github.com/thedavidweng/money">
    <img src="https://raw.githubusercontent.com/thedavidweng/money/main/public/Golden-Toad-logo.webp" alt="money" width="100">
  </a>
</p>

[`money`](https://github.com/thedavidweng/money) is an open-source, self-hosted, fully local personal finance backend. It borrows the same JSON-first CLI contract and safety gates from `monarchmoney-cli`, but stores data on your own machine with no Monarch subscription required.

## Disclaimer

`monarchmoney-cli` is an independent, community-maintained project and is **not affiliated with, sponsored by, or endorsed by Monarch Money, Inc.**

## Acknowledgements

This project builds on work and ideas from:

- [`hammem/monarchmoney`](https://github.com/hammem/monarchmoney) — the original Python API project for accessing Monarch Money data
- [`bradleyseanf/monarchmoneycommunity`](https://github.com/bradleyseanf/monarchmoneycommunity) — the maintained community fork documenting the current unofficial Monarch Money API surface

## Infrastructure

- **CI/CD:** [cli-workflow-template](https://github.com/thedavidweng/cli-workflow-template) — reusable GitHub Actions workflows
- **Docs:** [site](https://github.com/thedavidweng/site) — landing page and documentation

## License

[Apache License 2.0](LICENSE)

<p align="center">
  Built for the Monarch Money community.
</p>
