# EP-OPS-001 — Observability and Operational Foundation User Stories

| Field | Value |
|---|---|
| Epic | `EP-OPS-001` — Observability and operational foundation |
| Status | Planned — no implementation is claimed by this document |
| Milestone | `M0` — Engineering foundation |
| Artifact version | 1.0 |
| Delivery profile | Solo, part-time, local-first learning project |
| Dependencies | `EP-PLAT-001` stable platform, API, database, event, and worker boundaries |
| Authoritative deliverables | `DLV-OPS-001` and `DLV-OPS-002` |
| Exit evidence | Redacted structured telemetry, correlation and trace propagation, a baseline operational dashboard contract, alert ownership, and repeatable runbook evidence |

## 1. Outcome

Provide a small, safe operational foundation that lets authorized operators
understand service health, workflow progress, dependency failure, backlog age,
and integrity risk without treating operational telemetry as financial truth.

The epic establishes the telemetry and operational contracts needed by later
finance capabilities. It does not implement a finance workflow, invent a
business event, or replace audit evidence.

## 2. Learning objective

Learn how to add observability to a modular monolith without leaking sensitive
values, coupling bounded contexts, creating unbounded metric cardinality, or
allowing telemetry failure to create a misleading financial result.

## 3. Scope

- Define one correlation model for HTTP requests, commands, database
  transactions, outbox/inbox work, and external calls.
- Emit structured JSON logs with stable low-cardinality fields and explicit
  classification/redaction rules.
- Establish OpenTelemetry trace and metric instrumentation at technical
  boundaries already present in the repository.
- Define the baseline operational views, metric catalogue, alert taxonomy,
  ownership, suppression, escalation, and response targets.
- Create a runbook template and initial platform runbooks for the foundation
  failure modes.
- Produce safe, reproducible verification evidence with synthetic data only.

## 4. Explicit exclusions

- Finance-domain aggregates, commands, events, posting rules, reconciliation
  rules, or authoritative financial records.
- New application API capabilities beyond the minimum support-reference or
  health behavior required by the approved technical specification.
- Database schema changes for telemetry, dashboards, incidents, or audit
  evidence.
- Authentication, authorization, accounting-scope policy, or emergency-access
  implementation; telemetry must preserve the seams owned by `EP-IAM-001`.
- Replacing the transactional outbox, inbox, audit evidence, or domain event
  history with logs or traces.
- Live Azure Monitor, production alerting, production SLO qualification, or
  external on-call integration.
- Full business-capability dashboard coverage before those capabilities exist.
  Later capability items must add their owned metrics and runbooks against the
  contracts established here.

## 5. Architecture and ownership rules

1. Technical telemetry primitives belong under `internal/platform` or an
   equivalently narrow technical package; finance rules remain in their owning
   bounded context.
2. `cmd/api` and `cmd/worker` remain composition roots. They may configure
   exporters and lifecycle behavior but must not own shared telemetry policy.
3. Logs, traces, and metrics carry identifiers and classifications, not
   unrestricted payloads.
4. Metric labels use bounded cardinality. Aggregate IDs, customer IDs, free-form
   error messages, request bodies, SQL text, and secrets are not labels or log
   fields.
5. Operational telemetry is diagnostic evidence, not the authoritative record
   of a financial fact or audit result.
6. Local verification must not require Azure credentials, production data, or a
   remote monitoring service.

## 6. Child delivery item: DLV-OPS-001 — Structured telemetry foundation

**As the TALLY developer, I want consistent, redacted logs, traces, metrics,
and correlation across the API and worker, so that later workflows can be
diagnosed without exposing sensitive data or changing business outcomes.**

### 6.1 User Story 1 — Define telemetry context and propagation

**As the TALLY developer, I want one technical context model across HTTP,
commands, transactions, messages, and workers, so that later telemetry can
connect work without carrying sensitive business payloads.**

#### Outcome and scope

- `internal/platform/telemetry` owns identifier-only runtime context values.
- HTTP middleware accepts or generates `X-Correlation-Id`, preserves valid W3C
  trace context, and returns a sanitized correlation support reference.
- Command, database, outbox/inbox, replay, and worker seams bridge existing
  correlation and causation identifiers into `context.Context`.
- The existing event envelope and outbox schema remain unchanged.

#### Explicit exclusions

- Structured logs, span creation/export, metrics, exporters, dashboards, and
  telemetry persistence remain later `DLV-OPS-001` stories.
- No finance commands, authorization policy, audit evidence, API route, or
  database migration is introduced.

