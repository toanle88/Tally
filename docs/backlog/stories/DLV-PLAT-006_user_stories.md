# DLV-PLAT-006 — Request Fingerprint and Idempotency Foundation User Stories

| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-006` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | DLV-PLAT-006 platform foundation implemented; focused evidence passes, with the environment-dependent persistence gate pending |
| Dependency position | Builds on the shared identity and accounting-scope primitives from `DLV-PLAT-005`; provides a foundation for capability handlers and the outbox/inbox work in `DLV-PLAT-007`. |
| Exit evidence | Canonical fingerprint, durable reservation, same-content retry, changed-content conflict, transactional metadata finalization, concurrent ownership, boundary, and architecture evidence pass; Docker/SQLC persistence verification remains an environment-dependent gate. |

## 1. Purpose and scope

**As the TALLY platform maintainer, I want a scoped request-fingerprint and idempotency foundation, so that retried commands return established outcomes and cannot repeat or change a business effect.**

This item defines reusable platform behavior and contracts. It does not own a
finance aggregate or capability workflow. Owning-capability integration proves
how a finance transaction uses this foundation and is tracked by the relevant
capability delivery item.

The coordination-contract and durable persistence implementations provide
platform ownership, established-result lookup, terminal-result finalization,
lease handling, and database-level concurrency behavior. They do not claim
exactly-once financial/business effects, authorization policy, audit policy,
or a specific owning-capability transaction.

## 2. Approved boundaries

- The foundation package is `internal/platform/idempotency`.
- A canonical request fingerprint represents functional command content, using a deterministic canonicalization and stable cryptographic digest. Equivalent functional content produces the same fingerprint; material content changes produce a different fingerprint.
- The business idempotency identity is scoped by the applicable `AccountingScope` and command/business identity. It is distinct from aggregate version, event identity, correlation identity, and causation identity.
- For the same scope, identity, and fingerprint, the established in-progress or terminal command result is returned without repeating the business effect.
- Reusing an identity with a different canonical fingerprint returns `IDEMPOTENCY_CONFLICT` and produces no business effect.
- Idempotency metadata has an explicit durable reservation/finalization transaction boundary. An owning capability must later persist its business change and terminal result together.
- Stored metadata includes the scoped identity, canonical fingerprint, lifecycle/result status, and the stable result reference or response metadata needed to return the established result; it does not own finance state.
- Shared platform code provides technical coordination only. The owning finance bounded context remains responsible for authorization, audit, validation, aggregate invariants, and business effects.
- Outbox, inbox, workers, delivery retries, and replay orchestration remain deferred to `DLV-PLAT-007`.

## 3. Explicit exclusions

- Finance aggregates, capability commands or handlers, authorization policies, audit policy, migrations, or finance-owned persistence schemas.
- Event identity deduplication or consumer inbox behavior.
- External provider idempotency, delivery scheduling, retry policy, or replay tooling.
- Frontend screens or changes to existing OpenAPI idempotency-header contracts.
- Owning finance transactions, journal effects, authorization policy, audit persistence, and finance-level exactly-once proof.

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

### User Story 3 — Coordinate established results for identical retries

**As a platform maintainer, I want the foundation to coordinate identical retries, so owning command handlers can safely return established outcomes.**

- [x] The first accepted identity/fingerprint establishes one durable result record and one execution owner.
- [x] A repeat with the same scope, identity, and fingerprint returns the existing in-progress or terminal result.
- [x] The platform coordinator does not issue a second owner for an identical retry.
- [x] An ambiguous or in-progress result remains discoverable through the established identity and is never presented as a false success.
- [x] Tests cover retry before completion, retry after success, and retry after a terminal failure or rejection.

#### Coordination-contract prerequisite delivered

- [x] First execution ownership, same-fingerprint established-result lookup, terminal-result transition, and ambiguous/in-progress visibility are covered by `internal/platform/idempotency` tests.
- Owning-transaction integration and exactly-once business-effect protection remain follow-up capability scope.

### User Story 4 — Reject changed content at the platform boundary

**As a platform maintainer, I want changed content under an existing identity rejected, so that one business identity cannot establish conflicting outcomes.**

- [x] A same-scope identity with a different canonical fingerprint returns `ErrIdempotencyConflict` using the stable platform error meaning.
- [x] The platform conflict contains no request content and does not disclose protected data.
- [x] A conflict does not update stored result metadata or create a second owner.
- [x] The platform contract directs callers to preserve the identity and fingerprint pairing rather than overwrite it.
- [x] Tests cover material content changes and verify no platform state change.

The owning transport adapter remains responsible for mapping this platform
error to the existing `IDEMPOTENCY_CONFLICT` HTTP 409 contract.

### User Story 5 — Prove transactional, concurrent, and boundary behavior

> User Story 5 is complete for the platform foundation. Cross-process recovery,
> owning finance effects, authorization, audit integration, and finance-level
> exactly-once behavior remain capability follow-up scope.

**As the TALLY maintainer, I want focused verification of the idempotency foundation, so that owning capabilities have a durable and concurrency-safe platform contract.**

- [x] Concurrent first submissions for one scoped identity establish at most one platform owner; losers observe the established, in-progress, or deterministic conflict result.
- [x] Idempotency reservation and terminal metadata finalization have explicit transaction boundaries; rollback leaves no false terminal result and lease recovery is guarded by owner tokens.
- [x] Tests prove same-content retry, changed-content conflict, malformed identity, fingerprint boundaries, and terminal/in-progress result behavior.
- [x] Package and architecture checks show finance modules and adapters do not own shared idempotency behavior or access another module's schema directly.
- [x] Existing OpenAPI idempotency-header contracts remain unchanged and documented root verification commands run the focused checks reproducibly.

## 5. Definition of Done

- [x] The platform slices of all five stories and their acceptance criteria pass.
- [x] Canonicalization and fingerprint tests prove equivalent content stability and material-change distinction.
- [x] Identity validation, scope separation, result metadata, retry, conflict, transaction, concurrency, lease, and boundary evidence pass.
- [x] The platform does not create a second owner for an identical retry, and changed content creates no second platform effect.
- [x] Package ownership, architecture, and unchanged OpenAPI checks pass.
- [x] Owning finance effects, authorization policy, audit persistence, frontend behavior, and outbox/inbox workflow remain outside this delivery item.

The Docker-backed integration and pinned SQLC drift checks are release-gate
verification and are not marked as passed until they run successfully.

## 6. Traceability

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-006` |
| Direct requirement IDs | Contributes to `GFR-006` and `GFR-007`; supports `GFR-008` and the immutable lineage expectations of `GFR-013`. |
| Workflow IDs | `WF-7.13`, `WF-7.14` |
| Quality contribution | Deterministic retry, conflict safety, transaction atomicity, and concurrency safety for later command handlers. |
| Exit evidence | Same-content, changed-content, transactional metadata, concurrent ownership, boundary, package-boundary, and contract-preservation checks pass. |

## 7. Source references

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
