# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.5.0](https://github.com/thedavidweng/monarchmoney-cli/compare/v0.4.0...v0.5.0) (2026-06-18)


### Features

* add shell completion generation to Homebrew cask ([fba91e0](https://github.com/thedavidweng/monarchmoney-cli/commit/fba91e0fccae01c427e2d8f7d3fb2b433f52ff44))


### Bug Fixes

* address CI lint failure and review feedback ([6cbfb3f](https://github.com/thedavidweng/monarchmoney-cli/commit/6cbfb3fcaa94710210b88b9fda09b2376907cf0a))
* address Go production review findings ([c24158b](https://github.com/thedavidweng/monarchmoney-cli/commit/c24158b37a72039c426ad1fb103e6e02760a50c4))
* address review comments ([ff841de](https://github.com/thedavidweng/monarchmoney-cli/commit/ff841debde85a14cd5ef7e92e454f3416cadf137))
* defer CI inputs until template PR merges ([363a30b](https://github.com/thedavidweng/monarchmoney-cli/commit/363a30bb336950caa07a453539bbdfbb820d9236))
* golangci-lint v2 config and macOS symlink test ([52a6a61](https://github.com/thedavidweng/monarchmoney-cli/commit/52a6a618ae26c9ea06a32eac6852c9fc56af067e))
* move codecov ignore to root level ([ff8e781](https://github.com/thedavidweng/monarchmoney-cli/commit/ff8e7818361ab19e6f62b420dab312b7a1a287fd))
* set include-component-in-tag to false for plain v* tags ([da1c05a](https://github.com/thedavidweng/monarchmoney-cli/commit/da1c05a189786d3e2bffb4b20b7461f560eb69b1))


### Refactoring

* reduce TestStoreRoundTrip cyclomatic complexity for Go Report Card ([fddb3f9](https://github.com/thedavidweng/monarchmoney-cli/commit/fddb3f914b1fd6db67116a5858877141be31f60e))
* use reusable workflows from cli-workflow-template ([2b8e64c](https://github.com/thedavidweng/monarchmoney-cli/commit/2b8e64ca6e8a8904cdf3510e1a873a4486a112d4))


### Documentation

* add Go Report Card badge ([5e04872](https://github.com/thedavidweng/monarchmoney-cli/commit/5e048727d27e85cd54adc8624190e8151afe509e))
* add infrastructure links (CI/CD and docs) ([130740f](https://github.com/thedavidweng/monarchmoney-cli/commit/130740f756e9667a21d54f55f47ab7a24e2e8e84))
* add root-level docs for site sync (COMMANDS.md, JSON_SCHEMA.md, CONTEXT.md) ([71793da](https://github.com/thedavidweng/monarchmoney-cli/commit/71793da94936900757e7f9a56623e0f8090334f0))
* refresh README and fix CI badge ([e3c9814](https://github.com/thedavidweng/monarchmoney-cli/commit/e3c9814d29041c661e11683275506f80fa7c10f6))
* remove duplicate files, inline doc-sync rule ([bd496eb](https://github.com/thedavidweng/monarchmoney-cli/commit/bd496ebec19e4b6642cd36abafeba743fdc87026))
* restructure README to match canvas-cli pattern ([5b8179b](https://github.com/thedavidweng/monarchmoney-cli/commit/5b8179ba87c49e2ec922ce29e3806a7a08e7531e))
* standardize README badges ([76726f7](https://github.com/thedavidweng/monarchmoney-cli/commit/76726f7da041cab814465ffae2a014c51459498f))

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