#### Acceptance criteria

- [x] `TelemetryContext` validates optional trace/span identifiers and required
  correlation identifiers without storing payloads or mutable global state.
- [x] Missing HTTP correlation IDs are generated; malformed supplied IDs are
  rejected; valid W3C trace context can be extracted and injected.
- [x] Command causation and existing event correlation/causation IDs reach
  transaction, inbox, outbox, replay, and worker callbacks.
- [x] Retries and replay preserve business correlation/causation identity.
- [ ] Focused unit, propagation, negative, race, vet, and diff checks pass.
- [ ] A documented context model propagates `trace_id`, `span_id`,
  `correlation_id`, and `causation_id` across HTTP requests, command handling,
  database transactions, outbox/inbox dispatch, and external-call seams.

#### Implementation status

Implemented on `feat/dlv-ops-001-us1-telemetry-context`; non-race Go tests and
vet pass through the installed Windows Go toolchain. The focused Make gate and
race checks remain open because the Linux `go` command is unavailable and the
Windows toolchain has CGO disabled.

### 6.2 User Story 2 — Emit redacted structured logs

**As the TALLY developer, I want structured logs with stable, redacted fields,
so that operators can diagnose technical failures without exposing sensitive
business data.**

#### Acceptance criteria

- [x] Structured logs use RFC3339 UTC timestamps and stable fields for service,
  module, operation, result, error code, retryability, and data classification.
- [x] Material actions include pseudonymous actor and accounting-scope
  identifiers only when the owning authorization contract permits them.
- [x] Aggregate type, identifier, and version are represented as bounded
  diagnostic references; unrestricted request, event, SQL, or response bodies
  are not logged.
- [x] Tokens, credentials, bank details, payroll values, unrestricted tax
  identifiers, secrets, and verification credentials are rejected or redacted
  by deterministic negative tests.

#### Implementation status

Implemented on `feat/dlv-ops-001-us2-redacted-structured-logs-main`. The
technical logger uses Go `slog` with a strict JSON field allow-list, typed
UUID/aggregate references, existing telemetry-context propagation, and safe
API/worker lifecycle logging. Sensitive and unknown attributes are rejected;
raw errors and request/event/SQL/response payloads are not emitted.

Focused non-race tests and vet pass through the installed Windows Go toolchain.
The race gate remains open because the available toolchain has CGO disabled;
the repository-native Make gate remains open because Linux `go` is unavailable.
Strict review status: `APPROVE` at `91/100`, with no blocking findings. The
remaining review limitations are the unavailable race gate and the inherited
repository-native Make environment limitation.

### 6.3 User Story 3 — Instrument traces and bounded platform metrics

**As the TALLY developer, I want traces and bounded platform metrics at existing
technical seams, so that service health and workflow progress can be measured
without creating unbounded cardinality or changing domain ownership.**

#### Acceptance criteria

- [x] Required technical spans cover HTTP request, authorization decision seam,
  idempotency lookup seam, command handler, repository operation, PostgreSQL
  transaction, outbox claim/delivery, inbox handling, provider call, report
  job, and recovery action where the corresponding component exists.
- [x] SQL statement text and external payloads are normalized or redacted; the
  telemetry implementation does not persist raw sensitive payloads.
- [x] The approved metric catalogue is represented with bounded labels for
  latency, command outcomes, transactions, outbox/inbox backlog, and platform
  failure classes. Domain-specific metrics remain owned by later capabilities.

#### Implementation status

Implemented on `feat/dlv-ops-001-us3-traces-bounded-platform-metrics`. The
implementation evidence is recorded in
[`docs/verification/DLV-OPS-001-us3-traces-bounded-platform-metrics.md`](../../verification/DLV-OPS-001-us3-traces-bounded-platform-metrics.md).
Authorization, finance command-handler, provider, and report-job seams remain
explicitly deferred because those owning components do not yet exist.

Focused and full non-race tests, vet, and sqlc compile/diff checks pass through
the available Windows Go toolchain. The race gate remains open because the
available toolchain has CGO disabled; the repository-native Make gate remains
environment-limited because Linux `go` is unavailable.

#### 6.3.1 Delivery plan (Plan mode — 2026-09-21; implemented)

**Plan status:** Implemented. The acceptance criteria above are complete for
the corresponding seams that currently exist. This does not complete User
Story 4 or the parent delivery item.

