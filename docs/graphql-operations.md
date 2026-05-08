# GraphQL Operations

The following GraphQL operations have been identified from the reference implementation for use in the `monarch` CLI.

## Authentication & Identity
- **Login Mutation**: Used in `auth login`.
  - Endpoint: `https://api.monarch.com/auth/login/` (REST-ish POST)
- **GetIdentity**: Basic user/household info.
  - Operation: `query GetIdentity { ... }`

## Accounts
- **GetAccounts**: List all accounts.
  - Operation: `query GetAccounts { accounts { ... } }`
- **GetAccountDetails**: Specific account info.
  - Operation: `query GetAccountDetails($id: ID!) { ... }`
- **RefreshAccounts**: Trigger sync.
  - Operation: `mutation RefreshAccounts { ... }`

## Transactions
- **GetTransactions**: List/Search transactions.
  - Variables: `offset`, `limit`, `search`, `startDate`, `endDate`, etc.
  - Operation: `query GetTransactions($limit: Int!, $offset: Int!, ...) { ... }`
- **GetTransactionDetails**: Single transaction view.
  - Operation: `query GetTransactionDetails($id: ID!) { ... }`

## Budgets
- **GetBudgets**: Monthly budget data.
  - Variables: `month`, `year`.
  - Operation: `query GetBudgets($month: Int!, $year: Int!) { ... }`

## Service Endpoints
- Base API: `https://api.monarch.com/graphql`
- Login: `https://api.monarch.com/auth/login/`
- Uploads: `https://api.monarch.com/account-balance-history/upload/`
