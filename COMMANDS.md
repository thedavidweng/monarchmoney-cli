# Commands

`monarchmoney-cli` is designed to cover the full capability surface of the Monarch Money API, matching and exceeding the feature set of the [monarch-mcp-server](https://github.com/robcerda/monarch-mcp-server) project.

For task-oriented walkthroughs with real command output, see the [guides](README.md#documentation): [Configuration](docs/guides/configuration.md), [Ledger backup](docs/guides/ledger-backup.md) (`cache`, `hledger`), [Transactions workflow](docs/guides/transactions-workflow.md) (search → dry-run → confirm, splits, tags, rules), [Monthly review](docs/guides/monthly-review.md) (`budgets`, `cashflow`, `analyze`), and [Agent automation](docs/guides/agent-automation.md) (safe unattended setups).

## Core Domain Coverage

| Domain | Capability | CLI Command |
|---|---|---|
| **Accounts** | List, show, dated balances, holdings, history, refresh, net worth snapshots | `monarch accounts` |
| **Transactions** | List/export with advanced filters, search, summary, duplicates, splits, bulk-categorize | `monarch transactions` |
| **Rules** | List, create, update, delete auto-categorization rules | `monarch rules` |
| **Budgets** | List, show, set, reset, flexible, rollover | `monarch budgets` |
| **Cashflow** | Summary, category/merchant breakdown, grouped trends, spending totals | `monarch cashflow` |
| **Overview** | Net worth, cashflow, and recent transactions in one call | `monarch overview` |
| **Analysis** | Deterministic anomaly, subscription, merchant, and budget burn-rate analysis | `monarch analyze` |
| **Categories** | List, show, create, update, rollover, delete, reactivate, reorder, groups | `monarch categories` |
| **Goals** | Full lifecycle, events, contributions, budgets | `monarch goals` |
| **Investments** | Portfolio, accounts, holdings, securities, manual holdings | `monarch investments` |
| **Tags** | List, show, create, update, delete, reorder, set, add, clear | `monarch tags` |
| **Institutions** | List linked financial institutions | `monarch institutions` |
| **Merchants** | List, show, rename, delete | `monarch merchants` |
| **Household** | Show household, members, profile, preferences | `monarch household` |
| **Reports** | Grouped report data, saved reports | `monarch reports` |
| **Recurring** | List, streams, show, summary, update, create, stream-update, remove | `monarch recurring` |
| **Credit** | Get credit score history | `monarch credit` |
| **Subscription** | Show Monarch subscription details | `monarch subscription show` |
| **Attachments** | List, upload, download | `monarch transactions attachments` |
| **Receipts** | List, show, upload, download, delete, match, unmatch, update, settings | `monarch receipts` |
| **Auth** | Login, logout, MFA, session status and management | `monarch auth` |
| **Cache** | Local data cache (sync, search, stats, cleanup) | `monarch cache` |
| **Ledger Backup** | One-way regenerating plain-text ledger for hledger | `monarch hledger` |
| **Audit** | Audit log cleanup and management | `monarch audit` |
| **Doctor** | Verify environment and authentication | `monarch doctor` |
| **Version** | Print version information | `monarch version` |

## Read Commands

- `monarch accounts list`: List all accounts.
- `monarch accounts show <id>`: Show detailed account info.
- `monarch accounts types`: List available account types.
- `monarch accounts balance-at --date YYYY-MM-DD`: Get account balances as of a specific date.
- `monarch accounts history <id>`: Get balance history with `--from`/`--to`.
- `monarch accounts holdings <id>`: List investment holdings.
- `monarch accounts aggregate-snapshots`: Get net worth history over time.
- `monarch accounts snapshots`: Get net worth by account type.
- `monarch accounts recent-balances`: Get recent balances for all accounts.
- `monarch accounts refresh-status`: Check the refresh status of linked accounts.
- `monarch networth`: Top-level alias for `accounts aggregate-snapshots`.
- `monarch transactions list`: List latest transactions with advanced filters.
- `monarch transactions export`: Export transactions with the same pending, report visibility, goal, and notes filters as list.
- `monarch transactions search <query>`: Search transactions by text.
- `monarch transactions show <id>`: Get full transaction details.
- `monarch transactions summary`: Get aggregated spending summary.
- `monarch transactions splits <id>`: View split details for a transaction.
- `monarch transactions duplicates`: Find potential duplicate transactions.
- `monarch transactions attachments list <id>`: List attachments for a transaction.
- `monarch transactions attachments show <id> --id <attachment-id>`: Show an attachment for a transaction.
- `monarch transactions attachments download <id>`: Download attachments for a transaction.
- `monarch transactions attachments delete <id> --id <attachment-id>`: Delete an attachment from a transaction.
- `monarch receipts upload <file>`: Upload a receipt to the Monarch receipt inbox (see Mutation Commands).
- `monarch receipts list`: List receipt inbox entries with `--status`, `--source`, `--matched`/`--unmatched` filters.
- `monarch receipts show <id>`: Show receipt details including matched transaction IDs.
- `monarch receipts download <id>`: Download the receipt image.
- `monarch receipts settings`: Show receipt auto-categorize and notes preferences.
- `monarch rules list`: List all auto-categorization rules.
- `monarch budgets list`: View planned vs actual for a month.
- `monarch budgets show <category-id>`: Show budget details for a category.
- `monarch budgets settings`: Show budget system settings.
- `monarch budgets export`: Export budget data.
- `monarch cashflow summary`: View income, expenses, and savings rate.
- `monarch cashflow spending`: View spending breakdown with totals.
- `monarch cashflow list`: List cashflow transactions.
- `monarch cashflow categories`: View spending by category.
- `monarch cashflow merchants`: View spending by merchant.
- `monarch cashflow trends`: View aggregate trends by category or category group and period.
- `monarch overview`: Get a compact financial overview (net worth, cashflow, recent transactions) for the current month or a custom range via `--from`/`--to`.
- `monarch goals list`: List goals with progress, balance, and target.
- `monarch goals show <id>`: Show a goal.
- `monarch goals budgets`: View savings goal monthly budget amounts.
- `monarch goals budget <id>`: Show monthly budget amounts for a goal.
- `monarch goals events list <id>`: List events for a goal.
- `monarch goals events contribute <id>`: Contribute to a goal from an account.
- `monarch goals events withdraw <id>`: Withdraw from a goal to an account.
- `monarch investments portfolio`: View portfolio performance and holdings.
- `monarch investments performance`: View historical security performance.
- `monarch investments accounts`: List investment (brokerage) accounts.
- `monarch investments holdings list`: List holdings, optionally filtered by account.
- `monarch investments holdings show <id>`: Show a holding.
- `monarch investments securities <query>`: Search securities by name or ticker.
- `monarch investments security <id>`: Show a security.
- `monarch analyze anomalies`: Find category spending anomalies from transaction history.
- `monarch analyze subscriptions`: Summarize recurring subscription costs and potential overlap facts.
- `monarch analyze merchants --compare previous-month`: Compare merchant expenses period-over-period.
- `monarch analyze burn-rate`: Compare budget usage with elapsed month time.
- `monarch recurring list`: View recurring transactions.
- `monarch recurring streams`: List recurring streams with forecast details.
- `monarch recurring show <id>`: Show a recurring stream.
- `monarch recurring summary`: Summarize upcoming recurring income and expenses.
- `monarch credit history`: View credit score history.
- `monarch categories groups`: List category groups.
- `monarch categories show <id>`: Show a category.
- `monarch categories groups update <group-id>`: Update a category group (name, budget variability, rollover settings).
- `monarch categories groups create --name <name>`: Create a category group.
- `monarch categories groups delete <id>`: Delete a category group.
- `monarch categories groups reorder <id> --order N`: Move a category group to a new position.
- `monarch categories rollover <category-id>`: Show rollover settings for a category.
- `monarch institutions list`: List linked financial institutions.
- `monarch merchants list`: List merchants with `--search`, `--limit`, `--offset`, `--order-by` filters.
- `monarch merchants show <id>`: Show merchant details.
- `monarch household show`: Show the current household.
- `monarch household members`: List household members.
- `monarch household member <id>`: Show a household member.
- `monarch household me`: Show the current user profile.
- `monarch household preferences`: Show household preferences.
- `monarch reports data`: Query grouped transaction report data with `--from`/`--to`, `--group-by`, `--timeframe`, `--sort-by`.
- `monarch reports list`: List saved reports.
- `monarch reports show <id>`: Show a saved report.
- `monarch subscription show`: Show Monarch subscription details.
- `monarch auth status`: Check current authentication status.
- `monarch auth session path`: Print the session file path.
- `monarch doctor`: Verify environment, authentication, and API connectivity.
- `monarch hledger backup [FILE]`: Regenerate a complete hledger journal from the local cache (default `./monarch.journal`). Covers all accounts (including hidden and closed, with lifecycle flags), full transaction history with all metadata preserved as comment tags (notes, raw merchant names, tags, goal linkage, review state, hide-from-reports, recurring), closing balance assertions, and investment holdings as opening positions with holding names. Pending transactions are excluded; transfers become single two-posting transactions; every entry carries a `monarch-id:` tag. History gaps (balances not explained by cached transactions) surface as deterministic `opening balances` entries through `equity:monarch:opening`, so assertions always pass while gaps stay auditable. Reads only from the local cache — run `monarch cache sync --all` first for archive-complete history. The command warns when such gaps exist or the cache has not been synced for over 7 days.
- `monarch version`: Print version information.

## Mutation and Remote-Action Commands

All mutations are protected by the [Safety Model](./docs/safety.md).

- `monarch auth login`: Authenticate and persist session.
- `monarch auth logout`: Remove the local session token.
- `monarch accounts refresh [account-id...]`: Trigger a remote sync of all accounts (or specific ones).
- `monarch accounts create-manual`: Create a manual account.
- `monarch accounts update <id>`: Update account name or balance.
- `monarch accounts delete <id>`: Delete an account.
- `monarch accounts upload-history <id>`: Upload balance history for an account.
- `monarch transactions attachments upload <id> <file>`: Upload a file as a transaction attachment.
- `monarch receipts upload <file>`: Upload a receipt to the Monarch receipt inbox; Monarch's AI categorizes and matches it automatically.
- `monarch receipts delete <id>`: Delete an unmatched receipt.
- `monarch receipts match <id> --transaction <tx-id>`: Manually match a receipt to a transaction.
- `monarch receipts unmatch <id>`: Remove the transaction match from a receipt.
- `monarch receipts update <id>`: Correct extracted receipt details (merchant, date, subtotal, tax, tip, total).
- `monarch receipts settings update`: Update receipt auto-categorize and notes preferences.
- `monarch transactions create`: Manually add a transaction.
- `monarch transactions update <id>`: Modify transaction fields (notes, category, amount, date, merchant, hide-from-reports, mark-reviewed).
- `monarch transactions delete <id>`: Remove a transaction.
- `monarch transactions split <id>`: Split a transaction into parts.
- `monarch transactions unsplit <id>`: Remove all splits from a transaction.
- `monarch transactions goal link <id> --goal-id <goal-id>`: Link a transaction to a savings goal.
- `monarch transactions goal unlink <id>`: Remove the savings goal link from a transaction.
- `monarch transactions bulk-categorize`: Apply a category to multiple transactions.
- `monarch goals create --name <name>`: Create a savings goal.
- `monarch goals update <id>`: Update a savings goal.
- `monarch goals delete <id>`: Delete a savings goal.
- `monarch goals archive <id>`: Archive a savings goal.
- `monarch goals restore <id>`: Restore an archived savings goal.
- `monarch goals priorities --id <id...>`: Set goal priority order.
- `monarch goals link-account <id> --account <account-id>`: Link an account balance to a goal.
- `monarch goals unlink-account <id> --account <account-id>`: Unlink an account balance from a goal.
- `monarch goals events contribute <id>`: Contribute to a goal from an account.
- `monarch goals events withdraw <id>`: Withdraw from a goal to an account.
- `monarch goals events update <event-id>`: Update a goal event.
- `monarch goals events delete <event-id>`: Delete a goal event.
- `monarch goals budget set <id> --month YYYY-MM --amount N`: Set a monthly budget amount for a goal.
- `monarch transactions tags set <id>`: Set tags on a transaction.
- `monarch transactions tags add <id>`: Append tags to a transaction.
- `monarch transactions tags clear <id>`: Remove all tags.
- `monarch rules create`: Create an auto-categorization rule.
- `monarch rules update <id>`: Update an existing rule.
- `monarch rules delete <id>`: Delete a rule.
- `monarch budgets set <category-id>`: Set budget amount for a category.
- `monarch budgets set-group <group-id>`: Set budget amount for a category group.
- `monarch budgets create --month YYYY-MM`: Create a budget for a month.
- `monarch budgets clear --month YYYY-MM`: Clear all budget amounts for a month.
- `monarch budgets reset`: Reset budget for a month.
- `monarch budgets reset-rollover --month YYYY-MM (--category-id|--group-id)`: Reset rollover for a category or group.
- `monarch budgets flexible set <category-id>`: Set flexible budget amount.
- `monarch budgets flex-rollover set <category-id>`: Set flex-rollover budget amount.
- `monarch budgets flex-rollover show`: Show flexible budget rollover settings.
- `monarch categories create`: Create a new category.
- `monarch categories update <id>`: Update a category (name, icon, budget variability, exclude from budget).
- `monarch categories delete <id>`: Delete a category.
- `monarch categories reactivate <id>`: Restore a deleted category.
- `monarch categories reorder <id> --group <group-id> --order N`: Move a category within its group.
- `monarch categories delete-many <id...>`: Delete multiple categories.
- `monarch recurring update <id>`: Update a recurring transaction.
- `monarch recurring create --merchant <id>`: Create a recurring stream for a merchant.
- `monarch recurring stream-update <id>`: Update a recurring stream (frequency, amount, date, active).
- `monarch recurring remove <id>`: Mark a stream as not recurring.
- `monarch merchants update <id> --name <name>`: Rename a merchant.
- `monarch merchants delete <id> [--move-to <id>]`: Delete a merchant, optionally moving relations elsewhere.
- `monarch household me update`: Update the current user profile (display name, timezone).
- `monarch household preferences update`: Update household review preferences.
- `monarch reports create --name <name>`: Create a saved report.
- `monarch reports update <id> --name <name>`: Rename a saved report.
- `monarch reports delete <id>`: Delete a saved report.
- `monarch investments holdings create`: Create a manual holding.
- `monarch investments holdings update <id>`: Update a manual holding.
- `monarch investments holdings delete <id>`: Delete a manual holding.
- `monarch tags create`: Create a new tag.
- `monarch tags show <id>`: Show a tag.
- `monarch tags update <id>`: Update a tag name or color.
- `monarch tags delete <id>`: Delete a tag.
- `monarch tags reorder <id> --order N`: Move a tag to a new position.
- `monarch cache sync`: Sync a full-fidelity archive copy of your data into the local cache: accounts (type group, lifecycle flags, current balance), transactions (tags, splits, pending/review state, hide-from-reports and recurring flags, category groups, raw merchant names, goal linkage), and investment holdings. Syncs are cumulative upserts: existing history is preserved and only new or changed rows are written. Archives missing newer columns are upgraded in place without data loss; only pre-archive caches are rebuilt. Use `--limit N` to set page size (default 1000), `--all` to paginate through all matching transactions. When `backup_path` is set in the config file (or `MONARCH_BACKUP_PATH`), every successful sync also regenerates the hledger journal at that path; the JSON envelope then includes a `data.backup` field, and a regeneration failure surfaces as an envelope warning without failing the sync.
- `monarch cache search <query>`: Search transactions in local cache. Matches merchant, notes, category, raw merchant names (Plaid name and data-provider description), and tag names.
- `monarch cache stats`: Show cache statistics including last sync time and holding count.
- `monarch cache cleanup --before YYYY-MM-DD`: Delete old transactions from cache.
- `monarch audit cleanup`: Remove audit log files older than N days (default 30). Use `--older-than N` to customize.
- `monarch completion [bash|zsh|fish|powershell]`: Generate shell completion scripts.

## Safety & Audit

- **Dry-run**: Every mutation supports `--dry-run` to preview changes.
- **Confirmation**: Remote writes require the `--confirm` flag.
- **Read-only**: Use `MONARCH_READONLY=1` to block all mutations.
- **Audit Logs**: Every executed mutation is logged to `~/.monarchmoney-cli/audit/`. Use `monarch audit cleanup --older-than N` to remove logs older than N days (default 30).

## Feature Parity with monarch-mcp-server

This CLI covers all features provided by the [monarch-mcp-server](https://github.com/robcerda/monarch-mcp-server) MCP server:

| monarch-mcp-server Tool | monarchmoney-cli Command |
|---|---|
| `get_accounts` | `accounts list` |
| `get_account_holdings` | `accounts holdings <id>` |
| `get_account_balance_history` | `accounts history <id>` |
| `refresh_accounts` | `accounts refresh` |
| `get_net_worth` | `accounts aggregate-snapshots` |
| `get_net_worth_by_account_type` | `accounts snapshots` |
| `get_transactions` | `transactions list` (with advanced filters) |
| `search_transactions` | `transactions search` |
| `get_transaction_details` | `transactions show <id>` |
| `create_transaction` | `transactions create` |
| `update_transaction` | `transactions update <id>` |
| `delete_transaction` | `transactions delete <id>` |
| `get_transactions_needing_review` | `transactions list --needs-review` |
| `mark_transaction_reviewed` | `transactions update <id> --mark-reviewed` |
| `bulk_categorize_transactions` | `transactions bulk-categorize` |
| `get_transaction_splits` | `transactions splits <id>` |
| `split_transaction` | `transactions split <id>` |
| `get_transaction_rules` | `rules list` |
| `create_transaction_rule` | `rules create` |
| `update_transaction_rule` | `rules update <id>` |
| `delete_transaction_rule` | `rules delete <id>` |
| `get_budgets` | `budgets list` |
| `set_budget_amount` | `budgets set` |
| `get_cashflow` | `cashflow summary` |
| `get_spending_summary` | `cashflow spending` |
| `get_categories` | `categories list` |
| `get_category_groups` | `categories groups` |
| `get_tags` | `tags list` |
| `set_transaction_tags` | `transactions tags set <id>` |
| `create_tag` | `tags create` |
| `get_recurring_transactions` | `recurring list` |
| `get_transactions_summary` | `transactions summary` |
