# 0015 - Rule and merchant full input-surface parity

## Status

Accepted.

## Context

A capability-boundary audit of rule configuration found four gaps
against the web app: merchant-rename actions, raw-statement matching,
review-status actions, and merchant default categories. The CLI only
sent `setCategoryAction`/`addTagsAction` on rules and
`{merchantId, name}` on merchants.

The real input surface was recovered from the web bundle
(`app.monarch.com`, `__APP_VERSION__="v1.0.4858"`, which embeds the
full GraphQL introspection schema) cross-checked with
`monarch-money-ts` `TRANSACTION_RULE_FIELDS` and the Transaction Rules
help article. `CreateTransactionRuleInput` carries 27 fields and
`UpdateTransactionRuleInput` the same plus `id`; every one is accepted
by the existing `createTransactionRuleV2`/`updateTransactionRuleV2`
mutations, so no new endpoint is needed:

- Criteria: `merchantNameCriteria`, `originalStatementCriteria`,
  legacy `merchantCriteria` (each `{operator: eq|contains, value}`),
  `merchantCriteriaUseOriginalStatement`, `amountCriteria` (with
  `valueRange{lower,upper}` for `between`), `categoryIds`,
  `accountIds`, `criteriaOwnerUserIds`/`criteriaOwnerIsJoint`,
  `criteriaBusinessEntityIds`/`criteriaBusinessEntityIsUnassigned`.
- Actions: `setCategoryAction: ID`, `setMerchantAction: String`
  (merchant display name, not an ID), `addTagsAction`,
  `setHideFromReportsAction`, `reviewStatusAction`
  (`needs_review|reviewed`) with `needsReviewByUserAction`,
  `linkGoalAction`, `linkSavingsGoalAction`,
  `setLinkToPaydownBudgetAction`, `sendNotificationAction`,
  `actionSetOwner`/`actionSetOwnerIsJoint`,
  `actionSetBusinessEntity`/`actionSetBusinessEntityIsUnassigned`,
  `splitTransactionsAction{amountType: ABSOLUTE|PERCENTAGE,
  splitsInfo}`.
- `UpdateMerchantInput` accepts `defaultCategoryId`,
  `defaultCategoryApplicationMode` (`new_only|new_and_edits`),
  `showDefaultCategoryPrompt`, and `recurrence`; `Merchant`
  exposes `defaultCategory`, `defaultCategoryApplicationMode`,
  `showDefaultCategoryPrompt`. Query `merchants` accepts
  `filters{hasDefaultCategory}`, `includeIds`, and
  `includeMerchantsWithoutTransactions`.

No live write probe was available, so field types were verified
statically against the bundle schema instead of ADR-0014-style live
roundtrips.

## Decision

1. `rules create`/`update` expose the full input surface as flags,
   sharing one builder so create and update can never diverge; only
   flags actually passed are sent, preserving update patch semantics.
   Complex inputs follow the `transactions split --file` precedent:
   `--split-file` (`amountType` + `splitsInfo`) and merchant
   `--recurrence-file` (must contain `isRecurring`).
2. `rules list` returns the full `TransactionRuleV2` field set
   (including `originalStatementCriteria` and the legacy
   `merchantCriteria`, kept separate instead of merged) and
   `merchants list`/`show` return the default-category fields.
3. New enum/conditional validation fails fast with
   `INVALID_ARGUMENTS`: `--review-status`,
   `--default-category-mode`, split `amountType`, and
   `--amount-value-upper` (required with and only with
   `--amount-operator=between`).
4. `merchants update` drops the required `--name` flag and requires
   at least one update flag instead, so default-category-only edits
   work.
5. Drive-by fix in the touched file: `RuleReorderResult.MovedTo`
   serializes as `moved_to` instead of repeating `requested_order`.

## Consequences

- `rules.list`/`merchants.*` JSON gains additive fields; existing
   keys are unchanged.
- `mise run drift` operation count is unchanged (no new
  `.graphql` files); mock tests assert the exact variable payload
  of every new flag on both create and update paths.
- Split/recurrence file shapes are validated client-side only for
  `amountType`/`isRecurring`; deeper shape errors surface from the
  API.
