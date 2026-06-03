# Safety Model

`monarchmoney-cli` treats financial data with the highest level of caution. It implements a multi-tiered safety model to prevent accidental or unauthorized modifications.

## Operation Tiers

Operations are classified into tiers based on their impact:

| Tier | Description | Examples | Security Requirement |
|---|---|---|---|
| **Read** | Fetching data from API | `list`, `show`, `search` | Always allowed |
| **Remote Action** | Triggering remote sync | `accounts refresh` | Blocked by Read-only |
| **Mutation** | Modifying remote data | `update`, `set`, `create` | Requires `--confirm` |
| **Destructive** | Deleting remote data | `delete`, `reset` | Requires `--confirm` |

## Safety Layers

### 1. Read-only Mode
Read-only mode blocks all operations that are not in the **Read** tier. This is ideal for CI/CD environments or AI Agent integrations where you want to ensure no changes are made.

**Enable via flag:**
```bash
monarch accounts refresh --read-only
# Error: [READ_ONLY_VIOLATION] remote writes are blocked in read-only mode
```

**Enable via environment variable:**
```bash
export MONARCH_READONLY=1
monarch transactions delete tx_123 --confirm
# Error: [READ_ONLY_VIOLATION] remote writes are blocked in read-only mode
```

### 2. Dry-run Previews
Before executing a mutation, you can use the `--dry-run` flag to see exactly what would happen. This generates a "Mutation Plan" but does not send the final request to the API.

```bash
monarch budgets set --category cat_123 --amount 500 --dry-run
```

### 3. Explicit Confirmation
By default, any mutation or destructive command will fail if the `--confirm` flag is missing. This prevents accidental execution of dangerous commands.

```bash
monarch transactions delete tx_123
# Error: [CONFIRMATION_REQUIRED] this destructive operation requires --confirm to execute
```

## Audit Logging

Every mutation that is successfully executed (or attempted with `--confirm`) is logged locally for your review.

**Log Path:** `~/.monarchmoney-cli/audit/YYYY-MM-DD.jsonl`

**Log Format:**
```json
{
  "timestamp": "2026-05-08T22:12:00Z",
  "command": "transactions.update",
  "resource_id": "tx_123",
  "dry_run": false,
  "confirmed": true,
  "profile": "default",
  "result": "success",
  "error_code": ""
}
```

Audit records **never** contain sensitive credentials, session tokens, or private financial details (like account numbers).
