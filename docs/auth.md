# Authentication & Session Management

`monarchmoney-cli` uses the unofficial Monarch Money API. It handles authentication, MFA, and session persistence locally and securely.

## Authentication Flow

### Standard Login
Run the following command to start the interactive login process:
```bash
monarch auth login
```
You will be prompted for your email and password.

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

### Checking Status
To check if you have a valid local session:
```bash
monarch auth status
```

### Logging Out
To remove the local session token:
```bash
monarch auth logout
```

## Security Best Practices

1. **Permissions**: Ensure your `~/.monarchmoney-cli` directory has `0700` permissions.
2. **Environment Variables**: For scripts, you can use environment variables instead of interactive prompts:
   - `MONARCH_EMAIL`: Your account email address.
   - `MONARCH_PASSWORD`: Your account password.
   - `MONARCH_MFA_CODE`: A 6-digit MFA code (for single-use scripts).
   - `MONARCH_MFA_SECRET`: Your TOTP secret key for automatic code generation.
3. **Session Safety**: Never share your `session.json` file. It contains a long-lived token that grants access to your Monarch account.
