# DLV-PLAT-005 — Shared Finance Primitives User Stories

| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-005` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | Planned |
| Dependency position | Builds on `DLV-PLAT-001` and `DLV-PLAT-003`; may use the transport conventions from `DLV-PLAT-004`. No finance capability implementation is required. |
| Exit evidence | Shared primitives have unit and serialization tests; money uses exact decimal arithmetic; invalid precision, scope, identity, and version values are rejected deterministically. |

## 1. Purpose and scope

**As the TALLY developer, I want shared finance primitives with explicit invariants, so that finance modules represent money, currency, accounting scope, identity, and optimistic-concurrency versions consistently.**

This item establishes reusable domain-neutral primitives required by later finance capabilities. It does not implement a finance aggregate, command handler, repository, endpoint behavior, authorization policy, or integration workflow.

## 2. Approved boundaries

- Money must not use binary floating point. Arithmetic and serialization preserve exact decimal meaning and currency scale rules.
- Currency is an explicit ISO currency identity with deterministic validation and supported precision metadata.
- `AccountingScope` explicitly contains tenant, legal entity, ledger, accounting book, and functional currency identifiers; scope is never inferred from ambient request context.
- Stable identifiers are opaque, validated values with type-safe construction where practical; they do not encode business state.
- Aggregate versions support expected-version checks and monotonic advancement for optimistic concurrency.
- Established financial facts remain immutable; these primitives do not provide destructive mutation helpers.
- Shared packages contain technical/domain primitives only and do not own finance module behavior or persistence schemas.

## 3. Explicit exclusions

- Finance aggregates, domain commands, domain events, repositories, migrations, or capability-specific schemas.
- Request fingerprints and business idempotency (`DLV-PLAT-006`).
- Outbox, inbox, workers, retries, delivery, and replay (`DLV-PLAT-007`).
- Authentication, authorization, audit evidence, or accounting-scope policy decisions.
- Currency-rate mastering, revaluation, translation, or FX calculations owned by Multi-Currency.
- API endpoint behavior or frontend screens.

## 4. User stories

### User Story 1 — Implement exact-decimal money and currency primitives

**As a finance module developer, I want exact-decimal money and validated currency values, so that calculations cannot silently lose monetary precision.**

- [x] Money construction rejects malformed decimal values and values that violate the agreed currency precision policy.
- [x] Money arithmetic is deterministic, preserves currency identity, and rejects incompatible currencies or invalid results.
- [x] Currency values validate the agreed canonical representation and expose the precision policy selected in Definition of Ready.
- [x] Serialization uses the exact-decimal representation selected in Definition of Ready and never converts money through binary floating point.
- [x] Tests cover zero, permitted negative values, precision boundaries, rounding boundaries, and incompatible currencies.

### User Story 2 — Implement explicit accounting-scope identity

**As a finance module developer, I want an explicit accounting scope, so that ledger-bound facts cannot be attributed to an inferred or incomplete scope.**

- [ ] `AccountingScope` contains tenant, legal entity, ledger, accounting book, and functional currency identifiers.
- [ ] Construction rejects missing, malformed, or inconsistent scope components.
- [ ] Equality and serialization include every scope component, including functional currency.
- [ ] The primitive does not read ambient tenant, request, session, or authentication context.
- [ ] Tests prove distinct ledgers or accounting books remain distinct even under the same legal entity.

### User Story 3 — Implement stable identity primitives

**As a finance module developer, I want validated stable identifiers, so that records and references remain unambiguous across module boundaries.**

- [ ] Required identifier types have validated construction and canonical serialization.
- [ ] Empty, malformed, and wrong-type identifiers are rejected deterministically.
- [ ] Identifier equality is value-based and does not depend on display formatting.
- [ ] Identifiers do not carry mutable lifecycle state or infer ownership outside their declared type.
- [ ] Tests cover round-trip serialization, equality, invalid input, and type distinction.

### User Story 4 — Implement aggregate version primitives

**As a finance module developer, I want explicit aggregate versions, so that concurrent changes can be detected without overwriting an established outcome.**

- [ ] A new aggregate has a defined initial version, and the version representation and valid range are recorded in Definition of Ready.
- [ ] Version advancement is monotonic and rejects overflow or invalid transitions.
- [ ] Expected-version comparison distinguishes a matching version from a stale version.
- [ ] Serialization round-trips versions without loss or implicit conversion.
- [ ] Tests cover initial, matching, stale, advancement, invalid, and boundary values.

### User Story 5 — Prove serialization and boundary behavior

**As the TALLY maintainer, I want focused verification for the shared primitives, so that later modules can adopt them without duplicating incompatible rules.**

- [ ] Unit tests cover all primitive invariants and negative cases.
- [ ] Serialization tests cover the exact Go boundary and API representation selected in Definition of Ready, without changing the OpenAPI contract unnecessarily.
- [ ] Tests prove no binary floating-point money representation is used in implementation or serialized output.
- [ ] Package ownership and import checks show shared primitives do not depend on finance bounded-context adapters or schemas.
- [ ] A documented root command runs the focused verification reproducibly.

## 5. Definition of Ready

- [x] User Story 1 uses `shopspring/decimal`, caller-provided immutable currency metadata, configured currency scale from `0..12`, and the PostgreSQL-compatible `numeric(38,12)` domain ceiling.
- [x] User Story 1 is implemented in `internal/platform/money` with explicit constructors, accessors, arithmetic methods, stable errors, and canonical amount-text serialization. The API continues to represent amount and currency separately.
- [ ] Boundaries with `DLV-PLAT-004`, `DLV-PLAT-006`, `DLV-PLAT-007`, and finance capability items are preserved.
- [ ] Five stories are small enough for one or a short chain of reviewable changes.

## 6. Definition of Done

- [ ] All five stories and acceptance criteria pass.
- [x] User Story 1 focused unit, serialization, and boundary tests pass.
- [x] User Story 1 money has no binary floating-point implementation or serialization path.
- [x] User Story 1 documentation identifies the primitives, invariants, command, and ownership boundary.
- [x] No finance capability, idempotency behavior, integration workflow, or adjacent delivery item was marked complete.

## 7. Traceability

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-005` |
| Direct FR/GFR IDs | Contributes to `GFR-001` and `GFR-008`; this item does not complete either control or `GFR-013`. |
| Quality contribution | Shared type-safety, precision, serialization, and optimistic-concurrency foundation. |
| Exit evidence | Unit and serialization tests pass without floating-point money; invalid primitive and boundary cases are covered. |

## 8. Source references

- `docs/specs/finance_domain_model_ddd.md` — `AccountingScope`, money/currency precision, identity, and optimistic concurrency rules.
- `docs/specs/finance_delivery_plan_v1.0.md` — platform foundation backlog and DLV-PLAT-005 exit evidence.
- `docs/specs/technical_specifications/01_backend_module_specifications_v1.0.md`
- `docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md`
- `docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md`
- `docs/specs/technical_specifications/06_security_identity_authorization_specifications_v1.0.md`
- `docs/specs/technical_specifications/10_technical_traceability_decisions_v1.0.md`
- `docs/backlog/stories/DLV-PLAT-004_user_stories.md`
