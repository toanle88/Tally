# EP-COA-001 — COA Segment Configuration User Stories

| Field | Value |
|---|---|
| Epic | EP-COA-001 — COA segment configuration |
| Status | Locally complete and intentionally closed on 2026-10-10; implementation and focused local verification evidence are recorded for User Stories 1–5; runtime and release qualification remain deferred |
| Milestone | M1 — Identity and accounting configuration |
| Dependencies | EP-OMD-001, EP-IAM-001; platform foundation is transitive through those dependencies |
| Delivery items | DLV-FR-COA-001 through DLV-FR-COA-005 |
| Owning bounded context | COA Segment Accounting — internal/coa, coa schema |
| Primary users | Chart-of-Accounts Administrator; Finance Approver |
| Authoritative records | SegmentDefinition, SegmentCombination, SegmentChangeRequest |
| Exit evidence | Domain, persistence, API, authorization, audit, concurrency, UI, accessibility, and approved-decision application evidence for all five stories; deferred external and full-release qualification is not claimed |

This document remains the epic planning artifact. All five user stories now
have traceable implementation and local verification records below. Runtime,
external, and release qualification obligations remain explicitly deferred.

**Current closure status (2026-10-10):** The local implementation scope for
`EP-COA-001` is complete and intentionally closed. The separate PostgreSQL
runtime, downstream cross-context, production, live Entra, full audit-chain,
performance/capacity, and release qualification obligations remain open and
are not claimed by this closure.

## 1. Outcome

Deliver the COA Segment Accounting slice as the authoritative owner for
effective-dated segment definitions, segment values, segment combinations, and
governed segment changes. Authorized users can maintain the configuration,
validate proposed combinations without changing business state, request
controlled changes, and apply an approved Workflow decision only after COA
revalidates the current subject version and effective-date policy.

Historical consumers retain the segment-definition and combination version
applicable to their established facts. A later configuration change must not
rewrite an established journal, report, or other downstream business fact.

## 2. Learning objective

Learn how to implement a supporting bounded context with aggregate ownership,
effective dating, scoped uniqueness, optimistic concurrency, idempotent
commands, read-only validation, approval handoff, safe cross-context
references, and a narrow application boundary for applying an externally owned
approval decision.

## 3. Scope

- Maintain SegmentDefinition records with segment type, code, name, status,
  and effective date range.
- Maintain SegmentValue entities under their owning SegmentDefinition,
  including descriptions, status, and effective date range.
- Maintain SegmentCombination records and validate proposed combinations
  against current segment values, restrictions, rule versions, and effective
  dates.
- Create and track SegmentChangeRequest records for governed changes to a
  segment definition, value, combination, or assignment covered by the
  approved COA contract.
- Apply an approved or rejected segment-change decision through
  ApplySegmentChangeApprovalDecision after revalidating the current subject
  version and effective-date policy.
- Provide the approved COA worklist and record screens as the UI entry points
  for definitions, values, combinations, changes, approvals, and validation
  exceptions.
- Apply IAM authorization, explicit scope, material-action audit, safe error
  handling, correlation, idempotency, optimistic concurrency, and bounded
  operational telemetry at each applicable command boundary.

## 4. Explicit exclusions

- No ownership or implementation of General Ledger charts of accounts,
  accounts, journals, posting, reversal, or posting gates. GL remains the
  owner of those records and financial effects.
- No Workflow & Approvals implementation. Workflow owns approval requests and
  decisions; COA consumes the approved decision reference and applies it only
  through its own aggregate boundary.
- No ownership of Organization & Master Data records and no direct writes to
  the organization schema. COA consumes only approved identifiers, scope, and
  reference versions through the allowed boundary.
- No Audit Integrity chain implementation. COA uses the approved audit port;
  it does not write audit-chain storage directly.
- No direct access to another bounded context's adapter, repository, or
  database schema.
- No new public route, request shape, response type, permission identifier,
  event, or persistence schema outside the approved specifications. The
  existing COA OpenAPI paths and common command/result/problem contracts are
  the contract baseline.
- No production data, live Entra qualification, production audit-chain
  qualification, full performance/capacity qualification, or release
  qualification from local implementation evidence alone.

## 5. Ownership and domain rules

### 5.1 Owning component

COA Segment Accounting owns the SegmentDefinition,
SegmentCombination, and SegmentChangeRequest aggregates in internal/coa and
the coa PostgreSQL schema. The approved persistence contract names the
following authoritative tables:

- coa.segment_definition
- coa.segment_combination
- coa.segment_change_request

The owning module is the only writer of these records. Other capabilities
consume stable COA identifiers and the immutable versions needed for their own
facts through approved application or integration contracts.

### 5.2 Domain objects and invariants

- SegmentDefinition is an aggregate root containing SegmentValue entities.
  Its value objects are SegmentDefinitionId, SegmentType, Code, Name,
  EffectiveDateRange, and SegmentStatus.
- SegmentCombination is an aggregate root containing
  SegmentCombinationValue entities. Its value objects are
  SegmentCombinationId, ValidationStatus, and EffectiveDateRange.
- SegmentChangeRequest is an aggregate root with ChangeRequestId,
  ChangeType, ApprovalRequestId, ApprovalDecisionReference, and
  RequestedEffectiveDate value objects.
- Segment definitions and values preserve lifecycle status, effective-date
  meaning, applicable uniqueness, and the version used by historical
  consumers. The exact approval applicability remains policy-driven and must
  come from the approved Workflow contract.
- Combination validation returns the applicable rule/version, effective-date
  result, and rejection reasons without changing authoritative business
  state.
- A change request records the requested subject and version; it does not
  silently mutate the subject while awaiting approval.
- Workflow owns the approval decision. COA applies
  ApplySegmentChangeApprovalDecision only after revalidating the current
  segment-definition version and effective-date policy.
- Approved configuration changes do not rewrite established downstream facts.
  Consumers retain the source identifier and version applicable when their
  fact was established.
- State-changing commands use the platform idempotency and optimistic
  concurrency contracts. A repeated command with the same identity and
  fingerprint returns the established result; changed content under a reused
  identity returns a typed conflict and creates no second effect.
- Material state changes require authorization and attributable audit
  evidence. Audit evidence contains references and fingerprints, not raw
  restricted values.
