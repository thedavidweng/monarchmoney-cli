# 0007 - HTTP hardening, mutation no-retry, overview command

## Status

Accepted.

## Context

The GraphQL client (`internal/graphql/client.go`) and the REST login path (`internal/auth/login.go`) used Go's default `http.Client`, which silently follows 3xx redirects. A misconfigured endpoint, a hostile intermediary, or a future Monarch API change that redirects the GraphQL endpoint to an attacker-controlled host would let the bearer token leak to that host with no error surfaced to the user. The fork (`matteing/monarch-cli`) rejects redirects explicitly via `httpx.RejectRedirects`; this CLI did not.

The same client retried every call up to three times, including mutations. A transport failure mid-request is ambiguous: Monarch may have already applied the write. Retrying a `DeleteTransaction` or `SetBudget` after a network blip can duplicate the effect or silently succeed twice, leaving the user's books in a state the CLI cannot see. The fork separates read-retry from mutation-no-retry; this CLI did not.

The client also had no rate-limit handling. Monarch returns HTTP 429 with a `Retry-After` header under load; the CLI treated it as a generic non-200 and retried on a fixed backoff, ignoring the server's guidance.

Finally, the CLI had no single command that showed a user "where do I stand right now". The fork's `monarch overview` aggregates net worth, cashflow, and recent transactions in one call; this CLI required three separate invocations.

## Decision

Four changes, each narrowly scoped:

1. **Reject redirects.** `NewClient` sets `CheckRedirect` to a function returning `http.ErrUseLastResponse`, so any 3xx becomes a non-200 and surfaces as an `API_ERROR`. The login HTTP client does the same. A legitimate Monarch API call never redirects; a redirect is always a signal to stop.

2. **Mutation no-retry.** Add `Client.DoMutation`, which executes exactly one attempt. The `graphQLClient` interface in `internal/monarch/service.go` gains `DoMutation`, and every service method that issues a GraphQL mutation (`CreateManualAccount`, `UpdateAccount`, `DeleteAccount`, `RefreshAccounts`, `SetBudget`, `ResetBudget`, `UpdateFlexibleBudget`, `UpdateFlexRolloverSettings`, `CreateCategory`, `DeleteCategory`, `DeleteCategories`, `UpdateRecurringTransaction`, `CreateTransactionRule`, `UpdateTransactionRule`, `DeleteTransactionRule`, `CreateTag`, `UpdateTransaction`, `DeleteTransaction`, `UpdateTransactionSplits`, `CreateTransaction`, `SetTransactionTags`) calls `DoMutation` instead of `Do`. Reads keep the existing retry loop.

3. **Rate-limit and Retry-After.** HTTP 429 produces a new `RATE_LIMITED` error code with the parsed `Retry-After` value (seconds or HTTP-date) stored in a new `Error.RetryAfterMS` field. 5xx responses also parse `Retry-After` and stay retryable. The retry loop honours `RetryAfterMS` when present, capped at 10 seconds to avoid pathological waits. A new `errors.NewWithRetryAfter` constructor populates the field. `RATE_LIMITED` shares exit code 5 with the network errors, since both mean "try again later".

4. **`monarch overview` command.** A new `Service.GetFinancialOverview` fetches accounts, cashflow summary, and the 10 most recent transactions concurrently via `golang.org/x/sync/errgroup`, then computes net worth from visible, net-worth-included accounts. The `overview` CLI command renders the aggregate for humans or emits the JSON envelope. `--from`/`--to` default to the current month.

A new dependency, `golang.org/x/sync/errgroup`, is introduced for the concurrent fetch. It is the standard library's recommended primitive for "N goroutines, fail on first error" and carries no transitive dependencies beyond `golang.org/x/sync`.

## Consequences

### Positive

- A redirect can no longer exfiltrate the bearer token; it is a hard error.
- Mutations are safe to retry manually but never retried automatically, so a transport failure after a successful write surfaces as an error the user investigates, rather than a silent duplicate.
- Rate-limit responses carry the server's wait hint into the structured error envelope, so scripts and agents can back off correctly.
- Users get a single command for a financial snapshot; agents get one JSON envelope instead of three calls.

### Negative

- A mutation that fails after the server applied it now requires the user to reconcile manually. This is the correct failure mode (ambiguous outcome → human judgement) but is less convenient than a silent retry that happens to work.
- `golang.org/x/sync` is a new external dependency, the first outside the standard library for the monarch service layer. It is minimal and stable.

### Mitigations

- The mutation-no-retry error message names the operation and resource so the user knows exactly what to check.
- `errgroup` is the only `golang.org/x` dependency; if it ever needs to be removed, the three goroutines can be replaced with a hand-rolled `sync.WaitGroup` plus a channel for the first error, at the cost of ~15 lines.
