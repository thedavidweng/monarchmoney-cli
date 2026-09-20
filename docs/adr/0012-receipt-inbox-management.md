# 0012 - Receipt inbox management

## Status

Accepted.

## Context

The Monarch web app lets users upload receipt images (or forward receipt
emails) to a receipt inbox. Monarch's AI extracts merchant, amount, date,
and line items, then matches each receipt to a posted transaction: the
match adds a `Receipt Import` tag, writes notes, and attaches the image to
the transaction. The CLI only implemented the upload leg
(`Common_CreateBulkRetailSync` -> `POST /retail-sync/{id}/files` ->
`Common_StartRetailSync`), so users could neither inspect what the AI
extracted, download the image, find matched transactions, nor correct or
re-match receipts.

The real operation surface was recovered from web-app traffic as captured
by `erikrubstein/monarch-api2` (May 2026): `Common_RetailSyncsQueryWithTotal`
(`retailSyncsWithTotal`), `Common_RetailSyncQuery` (`retailSync`),
`Common_DeleteRetailSync` (`deleteUnmatchedRetailSync`),
`Common_MatchRetailTransaction`, `Web_UnmatchRetailTransaction`,
`Common_UpdateRetailOrder`, `Common_GetRetailExtensionSettings`, and
`Common_UpdateRetailVendorSettings`. Receipt sources are `user_import`
(scans/uploads) and `email_import` (forwarded mail); statuses are
`completed`, `failed`, `in_progress`, `pending`, `pending_matches`.

## Decision

1. **Mirror the upstream operations verbatim.** Eight new embedded queries
   under `queries/receipts/` use the exact upstream operation names and the
   shared `ReceiptFields on RetailSync` fragment (orders, line items,
   retail transactions with linked transaction IDs, attachments).
2. **Full inbox command surface.** `receipts list` (`--status`, `--source`,
   `--matched`/`--unmatched`, `--limit`/`--offset`), `receipts show`,
   `receipts download` (via the attachment `originalAssetUrl`, reusing the
   transaction-attachment downloader), `receipts delete` (destructive tier),
   `receipts match --transaction` / `receipts unmatch` (resolving the
   `retailTransactionId` through `GetReceipt` first, as upstream does),
   `receipts update` (merchant/date/subtotal/tax/tip/total), and
   `receipts settings` / `receipts settings update` (auto-categorize and
   notes preferences).
3. **Matched-transaction discovery without a new endpoint.** `show`/`list`
   expose `MatchedTransactionIDs()` so agents can join into
   `transactions show`; the notes side is covered by the existing
   `transactions list --has-notes` filter, which `transactions export`
   now also accepts (it previously dropped the flag).

## Consequences

- New public command paths: `receipts.list`, `receipts.show`,
  `receipts.download`, `receipts.delete`, `receipts.match`,
  `receipts.unmatch`, `receipts.update`, `receipts.settings`,
  `receipts.settings.update`. All mutations honor `--dry-run`/`--confirm`.
- The live endpoint suite probes `ListReceipts`, `GetReceipt`, and
  `GetReceiptSettings`, so a Monarch schema change fails fast with the
  endpoint name attached.
- `match`/`unmatch` cost one extra read (`GetReceipt`) per invocation to
  resolve the retail transaction ID; this matches upstream behavior and
  keeps the CLI flags in receipt-ID terms.