**Repository baseline at planning time:** `main` at `e26d6f9` contained the
telemetry context and redacted structured logger from User Stories 1 and 2.
The implementation below is now the scope authority for the selected seams;
the approved specifications remain the behavior authority.

**Outcome and learning objective:** Add reusable technical tracing and metrics
at boundaries already implemented by the platform, so operators can measure
request latency, command outcomes, transactions, integration backlog, and
typed platform failures without exposing payloads, creating unbounded metric
cardinality, changing financial outcomes, or moving ownership out of the
modular monolith.

**Owning component and boundaries:**

- `internal/platform/telemetry` owns tracer/meter setup, safe attributes and
  labels, span lifecycle helpers, and provider lifecycle hooks.
- Composition roots in `cmd/api` and `cmd/worker` provide service identity and
  lifecycle wiring; they do not own finance metrics or domain rules.
- Existing platform seams in `internal/platform/database`,
  `internal/platform/idempotency`, `internal/platform/integration`, and
  `internal/platform/worker` are instrumented through their existing ports.
- No domain aggregate, posted financial fact, audit record, event envelope,
  authorization policy, public API schema, or database schema is owned by this
  story. No telemetry table or migration is planned.

**Trace and metric contract for implementation:**

| Acceptance area | Planned treatment | Current-component boundary |
|---|---|---|
| Required spans | Create child spans with stable names and safe outcome attributes; carry the existing telemetry context and close spans on every success, error, cancellation, and panic path. | HTTP middleware; durable idempotency acquire/finalize; transaction begin/commit/rollback; integration publish/consume/reconcile/replay; outbox claim, delivery, lease, and establishment; worker host lifecycle. |
| Conditional spans | Add narrow instrumentation hooks only when the owning component is delivered; do not create placeholder finance or IAM code. | Authorization decision, command handler, provider call, and report job are not implemented in the current repository and remain explicit deferrals. Recovery instrumentation is limited to existing integration failure-recording/reconciliation paths. |
| Safe trace data | Permit stable module, operation, result, error-code, retryability, classification, route template, method, and status class attributes only. Normalize SQL to operation/database metadata; never attach SQL text, request/event/response bodies, credentials, tokens, bank/payroll/tax values, aggregate IDs, customer IDs, or external payloads. | Reuse the User Story 2 allow-list and add trace-specific validation tests rather than serializing arbitrary `slog` or error values. |
| `finance_http_request_duration_seconds` | Record a histogram using the canonical route template, HTTP method, and status class. Use a bounded fallback for unmatched routes. | Existing request middleware in `internal/platform/telemetry` and `cmd/api`. |
| `finance_command_total` | Record a counter with bounded module, operation, and result values. Provide the technical helper for future command handlers; instrument only a command seam that exists. | No command-handler seam exists in the current repository; finance command handlers are deferred. |
| `finance_db_transaction_duration_seconds` | Record a histogram with bounded module, operation, and result values around existing PostgreSQL transaction boundaries. | `internal/platform/database` and `internal/platform/integration`; generated query code remains generated. |
| `finance_outbox_pending_total` and `finance_outbox_oldest_age_seconds` | Publish gauges from read-only platform queries, grouped only by canonical bounded event type. Unknown or unsupported label values collapse to a safe bounded category. | `db/queries/platform/integration_outbox.sql` and its generated `platformdb` code, without changing the outbox schema or event facts. |
| `finance_inbox_failure_total` | Record a counter with bounded consumer and normalized error-code labels at the existing failure-recording seam. | `internal/platform/integration/coordination.go` and `db/queries/platform/integration_inbox.sql`; raw message IDs and failure text are excluded. |
| Domain metrics | Do not implement journal, receipt, payment, settlement, approval, close, or audit-domain metrics in this story. | Later owning capability epics add their own metrics against this contract. |

**Impact and non-functional behavior:** HTTP response and health semantics stay
unchanged. No frontend behavior or OpenAPI change is required. Existing event
correlation/causation and outbox/inbox ownership stay unchanged. Telemetry
recording is diagnostic and must not be allowed to turn a committed business
effect into a failure, repeat an idempotent operation, extend a transaction
indefinitely, or create an unbounded retry loop. Metric label normalization is
performed before recording, and raw identifiers are never used as labels.
Provider/exporter configuration must be injectable and locally testable; this
story does not add live Azure credentials, remote monitoring, dashboards, or
production retention policy. Exporter failure and complete shutdown/failure
injection qualification remain User Story 4 evidence.

**Ordered implementation steps:**

