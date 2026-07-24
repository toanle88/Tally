# Finance Platform Quality, Testing and Environment Plan

| Document-control field | Value |
|---|---|
| Version | 1.0 |
| Baseline date | 2026-07-24 |
| Status | Passed |
| Source baseline | Finance DDD v3.1; Functional PRD v1.5; UX v1.0; NFR v1.0; Solution/System Design v1.0; Technical Specifications v1.0 |
| Delivery profile | Solo, part-time learning project; local-first; low-cost Azure demonstrations |
| Owner | Learning Product and Delivery Owner |

> **Purpose:** Define quality gates, test layers, acceptance execution, environment progression, seed data, defect policy and milestone qualification.
>
> **Planning rule:** Iteration counts and elapsed-time ranges are planning assumptions, not commitments. Reforecast after M2 using observed completion rate, defect rate and available learning time.

## 1. Quality model

Quality evidence is delivered continuously. A later test layer does not replace an earlier one, and a milestone demonstration does not replace repeatable automated verification.

## 2. Quality gates

| Gate | Name | Pass condition |
|---|---|---|
| QG-01 | Traceability and source integrity | Every item has valid source IDs; source/checkpoint hashes match; no unapproved scope is introduced. |
| QG-02 | Domain and application correctness | Domain invariants, lifecycle transitions, corrections and application handlers pass deterministic tests. |
| QG-03 | Persistence and migration safety | PostgreSQL constraints, migrations, rollback/forward-fix, sqlc generation and repository tests pass. |
| QG-04 | API and integration contracts | OpenAPI, event contracts, idempotency, ordering and compatibility checks pass. |
| QG-05 | UX, accessibility and localization | Component, workflow, keyboard, screen-reader, contrast, locale and error-state checks pass. |
| QG-06 | Security, privacy and authorization | Authentication, scope authorization, SoD, secret handling, privacy and export controls pass. |
| QG-07 | Concurrency, idempotency and financial integrity | Duplicate, stale-version, concurrent update, correction and reconciliation tests pass. |
| QG-08 | Observability and operational readiness | Logs, metrics, traces, alerts, dashboards and runbook evidence are complete. |
| QG-09 | Performance, capacity and recovery qualification | Applicable NFR performance, capacity, availability, backup, restore and DR targets pass. |
| QG-10 | Release evidence and demonstration | All required tests, documentation, traceability, cost checks and milestone demo scenarios pass. |

## 3. Test layers

| Layer | Scope | Required evidence |
|---|---|---|
| Domain unit | Value objects, aggregates, invariants, state transitions and calculations | Deterministic table-driven Go tests |
| Application | Command handlers, authorization orchestration and outcomes | Handler tests with controlled repositories and clocks |
| Persistence integration | PostgreSQL constraints, locks, transactions, migrations and sqlc queries | Testcontainers tests using production-equivalent PostgreSQL major version |
| API contract | OpenAPI requests, responses, errors, pagination, idempotency and concurrency | Contract validation and generated-client compatibility |
| Event/worker | Outbox/inbox, ordering, retries, deduplication, leases and recovery | Crash and redelivery tests |
| Frontend component | daisyUI abstractions, forms, tables, permissions and states | Vitest, React Testing Library and MSW |
| Workflow E2E | User journeys through API and database effects | Playwright with versioned scenario data |
| Accessibility | Keyboard, focus, labels, contrast and screen-reader behavior | Automated checks plus manual screen-reader script |
| Security | Authentication, authorization, SoD, privacy, exports, secrets and dependencies | Automated tests and threat-based review |
| Financial integrity | Balance, uniqueness, ownership, immutable correction and reconciliation | Cross-layer assertions and reconciliation queries |
| Performance/capacity | NFR latency, throughput, volume and degradation behavior | Repeatable load scripts and result report |
| Recovery | Backup, restore, restart, replay, reconciliation and DR | Exercise log with before/after financial controls |

## 4. Acceptance-scenario execution plan

Every `FAC-*` scenario remains authoritative from Functional PRD v1.5. Delivery groups scenarios by their source acceptance section and milestone.

