# Guide: Own your history with a plain-text ledger

**Scenario:** You want your complete financial history out of any subscription, in plain text that outlives Monarch — readable by you, diffable by git, and usable by [hledger](https://hledger.org/) for reports Monarch can't run.

The design is one-way: Monarch stays the source of truth, the CLI copies it into a local SQLite archive (`cache sync`), and the journal is regenerated from scratch from that archive on every run. The journal has **zero sync state** — never edit it; keep annotations in your own file that includes it (shown below).

All output blocks in the guides were captured from real runs of `monarch` and then anonymized: IDs, names, and amounts are replaced, but the output shape is exactly what the tool emits.

## The pipeline

```
Monarch API  --cache sync-->  SQLite archive  --hledger backup-->  monarch.journal
              (the only networked step)        (offline, fast)
```

## Step 0: prerequisites

```console
$ monarch doctor
Monarch Money CLI Doctor
Version: v0.8.1-...
OS/Arch: darwin/arm64
Config Path: /Users/david/.monarchmoney-cli/config.yaml (Exists: false)
Session Path: /Users/david/.monarchmoney-cli/session.json (Exists: true, Auth: true, PermOK: true)
```

If `Auth` shows `false`, run `monarch auth login` first ([Authentication](../auth.md)).

## Step 1: build the archive

```console
$ monarch cache sync --from 2026-08-01 --json
{"ok":true,"data":{"accounts":70,"holdings":3,"status":"sync complete","transactions":21},"meta":{"command":"cache.sync","profile":"default","duration_ms":1020,"schema_version":"2026-05-08","request_id":"284beeaf-5f65-4113-a433-1feb773408ee"}}
```

For the first archive build use `--all` instead of `--from`: it paginates through your entire transaction history, so it takes longer. Later runs are incremental top-ups like the one above (`--limit N` changes page size). The first sync takes a while; later ones upsert.

Cache semantics worth knowing:

- Syncs are **cumulative**: rows returned by later syncs replace matching IDs and add new ones. Old rows are *never* removed automatically — remote deletions do not propagate.
- Prune explicitly when wanted: `monarch cache cleanup --before 2023-01-01`.
- Archives missing newer columns are upgraded in place on the next sync — history is never dropped. Only caches from before the archive schema are rebuilt; run `monarch cache sync --all` once after such a rebuild.
- The cache stores archive-grade detail (tags, splits, review state, category groups, raw merchant names, holdings), not just a query shortcut.

## Step 2: generate the journal

```console
$ monarch hledger backup ~/finances/monarch.journal --json
{"ok":true,"data":{"accounts":69,"file":"/Users/david/finances/monarch.journal","holdings":3,"status":"backup complete","transactions":3394},"meta":{"command":"hledger.backup","profile":"default","duration_ms":194,"schema_version":"2026-05-08","request_id":"d58b98f7-51ff-4203-8398-819209cc9482"}}
```

This reads only from the local cache — no network, no session needed. That makes it safe to re-run constantly and to use offline:

```console
$ monarch hledger backup
Wrote ./monarch.journal (69 accounts, 3394 transactions, 3 holdings).
Warning: 1 account(s) have cached history that does not explain their balance
(total $20.00); run 'monarch cache sync --all' to backfill before relying on
this backup
```

(Output with `backup_path` unset and a feed that predates the oldest cached
transaction.) The backup never silently pretends to be complete: if any
account's cached history does not reconcile to its Monarch balance, or the
cache has not been synced for more than 7 days, the command prints a warning —
and emits it in the JSON envelope's `meta.warnings` — pointing at
`monarch cache sync --all`.

## What's inside the journal

Account names are derived deterministically from Monarch data — type-group prefix plus slugified name — so there is no mapping configuration:

```text
commodity $

account assets:monarch:360-checking-9368
    ; monarch-id: 217017336352937868
account liabilities:monarch:mbna-rewards-world-elite-mastercard-3116
    ; monarch-id: 217017912067299289
```

Every transaction carries the same `monarch-id:` tag, so traceability survives account renames:

```text
2026-07-12 Maple Roast Coffee
    ; monarch-id: 249314440459785996
    expenses:coffee-shops  $6.75
    liabilities:monarch:mbna-rewards-world-elite-mastercard-3116  $-6.75
```

Every transaction also preserves the full Monarch record as hledger comment tags — notes, raw
Plaid/data-provider merchant names, user tags, goal linkage, review state, hide-from-reports and
recurring flags — so nothing observable in the cache is lost when you leave:

```text
2026-07-12 Maple Roast Coffee
    ; monarch-id: 249314440459785996
    ; note: team order
    ; plaid-name: SQ *COFFEE CO
    ; tag: work
    ; recurring:
    expenses:coffee-shops  $6.75
    liabilities:monarch:mbna-rewards-world-elite-mastercard-3116  $-6.75
```

Account declarations carry their Monarch type and lifecycle flags (`; monarch-type:`,
`; monarch-flags: closed,hidden`), and opening investment positions keep each holding's name.

The file ends with a closing-balance assertion per account (including hidden and closed accounts):

```text
2026-08-20 closing balances
    assets:monarch:360-checking-9368  $0.00 = $409.71
    liabilities:monarch:mbna-rewards-world-elite-mastercard-3116  $0.00 = $-1204.55
```

Institution feeds rarely reach back to an account's full lifetime, so cached history often doesn't reconcile to Monarch's current balance. Instead of failing the assertions, the generator emits the difference as a deterministic opening entry through `equity:monarch:opening` — gaps stay visible and auditable rather than silently hidden:

```text
2026-08-20 opening balances
    assets:monarch:advantage-savings-0611  $20.00
    equity:monarch:opening  $-20.00
```

Audit those true-ups any time:

```console
$ hledger reg -f ~/finances/monarch.journal equity:monarch:opening
```

Pending transactions are excluded; transfers are single two-posting entries that never touch P&L categories; splits become multiple postings; investment holdings become one opening-position transaction per brokerage (`N TICKER @ cost = value`) because Monarch's API exposes no per-trade quantity/price history.

## Step 3: extend it with `include`

Hand-editing the generated file is pointless — the next regeneration discards your edits. Keep a small journal of your own that includes it instead:

```text
;; ~/finances/main.journal
include monarch.journal

;; Your own annotations below — future-dated plans, accruals,
;; budgets-in-hledger, anything Monarch doesn't know about.
2026-09-01 * planned property tax
    expenses:housing  $320.00
    assets:monarch:360-checking-9368
```

Run all reports against `main.journal`, never against the generated file. Because regeneration is deterministic, `git diff` between two runs shows exactly what changed — commit both files and your financial history has version control.

## Step 4: keep it current automatically

Set `backup_path` in `~/.monarchmoney-cli/config.yaml`:

```yaml
backup_path: /Users/david/finances/monarch.journal
```

or export `MONARCH_BACKUP_PATH=/Users/david/finances/monarch.journal`. Now every successful sync regenerates the journal; the envelope gains a `data.backup` field:

```json
{"ok":true,"data":{"status":"sync complete","accounts":70,"holdings":3,"transactions":21,"backup":"/Users/david/finances/monarch.journal"},"meta":{"command":"cache.sync","schema_version":"2026-05-08","request_id":"..."}}
```

(The block above is illustrative — same shape as the sync output in step 1 plus `backup`; only a run with `backup_path` set emits that field.) A regeneration failure becomes an entry in `meta.warnings` without failing the sync itself.

A common cadence: `monarch cache sync --all && git -C ~/finances add -A && git -C ~/finances commit -m "monthly ledger refresh"` in a cron job or launchd timer.

## Next steps

- [Configuration](configuration.md) — where `backup_path` and `cache_path` live, precedence rules.
- [Monthly review](monthly-review.md) — what people typically check before refreshing the ledger.
- Design rationale: [ADR-0010](../adr/0010-one-way-regenerating-hledger-backup.md) (one-way regeneration) and [ADR-0011](../adr/0011-cache-as-archive-replica.md) (cache as archive replica).