- Every command carries the explicit tenant, accounting, legal-entity, or
  other operation-specific scope required by the authorization contract.
  Scope is never inferred from ambient context alone.

## 6. Traceability

| Source | Coverage in this epic |
|---|---|
| DLV-FR-COA-001 / FR-COA-001 | User Story 1 — Maintain segment definitions. |
| DLV-FR-COA-002 / FR-COA-002 | User Story 2 — Maintain segment values. |
| DLV-FR-COA-003 / FR-COA-003 | User Story 3 — Validate segment combinations. |
| DLV-FR-COA-004 / FR-COA-004 | User Story 4 — Request segment changes. |
| DLV-FR-COA-005 / FR-COA-005 | User Story 5 — ApplySegmentChangeApprovalDecision. |
| M1 | Identity and accounting configuration milestone. |
| EP-PLAT-001, EP-IAM-001, EP-OMD-001 | Platform, authorization, and organization/reference prerequisites. |
| DDD §§2.14, 8, 9, 10, 11 | COA aggregates, authorization, concurrency, effective dating, audit, and retention rules. |
| PRD §3.14 | COA users, authoritative records, controls, operations, and visible results. |
| UX §7.14 | COA worklist, screens, actionable states, lineage, evidence, and sensitivity behavior. |
| NFR-SEC-004, NFR-SEC-009–012, NFR-SEC-017 | Default-deny access, protection, attribution, and permission-preserving exports. |
| NFR-PRV-001–003, NFR-PRV-008, NFR-PRV-009 | Purpose, minimization, field-level access, synthetic data, and safe notifications. |
| NFR-REL-001–004, NFR-REL-008, NFR-REL-009, NFR-REL-014 | Durability, idempotency, conflict safety, historical version preservation, and single ownership. |
| NFR-AUD-001, NFR-AUD-003, NFR-AUD-004 | Complete, attributable, append-only material-action evidence. |
| NFR-PERF-001, NFR-PERF-002, NFR-PERF-005, NFR-PERF-006 | Validation, state-change, worklist, and authorized-search responsiveness. |
| NFR-ACC-005–007 | Contrast, zoom/reflow, validation announcements, and non-disruptive status updates. |
| NFR-MNT-006, NFR-MNT-008 | Authorized configuration change, recovery path, and synchronized documentation. |
| NFR-TST-001, NFR-TST-002, NFR-TST-004, NFR-TST-006, NFR-TST-007 | Functional, UX, fault/duplicate/concurrency, security, and accessibility evidence. |
| QG-01, QG-02, QG-03, QG-04, QG-05, QG-06, QG-08, QG-10 | M1 minimum quality gates. QG-07 and QG-09 remain later qualification obligations where applicable. |

## 7. User stories

### User Story 1 — Maintain segment definitions

**Delivery item:** DLV-FR-COA-001 / FR-COA-001
**Existing API operation:** coaMaintainSegmentDefinitions
**Method and path:** PUT /api/v1/coa-segments/configuration/maintain-segment-definitions
**Permission:** finance.coa.maintain.segment.definitions
**Primary screen:** COA-SCR-01; worklist entry COA-WS-01

As a Chart-of-Accounts Administrator, I want to define and maintain
effective-dated COA segments so that downstream accounting capabilities can
use an authoritative segment structure without losing historical version
meaning.

Acceptance criteria:

- [ ] An authorized command can create or maintain segment type, code, name,
  status, and effective date range as defined by FR-COA-001.
- [ ] Applicable scope, uniqueness, lifecycle, effective-date, approval, and
  validation rules are evaluated before persistence; invalid input does not
  partially change the aggregate.
- [ ] The accepted result identifies the authoritative segment-definition
  identity, version, effective interval, lifecycle/approval state, and
  validation outcome without exposing restricted values.
- [ ] If-Match, idempotency, correlation, authorization, and material-action
  audit follow the approved platform contracts. A stale version is a typed
  conflict and a replay returns the established result.
- [ ] A later definition change preserves the source version required by
  established downstream facts and does not rewrite those facts.
- [ ] COA-WS-01 and COA-SCR-01 show the definition identity, state, effective
  dates, approval/validation status, permitted next action, and safe conflict
  or denial outcome.

Suggested implementation steps:

1. Define SegmentDefinition and SegmentValue ownership boundaries and domain
   rules in internal/coa.
2. Add the coa-owned migration, repository interface, persistence mapping,
   expected-version handling, and transaction coordination.
3. Implement the existing OpenAPI operation and permission through the COA
   application handler, preserving CommandRequest, EstablishedResult, and
   problem-details contracts.
4. Build the COA-SCR-01 record view and the definition portion of COA-WS-01
   with shared accessible identity, state, lineage, action, and evidence
   components.
5. Add domain, persistence, API, authorization, concurrency, privacy,
   accessibility, and safe-observability evidence.

Required test evidence:

- [ ] Domain tests for definition fields, value-object validation, status
  transitions, effective-date boundaries, uniqueness, and version conflicts.
- [ ] Persistence tests proving coa schema ownership, constraints,
  transaction rollback, and preservation of historical source versions.
- [ ] API tests for permission/scope enforcement, idempotent retry, If-Match
  conflicts, typed problem details, correlation, audit failure, and safe
  response/error projections.
- [ ] Component and Playwright tests for COA-WS-01 and COA-SCR-01, including
  keyboard behavior, validation summaries, zoom/reflow, and safe status
  announcements.

Likely files and packages involved:

- New internal/coa domain, service, repository, projection, and focused test
  files for SegmentDefinition and its owned aggregate boundary.
- New coa-owned migration and query sources under db/migrations/coa and
  db/queries/coa, plus the corresponding sqlc target and generated package
  following the existing organization persistence pattern.
- New COA HTTP adapter and tests under internal/platform/httpapi, following
  the existing organization handler and problem-mapping conventions.
- cmd/api/identity_runtime.go for the COA operation ID, repository/service
  construction, durable idempotency configuration, and handler wiring.
- web/src/routes/route-registry.ts and web/src/routes/router.tsx for
  COA-WS-01 and COA-SCR-01, plus a new COA workspace/record component and
  focused component/accessibility tests.
