# monarchmoney-cli

<p align="center">
  <img src="public/Monarch-Money-Press-Kit/logo-dark.png" alt="Monarch Money CLI Logo" width="400">
</p>

<p align="center">
  <b>A local, agent-friendly command-line tool for Monarch Money.</b>
</p>

<p align="center">
  <a href="https://github.com/monarchmoney-cli/monarch/actions/workflows/test.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/monarchmoney-cli/monarch/test.yml?branch=main&label=ci"></a>
  <a href="https://github.com/monarchmoney-cli/monarch/releases"><img alt="Release" src="https://img.shields.io/github/v/release/monarchmoney-cli/monarch"></a>
  <a href="https://github.com/monarchmoney-cli/monarch/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/monarchmoney-cli/monarch"></a>
  <a href="https://goreportcard.com/report/github.com/monarchmoney-cli/monarch"><img alt="Go Report" src="https://goreportcard.com/badge/github.com/monarchmoney-cli/monarch"></a>
</p>

---

`monarchmoney-cli` is a production-quality Go implementation of a Monarch Money interface. It allows you to query, manage, and automate your financial data directly from your terminal or via AI Agents.

## ✨ Key Features

- 🤖 **Agent-First**: Stable JSON output, distinct stdout/stderr, and predictable exit codes.
- 🛡️ **Safety First**: Multi-tiered safety model with `--read-only`, `--dry-run`, and `--confirm` gates.
- 📜 **Auditable**: Local JSONL audit logs for every remote mutation.
- ⚡ **Performance**: Single-binary implementation in Go with optional SQLite caching.
- 🧩 **Comprehensive**: Full coverage of accounts, transactions, budgets, cashflow, and more.

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

## 🛡️ Safety & Security

Financial data is sensitive. `monarchmoney-cli` is designed to be **safe by default**:

- **Read-only**: Use `MONARCH_READONLY=1` to block all mutations.
- **Dry-run**: Preview any change with `--dry-run`.
- **Confirmation**: Remote writes require an explicit `--confirm` flag.
- **Local-first**: Your credentials and data stay on your machine.

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

## ⚖️ Disclaimer

`monarchmoney-cli` is an independent, community-maintained project and is **not affiliated with, sponsored by, or endorsed by Monarch Money, Inc.**

## 📄 License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for more information.

---

<p align="center">
  Built with ❤️ for the Monarch Money community.
</p>
