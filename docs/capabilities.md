# Capabilities

`monarchmoney-cli` is designed to cover the full capability surface of the Monarch Money API.

## Core Domain Coverage

| Domain | Capability | CLI Command |
|---|---|---|
| **Accounts** | List, show details, holdings, history, refresh | `monarch accounts` |
| **Transactions** | List, search, summary, duplicates, splits | `monarch transactions` |
| **Budgets** | List, set, reset, flexible, rollover | `monarch budgets` |
| **Cashflow** | Summary, category/merchant breakdown | `monarch cashflow` |
| **Categories** | List, groups, create, delete | `monarch categories` |
| **Tags** | List, create, set, add, clear | `monarch tags` |
| **Institutions** | List linked financial institutions | `monarch institutions` |
| **Recurring** | List and update recurring transactions | `monarch recurring` |
| **Credit** | Get credit score history | `monarch credit` |
| **Subscription** | Show Monarch subscription details | `monarch subscription` |
| **Attachments** | List, upload, download | `monarch transactions attachments` |

## Read Commands

- `monarch accounts list`: List all accounts.
- `monarch accounts show <id>`: Show detailed account info.
- `monarch accounts history <id>`: Get balance history with `--from`/`--to`.
- `monarch accounts holdings <id>`: List investment holdings.
- `monarch transactions list`: List latest transactions.
- `monarch transactions search <query>`: Search transactions by text.
- `monarch transactions summary`: Get aggregated spending summary.
- `monarch budgets list`: View planned vs actual for a month.
- `monarch cashflow summary`: View income, expenses, and savings rate.

## Mutation and Remote-Action Commands

All mutations are protected by the [Safety Model](./safety.md).

- `monarch auth login`: Authenticate and persist session.
- `monarch accounts refresh`: Trigger a remote sync of all accounts.
- `monarch accounts create-manual`: Create a manual account.
- `monarch transactions update`: Modify transaction notes or category.
- `monarch transactions create`: Manually add a transaction.
- `monarch transactions delete`: Remove a transaction.
- `monarch budgets set`: Set budget amount for a category.
- `monarch transactions tags add`: Append tags to a transaction.
- `monarch transactions attachments upload`: Upload a file to a transaction.

## Safety & Audit

- **Dry-run**: Every mutation supports `--dry-run` to preview changes.
- **Confirmation**: Remote writes require the `--confirm` flag.
- **Read-only**: Use `MONARCH_READONLY=1` to block all mutations.
- **Audit Logs**: Every executed mutation is logged to `~/.monarchmoney-cli/audit/`.