- contracts/openapi/paths/coa-segments.yaml and generated API artifacts are
  contract inputs/baselines. Do not change them unless a source-backed
  contract correction is approved.

Focused planning notes:

- The generated COA operations provided the original transport boundary; the
  initial `internal/coa` segment-definition slice now extends it. Generated
  scaffolding alone remains insufficient delivery evidence for the remaining
  COA stories.
- SegmentValue is an entity inside the SegmentDefinition aggregate in the DDD,
  but FR-COA-002 owns segment-value maintenance. User Story 1 should establish
  the aggregate boundary without delivering the separate value-maintenance
  workflow.
- The current sources name applicable uniqueness, lifecycle, effective-date,
  approval, and historical-version controls but do not define every concrete
  status value or uniqueness key. The approved User Story 1 decision is:
  `Draft -> Active`, `Active <-> Suspended`, and `Draft`, `Active`, or
  `Suspended` may transition to terminal `Retired`; within an accounting
  scope, `segmentType + code` may not overlap across effective date ranges.
  The approved command data fields are `action`, `segmentDefinitionId`,
  `segmentType`, `code`, `name`, `status`, `effectiveDateFrom`, and
  `effectiveDateTo`. Effective-date endpoints are inclusive, and date ranges
  for the same scoped type/code must not overlap. These decisions are the
  source-backed implementation baseline for this story and are not inferred
  from the generic CommandRequest at runtime.
- IAM remains the authorization decision point and Audit Integrity remains the
  audit-chain owner. The COA service must use those ports and fail closed when
  required authorization or audit evidence cannot be established.

Story-level definition of done:

- [ ] SegmentDefinition domain behavior, versioning, effective-date handling,
  and approved validation rules are implemented and tested.
- [ ] The coa schema, migration, repository, and generated persistence output
  preserve ownership, atomicity, revision/source-version history, and stale
  version behavior.
- [ ] The existing coaMaintainSegmentDefinitions operation is wired without
  changing its approved route, permission, common result, or problem contract.
- [ ] COA-WS-01 and COA-SCR-01 expose only authorized safe projections and
  provide accessible validation, conflict, denial, and next-action states.
- [ ] Required domain, persistence, API, authorization, audit, idempotency,
  concurrency, UI, and accessibility evidence is attached.

#### User Story 1 implementation evidence — 2026-10-01

Implementation is on branch `feat/coa-maintain-segment-definitions` and keeps
the approved COA OpenAPI operation, permission, and common command/result
contracts unchanged. The slice adds the `internal/coa` domain/service and
PostgreSQL adapter, the coa-owned migration and sqlc target, runtime wiring,
the HTTP adapter, and COA-WS-01/COA-SCR-01 with the explicitly labelled local
safe read adapter and live mutation boundary.

Verified locally:

- `GOCACHE=/tmp/tally-go-cache go test ./...` — PASS.
- `GOCACHE=/tmp/tally-go-cache go test -tags integration ./internal/platform/database -run '^$'` — PASS; integration-tagged persistence tests compile.
- `make db-migrate-inventory && make db-migrate-validate && make db-migrate-check` — PASS.
- `make persistence-check` — PASS; clean PostgreSQL 18 persistence integration
  tests passed, including COA migration initialization and repository
  revision/rollback coverage.
- `make sqlc-compile` and `make db-sqlc-generate` — PASS; generated COA query output is present under `internal/coa/coadb`.
- `pnpm exec vitest run src/app/coa-segment-workspace.test.tsx src/routes/route-registry.test.ts src/routes/router.test.tsx --pool=threads --maxWorkers=1` — PASS, 14 tests.
- `pnpm run build` — PASS.
- `pnpm exec playwright test --config=playwright.config.ts --grep='COA-'` — PASS, 2 COA accessibility/keyboard tests.

Qualification still open:

- `make db-sqlc-check` remains a post-commit gate because its repository guard
  requires generated output to be committed; `make sqlc-compile` and
  `make db-sqlc-generate` passed and the generated COA package is present.
- The full frontend Vitest run was not used as completion evidence: the first
  run exposed one corrected COA test selector and an existing worker-start
  timeout in `src/lib/auth/authenticated-fetch.test.ts`; a constrained
  single-worker rerun did not complete in the sandbox. Focused COA and route
  tests passed.
- Production Entra authorization, production Audit Integrity, performance,
  capacity, and release qualification remain outside this local slice.

### User Story 2 — Maintain segment values

**Delivery item:** DLV-FR-COA-002 / FR-COA-002
**Existing API operation:** coaMaintainSegmentValues
**Method and path:** PUT /api/v1/coa-segments/configuration/maintain-segment-values
**Permission:** finance.coa.maintain.segment.values
**Primary screen:** COA-SCR-02; worklist entry COA-WS-01

As a Chart-of-Accounts Administrator, I want to maintain effective-dated
values under an authoritative segment definition so that valid combinations
can be configured without invalidating established history.

Acceptance criteria:

- [ ] An authorized command can create, update, activate, suspend, and
  effective-date segment values as defined by FR-COA-002.
- [ ] The command validates the owning segment definition, applicable value
  uniqueness, lifecycle, effective-date, approval, and assignment rules
  before committing; invalid input has no partial effect.
- [ ] The result identifies the segment-value identity, owning definition,
  version, effective interval, lifecycle/approval state, and validation
  outcome without exposing restricted values.
- [ ] If-Match, idempotency, correlation, authorization, and audit behavior
  follow the existing command contract. Stale updates and changed-content
  replays return typed conflicts without silently overwriting newer values.
- [ ] Suspending or end-dating a value makes its future applicability
  explicit; it does not rewrite historical combinations or established
  downstream facts.
- [ ] COA-WS-01 and COA-SCR-02 show the owning definition, value status,
  effective dates, validation/approval state, next action, and safe failure
  information.

Suggested implementation steps:

1. Implement SegmentValue lifecycle and parent-definition invariants within
   the SegmentDefinition aggregate.
2. Add the coa repository transaction and constraints for parent identity,
   scoped uniqueness, effective intervals, and aggregate version.
3. Wire coaMaintainSegmentValues through the approved HTTP, authorization,
   idempotency, audit, and problem-details boundaries.
4. Build the value portion of COA-WS-01 and COA-SCR-02 using the same
   authoritative-record and accessibility patterns as the definition screen.
