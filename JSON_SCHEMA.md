# JSON Output Schema

`monarchmoney-cli` uses a standardized JSON envelope for all structured output. This ensures that AI Agents and automated scripts can reliably parse the results.

## Success Envelope

```json
{
  "ok": true,
  "data": { ... },
  "meta": {
    "command": "accounts.list",
    "profile": "default",
    "duration_ms": 125,
    "schema_version": "2026-05-08",
    "warnings": ["optional deprecation or migration notice"]
  }
}
```

- `ok`: Always `true` for successful operations.
- `data`: The command-specific results (object or array).
- `meta`: Diagnostic information about the request.
- `meta.request_id`: A UUID generated per invocation, identical across every envelope emitted by a single command run.
- `meta.warnings` (optional): Non-fatal notices about deprecated fields or migration advice. Emitted by commands that interact with legacy API fields (e.g., `transactions list`, `accounts history`).

## Error Envelope

```json
{
  "ok": false,
  "error": {
    "code": "AUTH_REQUIRED",
    "message": "not logged in",
    "category": "auth",
    "retryable": false
  },
  "meta": {
    "command": "accounts.list",
    "profile": "default",
    "duration_ms": 10,
    "schema_version": "2026-05-08"
  }
}
```

- `ok`: Always `false` when an error occurs.
- `error.code`: A machine-readable string (e.g., `API_ERROR`, `READ_ONLY_VIOLATION`).
- `error.message`: A human-readable description of the error.
- `error.category`: High-level error grouping (`auth`, `network`, `api`, `validation`, `safety`, `internal`).
- `error.retryable`: Boolean indicating if the operation can be safely retried.
- `error.retry_after_ms`: Present on `RATE_LIMITED` and retryable 5xx errors when the server supplies a `Retry-After` header. Milliseconds to wait before retrying.

## Exit Codes

The process exit code is derived from `error.code` (see `internal/errors`). A successful command exits `0`.

| Exit code | Error code | Category |
|---|---|---|
| 0 | (success) | — |
| 1 | `INTERNAL_ERROR` | internal |
| 1 | `RESOURCE_NOT_FOUND` | api |
| 2 | `INVALID_ARGUMENTS` | validation |
| 3 | `AUTH_REQUIRED` | auth |
| 3 | `AUTH_SESSION_EXPIRED` | auth |
| 3 | `AUTH_MFA_REQUIRED` | auth |
| 3 | `AUTH_MFA_INVALID` | auth |
| 4 | `READ_ONLY_VIOLATION` | safety |
| 5 | `NETWORK_UNREACHABLE` | network |
| 5 | `NETWORK_TIMEOUT` | network |
| 5 | `RATE_LIMITED` | api |
| 6 | `API_ERROR` | api |
| 6 | `API_SCHEMA_CHANGED` | api |
| 6 | `FEATURE_UNAVAILABLE` | api |
| 7 | `VALIDATION_FAILED` | validation |
| 10 | `CONFIRMATION_REQUIRED` | safety |

## Event Stream (NDJSON)

For `accounts refresh --wait`, the CLI emits a stream of progress events when the `--events` flag is set. Each line in the stream is a valid JSON envelope.

```json
{"ok":true,"data":{"status":"syncing","percent":20},"meta":{"command":"accounts.refresh.progress"}}
{"ok":true,"data":{"status":"syncing","percent":80},"meta":{"command":"accounts.refresh.progress"}}
{"ok":true,"data":{"status":"complete"},"meta":{"command":"accounts.refresh"}}
```
