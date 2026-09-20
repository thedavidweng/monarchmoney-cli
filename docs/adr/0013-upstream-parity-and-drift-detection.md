# 0013 - Upstream API parity gaps and drift detection

## Status

Accepted.

## Context

`erikrubstein/monarch-api2` (May 2026) captures the Monarch web app's real
GraphQL surface across 13 domains (~120 functions). The CLI covers a
subset. A function-level audit against that upstream produced three
groups:

**Missing domains (no CLI command at all):** household (7: members, me,
preferences), merchants (4: list/get/update/delete), reports (6: report
data, saved-report CRUD).

**Missing operations in covered domains:** goals (13 of 17: get, create,
update, delete, archive/restore, priorities, account link/unlink, events,
contribute/withdraw, set budget amount); budget (group amounts,
variability/rollover per category and group, list months, settings,
create/clear); categories (catalog, get, reactivate, reorder, group
create/delete/reorder); tags (get/update/delete/reorder); recurring
(get, occurrences, summary, create, remove); investments (holdings
list/get, security search/get, investment accounts, manual-holding CRUD);
transactions (unsplit, single-attachment get, attachment delete,
link-to-goal); receipts settings were added in ADR-0012.

**Divergent operation names (CLI works today but differs from upstream's
capture, so liveness must be proven, not assumed):** budgets
(`Common_GetJointPlanningData` vs `Common_BudgetDataQuery`), cashflow
(`GetCashflowSummary/Categories/Merchants`, `GetAggregatesGraph` vs
`Common_GetCashFlow*Aggregates`), transactions (`GetTransactionsList` vs
`Web_GetTransactionsList`), categories (`ManageGetCategoryGroups` vs
`Common_GetCategoryGroups`), goals budgets (`GetSavingsGoals` vs
`Common_SavingsGoalBudgetAmounts`), accounts show/list
(`AccountDetails_getAccount`/`GetAccounts` vs `Common_GetAccount`/
`Common_GetAccounts`). Neither capture is authoritative: the web app
drifts without notice, and either side may lag.

Separately, offline tests only validate what the repo believes the API
looks like (ADR-0009). Nothing failed in CI when an operation was
renamed, removed, or left orphaned.

## Decision

1. **Parity work proceeds domain by domain**, smallest-closed-loop first
   (tags, then merchants, then reports/household), each slice shipping
   query + service + CLI + tests + docs together. This ADR's gap list is
   the tracking source until slices land; no speculative stubs.
2. **Static drift gate runs on every push/PR.** `scripts/check-graphql-operations.sh`
   fails on duplicate operation names, `.graphql` operations unreferenced
   by Go code, and `OperationName` literals with no query definition (70
   operations green at adoption). `TestNoOrphanQueries` covers the file
   level; the script covers the operation level, including dynamically
   assigned names via string-literal matching.
3. **Live drift gate runs weekly and on demand.** `.github/workflows/api-drift.yml`
   runs the static checks plus `TestLiveEndpointAvailability` when the
   `MONARCH_LIVE_TOKEN` secret is present (the probe step exits 0 early
   without it, so forks stay green), uploads the probe report, and files
   a `needs-triage` issue on failure (deduplicated by title). Only the live gate can
   confirm or clear the divergent-name suspects above. The token is read
   inside the shell step because the `secrets` context is unavailable in
   job-level `if` conditions (a parse error otherwise fails the workflow
   before any job starts).

## Consequences

- A deprecated/removed Monarch operation now fails CI within a week at
  the latest (immediately on PRs that touch queries, via the static
  gate; via the live gate for server-side removals), with the endpoint
  name in the failure output.
- The workflow needs the `MONARCH_LIVE_TOKEN` repository secret to
  provide value beyond static checks; without it the live job skips
  silently and drift can only be caught manually via `mise run test-live`.
- Write probes stay behind `MONARCH_LIVE_WRITES=1` and are not enabled
  in the scheduled workflow, so the drift job is read-only.