5. Test parent-version conflicts, effective-date edges, denied fields,
   replayed commands, and downstream source-version preservation.

Required test evidence:

- [ ] Domain tests for value lifecycle, parent association, status changes,
  effective dates, uniqueness, and aggregate-version conflicts.
- [ ] Persistence tests for coa schema ownership, parent/child atomicity,
  constraints, rollback, and historical version retention.
- [ ] API tests for authorization, scope, idempotency, If-Match, typed
  rejection, audit failure, correlation, and safe negative disclosure.
- [ ] Component and Playwright tests for value maintenance, filtering,
  keyboard use, accessible validation, and responsive/reflow behavior.

#### User Story 2 implementation evidence — 2026-10-04

Implementation is on branch `feat/coa-maintain-segment-values`. The slice keeps
the approved `coaMaintainSegmentValues` contract and generated clients, adds
`SegmentValue` as a child entity of the COA-owned `SegmentDefinition`
aggregate, and stores current child state plus parent revision snapshots in
the `coa` schema. Mutations require the parent aggregate version, use the
existing lifecycle, enforce parent interval containment and normalized-value
range uniqueness, and commit child state, parent revision, audit reference,
and durable idempotency in one transaction. No assignment-specific rule was
invented because the approved model does not yet expose an assignment-rule
representation or validation port; approval remains policy-driven.

The HTTP adapter maps the existing operation to the value service and exact
permission. COA-WS-01 now filters and links safe value projections, and
COA-SCR-02 maintains value, description, lifecycle, and effective dates with
parent-version retry guidance. The local safe adapter remains explicitly
labelled because no approved COA read endpoint exists.

Verified locally:

- `GOCACHE=/tmp/tally-go-cache go test ./...` — PASS.
- `GOCACHE=/tmp/tally-go-cache go test -tags integration ./internal/platform/database -run '^$'` — PASS; integration-tagged persistence tests compile.
- `make db-migrate-validate` and `make db-migrate-check` — PASS.
- `make sqlc-compile`, `make db-sqlc-generate`, and `GOCACHE=/tmp/tally-go-cache make sqlc-check` — PASS; the generated COA value query/model is present.
- `GOCACHE=/tmp/tally-go-cache make api-check`, `make api-generate-check`, and `make api-ts-check` — PASS; the existing COA value contract and generated artifacts remain aligned.
- `pnpm exec vitest run src/app/coa-segment-workspace.test.tsx src/routes/route-registry.test.ts src/routes/router.test.tsx --pool=threads --maxWorkers=1` — PASS, 17 tests.
- `pnpm build` — PASS.
- `pnpm exec playwright test --config=playwright.config.ts --grep='COA-'` — PASS, 3 COA accessibility/keyboard tests.

Qualification still open:

- `make persistence-check` was blocked because the Docker daemon was
  unavailable in the execution environment. The integration-tagged tests
  compile, but clean PostgreSQL execution is not claimed here.
- `make db-sqlc-check` remains a post-commit repository guard; the direct
  `sqlc-check` drift check, SQLC compile, and generation passed.
- Production Entra authorization, production Audit Integrity, performance,
  capacity, recovery, and release qualification remain outside this local
  slice.

### User Story 3 — Validate segment combinations

**Delivery item:** DLV-FR-COA-003 / FR-COA-003
**Existing API operation:** coaValidateSegmentCombinations
**Method and path:** POST /api/v1/coa-segments/actions/validate-segment-combinations
**Permission:** finance.coa.validate.segment.combinations
**Primary screen:** COA-SCR-03; worklist entry COA-WS-01

As a Chart-of-Accounts Administrator or authorized finance user, I want to
validate a proposed segment combination so that I can see whether it is
allowed and effective before another business action relies on it.

Implementation baseline for the M1 read-only slice:

- The existing generic `CommandRequest` carries `accountingScopeId`,
  `businessDate`, and `data.segmentValues[]` references containing a
  `segmentDefinitionId` and `segmentValueId`. The existing generic
  `EstablishedResult` carries `validationStatus`, `effectiveDateResult`,
  `sourceVersions`, `invalidValues`, `restrictions`, `rejectionReasons`, and
  `nextAction` in its action-specific `data` map.
- Validation reads the COA-owned definition/value aggregate through one
  consistent source snapshot. A selected value reports its parent definition
  aggregate version and revision because SegmentValue has no independent
  aggregate version in the approved M1 model.
- M1 restrictions are the source-backed scope, lifecycle, missing-value,
  duplicate-selection, and effective-date checks. A separate rule-catalog
  identifier/version and persisted SegmentCombination record are deferred
  until an approved source-backed representation exists.
- The operation is read-only with no COA mutation, audit revision, outbox
  event, migration, or generated API contract change. Idempotency remains
  required for deterministic established results and changed-content conflict.

Acceptance criteria:

- [x] An authorized request evaluates the proposed combination against the
  current segment definitions, values, restrictions, applicable source
  versions, and requested effective date. The approved M1 model has no
  separate rule-catalog version to evaluate.
- [x] The response identifies validation status, applicable source versions,
  effective-date result, invalid values, restrictions, and rejection reasons
  required by the available FR-COA-003 contract. A separate rule-catalog
  version remains deferred until an approved source-backed representation
  exists.
- [x] Validation does not create, update, activate, suspend, approve, or
  otherwise change authoritative business state.
- [x] The request preserves the approved idempotency, correlation,
  authorization, scope, and safe problem-details contract. Repeated
  identical validation requests return a deterministic established result;
  changed content under a reused identity returns a typed conflict.
- [x] A concurrent definition/value version change is not hidden: the result
  either identifies the source version used or returns a typed conflict or
  retryable outcome according to the approved command contract.
- [x] COA-SCR-03 displays valid/invalid outcomes, invalid values,
  restrictions, effective-date reasons, source/rule versions, and a clear
  next action without becoming a second mutation surface.

Suggested implementation steps:

1. Define the pure combination-validation policy and result types within the
   COA application/domain boundary.
2. Read definitions and values through COA-owned repositories and establish a
   consistent source-version snapshot for the validation.
3. Implement the existing validation operation without adding a persistence
   side effect or a new public route.
4. Build COA-SCR-03 with safe aggregate errors, deterministic result display,
   accessible announcements, and retry/conflict handling.
