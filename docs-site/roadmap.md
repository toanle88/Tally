# Roadmap and progress

The roadmap is organized into nine milestones, from engineering foundation (M0) through full qualification (M9). Progress means acceptance criteria and evidence pass; partial percentage completion does not close a milestone.

## Current checkpoint: M0

Completed foundation deliveries include the monorepo, Docker/PostgreSQL workflow, migrations and typed persistence foundations, OpenAPI workflow, shared exact-decimal finance primitives, request fingerprint/idempotency foundations, and the completed DLV-PLAT-007 event-envelope, durable outbox/inbox, transactional coordination, lease-safe dispatch, typed retry, worker lifecycle, crash recovery, delivery ordering, and replay foundations. Semantic payload safety remains a separate deferred follow-up.

## Milestones

| Milestone | Outcome |
| --- | --- |
| M0 | Engineering foundation |
| M1 | Identity and accounting configuration |
| M2 | First posted journal vertical slice |
| M3 | Approval and period controls |
| M4 | Receivables and billing |
| M5 | Payables and payment execution |
| M6 | Bank and cash reconciliation |
| M7 | Assets and revenue |
| M8 | Currency, intercompany, and reporting |
| M9 | Tax, payroll, audit, and qualification |

M2 is the recommended stopping point for a strong portfolio-sized outcome: a complete journal posting and reversal slice across domain, persistence, API, UI, authorization, tests, and operations.

See the [delivery plan](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_delivery_plan_v1.0.md) and [live roadmap](https://github.com/toanle88/Tally/blob/main/ROADMAP.md) for authoritative status.
