# JSON Output Schema

All commands supporting the `--json` flag wrap their results in a standard envelope.

## Success Envelope
```json
{
  "ok": true,
  "data": { ... },
  "meta": {
    "command": "command.name",
    "profile": "default",
    "duration_ms": 123,
    "schema_version": "2026-05-08",
    "request_id": "uuid"
  }
}
```

## Error Envelope
```json
{
  "ok": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "category": "category",
    "retryable": true
  },
  "meta": { ... }
}
```

## Error Codes
- `AUTH_REQUIRED`: Not logged in or session expired.
- `READ_ONLY_VIOLATION`: Mutation attempted in read-only mode.
- `CONFIRMATION_REQUIRED`: Mutation attempted without `--confirm`.
- `API_ERROR`: Monarch API returned an error or unexpected status.
- `NETWORK_UNREACHABLE`: Could not connect to Monarch servers.
- `VALIDATION_FAILED`: Input arguments or flag values are invalid.
- `RESOURCE_NOT_FOUND`: Specified ID does not exist.
