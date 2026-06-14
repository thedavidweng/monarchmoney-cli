# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.4.0] - 2026-06-12


### ✨ New Features

- Add Cobra shell completion, command groups, examples, flag completions
- Add Windows platform support
- Add XDG_STATE_HOME support on Linux with legacy fallback

### 🐛 Bug Fixes

- Address Greptile review comments
- Update cosign signing to v3 bundle format
- Address Copilot review suggestions
- Address PR review — Windows path safety, perm test, doctor coverage
- Move defaultDirFor to shared paths.go for Windows compilation
- Windows cache tests — Close() for SQLite file locks, skip perm check
- Windows SQLite file locks and cross-platform test compatibility

### 📝 Documentation

- Sync completion command and CLI changes with docs

### 📦 Dependencies

- Maximize GoReleaser capabilities
- Add golangci-lint configuration

### 🔧 Chores

- Bump version to v0.4.0
## [0.3.2] - 2026-06-03


### ♻️ Refactoring

- **cli**: Extract shared command bootstrap and error handling
- **cli**: Centralize mutation lifecycle into deps.Mutate()
- **test**: Consolidate shared test helpers into testutil package

### ✨ New Features

- Release v0.3.2 — production hardening

### 🐛 Bug Fixes

- **test**: Make TestAnalyzeBurnRateJSON date-independent

### 🔧 Chores

- Set up agent skills configuration
- Minimize AGENTS.md with progressive disclosure
- Relicense to Apache License 2.0

### 🧪 Tests

- **cli**: Add coverage for accounts, transactions, budgets, and cashflow commands
- **cli**: Add transactions.show and categories.list coverage
## [0.3.1] - 2026-05-18


### ✨ New Features

- Add monarch butterfly ASCII art to startup
- **tests**: Add E2E CLI tests with auto-discovery and coverage

### 🐛 Bug Fixes

- Add workflow permissions, fix session path injection, show help after banner
- Move banner to version command, restore default help behavior
- Add spacing between banner and version text
- **homebrew**: Add post-install hook to clear quarantine attrs

### 📝 Documentation

- Add Related section linking to money app
## [0.3.0] - 2026-05-10


### ✨ New Features

- Add goals, investments, trends, balance-at, and enrich existing queries

### 📝 Documentation

- Add Monarch Money referral link and fix brew install instructions
- Strip r_source param from referral link

### 🔧 Chores

- Bump version to v0.3.0
## [0.2.2] - 2026-05-09


### 🔧 Chores

- Bump version to v0.2.2
## [0.2.1] - 2026-05-09


### 🐛 Bug Fixes

- Switch back to brews (formula) from homebrew_casks
## [0.2.0] - 2026-05-09


### 🔧 Chores

- Bump version to v0.2.0
## [0.1.4] - 2026-05-09


### ✨ New Features

- Add transaction rules, splits, bulk-categorize, advanced filters, and spending summary

### 🐛 Bug Fixes

- Resolve test report issues - implement attachments list, networth command, remove stubs
## [0.1.2] - 2026-05-09


### ✨ New Features

- Show session path after successful login for transparency

### 🐛 Bug Fixes

- Parse and display detailed API error messages during login
- Update goreleaser homebrew config

### 🔧 Chores

- Fix goreleaser v2 deprecations
- Bump version to 0.1.2-dev
- Release 0.1.2
## [0.1.1] - 2026-05-09


### ✨ New Features

- Add interactive MFA support and env var bindings for login

### 📦 Dependencies

- Upgrade GitHub Actions to node24
## [0.1.0] - 2026-05-09


### ✨ New Features

- Complete Phase 1 - Project Foundation
- Complete Phase 2 - Auth & GraphQL Client
- Complete Phase 3 - Read-only MVP
- Complete Phase 4 - Full Read Coverage
- Complete Phase 5 - Safety & Mutations
- Complete Phase 6 - Quality & Polish
- Complete missing Phase 4 read capabilities (institutions, attachments, search)
- Complete missing Phase 5 mutations (manual accounts, history upload, tx create/split, budget reset)
- Complete Phase 6 - Distribution (GoReleaser, CI, completions)
- Complete Phase 8 - SQLite Cache (sync, search, stats)
- Complete Phase 10 - Hardening & v1.0 (docs, tests, tree freeze)
- Complete go rewrite of monarchmoney-cli with full capability coverage and security features

### 📝 Documentation

- Complete Phase 0 - Discovery & API Inventory

### 📦 Dependencies

- Automate release to homebrew-tap and add go install documentation

### 🔧 Chores

- Apply audit fixes and prepare for history cleanup
- Cleanup .gitignore after history purge
- Release v0.1.0