1. Reconfirm the OpenTelemetry API/provider contract against the pinned Go
   dependencies, define stable span names and bounded label registries, and
   decide the local no-remote provider/test-recorder wiring without adding
   Azure credentials.
2. Extend `internal/platform/telemetry` with the smallest reusable tracer and
   meter helpers. Reuse the existing context and redaction rules; reject or
   collapse invalid, unknown, free-form, or high-cardinality labels.
3. Add HTTP child-span and request-duration instrumentation while preserving
   correlation middleware, safe request logging, route behavior, and response
   writer compatibility.
4. Instrument existing idempotency, transaction, repository/query, and
   integration seams. Add only read-only generated queries needed for pending
   and oldest outbox/inbox measurements; regenerate and check sqlc output
   rather than editing generated files manually.
5. Instrument dispatcher, replay, failure-recording, and worker lifecycle
   outcomes with typed result/failure classes. Ensure cancellation, lease loss,
   retry, duplicate delivery, and recovery paths close spans and update only
   bounded counters/gauges.
6. Wire provider creation and bounded shutdown hooks in `cmd/api` and
   `cmd/worker`, keeping telemetry setup separate from business transaction
   success and preserving existing lifecycle logs.
7. Add focused unit/integration/negative verification, a repository-native
   verification command, and a dedicated evidence document recording covered
   seams, metric/span inventory, unimplemented conditional seams, command
   results, and any toolchain limitations.

**Likely files and packages:**

- `internal/platform/telemetry` — new tracing/metrics helpers and tests beside
  `context.go`, `http.go`, and `logging.go`.
- `cmd/api/main.go`, `cmd/worker/main.go` — provider and lifecycle wiring.
- `internal/platform/database`, `internal/platform/idempotency`,
  `internal/platform/integration`, and `internal/platform/worker` — existing
  technical seam instrumentation and tests.
- `db/queries/platform/integration_outbox.sql` and
  `db/queries/platform/integration_inbox.sql`, with generated
  `internal/platform/database/platformdb/*` output only when read-only metric
  queries are required.
- `Makefile`, `scripts/verify/`, and `scripts/README.md` — focused verification
  command and documentation.
- `docs/verification/DLV-OPS-001-us3-traces-bounded-platform-metrics.md` —
  implementation evidence and verification limitations.

**Required test evidence:**

- Unit tests for span names, attribute allow-listing, SQL normalization,
  payload exclusion, label normalization, bounded-cardinality behavior, and
  status/route classification.
- HTTP tests proving trace context and correlation continuity, request timing,
  status-class recording, and unchanged health/error responses.
- Integration tests with synthetic PostgreSQL/outbox/inbox data proving
  transaction and worker/integration context continuity, backlog gauges, and
  typed failure counters without exposing payloads or identifiers.
- Negative tests scanning captured spans/metrics/logs for request bodies,
  event bodies, SQL statements, credentials, secrets, unrestricted errors, and
  aggregate/customer/message identifiers.
- Focused Go test/vet checks, relevant persistence/sqlc drift checks,
  `go test ./...`, `git diff --check`, and the new verification script. Run the
  race gate when the available toolchain supports it and record a limitation if
  CGO or the repository-native Go command is unavailable.

**Definition of done for this story:**

- The three acceptance criteria are evidenced for every currently implemented
  corresponding seam, with absent authorization/provider/report/finance seams
  explicitly listed as deferred rather than simulated.
- The approved platform metric subset is present with bounded labels, and no
  domain-specific metric ownership is introduced early.
- Spans and metrics contain no raw sensitive payloads or unrestricted SQL, no
  telemetry path changes authoritative business results, and no schema/API/
  event ownership boundary is bypassed.
- Focused and repository checks pass, the evidence document is complete, and a
  strict branch-diff review has no blocking findings.

**Risks and open decisions:**

- The current `go.mod` directly requires OpenTelemetry API packages but does
  not yet declare a tracing/metrics SDK or exporter. Implementation must choose
  the smallest pinned provider/test-recorder arrangement consistent with the
  approved standard; it must not silently introduce live Azure export.
- The approved catalogue names `event_type` and `consumer` labels, while the
  current envelope accepts general identifier text. A finite registry or
  `unknown`/`other` collapse is required before recording; raw event or
  consumer text must not become an unbounded label.
- Database backlog gauges can add read-only query code but must not add
  telemetry persistence or mutate immutable outbox event facts.
- `EP-IAM-001` owns authorization and later capability epics own command,
  provider, reporting, and business metrics. Instrumentation must remain
  adapter-level and wait for those seams.

