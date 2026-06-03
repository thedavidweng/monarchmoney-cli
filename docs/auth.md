# Authentication & Session Management

`monarchmoney-cli` uses the unofficial Monarch Money API. It handles authentication, MFA, and session persistence locally and securely.

## Authentication Flow

### Standard Login
Run the following command to start the interactive login process:
```bash
monarch auth login
```
You will be prompted for your email and password.
After login completes, the CLI prints the account email, the login timestamp, and the local session file path so you can confirm exactly which account was stored.

### MFA Support
If your account has Multi-Factor Authentication enabled:
1. The CLI will detect the requirement and prompt you for the 6-digit code interactively.
2. Alternatively, you can provide the code via the `--mfa-code` flag.

**Automatic MFA:**
If you have your TOTP secret key, you can automate the process:
```bash
monarch auth login --email user@example.com --password "..." --mfa-secret "YOUR_SECRET"
```

## Session Persistence

Once authenticated, a session token is stored locally. This token is used for all subsequent commands.

- **Storage Path**: `~/.monarchmoney-cli/session.json`
- **Security**: The file is saved with `0600` permissions (read/write by owner only).
- **Contents**: The session file stores the token, account email, timestamps, and profile metadata needed for `auth status`.

### Checking Status
To check if you have a valid local session:
```bash
monarch auth status
```
This command now performs a live Monarch identity check by default, so it can tell you whether the stored session is still valid.
If the token has expired or been revoked, the command reports `AUTH_SESSION_EXPIRED` and keeps the stored email visible so you know which account needs to be re-authenticated.

### Logging Out
To remove the local session token:
```bash
monarch auth logout
```

## Session Status and Local Commands

- The session token is stored at `~/.monarchmoney-cli/session.json` by default.
- `auth status` reports the stored email, the last login time, and whether the session is still valid.
- Cache-backed commands such as `cache stats`, `cache search`, and `networth` continue to work from local data without requiring a live session check.

## Local Cache Database

The local cache lives at `~/.monarchmoney-cli/cache/monarch.sqlite`.

- It is a standard SQLite database, not AES-256 encrypted.
- The cache file is created with `0600` permissions and the directory is created with `0700` permissions.
- It may contain cached account and transaction data, so treat it as sensitive local data.
- `cache sync` is manual. It upserts all accounts and up to the latest 1000 transactions, optionally filtered with `--from YYYY-MM-DD`.
- The transaction cache is cumulative: rows returned by later syncs replace matching IDs and add new IDs, but old cached rows are not removed automatically.
- The cache is not a complete mirror of Monarch. Remote deletions are not reconciled locally; use `cache cleanup --before YYYY-MM-DD` to explicitly prune old cached transactions.
- If you want stronger at-rest protection, rely on full-disk encryption such as FileVault or store the profile on an encrypted volume.

## Security Best Practices

1. **Permissions**: Ensure your `~/.monarchmoney-cli` directory has `0700` permissions.
2. **Environment Variables**: For scripts, you can use environment variables instead of interactive prompts:
   - `MONARCH_EMAIL`: Your account email address.
   - `MONARCH_PASSWORD`: Your account password.
   - `MONARCH_MFA_CODE`: A 6-digit MFA code (for single-use scripts).
   - `MONARCH_MFA_SECRET`: Your TOTP secret key for automatic code generation.
   - `MONARCH_USER_AGENT`: Override the default HTTP User-Agent string.
3. **Session Safety**: Never share your `session.json` file. It contains a long-lived token that grants access to your Monarch account.

### Credential Security

**Prefer environment variables over CLI flags.** The `--password` and `--mfa-secret` flags expose credentials in the process table (`/proc/PID/cmdline`), which is visible to other processes on the system. Environment variables are safer for scripts and automation:

```bash
# Safer: credentials via environment variables
export MONARCH_PASSWORD="..."
export MONARCH_MFA_SECRET="..."
monarch auth login --email user@example.com

# Less safe: credentials visible in process table
monarch auth login --email user@example.com --password "..." --mfa-secret "..."
```

The interactive prompt (default behavior, no flags) is the most secure option for manual use, as it reads the password via `term.ReadPassword()` without echoing.