| Acceptance group | Scenario count | Primary milestone | Test pack |
|---|---:|---|---|
| FAC-14-1 | 9 | M2 | `tests/acceptance/fac-14-1` |
| FAC-14-2 | 20 | M7 | `tests/acceptance/fac-14-2` |
| FAC-14-3 | 21 | M3 | `tests/acceptance/fac-14-3` |
| FAC-14-4 | 8 | M9 | `tests/acceptance/fac-14-4` |
| FAC-14-5 | 4 | M9 | `tests/acceptance/fac-14-5` |
| FAC-14-6 | 22 | M4 | `tests/acceptance/fac-14-6` |
| FAC-14-7 | 6 | M8 | `tests/acceptance/fac-14-7` |
| FAC-14-8 | 3 | M3 | `tests/acceptance/fac-14-8` |
| FAC-14-9 | 3 | M9 | `tests/acceptance/fac-14-9` |
| FAC-14-10 | 8 | M3 | `tests/acceptance/fac-14-10` |
| FAC-14-11 | 6 | M8 | `tests/acceptance/fac-14-11` |
| FAC-14-12 | 6 | M7 | `tests/acceptance/fac-14-12` |
| FAC-14-13-1 | 5 | M5 | `tests/acceptance/fac-14-13-1` |
| FAC-14-13-2 | 13 | M5 | `tests/acceptance/fac-14-13-2` |
| FAC-14-13-3 | 5 | M4 | `tests/acceptance/fac-14-13-3` |
| FAC-14-13-4 | 5 | M6 | `tests/acceptance/fac-14-13-4` |
| FAC-14-13-5 | 5 | M8 | `tests/acceptance/fac-14-13-5` |
| FAC-14-13-6 | 5 | M8 | `tests/acceptance/fac-14-13-6` |
| FAC-14-13-7 | 5 | M7 | `tests/acceptance/fac-14-13-7` |
| FAC-14-13-8 | 5 | M7 | `tests/acceptance/fac-14-13-8` |
| FAC-14-13-9 | 5 | M8 | `tests/acceptance/fac-14-13-9` |
| FAC-14-13-10 | 5 | M9 | `tests/acceptance/fac-14-13-10` |
| FAC-14-13-11 | 5 | M9 | `tests/acceptance/fac-14-13-11` |
| FAC-14-13-12 | 5 | M3 | `tests/acceptance/fac-14-13-12` |
| FAC-14-13-13 | 5 | M9 | `tests/acceptance/fac-14-13-13` |
| FAC-14-13-14 | 5 | M9 | `tests/acceptance/fac-14-13-14` |
| FAC-14-13-15 | 5 | M9 | `tests/acceptance/fac-14-13-15` |

## 5. Environment plan

| Environment | Purpose | Data | Lifetime | Promotion gate |
|---|---|---|---|---|
| Local | Daily development and debugging | Synthetic, resettable | Continuous on learner machine | Unit and local integration tests |
| CI | Pull-request verification | Ephemeral generated scenarios | Per workflow run | All required checks green |
| Azure dev | Terraform, identity, networking and managed-service exercises | Synthetic, non-sensitive | Created only when needed | Terraform plan review and cost check |
| Azure demo | Milestone demonstration and recovery exercise | Versioned synthetic milestone dataset | Ephemeral; destroy after use | Milestone quality gates |
| Qualification | Full NFR exercise when pursued | Generated baseline-scale dataset | Temporary dedicated exercise | QG-09 and QG-10 |

## 6. Test-data strategy

Versioned scenario packs must include:

- multiple tenants, legal entities, ledgers, books and fiscal periods;
- transaction, functional and presentation currencies;
- active, pending, failed, returned, reversed, reconciled and corrected states;
- duplicate identities with same and changed fingerprints;
- expected-version conflicts and deterministic concurrency barriers;
- legal-hold and sensitive-data cases;
- period close, reopen, reclose and expired-control cases;
- high-volume generators separate from human-readable demo fixtures; and
- reconciliation control totals before and after failure/recovery exercises.

No production personal, payroll, tax or bank data is permitted.

## 7. Defect policy

| Severity | Definition | Milestone rule |
|---|---|---|
| Critical | Financial misstatement, duplicate effect, unauthorized access, data loss or unrecoverable integrity failure | Blocks all releases |
| High | Required workflow, correction, recovery, security or accessibility path cannot complete | Blocks affected milestone |
| Medium | Workaround exists and no financial/security integrity is at risk | May defer with recorded owner and target milestone |
| Low | Cosmetic or minor usability issue | May defer through normal backlog |

## 8. NFR qualification sequencing

- Security, privacy, accessibility, maintainability and observability are tested from M0/M1 onward.
- Reliability, idempotency and concurrency are mandatory from the first financial state change in M2.
- Capability-specific performance smoke tests run at every milestone.
- Full baseline capacity, availability and recovery qualification is scheduled at M9 or earlier if the learner chooses to pursue production-like evidence.
- A requirement not exercised is reported as **not yet qualified**, never as passed by documentation alone.

## 9. Release evidence package

Each milestone produces:

- source and build identifiers;
- migration and configuration versions;
- requirement/workflow/acceptance coverage report;
- test results and defect disposition;
- accessibility and security evidence;
- operational dashboard and runbook checks;
- cost and resource inventory;
- demo script and screenshots or recording; and
- checkpoint hashes for changed documents.
## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `3321a68aec5eb4b78f80cda779c1cae9ac94538aba4f5c7bd5b430fe18229ca2` |
| Review status | Passed |
| Reuse rule | Re-run structural checks when this body hash and all source hashes remain unchanged. Run targeted semantic review for localized backlog, estimate, dependency, milestone, gate, risk or cost changes. Run the full suite for requirement, workflow, acceptance, architecture, technical-specification or source-hash changes. |

### Checks recorded

- All 10 quality gates are defined.
- All 199 acceptance scenarios are assigned through 27 acceptance groups.
- Environment, test-data, defect and qualification policies are defined.
- Unexecuted NFR qualification cannot be reported as passed.