5. Add normal, boundary, invalid-value, effective-date, concurrent-change,
   duplicate-request, and dependency-unavailable tests.

Required test evidence:

- [x] Domain/application tests prove allowed, disallowed, inactive,
  out-of-range, duplicate, restricted, and mixed-version combinations.
- [x] API tests prove the operation is read-only with respect to business
  state, is authorized and scoped, returns deterministic results, and
  preserves idempotency/correlation/problem contracts.
- [x] Persistence or repository tests prove no combination or definition
  mutation occurs during validation and that source versions are consistent.
- [x] Component and Playwright tests cover invalid-value summaries,
  effective-date reasons, focus behavior, keyboard access, and zoom/reflow.

#### User Story 3 implementation evidence — 2026-10-06

The M1 read-only slice is implemented on branch
`feat/coa-validate-segment-combinations`. It keeps the existing generic API
operation and generated contract, adds COA-owned snapshot validation with
durable idempotency in the PostgreSQL runtime, wires exact IAM permission and
scope checks, and adds COA-SCR-03 with a read-only result surface. The scope
boundary also prevents source-version or lifecycle detail from being returned
for definitions outside the requested accounting scope.

Verified locally:

- `GOCACHE=/tmp/tally-go-cache go test ./...` — PASS.
- `GOCACHE=/tmp/tally-go-cache go test -race ./internal/coa ./internal/platform/httpapi` — PASS.
- `make db-migrate-validate` — PASS; no migration was required for this read-only slice.
- `GOCACHE=/tmp/tally-go-cache OPENAPI_CHECK_TIMEOUT_SECONDS=120 make api-check` — PASS; generated artifacts remain aligned.
- `pnpm -C web exec vitest run src/app/coa-segment-workspace.test.tsx src/routes/route-registry.test.ts src/routes/router.test.tsx --pool=threads --maxWorkers=1` — PASS, 20 tests.
- `pnpm -C web exec tsc -p tsconfig.playwright.json --noEmit` — PASS.
- `pnpm -C web build` — PASS.
- `pnpm -C web exec playwright test --config=playwright.config.ts --grep='COA-'` — PASS, 4 COA accessibility/keyboard tests.

Qualification still open:

- `make persistence-check` remains blocked because Docker is unavailable in
  the execution environment; the PostgreSQL snapshot adapter was compiled,
  but clean PostgreSQL execution is not claimed here.
- No separate rule catalog/version or persisted `SegmentCombination` record
  is implemented because the approved M1 sources do not define one.
- COA-SCR-03 still reads local synthetic safe projections because no approved
  COA read endpoint exists; only the validation operation is live-wired.
- Production Entra authorization, Audit Integrity, performance, capacity,
  recovery, and release qualification remain outside this local slice.

### User Story 4 — Request segment changes

**Delivery item:** DLV-FR-COA-004 / FR-COA-004
**Existing API operation:** coaRequestSegmentChanges
**Method and path:** POST /api/v1/coa-segments/actions/request-segment-changes
**Permission:** finance.coa.request.segment.changes
**Primary screen:** COA-SCR-04; worklist entry COA-WS-01

As a Chart-of-Accounts Administrator, I want to request a governed change to a
segment definition, value, combination, or approved assignment so that the
change can be reviewed and applied with a traceable effective date and source
version.

Acceptance criteria:

- [x] An authorized command creates a SegmentChangeRequest for the supported
  change type and subject, including the requested effective date and
  subject version required by FR-COA-004.
- [x] The request exposes its own reference, subject identity/version,
  requested effective date, approval status, and any validation conflict or
  rejection without mutating the subject configuration prematurely.
- [x] A missing, unauthorized, stale, invalid, or unsupported subject is
  rejected atomically with a typed result and no partial request or subject
  change.
- [x] Idempotency, correlation, explicit scope, authorization, and material
  action audit are enforced. A repeated command returns the established
  request; changed content under a reused identity returns a conflict.
- [x] Workflow remains the owner of the approval decision. COA records the
  approval request reference and exposes the current decision/application
  state without inventing a new approval policy.
- [ ] COA-WS-01 and COA-SCR-04 show the requested change, subject version,
  requested effective date, approval state, impacted-record information
  available from the approved contract, conflict/rejection, and next action.

Suggested implementation steps:

1. Define SegmentChangeRequest lifecycle and subject/version references in
   internal/coa without duplicating the target aggregate.
2. Add the coa-owned request table, repository, constraints, and transaction
   coordination for atomic request creation.
3. Connect the existing operation to the approved authorization, validation,
   idempotency, audit, and Workflow decision-reference ports.
4. Build COA-SCR-04 and its worklist states using the authoritative request
   record, not a second mutation model.
5. Test request creation, duplicate replay, stale subject, invalid change,
   unauthorized scope, dependency failure, safe errors, and approval handoff.

Required test evidence:

- [x] Domain tests for change types, subject/version capture, requested
  effective dates, lifecycle states, and conflict behavior.
- [ ] Persistence tests for atomic request creation, uniqueness/fingerprint
  constraints, coa schema ownership, and rollback on audit/dependency failure.
- [x] API tests for authorization, scope, idempotency, typed conflicts,
  correlation, audit evidence, and no premature subject mutation.
- [x] Component and Playwright tests for request review, approval status,
  impact/exception presentation, keyboard behavior, and safe announcements.

#### User Story 4 implementation evidence — 2026-10-07

Implementation is on branch `feat/coa-request-segment-changes`.

Approved implementation decisions recorded during delivery:

- This slice supports governed requests for existing `SegmentDefinition` and
  `SegmentValue` subjects. Combination and assignment requests return a typed
  unsupported-subject result because their authoritative models and Workflow
  contract are not defined in the approved M1 sources.
- The command requires a Workflow-created `approvalRequestId`. COA validates
  and records the opaque reference; it does not create approval policy,
  decide the request, or implement Workflow.
- `proposedChange` reuses the existing definition/value maintenance fields.
  The request stores a canonical proposal snapshot and never mutates the
  source subject. A value request captures the owning definition aggregate
  version as its subject version.

Implemented:

- `internal/coa/segment_change_request.go` owns command validation, subject
  version/fingerprint checks, lifecycle projection, authorization, idempotent
  request creation, and audit evidence.