#### 6.3.2 Traceability for the delivery plan

| Identifier | Relationship to User Story 3 |
|---|---|
| `M0` | Engineering foundation milestone. |
| `EP-OPS-001` | Parent observability and operational foundation epic. |
| `DLV-OPS-001` | Structured telemetry foundation delivery item. |
| `GFR-012` | Technical outcomes distinguish intermediate, exception, reconciliation, and terminal integration states. |
| `ARC-OBS-001` | Trace, correlation, and causation identifiers cross technical boundaries. |
| `ARC-OBS-002` | Platform and later business health metrics use separate ownership. |
| `ARC-PRV-001` / `NFR-SEC-010` | Telemetry data minimization and secret/credential exclusion. |
| `NFR-OBS-003`, `NFR-OBS-004`, `NFR-OBS-008` | Safe support diagnostics, no sensitive telemetry, and typed operational outcomes. |
| `NFR-MNT-001`, `NFR-TST-003` | Source-to-verification traceability and evidence for each applicable requirement. |
| `QG-01`, `QG-08`, `QG-10` | Traceability, observability readiness, and release evidence gates. |

This record preserves the story's bounded scope and explicitly separates
implemented platform seams from deferred authorization, provider, reporting,
and finance capability seams.

### 6.4 User Story 4 — Prove telemetry failure and sensitive-data boundaries

**As the TALLY maintainer, I want telemetry failure and sensitive-data tests,
so that diagnostics cannot create a false business result, duplicate an
operation, or leak protected values.**

#### Acceptance criteria

- [x] Telemetry exporter failure, timeout, or shutdown does not create a
  successful business result, duplicate authoritative operation, or unbounded
  retry loop.
- [x] Tests prove context propagation, redaction, bounded labels, exporter
  failure behavior, and clean shutdown without a remote telemetry service.

#### Implementation status

Implemented on `codex/dlv-ops-001-us4-telemetry-failure-sensitive-data` with
local synthetic trace and metric exporter failure injection. Provider shutdown
is at-most-once and deadline-bounded; telemetry errors remain diagnostic and
cannot replace an application result. Evidence is recorded in
[`docs/verification/DLV-OPS-001-us4-telemetry-failure-sensitive-data.md`](../../verification/DLV-OPS-001-us4-telemetry-failure-sensitive-data.md).

Live Azure export, remote monitoring, production alerting, and retention
qualification remain deferred. This story does not mark `EP-OPS-001`,
`DLV-OPS-001`, `M0`, or `QG-08` complete as a whole.

### 6.5 Contract and impact analysis

| Area | Planned impact |
|---|---|
| Application commands/queries | No new finance command or query. Technical context is attached at existing boundaries. |
| API | Preserve existing health semantics. If a support reference is exposed, it is a sanitized correlation reference, not a payload dump. |
| Database | No telemetry tables or migrations. Existing transaction and query seams may be instrumented. |
| Events/workers | Propagate technical context through existing outbox/inbox and worker lifecycle contracts without changing event ownership or payload meaning. |
| Authorization | No policy implementation. Authorization decisions may create spans with safe outcome categories. |
| Frontend | No new finance behavior. Existing client error handling may preserve a safe support reference when the API provides one. |
| Observability | This item owns the structured telemetry, context, redaction, instrumentation, and metric contracts. |

### 6.6 Suggested implementation steps

1. Confirm the trace/correlation field contract and data-classification rules
   against the technical observability specification.
2. Add the smallest reusable platform telemetry package and configure it from
   the API and worker composition roots.
3. Instrument existing HTTP, database, idempotency, integration, and worker
   seams without adding domain ownership or new persistence.
4. Add redaction, bounded-cardinality, propagation, exporter-failure, and
   shutdown tests using synthetic values.
5. Record the metric/span inventory and verification results in a dedicated
   evidence document.

### 6.7 Required test evidence

- Unit tests for field normalization, classification, redaction, and context
  propagation.
- Integration tests for API-to-database and outbox/worker context continuity.
- Negative tests proving secrets, full payloads, unrestricted SQL, and
  unbounded identifiers cannot enter telemetry.
- Failure tests proving exporter unavailability is bounded and cannot change
  an authoritative operation's result.
- `go test ./...`, focused telemetry checks, and `git diff --check`.

## 7. Child delivery item: DLV-OPS-002 — Baseline dashboard and runbook foundation

