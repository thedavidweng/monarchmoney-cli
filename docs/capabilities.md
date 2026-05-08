# Capability Matrix

This document maps the capabilities from the reference Python library to the `monarch` CLI commands.

## Authentication
| Source Capability | CLI Command | Safety Tier | Phase |
|-------------------|-------------|-------------|-------|
| interactive login | `auth login` | auth | P0 |
| non-interactive login | `auth login --email` | auth | P0 |
| MFA code support | `auth login --mfa-code` | auth | P0 |
| TOTP from secret | `auth login --mfa-secret` | auth | P0 |
| Session persistence | `auth status` | auth | P0 |

## Accounts
| Source Capability | CLI Command | Safety Tier | Phase |
|-------------------|-------------|-------------|-------|
| get_accounts | `accounts list` | read | P1 |
| get_account_holdings | `accounts holdings` | read | P1 |
| get_account_history | `accounts history` | read | P1 |
| request_accounts_refresh | `accounts refresh` | remote_action | P1 |
| is_accounts_refresh_complete | `accounts refresh-status` | read | P1 |
| create_manual_account | `accounts create-manual` | mutation | P2 |
| update_account | `accounts update` | mutation | P2 |
| delete_account | `accounts delete` | destructive | P2 |

## Transactions
| Source Capability | CLI Command | Safety Tier | Phase |
|-------------------|-------------|-------------|-------|
| get_transactions | `transactions list/search` | read | P1 |
| get_transaction_details | `transactions show` | read | P1 |
| get_transactions_summary | `transactions summary` | read | P1 |
| find_duplicate_transactions | `transactions duplicates` | read | P1 |
| create_transaction | `transactions create` | mutation | P2 |
| update_transaction | `transactions update` | mutation | P2 |
| delete_transaction | `transactions delete` | destructive | P2 |
| update_transaction_splits | `transactions split` | mutation | P2 |

## Categories & Tags
| Source Capability | CLI Command | Safety Tier | Phase |
|-------------------|-------------|-------------|-------|
| get_transaction_categories | `categories list` | read | P1 |
| get_transaction_category_groups | `categories groups` | read | P1 |
| create_transaction_category | `categories create` | mutation | P2 |
| delete_transaction_category | `categories delete` | destructive | P2 |
| get_transaction_tags | `tags list` | read | P1 |
| set_transaction_tags | `transactions tags set` | mutation | P2 |

## Budgets & Cashflow
| Source Capability | CLI Command | Safety Tier | Phase |
|-------------------|-------------|-------------|-------|
| get_budgets | `budgets list` | read | P1 |
| set_budget_amount | `budgets set` | mutation | P2 |
| get_cashflow_summary | `cashflow summary` | read | P1 |
| get_cashflow | `cashflow list` | read | P1 |