- `internal/coa/postgres_segment_change_request_repository.go` persists the
  request in the COA transaction, locks and re-reads the subject, writes the
  audit reference, and coordinates durable idempotency. The memory repository
  mirrors duplicate and rollback behavior for local/UI runtime use.
- `db/migrations/coa/00003_create_segment_change_request_table.sql` adds the
  COA-owned request table, status constraints, subject index, and pending
  subject/proposal fingerprint uniqueness constraint. SQLC query source and
  generated COA artifacts are present.
- `internal/platform/httpapi/coa_handler.go` wires the existing
  `CoaRequestSegmentChanges` operation and common command/result/problem
  contracts. Runtime wiring uses the exact COA permission and the existing
  audit boundary.
- `COA-SCR-04` is registered and provides definition/value request entry,
  Workflow reference capture, safe pending/not-applied projection, and an
  explicit no-premature-mutation boundary. The screen still uses the existing
  local safe adapter for reads because no approved COA request-read endpoint
  exists.

Verified locally:

- `GOCACHE=/tmp/tally-go-cache go test ./...` — PASS.
- `GOCACHE=/tmp/tally-go-cache go test -race ./internal/coa ./internal/platform/httpapi` — PASS.
- `make db-migrate-validate`, `make db-migrate-check`, `make sqlc-compile`,
  and `make sqlc-check` — PASS.
- `GOCACHE=/tmp/tally-go-cache make api-generate-check`, `make api-ts-check`,
  and `GOCACHE=/tmp/tally-go-cache OPENAPI_CHECK_TIMEOUT_SECONDS=180 make api-check` — PASS.
- `pnpm exec tsc -p tsconfig.json --noEmit`, Playwright TypeScript checking,
  `pnpm build`, focused Vitest (22 tests), and focused Playwright COA tests
  (5 tests) — PASS.
- The integration test source compiles with
  `go test -tags=integration ./internal/platform/database -run '^$'` and
  covers PostgreSQL atomicity, duplicate replay, and audit rollback.

Qualification still open:

- `make persistence-check` is blocked because the Docker daemon is unavailable;
  clean PostgreSQL execution and the integration test runtime are not claimed.
- `make db-sqlc-check` intentionally rejects the uncommitted generated COA
  artifacts in this working tree; the deterministic `make sqlc-check` passed.
- COA-WS-01 is not an authoritative request worklist yet, and impact analysis
  and the Story 5 application result are still shown through
  the local safe adapter until an approved COA request-read endpoint exists.

### User Story 5 — Apply segment-change approval decision

#### Implementation status

User Story 5 is implemented for the approved M1 application slice. The
Workflow-owned decision is accepted through the existing generic command
boundary with approval-request/decision references, policy and decision
versions, candidate fingerprint, approver, and decision timestamp. COA
revalidates the stored request, current subject version/fingerprint, scope,
effective-date and lifecycle rules, and definition/value uniqueness before
recording an atomic applied, unchanged, rejected, or conflict result.

Implemented:

- `internal/coa/segment_change_approval.go` owns the application state machine,
  typed conflict/rejection results, authorization, replay/idempotency handling,
  audit evidence, and memory behavior for definitions and values.
- `internal/coa/postgres_segment_change_approval_repository.go` applies the
  definition/value change, revision, request terminal state, audit reference,
  and durable idempotency result in one COA transaction with subject locking.
- `db/migrations/coa/00004_apply_segment_change_approval_decision.sql` and the
  regenerated SQLC artifacts persist approval/application metadata and terminal
  lifecycle constraints.
- The existing
  `coaApplySegmentChangeApprovalDecision` HTTP operation is wired to the exact
  COA permission and typed problem results. `COA-SCR-04` captures the generic
  Workflow decision evidence and presents applied, rejected, unchanged, and
  conflict outcomes; `COA-WS-01` links safe request projections to the screen.

Verified locally:

- Domain/application coverage includes approved definition application and
  replay, rejected no-mutation, unchanged approved, stale conflict, invalid
  candidate reference, and changed-content idempotency conflict.
- HTTP coverage exercises request creation followed by approval application,
  resulting subject version, and subject mutation only at the application
  boundary. Frontend component and focused accessibility coverage exercise
  decision capture and applied-result presentation.
- Migration, SQLC, OpenAPI/TypeScript contract, Go, frontend, and focused
  Playwright checks are recorded in the delivery handoff for this change.

Qualification and deferred scope:

- PostgreSQL runtime/integration execution remains unclaimed because the local
  Docker daemon is unavailable. The persistence implementation and migration
  checks passed; clean-database atomicity, rollback, and concurrent runtime
  evidence require the integration environment.
- No COA event/outbox publication was added because the current approved
  technical event inventory does not define a COA segment-application
  publication contract. Any future publication must use the transactional
  outbox and an approved versioned event contract.
- Combination and assignment application remain explicitly unsupported until
  their owning aggregate/contract is approved. Authoritative request reads,
  impact analysis, and a full external Workflow delivery/recovery harness also
  remain outside this slice.

**Delivery item:** DLV-FR-COA-005 / FR-COA-005
**Existing API operation:** coaApplySegmentChangeApprovalDecision
**Approved application handler contract:** apply-segment-change-approval-decision.v1
**Method and path:** POST /api/v1/coa-segments/actions/apply-segment-change-approval-decision
**Permission:** finance.coa.apply.segment.change.approval.decision
**Primary screen:** COA-SCR-04; worklist entry COA-WS-01

As an authorized COA process or actor, I want to apply the Workflow-owned
approval decision for a segment change so that the authoritative COA state
reflects the approved outcome only when the current subject and policy still
match.

Acceptance criteria:

- [ ] The operation accepts the approved decision reference and applies the
  DDD-defined preconditions, invariants, approval rules, expected-version
  rules, and correction semantics for FR-COA-005.
- [ ] Before applying an approval, COA revalidates the current
  SegmentChangeRequest, subject version, effective-date policy, and any
  applicable uniqueness or combination restrictions.
- [ ] An approved decision applies the governed change atomically; a rejected
  decision leaves the subject configuration unchanged while recording the
  decision outcome. Unchanged and conflict outcomes are explicit and
  distinguishable.