**As an authorized operator, I want baseline health views, actionable alerts,
and repeatable runbooks, so that I can identify and recover from platform
problems without guessing or requesting sensitive data.**

### 7.1 User Story 1 — Define baseline operational health views

**As an authorized operator, I want separate health views for service
availability, pending work, errors, exceptions, and capacity, so that I can
identify the kind of operational problem before choosing a recovery action.**

#### Acceptance criteria

- [x] A dashboard contract defines panels for API health/latency, PostgreSQL
  health, outbox/inbox backlog and age, error classes, capacity, and current
  operational exceptions.
- [x] The dashboard contract identifies source metrics, bounded dimensions,
  freshness target, owner, and safe behavior when data is missing.
- [x] Health views distinguish availability, latency, errors, pending work,
  aging, exceptions, and capacity; they do not collapse all failures into one
  generic error rate.

#### Implementation status

Implemented on `codex/dlv-ops-002-us1-baseline-operational-health-views`.
The provider-neutral contract is recorded in
[`docs/operations/dashboard-contract-v1.md`](../../operations/dashboard-contract-v1.md)
and the local verification evidence is recorded in
[`docs/verification/DLV-OPS-002-us1-baseline-operational-health-views.md`](../../verification/DLV-OPS-002-us1-baseline-operational-health-views.md).
The focused dashboard-contract gate passed with synthetic negative cases for
missing ownership, unbounded dimensions, collapsed error classes, and unsafe
missing-data behavior. This does not complete `DLV-OPS-002`, `EP-OPS-001`,
`QG-08`, or live production monitoring qualification.

### 7.2 User Story 2 — Define alert severity, ownership, and escalation

**As an authorized operator, I want alerts to identify severity, ownership,
response targets, and escalation paths, so that operational failures receive a
clear and safe response.**

#### Acceptance criteria

- [x] Alert definitions use P1–P4 severity, owner, response target, runbook
  link, suppression/maintenance behavior, escalation path, and completion
  evidence.
- [x] Alert rules cover integrity uncertainty, critical control failure,
  outbox/backlog age, database saturation, dependency failure, and capacity
  risk without asserting unsupported production thresholds.

#### Implementation status

Implemented on `codex/dlv-ops-002-us2-alert-severity-ownership-escalation`.
The provider-neutral `alert.v1` contract is recorded in
[`docs/operations/alert-contract-v1.md`](../../operations/alert-contract-v1.md)
and the local verification evidence is recorded in
[`docs/verification/DLV-OPS-002-us2-alert-severity-ownership-escalation.md`](../../verification/DLV-OPS-002-us2-alert-severity-ownership-escalation.md).
The focused alert-contract gate validates all six alert categories, complete
ownership/response/runbook/suppression/escalation/completion metadata, and
negative cases for missing or unsafe metadata and unsupported thresholds.
This does not complete `DLV-OPS-002`, `EP-OPS-001`, `QG-08`, User Story 3
runbook content, or live production monitoring qualification.

### 7.3 User Story 3 — Establish the runbook template and initial platform runbooks

**As an authorized operator, I want safe, repeatable runbooks for foundation
failures, so that recovery preserves evidence and does not create an unsafe
financial or infrastructure change.**

#### Acceptance criteria

- [x] A runbook template captures owner, prerequisites, detection, decision
  points, safe commands, evidence to preserve, escalation, recovery checks,
  reconciliation requirements, and closure criteria.
- [x] Initial platform runbooks cover failed migration, outbox backlog/poison
  item, database restore, telemetry/monitoring failure, and capacity
  saturation. Business-specific payment, close, filing, and audit runbooks are
  assigned to their owning future delivery items.
- [x] Runbooks explicitly prohibit direct destructive financial edits,
  unreviewed portal changes, secret disclosure, and treating telemetry as
  authoritative financial evidence.

#### 7.3.1 Delivery plan

**Plan status:** Implementation in progress on
`codex/dlv-ops-002-us3-runbook-foundation`.

**Outcome and learning objective:** Establish a provider-neutral, repeatable
operational response pattern that preserves evidence and financial integrity
while teaching safe migration, integration, recovery, telemetry, and capacity
operations in the local-first learning environment.

**Owning area and boundaries:** Platform Operations owns the runbook template,
initial platform procedures, alert links, contract verification, and evidence
record. No finance bounded context, application API, database schema, event,
worker behavior, authorization policy, or infrastructure resource is changed.
Authorization enforcement remains owned by `EP-IAM-001`; authoritative finance,
audit, outbox/inbox, and recovery records remain owned by their existing
components.

