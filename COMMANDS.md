# monarch Command Reference

## Authentication
- `auth login`: Interactive login to Monarch Money.
- `auth status`: Check session validity and profile.
- `auth logout`: Remove local session data.

## Accounts
- `accounts list`: List all financial accounts.
- `accounts holdings <id>`: List investment holdings for an account.
- `accounts history <id>`: Get balance history for an account.
- `accounts refresh`: Trigger a remote sync of all accounts.
- `accounts create-manual`: Create a new manual tracking account.
- `accounts update <id>`: Update account metadata or balance.
- `accounts delete <id>`: Remove an account (requires --confirm).
- `accounts upload-history <id> <file>`: Upload CSV balance history.

## Transactions
- `transactions list`: List recent transactions.
- `transactions search <query>`: Search transactions by merchant or notes.
- `transactions show <id>`: Get detailed information for one transaction.
- `transactions summary`: Get spending summary for a date range.
- `transactions duplicates`: Identify potential duplicate transactions.
- `transactions splits <id>`: View split details for a transaction.
- `transactions create`: Manually log a new transaction.
- `transactions update <id>`: Update transaction notes or category.
- `transactions delete <id>`: Remove a transaction (requires --confirm).
- `transactions split <id>`: Define split categories for a transaction.
- `transactions export`: Export transactions to JSON or CSV.
- `transactions tags set <id>`: Assign tags to a transaction.
- `transactions attachments list <id>`: List files attached to a transaction.
- `transactions attachments download <tx-id> --id <att-id>`: Download an attachment.

## Categories & Tags
- `categories list`: List all transaction categories.
- `categories create`: Create a new category.
- `categories delete <id>`: Remove a category.
- `categories delete-many --file <file>`: Bulk delete categories.
- `tags list`: List all available tags.
- `tags create`: Create a new transaction tag.

## Budgets & Cashflow
- `budgets list`: View planned vs actual spending by category.
- `budgets set <category-id>`: Update a budget goal.
- `budgets reset`: Reset budget data for a specific month.
- `cashflow summary`: High-level income/expense/savings report.
- `cashflow categories`: Spending breakdown by category.
- `cashflow merchants`: Spending breakdown by merchant.

## Recurring & Credit
- `recurring list`: View subscription and recurring bills.
- `credit history`: View credit score tracking data.

## System
- `doctor`: Check local environment and connectivity.
- `version`: Print version information.
- `cache sync`: Update local SQLite cache.
- `cache search`: Search locally cached data.
- `cache stats`: View cache utilization.