- [ ] A stale subject, superseded request, duplicate decision, invalid
  approval reference, policy conflict, or unavailable required dependency
  returns a typed result and does not create a partial configuration effect.
- [ ] Repeating the same decision identity and fingerprint returns the
  established result. Reusing the identity with changed content returns a
  conflict and does not apply a second effect.
- [ ] The applied decision, resulting COA version/state, approval reference,
  effective-date result, and conflict/rejection are visible in COA-SCR-04 and
  COA-WS-01 without exposing restricted values.
- [ ] The approved application-handler and transactional integration
  boundaries are replay-safe. Any integration publication uses the
  transactional outbox and the approved versioned contract; no direct
  cross-schema write is introduced.

Suggested implementation steps:

1. Define the COA application handler for
   ApplySegmentChangeApprovalDecision and the decision/result state machine.
2. Revalidate the current request, subject aggregate version, effective-date
   policy, and applicable uniqueness rules inside one owning-module
   transaction.
3. Persist the resulting COA state and audit reference atomically with
   idempotency and conflict handling.
4. Implement the approved apply-segment-change-approval-decision.v1 boundary
   and transactional outbox behavior only where the existing integration
   specification requires it.
5. Complete the COA-SCR-04 applied/rejected/unchanged/conflict states and add
   restart, replay, concurrent-approval, and dependency-failure evidence.

Required test evidence:

- [ ] Domain/application tests for approved, rejected, unchanged, stale,
  superseded, invalid, duplicate, and conflict outcomes.
- [ ] Persistence tests for atomic application, expected-version checks,
  uniqueness/effective-date constraints, rollback, and preservation of
  prior established versions.
- [ ] API and handler tests for authorization, idempotency, replay, approval
  reference validation, correlation, audit failure, outbox transactionality,
  and safe problem details.
- [ ] Integration tests for ordered/replayed application-handler delivery,
  duplicate suppression, owned exceptions, and recovery without a second
  business effect.
- [ ] Component and Playwright tests for applied, rejected, unchanged, and
  conflict states, including accessible status announcements and safe
  next-action guidance.

## 8. Cross-cutting delivery contract

### 8.1 Existing API surface

The repository already contains the following approved COA OpenAPI
operations. Implementation must preserve their operation IDs, paths,
permissions, common CommandRequest/EstablishedResult/problem-details types,
declared status classes, correlation behavior, and idempotency headers.

| Operation | Method and path | Permission | Expected-version header |
|---|---|---|---|
| coaMaintainSegmentDefinitions | PUT /coa-segments/configuration/maintain-segment-definitions | finance.coa.maintain.segment.definitions | If-Match |
| coaMaintainSegmentValues | PUT /coa-segments/configuration/maintain-segment-values | finance.coa.maintain.segment.values | If-Match |
| coaValidateSegmentCombinations | POST /coa-segments/actions/validate-segment-combinations | finance.coa.validate.segment.combinations | None declared; source version must follow the approved command/application contract |
| coaRequestSegmentChanges | POST /coa-segments/actions/request-segment-changes | finance.coa.request.segment.changes | None declared; subject version is part of the approved command semantics |
| coaApplySegmentChangeApprovalDecision | POST /coa-segments/actions/apply-segment-change-approval-decision | finance.coa.apply.segment.change.approval.decision | None declared; current subject version is revalidated by COA |

All five operations require the approved idempotency-key contract. The
validation operation is business-state read-only even though its request
contract requires an idempotency key.

### 8.2 Authorization, audit, and privacy

- IAM remains the authorization decision point. Each operation checks its
  exact COA permission, explicit accounting/business scope, and the
  operation-specific dimensions that apply; the default is deny.
- Material configuration changes, change requests, approval application,
  denied attempts, and sensitive access attempts use the audit boundary with
  actor, authentication subject, scope, purpose, aggregate/action,
  authorization/approval outcome, correlation, causation, and before/after
  fingerprints.
- Raw restricted values, policy payloads, secrets, credentials, or unrelated
  personal/bank/tax data are not audit fields, logs, traces, metrics,
  notifications, errors, exports, or accessible names.
- If required authorization, audit, or policy evidence is unavailable, the
  material action fails closed and does not commit a state change.
- Worklist counts, filters, bulk actions, and any future export projection
  apply row and field permissions before calculating or rendering results.

### 8.3 Consistency, concurrency, and integration

- Each aggregate mutation is atomic within the coa owning boundary and uses
  optimistic concurrency and the existing idempotency/fingerprint contract.
- Validation reads a consistent source-version view and has no authoritative
  mutation effect.
- A later definition/value change preserves the source version used by
  established downstream facts. Dependent modules never write COA tables.
- Workflow approval references are immutable inputs to COA application;
  applying a decision always revalidates the current COA state.
- Approved cross-context publication, when required by the approved
  specification, is durable and replay-safe through the transactional
  outbox. A delivery failure does not create a second COA mutation or erase
  the accepted source version.
- Exceptions identify the source aggregate, version, correlation,
  dependency, retry eligibility, and next action without exposing restricted
  values.

### 8.4 UI and observability

- COA-WS-01 groups work by actionable state and exposes authorized scope,
  owner, age, approval, exception, and next action.
- COA-SCR-01 through COA-SCR-04 use the shared identity, state, action,
  lineage, evidence, and sensitivity components. Search links to the
  authoritative record and does not create another mutation surface.
- Validation, denied, stale, unavailable, approval, applied, unchanged, and
  conflict outcomes are typed, localized through the existing UI contract,
  and announced safely to assistive technology.
- Logs, traces, and metrics are bounded operational telemetry, not
  authoritative audit evidence. They include stable support references,
  operation/result classification, scope-safe identifiers, versions, and
  correlation without embedding request payloads or secrets.

## 9. Dependencies and sequencing

1. Confirm the EP-PLAT-001 command, database, migration, sqlc, OpenAPI,
   idempotency, audit-context, outbox, and verification foundations.
2. Confirm the EP-IAM-001 permission, scope, field-authorization, and audit
   ports. COA does not duplicate IAM policy evaluation.
3. Confirm the EP-OMD-001 reference boundary needed for explicit
   organization/accounting scope. COA does not access organization tables.
4. Implement User Story 1, then User Story 2, so definitions and values exist
   before combination validation.
