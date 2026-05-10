# monarch Command Reference

## Authentication
- `auth login`: Interactive login to Monarch Money.
- `auth status`: Check session validity and profile.
- `auth logout`: Remove local session data.

## Accounts
- `accounts list`: List all financial accounts.
- `accounts show <id>`: Show detailed account info.
- `accounts holdings <id>`: List investment holdings for an account.
- `accounts balance-at --date YYYY-MM-DD`: Get account display balances for a specific date.
  - `--account-id`: Limit to account IDs (repeatable).
- `accounts history <id>`: Get balance history for an account.
- `accounts refresh`: Trigger a remote sync of all accounts.
- `accounts refresh-status`: Check refresh progress.
- `accounts create-manual`: Create a new manual tracking account.
- `accounts update <id>`: Update account metadata or balance.
- `accounts delete <id>`: Remove an account (requires --confirm).
- `accounts upload-history <id> <file>`: Upload CSV balance history.
- `accounts snapshots`: Get net worth snapshots by account type.
- `accounts aggregate-snapshots`: Get aggregate net worth history.
- `accounts recent-balances`: Get recent balance data for all accounts.
- `accounts types`: List available account types.

## Transactions
- `transactions list`: List recent transactions.
  - `--category-id`: Filter by category ID (repeatable).
  - `--account-id`: Filter by account ID (repeatable).
  - `--tag-id`: Filter by tag ID (repeatable).
  - `--goal-id`: Filter by goal ID (repeatable).
  - `--needs-review`: Filter for transactions needing review.
  - `--has-notes`: Filter for transactions with notes.
  - `--is-split`: Filter for split transactions.
  - `--is-recurring`: Filter for recurring transactions.
  - `--pending true|false`: Filter by pending status.
  - `--hide-from-reports true|false`: Filter by report visibility.
  - `--from`/`--to`: Date range filter.
  - `--limit`/`--offset`: Pagination.
- `transactions search <query>`: Search transactions by merchant or notes.
- `transactions show <id>`: Get detailed information for one transaction.
- `transactions summary`: Get spending summary for a date range.
- `transactions duplicates`: Identify potential duplicate transactions.
- `transactions splits <id>`: View split details for a transaction.
- `transactions split <id>`: Define split categories for a transaction (--file).
- `transactions create`: Manually log a new transaction.
- `transactions update <id>`: Update a transaction.
  - `--notes`: Update notes.
  - `--category`: Set category ID.
  - `--amount`: Update amount.
  - `--date`: Update date.
  - `--merchant`: Update merchant name.
  - `--hide-from-reports`: Hide from reports.
  - `--needs-review`: Set needs-review flag.
  - `--mark-reviewed`: Mark as reviewed (shortcut for --needs-review=false).
- `transactions delete <id>`: Remove a transaction (requires --confirm).
- `transactions bulk-categorize`: Apply a category to multiple transactions.
  - `--id`: Transaction IDs (repeatable).
  - `--category-id`: Category to apply.
  - `--mark-reviewed`: Also mark as reviewed (default: true).
- `transactions export`: Export transactions to JSON or CSV.
  - `--pending true|false`, `--hide-from-reports true|false`, and `--goal-id` use the same filters as `transactions list`.
- `transactions tags set <id>`: Assign tags to a transaction.
- `transactions tags add <id>`: Append tags to a transaction.
- `transactions tags clear <id>`: Remove all tags from a transaction.
- `transactions attachments list <id>`: List files attached to a transaction.
- `transactions attachments download <tx-id> --id <att-id>`: Download an attachment.

## Rules
- `rules list`: List all auto-categorization rules.
- `rules create`: Create a new rule.
  - `--merchant-operator`: Match operator (eq, contains).
  - `--merchant-value`: Merchant name/pattern to match.
  - `--amount-operator`: Amount comparison (gt, lt, eq, between).
  - `--amount-value`: Amount threshold.
  - `--amount-is-expense`: Whether amount is expense (default: true).
  - `--set-category-id`: Category ID to assign.
  - `--add-tag-id`: Tag IDs to add (repeatable).
  - `--account-id`: Limit to account IDs (repeatable).
  - `--apply-to-existing`: Apply to existing transactions.
- `rules update <id>`: Update an existing rule (same flags as create).
- `rules delete <id>`: Delete a rule (requires --confirm).

## Categories & Tags
- `categories list`: List all transaction categories.
- `categories groups`: List category groups.
- `categories create`: Create a new category.
- `categories delete <id>`: Remove a category.
- `categories delete-many --file <file>`: Bulk delete categories.
- `tags list`: List all available tags.
- `tags create`: Create a new transaction tag.

## Budgets & Cashflow
- `budgets list`: View planned vs actual spending by category.
- `budgets show <category-id>`: Show budget for a specific category.
- `budgets set <category-id>`: Update a budget goal.
- `budgets reset`: Reset budget data for a specific month.
- `budgets flexible set`: Set flexible budget amount.
- `budgets flex-rollover set`: Configure flex rollover settings.
- `cashflow list`: Get cashflow records by period.
- `cashflow summary`: High-level income/expense/savings report.
- `cashflow categories`: Spending breakdown by category.
- `cashflow merchants`: Spending breakdown by merchant.
- `cashflow trends`: Aggregate cashflow by category or category group over month, quarter, or year.
  - `--from`/`--to`: Required date range.
  - `--group-by`: `category` or `category-group`.
  - `--period`: `month`, `quarter`, or `year`.
  - `--account-id`/`--category-id`: Optional filters (repeatable).
- `cashflow spending`: Spending summary with totals (income, expenses, net).
- `analyze anomalies`: Find category spending anomalies against historical expense averages.
- `analyze subscriptions`: Summarize recurring subscriptions and potential overlap facts.
- `analyze merchants`: Compare merchant expenses against a previous period.
- `analyze burn-rate`: Compare budget usage with elapsed month time.

## Recurring & Credit
- `recurring list`: View subscription and recurring bills.
- `recurring update <id>`: Update recurring transaction amount.
- `credit history`: View credit score tracking data.

## Goals & Investments
- `goals list`: List goal IDs and names.
- `investments portfolio`: Get portfolio performance and aggregate holdings.
  - `--from`/`--to`: Optional performance date range.
  - `--account-id`: Filter by investment account ID (repeatable).
- `investments performance`: Get historical performance for securities.
  - `--security-id`: Security IDs to include (repeatable, required).
  - `--from`/`--to`: Required date range.
  - `--values`: Include chart value fields.

## Other
- `subscription show`: Show Monarch subscription details.
- `institutions list`: List linked financial institutions.

## System
- `doctor`: Check local environment and connectivity.
- `version`: Print version information.
- `cache sync`: Manually upsert accounts and the latest cached transaction page into the local SQLite cache.
- `cache search`: Search locally cached transactions.
- `cache stats`: View cache utilization.
- `cache cleanup`: Explicitly remove cached transactions before a required cutoff date.
