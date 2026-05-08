# Authentication Flow

The `monarch` CLI follows the authentication pattern established by the community Python library.

## 1. Login Procedure
- **Endpoint**: `POST https://api.monarch.com/auth/login/`
- **Payload**:
  ```json
  {
    "username": "email@example.com",
    "password": "yourpassword"
  }
  ```
- **Response**:
  - Success (200 OK): Returns a token and user metadata.
  - MFA Required (401 Unauthorized + Error Code): Indicates that a two-factor code is needed.

## 2. Multi-Factor Authentication (MFA)
- If MFA is enabled, the first login attempt will fail, signaling the need for a code.
- The user provides the 6-digit TOTP code (manually or via `MONARCH_MFA_SECRET`).
- **Endpoint**: `POST https://api.monarch.com/auth/login/` (same endpoint, but with `mfa_token` or similar headers/payload fields as identified in Python code).

## 3. Session Persistence
- Once authenticated, the server returns a token.
- **Header**: `Authorization: Token <token>`
- The CLI stores this token in `~/.monarchmoney-cli/session.json`.
- File permissions must be `0600` to protect the token.

## 4. Session Validity
- `monarch auth status` can verify the token by calling a lightweight GraphQL query like `GetIdentity`.
- If the query returns an unauthorized error, the session is marked as expired.