5. Implement User Story 3 as the read-only validation slice over the
   definition/value source versions.
6. Implement User Story 4 to establish a governed request and Workflow
   decision-reference boundary.
7. Implement User Story 5 after the request state and approved decision
   contract are stable. WFA remains a separate epic; no WFA implementation is
   pulled into COA.
8. Qualify the COA-to-GL handoff only through approved identifiers,
   combinations, source versions, and application contracts. Do not start GL
   implementation by writing COA tables directly.

## 10. Risks and open decisions

- The current COA sources identify applicable controls but do not define every
  approval-policy detail for each change type. Do not invent mandatory
  approval behavior; resolve applicability through the approved Workflow
  policy/contract before implementation.
- The PRD permits changes to definitions, values, combinations, and
  assignments, while the DDD names three COA aggregate roots. The
  implementation must map assignment-related behavior to an approved COA
  aggregate/contract or record a source-backed decision before adding fields
  or a new aggregate.
- Combination restriction and uniqueness rules need an explicit source-backed
  representation before persistence constraints are designed. Do not infer
  cross-context rules from the generated OpenAPI contract.
- The generated OpenAPI COA adapters and client types remain the contract
  baseline. User Stories 1–5 now have owning-module implementations and
  capability UI; deferred runtime and external qualification remain separate
  obligations.
- M1 requires QG-01, QG-02, QG-03, QG-04, QG-05, QG-06, QG-08, and QG-10.
  Later QG-07 concurrency/financial-integrity and QG-09
  performance/capacity/recovery qualification must not be silently claimed
  from the M1 slice.
- The exact event/publication payload is not expanded in the current COA
  path contract. Use only approved integration specifications and the
  transactional outbox; do not create an unapproved event as part of the
  initial slice.

## 11. Required verification evidence

- [ ] Focused domain and application tests for all five COA stories,
  including lifecycle, effective-date, uniqueness, validation, approval
  application, conflict, and correction behavior.
- [ ] coa-owned PostgreSQL migrations, constraints, sqlc/repository tests,
  rollback/forward-fix evidence, and schema-boundary checks.
- [ ] HTTP contract tests for every approved operation, exact permission and
  scope behavior, typed problem details, correlation, idempotency, If-Match
  where declared, safe errors, and no sensitive leakage.
- [ ] Apply-decision handler and transactional outbox/integration tests for
  ordering, duplicate delivery, replay, dependency failure, owned
  exceptions, and recovery without a second business effect.
- [ ] UI component and Playwright evidence for COA-WS-01 and COA-SCR-01
  through COA-SCR-04, including keyboard, screen reader, validation,
  announcements, contrast, zoom/reflow, and safe state presentation.
- [ ] Negative scans proving secrets, raw restricted values, unrestricted
  errors, and raw policy payloads do not appear in records, responses, logs,
  traces, notifications, exports, or evidence projections.
- [ ] OpenAPI lint/bundle, generated Go/TypeScript drift, migration, sqlc,
  repository-boundary, and frontend checks pass without unapproved contract
  changes.
- [ ] Supported Go and frontend tests, vet/race checks where applicable,
  build checks, and git diff --check pass.
- [ ] Synthetic or deidentified data is used for non-production tests. No
  live Entra, production audit-chain, or full release qualification claim is
  made by local evidence.

## 12. Definition of done

The local EP-COA-001 delivery scope is complete: all five child stories have
reviewable implementation and focused local verification evidence; the coa
schema and internal/coa module preserve bounded-context ownership; approved
API and generated artifacts remain synchronized; effective dates, versions,
approval references, idempotency, concurrency, failure behavior, privacy,
authorization, audit, accessibility, and safe downstream handoff evidence are
recorded in the story sections above.

EP-COA-001 is marked locally complete and intentionally closed in the
roadmap. This closure does not claim PostgreSQL runtime execution where the
Docker environment was unavailable, downstream cross-context qualification,
production data, live Entra behavior, full audit-chain qualification,
performance/capacity qualification, or release qualification. Any remaining
unchecked evidence item is a deferred qualification or environment-dependent
follow-up, not a claim of production readiness.

## 13. Source references

- [Finance Domain Model & Use Cases — DDD Baseline](../../specs/finance_domain_model_ddd.md)
- [Finance Functional PRD](../../specs/prd/01_finance_functional_prd_v1.5.md)
- [Finance Functional Requirements Catalog](../../specs/prd/02_finance_functional_requirements_catalog_v1.5.md)
- [Finance Functional Traceability and Acceptance](../../specs/prd/03_finance_functional_traceability_acceptance_v1.5.md)
- [Finance UX Workflow Specification](../../specs/finance_ux_workflow_specification_v1.0.md)
- [Finance Nonfunctional Requirements](../../specs/finance_nonfunctional_requirements_v1.0.md)
- [Finance Delivery Plan](../../specs/finance_delivery_plan_v1.0.md)
- [Solution Architecture Overview](../../specs/system_design/01_solution_architecture_overview_v1.0.md)
- [Application Module Design](../../specs/system_design/02_application_module_design_v1.0.md)
- [Data and Integration Architecture](../../specs/system_design/03_data_integration_architecture_v1.0.md)
- [Architecture Traceability Decisions](../../specs/system_design/05_architecture_traceability_decisions_v1.0.md)
- [Backend Module Technical Specifications](../../specs/technical_specifications/01_backend_module_specifications_v1.0.md)
- [API OpenAPI Technical Specifications](../../specs/technical_specifications/02_api_openapi_specifications_v1.0.md)
- [Database Persistence Technical Specifications](../../specs/technical_specifications/03_database_persistence_specifications_v1.0.md)
- [Events, Workers, and Integration Technical Specifications](../../specs/technical_specifications/04_events_workers_integration_specifications_v1.0.md)
- [Frontend UI Technical Specifications](../../specs/technical_specifications/05_frontend_ui_technical_specifications_v1.0.md)
- [Security and Identity Authorization Technical Specifications](../../specs/technical_specifications/06_security_identity_authorization_specifications_v1.0.md)
- [Testing, Performance, and Recovery Specifications](../../specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md)
- [Existing COA OpenAPI paths](../../../contracts/openapi/paths/coa-segments.yaml)
