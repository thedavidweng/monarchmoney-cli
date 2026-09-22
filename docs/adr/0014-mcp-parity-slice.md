# 0014 - MCP-server parity slice: debt, rule order, sync health, whoami, stream review, goal contributions

## Status

Accepted.

## Context

`robcerda/monarch-mcp-server` (58 tools) maps six operations the CLI lacked:
debt paydown (`GetDebtPaydown`), rule reorder
(`updateTransactionRuleOrderV2`), credential sync health
(`GetCredentialSyncHealth`), identity plus plan capabilities (`GetWhoAmI`
with a separate `businessEntities` probe), recurring-stream review
(`reviewRecurringStream`), and per-account goal contributions
(`monthlyBudgetAmounts{accountBreakdown}` with
`Common_UpdateSavingsGoal(accountBudgetAmounts)`).

All six were verified live against `https://api.monarch.com/graphql`
before implementation: four reads returned real data, the reorder
mutation executed as a no-op, the review mutation answered
`Stream not found` for a bogus id, and the contribution read/write
path completed a create/set/read/delete roundtrip with no residue.
Two upstream texts could not be used verbatim: the reorder payload
has no `errors` field (selecting it returns HTTP 400), and goal
creation requires `type` (omitting it returns HTTP 400).

## Decision

1. Six closed loops ship query + service + CLI + mock tests + live
   probes together: `debt paydown`, `rules reorder`, `institutions
   health`, top-level `whoami`, `recurring review`, `goals
   contributions` with nested `contributions set`.
2. The capabilities probe stays a separate query so a plan-gated
   schema never fails `whoami`; a probe failure reports
   `business_entities_available: false`.
3. The reorder selection omits the `errors` field and reports the
   landed position read back from `transactionRules`, never echoing
   the requested order.
4. Live probes cover the new reads unconditionally, goal
   contributions behind existing goals, and the reorder no-op behind
   `MONARCH_LIVE_WRITES=1`. Stream review has no state-neutral live
   target and stays mock-tested.

## Consequences

- `mise run drift` tracks 77 operations; the weekly live gate covers
  the new reads.
- `goals create` without `--type` still fails live with HTTP 400;
  that pre-existing gap is out of scope and unchanged.
