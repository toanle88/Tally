# DLV-PLAT-006 — Request Fingerprint and Idempotency Foundation User Stories

| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-006` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | User Stories 1–2 implementation complete; verification evidence pending |
| Dependency position | Builds on the shared identity and accounting-scope primitives from `DLV-PLAT-005`; provides a foundation for capability handlers and the outbox/inbox work in `DLV-PLAT-007`. |
| Exit evidence | Canonical fingerprint, same-content retry, changed-content conflict, transactional, concurrent, and boundary tests pass without duplicate business effects. |

## 1. Purpose and scope

**As the TALLY platform maintainer, I want a scoped request-fingerprint and idempotency foundation, so that retried commands return established outcomes and cannot repeat or change a business effect.**

This item defines reusable platform behavior and contracts. It does not implement a finance aggregate or capability workflow.

## 2. Approved boundaries

- The foundation package is `internal/platform/idempotency`.
- A canonical request fingerprint represents functional command content, using a deterministic canonicalization and stable cryptographic digest. Equivalent functional content produces the same fingerprint; material content changes produce a different fingerprint.
- The business idempotency identity is scoped by the applicable `AccountingScope` and command/business identity. It is distinct from aggregate version, event identity, correlation identity, and causation identity.
- For the same scope, identity, and fingerprint, the established in-progress or terminal command result is returned without repeating the business effect.
- Reusing an identity with a different canonical fingerprint returns `IDEMPOTENCY_CONFLICT` and produces no business effect.
- Idempotency result metadata is persisted in the same local transaction as the owning business change. A result must not claim success when the business change did not commit.
- Stored metadata includes the scoped identity, canonical fingerprint, lifecycle/result status, and the stable result reference or response metadata needed to return the established result; it does not own finance state.
- Shared platform code provides technical coordination only. The owning finance bounded context remains responsible for authorization, validation, aggregate invariants, and business effects.
- Outbox, inbox, workers, delivery retries, and replay orchestration remain deferred to `DLV-PLAT-007`.

## 3. Explicit exclusions

- Finance aggregates, capability commands or handlers, authorization policies, audit policy, migrations, or finance-owned persistence schemas.
- Event identity deduplication or consumer inbox behavior.
- External provider idempotency, delivery scheduling, retry policy, or replay tooling.
- Frontend screens or changes to existing OpenAPI idempotency-header contracts.

## 4. User stories

### User Story 1 — Define canonical request fingerprinting

**As a platform and finance module developer, I want functional request content canonicalized deterministically, so that semantically equivalent retries can be recognized.**

- [x] Canonicalization defines included functional fields, excluded transport/non-functional fields, ordering, normalization, and representation for supported values.
- [x] Equivalent functional content produces byte-identical canonical input and the same fingerprint.
- [x] Material changes to functional content produce a different fingerprint.
- [x] Fingerprint serialization is stable and suitable for persistence, comparison, logging, and audit references without exposing unnecessary sensitive request content.
- [x] Tests cover field ordering, omitted versus explicit values where applicable, Unicode/encoding, nested collections, nullability, and boundary-sized requests.

### User Story 2 — Define scoped idempotency identity and stored command-result metadata

**As a platform developer, I want a validated scoped idempotency identity and result record, so that distinct business scopes and command outcomes cannot be confused.**

- [x] Missing, malformed, empty, or otherwise invalid identities are rejected deterministically with stable validation behavior.
- [x] Identity equality includes the applicable accounting scope and business/command identity; identical text in different scopes remains distinct.
- [x] Documentation and types distinguish business identity/fingerprint from aggregate version, event identity, correlation ID, and causation ID.
- [x] Stored result metadata can represent in-progress and terminal outcomes and retains the canonical fingerprint and stable result reference/response metadata.
- [x] Result metadata does not mutate or replace an established financial fact and does not make the platform package the owner of a finance aggregate.

### User Story 3 — Return the established result for identical retries

**As a command handler, I want an identical retry to return the established in-progress or terminal result, so that network retries are safe and the business effect occurs once.**

- [ ] The first accepted identity/fingerprint establishes one result record and coordinates execution with the owning business transaction.
- [ ] A repeat with the same scope, identity, and fingerprint returns the existing in-progress or terminal result.
- [ ] Identical retries do not invoke or commit the owning business effect a second time.
- [ ] An ambiguous or in-progress result remains discoverable through the established identity and is never presented as a false success.
- [ ] Tests cover retry before completion, retry after success, and retry after a terminal failure or rejection.

### User Story 4 — Reject changed content under the same identity

**As a command handler, I want changed content under an existing identity rejected, so that one business identity cannot establish conflicting outcomes.**

- [ ] A same-scope identity with a different canonical fingerprint returns `IDEMPOTENCY_CONFLICT` using the existing API error meaning and HTTP status.
- [ ] The conflict exposes only the result/fingerprint reference permitted by the contract and does not disclose protected request content.
- [ ] A conflict does not update the stored result metadata, aggregate state, events, or other business state.
- [ ] The caller is directed to create a new business identity/action rather than silently overwriting or merging content.
- [ ] Tests cover changes to each material content category and verify no state change.

### User Story 5 — Prove transactional, concurrent, and boundary behavior

**As the TALLY maintainer, I want focused verification of the idempotency foundation, so that transaction failures and concurrent submissions cannot duplicate business effects.**

- [ ] Concurrent first submissions for one scoped identity establish at most one result; losers observe the established result or a deterministic conflict.
- [ ] Idempotency metadata commits atomically with the owning business change; rollback leaves no false result, and recovery/transaction retry retains the committed result.
- [ ] Tests prove same-content retry, changed-content conflict, malformed identity, fingerprint boundaries, and terminal/in-progress result behavior.
- [ ] Package and architecture checks show finance modules and adapters do not own shared idempotency behavior or access another module's schema directly.
- [ ] Existing OpenAPI idempotency-header contracts remain unchanged and a documented root verification command runs the focused checks reproducibly.

## 5. Definition of Ready

- [ ] Canonicalization inputs, exclusions, normalization, digest representation, and sensitive-data handling are approved.
- [ ] The exact scoped identity components and result lifecycle/status values are approved without conflating them with existing identity primitives or aggregate versions.
- [ ] Transaction ownership and persistence boundary are agreed with the owning capability; no shared finance schema is introduced by this foundation item.
- [ ] `IDEMPOTENCY_CONFLICT` remains the stable conflict code and existing OpenAPI contracts are the compatibility baseline.
- [ ] Boundaries with `DLV-PLAT-005`, `DLV-PLAT-007`, and finance capability items are preserved.
- [ ] Five stories are small enough for one or a short chain of reviewable changes.

## 6. Definition of Done

- [ ] All five stories and acceptance criteria pass.
- [ ] Canonicalization and fingerprint tests prove equivalent content stability and material-change distinction.
- [ ] Identity validation, scope separation, result metadata, retry, conflict, transaction, concurrency, and recovery evidence pass.
- [ ] No business effect is repeated for an identical retry, and changed content produces no effect.
- [ ] Package ownership, architecture, and unchanged OpenAPI checks pass.
- [ ] No finance capability, authorization policy, frontend behavior, outbox/inbox workflow, or verification evidence file is marked complete by this documentation item.

## 7. Traceability

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-006` |
| Direct requirement IDs | Contributes to `GFR-006` and `GFR-007`; supports `GFR-008` and the immutable lineage expectations of `GFR-013`. |
| Workflow IDs | `WF-7.13`, `WF-7.14` |
| Quality contribution | Deterministic retry, conflict safety, transaction atomicity, and concurrency safety for later command handlers. |
| Exit evidence | Same-content, changed-content, transaction, concurrent, recovery, package-boundary, and contract-preservation tests pass. |

## 8. Source references

- `docs/specs/finance_delivery_plan_v1.0.md` — platform foundation backlog and GFR mappings.
- `docs/specs/finance_domain_model_ddd.md` — command fingerprint, duplicate delivery, concurrency, recovery, and immutable-fact rules.
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` — `GFR-006`, `GFR-007`, and `GFR-008`.
- `docs/specs/prd/01_finance_functional_prd_v1.5.md` — `WF-7.13` and `WF-7.14`.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — safe repeat and identity-content conflict behavior.
- `docs/specs/technical_specifications/01_backend_module_specifications_v1.0.md` — `IDEMPOTENCY_CONFLICT` error contract and transaction boundary.
- `docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md` — existing idempotency-header contract.
- `docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md` — local transaction and persistence conventions.
- `docs/specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md` — concurrency and recovery verification expectations.
- `docs/backlog/stories/DLV-PLAT-005_user_stories.md` — shared identity and accounting-scope foundation boundary.
