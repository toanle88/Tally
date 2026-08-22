# DLV-PLAT-007 — PostgreSQL Outbox/Inbox and Worker Foundation User Stories

| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-007` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | Planned |
| Dependency position | Builds on `DLV-PLAT-003` persistence conventions and `DLV-PLAT-006` idempotency coordination; provides the integration-delivery foundation for later bounded contexts. |
| Exit evidence | Versioned event contracts, transactional outbox/inbox persistence, lease-safe dispatch, retry and poison-work handling, worker lifecycle behavior, crash recovery, duplicate-delivery, ordering, and replay tests pass. |

## 1. Outcome

**As the TALLY platform maintainer, I want durable outbox/inbox and worker foundations, so that integration events survive process failure, are delivered at least once, and establish at most one local business effect per receiving context.**

This delivery item establishes the PostgreSQL-backed integration transport and
worker contracts used by later bounded contexts. The owning source and receiving
contexts remain responsible for their domain effects, authorization, audit
evidence, and business invariants.

## 2. Learning objective

Learn how a modular monolith preserves transactional event publication,
consumer deduplication, lease ownership, retry safety, and replay evidence
without introducing an external broker.

## 3. Scope

- Define the versioned integration-event envelope and minimal payload contract.
- Create the PostgreSQL `integration.outbox` and `integration.inbox` persistence foundation.
- Define atomic source-side publication and receiving-side consumption boundaries.
- Implement the initial outbox dispatcher and worker-host lifecycle contracts.
- Prove duplicate delivery, crash recovery, ordering, retry, lease fencing, and replay behavior.

## 4. Explicit exclusions

- Finance aggregates, capability commands, receiving-context domain effects, or accounting ownership.
- Authentication, authorization policy, audit policy, or user-facing workflow screens.
- Provider-specific payment, bank, tax, payroll, or document workers.
- External brokers such as Azure Service Bus, Kafka, or RabbitMQ; adoption requires a separate ADR.
- Operational dashboards and runbooks owned by `DLV-OPS-001` and `DLV-OPS-002`.
- Full finance-level exactly-once proof, which belongs to the owning capability integrations.

## 5. Owning bounded context & architecture

- **Owning bounded context / module:** Platform integration adapter; source and receiving bounded contexts own business interpretation and effects.
- **Persistence ownership:** `integration.outbox` and `integration.inbox` are platform integration schemas. No finance module may access another module's adapter or schema directly.
- **Initial worker host:** `cmd/worker` hosts all workers initially, with independent concurrency limits, database-pool budgets, shutdown deadlines, and metrics namespaces.
- **Transport:** PostgreSQL durable state is the initial transport. Registered in-process consumers and approved external adapters are invoked by the dispatcher.

## 6. User stories

### User Story 1 — Define the versioned event envelope and safe payload contract

**As a platform and bounded-context developer, I want a stable event envelope, so that every published event carries enough identity and lineage for validation, delivery, deduplication, and replay.**

- [ ] The envelope defines `messageId`, `eventType`, `eventVersion`, `occurredAt`, `sourceContext`, `aggregateId`, `aggregateVersion`, `accountingScopeId` where applicable, `correlationId`, `causationId`, `dataClassification`, `payloadFingerprint`, and minimal `data`.
- [ ] Event identity, semantic contract version, source aggregate version, scope, correlation, causation, and payload fingerprint are distinct concepts.
- [ ] Unknown contract versions, invalid scope, malformed identity, and unsupported semantic transformations produce an explicit validation outcome without changing domain state.
- [ ] Payloads contain only the minimum facts required by approved consumers and exclude secrets, tokens, full bank-account numbers, payroll details, and unrestricted remittance text.
- [ ] Serialization and fingerprinting are deterministic and preserve the original event identity across retry and replay.

### User Story 2 — Persist durable PostgreSQL outbox and inbox records

**As a platform developer, I want durable outbox and inbox records, so that event publication and consumer identity survive process restart and database recovery.**

- [ ] A migration creates `integration.outbox` with event identity, source aggregate/version, scope, lineage references, payload, fingerprint, availability, claim, attempt, error, and establishment fields.
- [ ] A migration creates `integration.inbox` with consumer/message identity, message fingerprint, processing state, result reference, receipt time, and establishment time.
- [ ] Outbox uniqueness prevents duplicate publication for the same source context, aggregate, aggregate version, and event type.
- [ ] Inbox primary-key identity is `(consumer_name, message_id)`; state values are limited to `processing`, `established`, and `failed`.
- [ ] Queries support due-outbox selection, lease expiry, inbox reconciliation, and scoped duplicate lookup without bypassing schema ownership.

### User Story 3 — Coordinate transactional publication and consumption

**As an owning-context developer, I want explicit transaction boundaries, so that acknowledged local effects cannot be separated from their integration evidence.**

- [ ] The source business effect and its outbox record commit in one database transaction.
- [ ] A receiving inbox record, receiving local effect, and any resulting outbox records commit in one database transaction.
- [ ] A same-fingerprint duplicate delivery returns the established inbox result and repeats no local business effect.
- [ ] A different fingerprint for the same consumer/message identity returns an identity-content conflict, preserves existing evidence, and raises an integrity outcome.
- [ ] A processing or failed inbox item remains discoverable and does not fabricate success; reconciliation checks the established local result before retrying.
- [ ] Platform integration code does not become the owner of finance aggregates, accounting effects, authorization, or audit policy.

### User Story 4 — Dispatch due outbox work with leases and typed retries

**As a platform operator, I want lease-safe dispatch and controlled retries, so that multiple workers can process due events without stale workers overwriting newer outcomes.**

- [ ] Claiming selects due, unestablished records using `FOR UPDATE SKIP LOCKED`, orders by availability and outbox identity, and assigns an owner token and lease.
- [ ] The default claim duration is 30 seconds, renewal occurs before two-thirds of lease consumption, and the duration remains longer than the measured p99 handler duration.
- [ ] Establishment and rescheduling require the current claim owner; an expired or superseded worker cannot overwrite a newer claim.
- [ ] Only typed transient dependency failures are retried, using the default delays of 5 seconds, 30 seconds, 2 minutes, 10 minutes, and 30 minutes unless an approved adapter policy overrides them.
- [ ] Domain rejection, authorization denial, idempotency conflict, and data-integrity mismatch are not automatically retried.
- [ ] Work that reaches ten failed attempts becomes a managed exception with retained evidence and is never silently deleted.

### User Story 5 — Prove worker lifecycle, crash recovery, duplicate delivery, and replay

**As the TALLY maintainer, I want focused integration verification, so that the platform can recover safely without duplicate effects or lost event evidence.**

- [ ] The worker host starts and stops workers with independent limits, pool budgets, metrics namespaces, and a bounded shutdown deadline.
- [ ] Tests cover crashes before source commit, after source commit, before consumer establishment, and after consumer local-effect processing.
- [ ] Tests cover concurrent claims, lease expiry, stale-worker fencing, worker restart, retry exhaustion, and managed poison work.
- [ ] Tests cover duplicate, out-of-order, delayed, unknown-version, invalid-scope, missing-prerequisite, and changed-fingerprint deliveries.
- [ ] Replay selects an immutable event range and consumer generation, preserves original event identities, retains existing inbox evidence, and reproduces expected projections without duplicate business effects.
- [ ] Focused verification proves package boundaries, migration/generated-code consistency, and the crash-before/after-commit and duplicate-delivery exit evidence.

## 7. Definition of Ready

- [ ] The event envelope and payload rules in the approved integration technical specification are treated as the contract baseline.
- [ ] PostgreSQL remains the initial transport; no broker design is introduced without an ADR.
- [ ] The transaction harness can inject failure at source commit, consumer commit, and worker establishment boundaries.
- [ ] Test fixtures can represent valid, duplicate, delayed, out-of-order, invalid, and poison events without creating a finance capability.
- [ ] The worker process and package ownership boundaries are agreed with `DLV-OPS-001`, `DLV-OPS-002`, and later capability delivery items.

## 8. Definition of Done

- [ ] All five stories and their acceptance criteria pass.
- [ ] Outbox and inbox migrations, constraints, indexes, and generated database artifacts are verified.
- [ ] Source publication and receiving consumption transaction boundaries are demonstrated with failure injection.
- [ ] Dispatcher leases, owner-token fencing, retry classification, backoff, and managed poison outcomes are tested.
- [ ] Duplicate, ordering, replay, restart, and crash-recovery evidence passes without duplicate local effects or silently lost event evidence.
- [ ] Worker lifecycle, package ownership, observability boundaries, and documentation are reviewed.
- [ ] No finance capability, external broker, frontend behavior, or unrelated M0 delivery item is marked complete.

## 9. Traceability identifiers

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Requirement IDs | Contributes to `GFR-006`, `GFR-007`, `GFR-008`, `GFR-009`, `GFR-012`, `GFR-013`, and `GFR-014`; does not complete capability-specific controls by itself. |
| Workflow IDs | `WF-7.13`, `WF-7.14` |
| NFR IDs | `NFR-REL-001`, `NFR-REL-002`, `NFR-INT-003`, `NFR-REC-001`, `NFR-REC-005`, `NFR-REC-007`, and applicable `NFR-TST-*` obligations. |
| Quality contribution | Durable integration evidence, at-least-once delivery, bounded-context inbox deduplication, lease-safe workers, and replay/recovery safety. |
| Exit evidence | Crash-before/after-commit and duplicate-delivery tests pass, with explicit outcomes for retry, ordering, invalid events, poison work, and replay. |

## 10. Source references

- `docs/specs/finance_delivery_plan_v1.0.md` — DLV-PLAT-007 backlog and M0 exit evidence.
- `docs/specs/finance_domain_model_ddd.md` — cross-context event invariants, event interpretation, ordering, replay, concurrency, and recovery.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — `WF-7.13` cross-context event interpretation, ordering, and replay.
- `docs/specs/finance_nonfunctional_requirements_v1.0.md` — reliability, integration, recovery, and test obligations.
- `docs/specs/system_design/03_data_integration_architecture_v1.0.md` — PostgreSQL integration transport, ordering, replay, and recovery architecture.
- `docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md` — outbox/inbox tables, constraints, and indexes.
- `docs/specs/technical_specifications/04_events_workers_integration_specifications_v1.0.md` — envelope, claiming, inbox, worker, retry, and broker-adoption contracts.
- `docs/specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md` — fault injection, duplicate, lease, recovery, and replay verification.

## 11. Dependencies

- `DLV-PLAT-003` — Goose, pgx, sqlc, migration, and persistence verification workflow.
- `DLV-PLAT-006` — request fingerprint, scoped identity, established-result lookup, and owner-token coordination foundations.
- `DLV-OPS-001` and `DLV-OPS-002` — later structured telemetry, dashboards, and runbook completion.
- Future bounded-context delivery items — source effects, receiving effects, authorization, audit, and business-level exactly-once behavior.