**Initial runbook set:**

- `RUN-001` — Failed migration.
- `RUN-002` — Outbox backlog or poison item.
- `RUN-003` — Database restore.
- Named platform scenario — Telemetry and monitoring failure. This remains
  unnumbered because the approved catalog defines only `RUN-001` through
  `RUN-010`.
- `RUN-010` — Capacity saturation.

**Safety and operational treatment:** Runbooks use repository-supported
inspection and verification commands, gate state-changing actions behind
authorization and approval, preserve bounded diagnostic evidence, and direct
unsupported or missing operator capabilities to escalation. Shared migration
recovery uses reviewed forward-fix or isolated restore; outbox recovery keeps
event facts immutable and checks established results before retry/replay;
restore procedures require isolated validation and reconciliation; telemetry is
never financial truth; and capacity response protects writes without silently
dropping work.

**Verification plan:** `scripts/verify/runbook-contract.sh` validates the
template, five runbooks, mandatory sections, delivered alert links, safe
commands, approved identifiers, and unsafe-instruction negative cases. The
repository-native `runbook-contract-check` target is supplemented by the
existing dashboard, alert, telemetry, worker, migration, and diff checks.

**Traceability:** `M0`, `EP-OPS-001`, `DLV-OPS-002`, `GFR-012`,
`ARC-OBS-001`, `ARC-OBS-002`, `ARC-PRV-001`, `ARC-CAP-001`, `ARC-REC-001`,
`ARC-TST-001`, `NFR-OBS-001`–`NFR-OBS-012`, `NFR-MNT-001`, `NFR-MNT-007`,
`NFR-MNT-010`, applicable `NFR-REC-*`, `NFR-TST-003`, `NFR-TST-008`,
`NFR-TST-009`, `QG-01`, `QG-08`, and `QG-10`.

#### 7.3.2 Implementation status

Implemented on `codex/dlv-ops-002-us3-runbook-foundation`. The runbook
template, five initial platform runbooks, delivered alert links, contract
verifier, Make target, scripts documentation, and verification evidence are
present in the working tree. `make runbook-contract-check`,
`make alert-contract-check`, `make dashboard-contract-check`,
`make repository-integrity-check`, `make db-migrate-check`, and
`git diff --check` passed. Existing Go-dependent telemetry, worker, and
migration-validation gates remain environment-blocked because Linux `go` is
unavailable and the installed Windows Go binary cannot be invoked in this WSL
session; this limitation is recorded in the verification evidence.

This completes User Story 3 documentation and local contract evidence. It does
not complete User Story 4, `DLV-OPS-002`, `EP-OPS-001`, `QG-08`, or production
operational qualification.

### 7.4 User Story 4 — Produce operational readiness evidence

**As the TALLY maintainer, I want reproducible verification evidence for
dashboard, alert, and runbook contracts, so that operational readiness can be
reviewed without Azure or production credentials.**

#### Acceptance criteria

- [ ] A local verification fixture proves dashboard/alert/runbook contracts,
  missing-data behavior, and failure propagation without Azure or production
  credentials.
- [ ] Evidence identifies the commit, commands, tool versions, result, safe
  failure location, deferred qualification scope, and no raw telemetry
  payloads, secrets, connection strings, or sensitive values.

### 7.5 Contract and impact analysis

| Area | Planned impact |
|---|---|
| Application/API | No new finance route. Existing health and support-reference contracts may be documented for dashboard use. |
| Database | No dashboard or incident schema in this M0 item. Restore and migration runbooks reference approved operational procedures. |
| Infrastructure | Local dashboard/contract verification only. Azure Monitor and production wiring remain deferred qualification. |
| Authorization | Dashboard access is described as authorized-operations scope; enforcement remains with `EP-IAM-001`. |
| Frontend | No operational console UI is required for this item; dashboard format/provider is an implementation choice within the approved contract. |
| Documentation | Owns dashboard definitions, alert catalogue, runbook template, initial runbooks, and verification evidence. |

### 7.6 Suggested implementation steps

1. Translate the approved metric catalogue and NFR freshness requirements into
   a versioned dashboard/panel contract.
2. Define alert severity, ownership, suppression, escalation, and safe
   missing-data behavior.
3. Create the reusable runbook template and the initial platform runbooks.
4. Add contract fixtures and negative cases for unowned alerts, missing runbook
   links, unsupported thresholds, and unsafe commands.
5. Record local evidence and list the external monitoring and quarterly alert
   exercises that remain deferred.

