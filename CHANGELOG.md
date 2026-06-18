# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.5.0](https://github.com/thedavidweng/monarchmoney-cli/compare/monarchmoney-cli-v0.4.0...monarchmoney-cli-v0.5.0) (2026-06-18)


### Features

* add Cobra shell completion, command groups, examples, flag completions ([dba40ae](https://github.com/thedavidweng/monarchmoney-cli/commit/dba40ae46d747ac7788ccf1bafaa026252305faa))
* add goals, investments, trends, balance-at, and enrich existing queries ([019cb86](https://github.com/thedavidweng/monarchmoney-cli/commit/019cb86282d04a6c8d27d7639e4839807cc370f5))
* add interactive MFA support and env var bindings for login ([40e7e86](https://github.com/thedavidweng/monarchmoney-cli/commit/40e7e86d48baf22cb314e72f2c6cccca41dc8932))
* add monarch butterfly ASCII art to startup ([98bd7a1](https://github.com/thedavidweng/monarchmoney-cli/commit/98bd7a1e8c5f8dacfab6b46ac3260983d6df84ab))
* add shell completion generation to Homebrew cask ([fba91e0](https://github.com/thedavidweng/monarchmoney-cli/commit/fba91e0fccae01c427e2d8f7d3fb2b433f52ff44))
* add transaction rules, splits, bulk-categorize, advanced filters, and spending summary ([4aa9ad3](https://github.com/thedavidweng/monarchmoney-cli/commit/4aa9ad3a99ab4bb9fda79553810cb352e931121b))
* add Windows platform support ([8d9d974](https://github.com/thedavidweng/monarchmoney-cli/commit/8d9d97444e73ad7bca58a844caaa9f10f12b2fca))
* add XDG_STATE_HOME support on Linux with legacy fallback ([2d2c3fd](https://github.com/thedavidweng/monarchmoney-cli/commit/2d2c3fd63870b4e74aa5040444d741f77e2e1144))
* complete go rewrite of monarchmoney-cli with full capability coverage and security features ([dbad114](https://github.com/thedavidweng/monarchmoney-cli/commit/dbad1148d89a76111f59eae493ef0484d9972860))
* complete missing Phase 4 read capabilities (institutions, attachments, search) ([4083eff](https://github.com/thedavidweng/monarchmoney-cli/commit/4083eff0ded84bb64eb37f31c2a58f542cfa631f))
* complete missing Phase 5 mutations (manual accounts, history upload, tx create/split, budget reset) ([a23eb5a](https://github.com/thedavidweng/monarchmoney-cli/commit/a23eb5a6ec960513ad5d95cf0f77bb5f3e2190fb))
* complete Phase 1 - Project Foundation ([c321ce1](https://github.com/thedavidweng/monarchmoney-cli/commit/c321ce112bdf3250a37c471f7bd977413b642e63))
* complete Phase 10 - Hardening & v1.0 (docs, tests, tree freeze) ([6e8ff20](https://github.com/thedavidweng/monarchmoney-cli/commit/6e8ff20eedd968e8b8dd50e0cb032634ad577fe9))
* complete Phase 2 - Auth & GraphQL Client ([92f72dd](https://github.com/thedavidweng/monarchmoney-cli/commit/92f72dd116e4e0e5590e826edea1cd80a1b535a4))
* complete Phase 3 - Read-only MVP ([25b0fd4](https://github.com/thedavidweng/monarchmoney-cli/commit/25b0fd4522b97e9b4d137fed062f1f978b4ef8b5))
* complete Phase 4 - Full Read Coverage ([0607939](https://github.com/thedavidweng/monarchmoney-cli/commit/0607939beb90eaaf329fd132c270f2b6dc1c9a71))
* complete Phase 5 - Safety & Mutations ([64dff5b](https://github.com/thedavidweng/monarchmoney-cli/commit/64dff5b8928b89b93b7b8ae8854df2fcdca9dc2b))
* complete Phase 6 - Distribution (GoReleaser, CI, completions) ([6345d96](https://github.com/thedavidweng/monarchmoney-cli/commit/6345d96c6332fe21e9b030c1cf4d904a019b976c))
* complete Phase 6 - Quality & Polish ([6a5763c](https://github.com/thedavidweng/monarchmoney-cli/commit/6a5763cfc22b071e95df019d09baebd647d8d62a))
* complete Phase 8 - SQLite Cache (sync, search, stats) ([dc2ff61](https://github.com/thedavidweng/monarchmoney-cli/commit/dc2ff6164aedf622fa2fd1a08353950ff684e118))
* release v0.3.2 — production hardening ([a15bc52](https://github.com/thedavidweng/monarchmoney-cli/commit/a15bc52255406160569e001974e5383ec3e16285))
* show session path after successful login for transparency ([3a5bb66](https://github.com/thedavidweng/monarchmoney-cli/commit/3a5bb6666891e353dd575c13c6536771c32a0646))
* **tests:** add E2E CLI tests with auto-discovery and coverage ([329d50d](https://github.com/thedavidweng/monarchmoney-cli/commit/329d50d30b1189ce28ff15207412b1e73e903219))


### Bug Fixes

* add spacing between banner and version text ([15fe441](https://github.com/thedavidweng/monarchmoney-cli/commit/15fe4411c806021ed7aa7ff7d341ec0b636dd20f))
* add workflow permissions, fix session path injection, show help after banner ([6a353ce](https://github.com/thedavidweng/monarchmoney-cli/commit/6a353ce145841105858b3928950e02ac06c576bc))
* address CI lint failure and review feedback ([6cbfb3f](https://github.com/thedavidweng/monarchmoney-cli/commit/6cbfb3fcaa94710210b88b9fda09b2376907cf0a))
* address Copilot review suggestions ([2d36adc](https://github.com/thedavidweng/monarchmoney-cli/commit/2d36adcb5263b0313b41c91d1bddd672f1bca847))
* address Go production review findings ([c24158b](https://github.com/thedavidweng/monarchmoney-cli/commit/c24158b37a72039c426ad1fb103e6e02760a50c4))
* address Greptile review comments ([5652aa2](https://github.com/thedavidweng/monarchmoney-cli/commit/5652aa2cdf63604008dbf261411214dc623a153b))
* address PR review — Windows path safety, perm test, doctor coverage ([c2065e5](https://github.com/thedavidweng/monarchmoney-cli/commit/c2065e5f0b28798dc18b6a6d929f7d1740f1bddb))
* address review comments ([ff841de](https://github.com/thedavidweng/monarchmoney-cli/commit/ff841debde85a14cd5ef7e92e454f3416cadf137))
* defer CI inputs until template PR merges ([363a30b](https://github.com/thedavidweng/monarchmoney-cli/commit/363a30bb336950caa07a453539bbdfbb820d9236))
* golangci-lint v2 config and macOS symlink test ([52a6a61](https://github.com/thedavidweng/monarchmoney-cli/commit/52a6a618ae26c9ea06a32eac6852c9fc56af067e))
* **homebrew:** add post-install hook to clear quarantine attrs ([84189fe](https://github.com/thedavidweng/monarchmoney-cli/commit/84189fe21ae7d5b23842b77dc6cfbdf8a65c26e3))
* move banner to version command, restore default help behavior ([5bc89e0](https://github.com/thedavidweng/monarchmoney-cli/commit/5bc89e0ae195bb0060fa26c8ca6e8f017cd7af88))
* move codecov ignore to root level ([ff8e781](https://github.com/thedavidweng/monarchmoney-cli/commit/ff8e7818361ab19e6f62b420dab312b7a1a287fd))
* move defaultDirFor to shared paths.go for Windows compilation ([154428a](https://github.com/thedavidweng/monarchmoney-cli/commit/154428a98343fef120179ee39ab018b0fdadaded))
* parse and display detailed API error messages during login ([38393bd](https://github.com/thedavidweng/monarchmoney-cli/commit/38393bd680b3e2c6246529527cbb2d89b26c685d))
* resolve test report issues - implement attachments list, networth command, remove stubs ([4c982d5](https://github.com/thedavidweng/monarchmoney-cli/commit/4c982d539b2d744d00fc1e9b2893658c4e16ad8c))
* switch back to brews (formula) from homebrew_casks ([ddc8ec7](https://github.com/thedavidweng/monarchmoney-cli/commit/ddc8ec7cf0b4fb3c9f200c9b86c0c4d37899d98d))
* **test:** make TestAnalyzeBurnRateJSON date-independent ([bed90f5](https://github.com/thedavidweng/monarchmoney-cli/commit/bed90f5c6257ae0b8484a5308616bc6b0656cb9f))
* update cosign signing to v3 bundle format ([bdeca97](https://github.com/thedavidweng/monarchmoney-cli/commit/bdeca979a699e0d3d7fa17032f552146f5946d06))
* update goreleaser homebrew config ([9c649cb](https://github.com/thedavidweng/monarchmoney-cli/commit/9c649cb4f09c296ce68a9b07c2039e1a0f3b3ea1))
* Windows cache tests — Close() for SQLite file locks, skip perm check ([2b81874](https://github.com/thedavidweng/monarchmoney-cli/commit/2b8187464504c24e33d7c8fd967c1305c9ab6ba2))
* Windows SQLite file locks and cross-platform test compatibility ([e59e382](https://github.com/thedavidweng/monarchmoney-cli/commit/e59e382a57a7398a7cb7e12feb6d54529edd20e8))


### Refactoring

* **cli:** centralize mutation lifecycle into deps.Mutate() ([856ae28](https://github.com/thedavidweng/monarchmoney-cli/commit/856ae28de39db6d5b96794f7602d69684af6dea9))
* **cli:** extract shared command bootstrap and error handling ([8c8deb4](https://github.com/thedavidweng/monarchmoney-cli/commit/8c8deb4269d746d8fa1df231d994987ba8f9a89a))
* reduce TestStoreRoundTrip cyclomatic complexity for Go Report Card ([fddb3f9](https://github.com/thedavidweng/monarchmoney-cli/commit/fddb3f914b1fd6db67116a5858877141be31f60e))
* **test:** consolidate shared test helpers into testutil package ([876ce58](https://github.com/thedavidweng/monarchmoney-cli/commit/876ce58c940a35f52c74045674cc5b60ed81ec9e))
* use reusable workflows from cli-workflow-template ([2b8e64c](https://github.com/thedavidweng/monarchmoney-cli/commit/2b8e64ca6e8a8904cdf3510e1a873a4486a112d4))


### Documentation

* add Go Report Card badge ([5e04872](https://github.com/thedavidweng/monarchmoney-cli/commit/5e048727d27e85cd54adc8624190e8151afe509e))
* add infrastructure links (CI/CD and docs) ([130740f](https://github.com/thedavidweng/monarchmoney-cli/commit/130740f756e9667a21d54f55f47ab7a24e2e8e84))
* add Monarch Money referral link and fix brew install instructions ([632c184](https://github.com/thedavidweng/monarchmoney-cli/commit/632c1846cf1005910fdbf92720b599ad98109512))
* add Related section linking to money app ([dc98c9e](https://github.com/thedavidweng/monarchmoney-cli/commit/dc98c9e1e521a8eff9d1995700c5c1a9b794c15a))
* add root-level docs for site sync (COMMANDS.md, JSON_SCHEMA.md, CONTEXT.md) ([71793da](https://github.com/thedavidweng/monarchmoney-cli/commit/71793da94936900757e7f9a56623e0f8090334f0))
* complete Phase 0 - Discovery & API Inventory ([6d03d3b](https://github.com/thedavidweng/monarchmoney-cli/commit/6d03d3b4e11983e8ceb2cc6db68172dff2f9e64b))
* refresh README and fix CI badge ([e3c9814](https://github.com/thedavidweng/monarchmoney-cli/commit/e3c9814d29041c661e11683275506f80fa7c10f6))
* remove duplicate files, inline doc-sync rule ([bd496eb](https://github.com/thedavidweng/monarchmoney-cli/commit/bd496ebec19e4b6642cd36abafeba743fdc87026))
* restructure README to match canvas-cli pattern ([5b8179b](https://github.com/thedavidweng/monarchmoney-cli/commit/5b8179ba87c49e2ec922ce29e3806a7a08e7531e))
* standardize README badges ([76726f7](https://github.com/thedavidweng/monarchmoney-cli/commit/76726f7da041cab814465ffae2a014c51459498f))
* strip r_source param from referral link ([2e96a5e](https://github.com/thedavidweng/monarchmoney-cli/commit/2e96a5eac83b4e2fa54c6d0a4ea812d67fa0aeaa))
* sync completion command and CLI changes with docs ([72fa6c0](https://github.com/thedavidweng/monarchmoney-cli/commit/72fa6c041e6d958d55e4378027bb6c80f0edae40))

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
