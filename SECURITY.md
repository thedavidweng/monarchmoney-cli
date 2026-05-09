# Security Policy

## Session Security
- The `monarch` CLI stores session tokens in `~/.monarchmoney-cli/session.json`.
- This file is created with `0600` permissions (read/write by owner only).
- The parent directory `~/.monarchmoney-cli/` is created with `0700` permissions.

## Audit Logs
- Every remote mutation (write/delete) is logged in `~/.monarchmoney-cli/audit/`.
- Logs include the command, timestamp, and target resource ID.
- Secrets are redacted from log entries.

## Read-only Mode
- Using the `--read-only` flag or setting `MONARCH_READONLY=1` blocks all mutation commands at the CLI safety layer before any network call is made.

## MFA Support
- Secure password prompting via terminal.
- TOTP generation from secrets stored in environment variables (use with caution).
