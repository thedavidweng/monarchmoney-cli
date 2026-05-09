# monarchmoney-cli

<p align="center">
  <img src="public/Monarch-Money-Press-Kit/logo-light.png" alt="Monarch Money Logo" width="400">
</p>

<p align="center">
  <b>A local, agent-friendly command-line tool for Monarch Money.</b>
</p>

<p align="center">
  <a href="https://github.com/thedavidweng/monarchmoney-cli/actions/workflows/test.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/thedavidweng/monarchmoney-cli/test.yml?branch=main&label=ci"></a>
  <a href="https://github.com/thedavidweng/monarchmoney-cli/releases"><img alt="Release" src="https://img.shields.io/github/v/release/thedavidweng/monarchmoney-cli"></a>
  <a href="https://github.com/thedavidweng/monarchmoney-cli/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/thedavidweng/monarchmoney-cli"></a>
  <a href="https://goreportcard.com/report/github.com/thedavidweng/monarchmoney-cli"><img alt="Go Report" src="https://goreportcard.com/badge/github.com/thedavidweng/monarchmoney-cli"></a>
</p>

---

`monarchmoney-cli` is a production-focused Go implementation of a Monarch Money interface. It allows you to query, manage, and automate your financial data directly from your terminal or via AI Agents.

## ✨ Key Features

- 🤖 **Agent-First**: Stable JSON output, distinct stdout/stderr, and predictable exit codes.
- 🛡️ **Safety First**: Multi-tiered safety model with `--read-only`, `--dry-run`, and `--confirm` gates.
- 📜 **Auditable**: Local JSONL audit logs for every remote mutation.
- ⚡ **Performance**: Single-binary implementation in Go with optional SQLite caching.
- 🧩 **Comprehensive**: Coverage across accounts, transactions, budgets, cashflow, and more.

## 🚀 Quick Start

### 1. Install

**Via Homebrew (macOS/Linux):**
```bash
brew tap thedavidweng/homebrew-tap
brew install monarchmoney-cli
```

**Via Go (Cross-platform):**
```bash
go install github.com/thedavidweng/monarchmoney-cli/cmd/monarch@latest
```

### 2. Verify Environment
```bash
monarch doctor
```

### 3. Login
```bash
monarch auth login
```

### 4. Query Data
```bash
# List all accounts in JSON format
monarch accounts list --json

# Search for transactions
monarch transactions search "Amazon" --from 2024-01-01 --json
```

## 📖 Documentation

Detailed guides are available in the [`/docs`](./docs) directory:

- 🛠️ **[Capabilities](./docs/capabilities.md)**: Full list of supported commands and features.
- 🔐 **[Safety Model](./docs/safety.md)**: How we protect your financial data.
- 🔑 **[Authentication](./docs/auth.md)**: MFA support and session management.
- 🤖 **[Agent Integration](./docs/agent-guide.md)**: Guide for connecting with AI Agents.
- 📊 **[JSON Schema](./docs/json-schema.md)**: Stable output contract details.

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## ⚖️ Disclaimer

`monarchmoney-cli` is an independent, community-maintained project and is **not affiliated with, sponsored by, or endorsed by Monarch Money, Inc.**

## 📑 Acknowledgements

This project builds on work and ideas from the following projects:

- [`hammem/monarchmoney`](https://github.com/hammem/monarchmoney) — The original Python API project for accessing Monarch Money data.
- [`bradleyseanf/monarchmoneycommunity`](https://github.com/bradleyseanf/monarchmoneycommunity) — The maintained community fork that documents and extends the current unofficial Monarch Money API surface.

## 📄 License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for more information.

---

<p align="center">
  Built with ❤️ for the Monarch Money community.
</p>
