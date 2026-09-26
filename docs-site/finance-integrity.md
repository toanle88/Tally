# Finance integrity

Financial correctness is TALLY’s primary architecture driver. Convenience must
not turn an established accounting fact into an editable row or turn a retry
into a second financial effect.

## Non-negotiable rules

| Rule | Meaning in TALLY |
| --- | --- |
| Exact money | Authoritative amounts and rates use exact decimal semantics. Binary floating point is not a money type. Amount and currency travel together, with currency-specific precision and rounding evidence. |
| One owner | A bounded context owns each aggregate and accounting effect. GL is the sole producer of final journal entries; subledgers submit published posting requests. |
| Immutable facts | Posted or otherwise established facts are not destructively edited. Their current display state is derived from immutable lifecycle and correction facts. |
| Explicit correction | Reversal, adjustment, amendment, return, unapplication, replacement, supersession, or compensation is a new linked action with its own authorization and evidence. |
| Safe repeat | A retried state change carries a business identity and canonical fingerprint. The same identity and content returns the established result; changed content conflicts without a new effect. |
| Concurrency honesty | Expected aggregate versions and deterministic locks prevent silent overwrites. A stale write becomes a typed conflict with a deliberate refresh/retry path. |
| Durable publication | State and outbox publication intent commit atomically. Inbox identity, consumer effect, and resulting publication commit atomically on the receiving side. |
| Evidence | Material changes carry actor, scope, authorization/policy references, correlation/causation, fingerprints, and an audit reference through an owning port. Operational telemetry is useful evidence about operation, but is not the audit record. |

## Why this matters

Finance workflows cross modules and external systems. A payment provider can
retry, a bank file can be duplicated, an approval can become stale, a period
can close while work is in flight, and a user can lose permission between
screen render and submit. TALLY therefore models pending, rejected, denied,
conflicted, unavailable, reconciled, reversed, and terminal outcomes
separately instead of presenting every request as success or failure.

## What is implemented in the current checkout

The repository has focused evidence for exact-decimal primitives, accounting
scope, aggregate versions, request fingerprints, scoped idempotency, durable
outbox/inbox coordination, dispatch/retry/replay, and IAM audit/evidence
boundaries. It does not yet implement the GL, AR, AP, payments, close, or
other finance aggregates that will use these controls.

The [DDD baseline](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_domain_model_ddd.md)
is authoritative for domain language and invariants. Focused evidence is
indexed in [current status and evidence](/current-status).
