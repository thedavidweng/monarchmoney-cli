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
- `meta.request_id` (optional): Reserved for future use.
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
- `error.category`: High-level error grouping (`auth`, `network`, `api`, `validation`, `safety`, `config`, `internal`).
- `error.retryable`: Boolean indicating if the operation can be safely retried.

## Event Stream (NDJSON)

For `accounts refresh --wait`, the CLI emits a stream of progress events when the `--events` flag is set. Each line in the stream is a valid JSON envelope.

```json
{"ok":true,"data":{"status":"syncing","percent":20},"meta":{"command":"accounts.refresh.progress"}}
{"ok":true,"data":{"status":"syncing","percent":80},"meta":{"command":"accounts.refresh.progress"}}
{"ok":true,"data":{"status":"complete"},"meta":{"command":"accounts.refresh"}}
```
