# 0009 - Upload endpoints and live endpoint verification

## Status

Accepted.

## Context

The upstream Python client (`bradleyseanf/monarchmoneycommunity`) implements four capabilities the CLI lacked or stubbed:

1. `upload_account_balance_history` — the CLI posted the CSV with `file` + `account_id` form fields and never triggered parsing. Monarch's actual contract is a multipart POST to `/account-balance-history/upload/` with `files` + `account_files_mapping` fields, followed by the `Web_ParseUploadBalanceHistorySession` mutation and polling `Web_GetUploadBalanceHistorySession` until the session reports `completed`. Without the parse session the upload never took effect.
2. `upload_attachment` — the CLI returned `FEATURE_UNAVAILABLE` even though Monarch supports it via `Common_GetTransactionAttachmentUploadInfo` → Cloudinary upload → `Common_AddTransactionAttachment`.
3. `upload_receipt_to_inbox` — not implemented (`Common_CreateBulkRetailSync` → `/retail-sync/{id}/files` → `Common_StartRetailSync`).
4. `find_duplicate_transactions` — already implemented as `transactions duplicates`.

Separately, all offline tests exercise the CLI against mocks or local HTTP doubles, so they validate what we *believe* the Monarch API looks like. Nothing detected when Monarch changes an operation, a field, or an endpoint: drift surfaced only as user-facing runtime failures.

## Decision

1. **Implement the three upload flows exactly as the upstream client performs them**, including the balance-history parse session and completion polling (10s interval, 300s timeout). The attachment upload stub and its `FEATURE_UNAVAILABLE` error path are removed.
2. **Add a live endpoint availability suite** (`internal/monarch/live_test.go`, `mise run test-live`). It skips unless `MONARCH_LIVE_TOKEN` is set; with the variable set it calls every read operation against the real API and fails per-endpoint so schema drift is pinpointed. Setting `MONARCH_LIVE_WRITES=1` additionally exercises the write/upload endpoints (rule create/delete roundtrip, balance-history upload, attachment upload, receipt inbox upload), which mutate the account behind the token.
3. **Add a static orphan-query test** that fails when an embedded `.graphql` file is never loaded through `queries.Get()`, preventing dead query files.

## Consequences

- New public command paths: `receipts.upload`, and `transactions.attachments.upload` now executes instead of erroring.
- The `FEATURE_UNAVAILABLE` error code is retired: no code path emits it anymore. It is removed from `internal/errors`, the exit-code table in JSON_SCHEMA.md, and the errors package tests.
- The live suite requires a real session token and network access; it stays skipped in CI and offline runs.
- With `MONARCH_LIVE_WRITES=1` the probed account receives a probe rule (deleted afterwards), a balance-history row, one transaction attachment, and one receipt-inbox entry.

## Update (2026-10-27)

The orphan-query source-grep test was removed on 2026-10-27; unused query files are reviewed during maintenance.
