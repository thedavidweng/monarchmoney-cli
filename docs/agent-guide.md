# Agent Integration Guide

`monarchmoney-cli` is designed from the ground up to be "Agent-friendly." This means it provides stable, predictable interfaces for AI Agents (like Gemini CLI, OpenClaw, or Hermes) to interact with your financial data.

## Core Principles for Agents

### 1. Always use `--json`
Agents should always pass the `--json` flag. This ensures the output is machine-readable and wrapped in a standard success/error envelope.

### 2. Distinct Output Streams
- **stdout**: Contains only the JSON result.
- **stderr**: Contains diagnostic messages, progress updates, and warnings.
Agents should only parse `stdout` for data.

### 3. Safe Defaults
When configuring an agent to use `monarch`, it is recommended to set `MONARCH_READONLY=1` in the environment. This ensures the agent cannot make any changes unless the user explicitly grants permission.

## Integration Examples

### Simple Query
**Goal**: List the top 5 transactions.
**Command**: `monarch transactions list --limit 5 --json`

Use the read-only domain commands when the user asks for point-in-time balances,
goal IDs, investment holdings, or grouped cashflow trends:

```bash
monarch accounts balance-at --date 2026-05-10 --json
monarch cashflow trends --from 2026-01-01 --to 2026-03-31 --group-by category-group --period month --json
monarch goals list --json
monarch investments portfolio --json
```

### Deterministic Analysis
Use `monarch analyze` when the task asks for comparisons, anomaly detection,
subscription totals, merchant changes, or budget burn-rate. These commands do
the grouping and arithmetic in code so agents do not need to infer results from
raw transactions.

Examples:

```bash
monarch analyze anomalies --month 2026-05 --json
monarch analyze subscriptions --json
monarch analyze merchants --compare previous-month --limit 20 --json
monarch analyze burn-rate --month 2026-05 --json
```

The analysis commands are deterministic and read-only. They do not use AI, make
subjective recommendations, or mutate Monarch data.

### Mutation with User Gate
**Goal**: Categorize a transaction.
**Flow**:
1. Agent runs: `monarch transactions update tx_123 --category cat_food --dry-run --json`
2. Agent presents the dry-run plan to the user.
3. If user approves, Agent runs: `monarch transactions update tx_123 --category cat_food --confirm --json`

## Error Handling

Agents should check the `ok` field in the JSON envelope and the process exit code.

| Exit Code | Category | Agent Action |
|---|---|---|
| 3 | Auth Error | Prompt user to run `monarch auth login` |
| 4 | Read-only | Explain that the operation is blocked by security settings |
| 10 | Confirmation | Ask user for explicit permission to use `--confirm` |

## Environment Configuration

For non-interactive agent environments, provide credentials via environment variables:

```bash
export MONARCH_EMAIL="agent-runtime@example.com"
export MONARCH_PASSWORD="..."
export MONARCH_MFA_SECRET="..."
export MONARCH_READONLY="1"
export MONARCH_USER_AGENT="MyAgent/1.0"
```

## Shell Completion

`monarch` supports shell completion for Bash, Zsh, Fish, and PowerShell:

```bash
# Bash
source <(monarch completion bash)

# Zsh
monarch completion zsh > "${fpath[1]}/_monarch"

# Fish
monarch completion fish > ~/.config/fish/completions/monarch.fish

# PowerShell
monarch completion powershell | Out-String | Invoke-Expression
```

## Command Groups

Commands are organized into groups shown in `--help`:

- **Core Commands**: `accounts`, `transactions`, `budgets`, `cashflow`, `rules`, `categories`, `tags`, `goals`, `investments`, `institutions`, `recurring`, `credit`, `subscription`
- **Analysis & Insights**: `analyze`
- **Utilities**: `auth`, `doctor`, `cache`, `audit`, `completion`, `version`
