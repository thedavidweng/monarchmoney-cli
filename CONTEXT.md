# monarchmoney-cli Domain Glossary

## Core Creed

**monarchmoney-cli is a replacement for the Monarch Money web app**, additionally providing automation convenience and agent-friendliness.

## Core Concepts

**monarchmoney-cli** — A local CLI tool for Monarch Money, used to query, manage, and automate personal finance data. The installed binary is `monarch`.

## Users

**Personal finance user** — Someone who needs to manage their Monarch Money account from the terminal.

**Agent** — An automation agent. Requires deterministic behavior.

## Ledger Backup

**Ledger backup** — A complete hledger journal regenerated wholesale from local data, serving as a plain-text copy of the user's financial history. One-way: Monarch is the source of truth, the journal is a disposable derived artifact.
_Avoid_: sync, import, bidirectional, incremental append

**Plain-text history** — The user goal motivating the ledger backup: owning complete financial history in plain text, independent of Monarch, enabling eventual migration away from Monarch.

**Archive completeness** — The property that a ledger backup can reconstruct the user's financial life without Monarch: full transaction history with all metadata (notes, raw merchant names, tags, splits, review state, goal linkage, hide-from-reports and recurring flags), all accounts (including hidden and closed) with their lifecycle flags, balances, and investment holdings.

**Derived account name** — An hledger account name deterministically generated from Monarch data (account type group + slugified display name), with no user configuration.
_Avoid_: account mapping, mapping file

## Command Design Decisions

**Safety gates** — A three-tier safety mechanism: `--read-only`, `--dry-run`, `--confirm`.

**Audit log** — Every remote write operation is recorded to a JSONL file.

**Cache** — The local SQLite replica of all Monarch data. Source for offline queries and ledger backups; refreshed by cache sync.
_Avoid_: query accelerator
