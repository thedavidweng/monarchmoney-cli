# monarchmoney-cli

[![CI](https://img.shields.io/github/actions/workflow/status/thedavidweng/monarchmoney-cli/test.yml?branch=main&label=ci)](https://github.com/thedavidweng/monarchmoney-cli/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/thedavidweng/monarchmoney-cli)](https://github.com/thedavidweng/monarchmoney-cli/releases)
[![License](https://img.shields.io/github/license/thedavidweng/monarchmoney-cli)](https://github.com/thedavidweng/monarchmoney-cli/blob/main/LICENSE)
[![Go Report](https://goreportcard.com/badge/github.com/thedavidweng/monarchmoney-cli)](https://goreportcard.com/report/github.com/thedavidweng/monarchmoney-cli)

Agent-friendly CLI for Monarch Money. Query, manage, and automate your financial data from the terminal or via AI agents.

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

## Disclaimer

`monarchmoney-cli` is an independent, community-maintained project and is **not affiliated with, sponsored by, or endorsed by Monarch Money, Inc.**

## Infrastructure

- **CI/CD:** [cli-workflow-template](https://github.com/thedavidweng/cli-workflow-template) — reusable GitHub Actions workflows
- **Docs:** [site](https://github.com/thedavidweng/site) — landing page and documentation

## License

[Apache License 2.0](LICENSE)