### 7.7 Required test evidence

- Dashboard and alert contract validation.
- Negative fixtures for missing owner, severity, runbook, suppression, or
  escalation metadata.
- Runbook review checklist and safe-command/secret scan.
- Synthetic missing-data and stale-data behavior tests.
- Verification record with `QG-08` evidence and explicit production
  qualification deferral.

## 8. Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Observability and operational foundation is an M0 epic. |
| `EP-OPS-001` | Parent epic for both delivery items. |
| `DLV-OPS-001` | Structured logging, correlation, and OpenTelemetry. |
| `DLV-OPS-002` | Baseline operational dashboard and runbook template. |
| `GFR-012` | Cross-context outcomes expose intermediate, exception, reconciliation, and terminal states. |
| `NFR-OBS-001`–`NFR-OBS-008` | Health views, detection, support references, safe telemetry, SLO reporting, age visibility, alert ownership, and typed failure classes. |
| `NFR-SEC-010` | Secrets and provider credentials never appear in telemetry or diagnostics. |
| `NFR-MNT-001` | Traceability from behavior to source and verification evidence. |
| `NFR-MNT-007` | Runbooks for Class A workflows, critical alerts, recovery, security, and audit-integrity incidents. |
| `NFR-MNT-010` | No unresolved critical operational, security, recovery, or audit defect at release. |
| `NFR-TST-003` | Every NFR has verification method, owner, result, and evidence or approved deferral. |
| `NFR-TST-009` | Release evidence includes runbooks, support readiness, service-objective readiness, and exceptions. |
| `QG-01` | Traceability and source integrity. |
| `QG-08` | Observability and operational readiness. |
| `QG-10` | Release evidence and demonstration. |

## 9. Dependencies and handoffs

- `DLV-PLAT-001` supplies the API and worker composition roots and root command
  conventions.
- `DLV-PLAT-002` and `DLV-PLAT-003` supply the PostgreSQL, migration, and query
  boundaries to instrument without owning their schemas.
- `DLV-PLAT-006` and `DLV-PLAT-007` supply idempotency, outbox/inbox, and worker
  seams for context propagation and backlog metrics.
- `DLV-CI-001` supplies pull-request verification; the observability checks
  must be added through repository commands rather than duplicated YAML logic.
- `EP-IAM-001` later owns dashboard authorization, accounting-scope access,
  emergency access, and privileged operational actions.
- Finance capability epics later own business metrics, workflow alerts,
  reconciliation indicators, and capability-specific runbooks.

## 10. Epic definition of done

- [ ] Both child delivery items pass their acceptance criteria and review.
- [ ] Structured telemetry is redacted, correlated, bounded, and tested across
  the existing API, database, integration, and worker seams.
- [ ] Baseline dashboard, alert, and runbook contracts are versioned and have
  local reproducible evidence.
- [ ] Every alert in the M0 catalogue has an owner, response target, runbook,
  suppression rule, escalation path, and closure evidence.
- [ ] No secret, raw sensitive payload, unrestricted SQL, or production data is
  present in telemetry, dashboards, runbooks, or evidence.
- [ ] The repository remains a modular monolith; no finance API, database
  schema, event ownership, or authorization boundary is bypassed.
- [ ] Live Azure monitoring, production alert exercises, full Class A business
  runbooks, and release qualification are explicitly recorded as deferred
  scope where their owning capabilities do not yet exist.
- [ ] Required tests, traceability, documentation, and `QG-08` evidence pass.

## 11. Required evidence

- Telemetry field, propagation, metric, and span inventory.
- Redaction and sensitive-data negative-test output.
- Dashboard, alert, and runbook contract validation output.
- Safe runbook review with owner, escalation, evidence, and closure checks.
- Commit, tool versions, command results, and deferred qualification record.

## 12. Source references

- `docs/specs/finance_delivery_plan_v1.0.md`
- `docs/specs/finance_nonfunctional_requirements_v1.0.md`
- `docs/specs/technical_specifications/08_observability_operations_specifications_v1.0.md`
- `docs/specs/system_design/04_security_deployment_operations_v1.0.md`
- `docs/specs/system_design/05_architecture_traceability_decisions_v1.0.md`
- `docs/backlog/epic-template.md`
- `docs/backlog/story-template.md`

## 13. Planning status

This is a planning artifact. No branch is created and no roadmap item is
marked complete by adding this document.

Suggested implementation branch: `codex/ep-ops-001-observability-foundation`
