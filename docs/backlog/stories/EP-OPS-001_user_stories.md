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

#### Implementation status

Implemented on `feat/dlv-ops-001-us1-telemetry-context`; non-race Go tests and
vet pass through the installed Windows Go toolchain. The focused Make gate and
race checks remain open because the Linux `go` command is unavailable and the
Windows toolchain has CGO disabled.

### 6.2 Acceptance criteria

- [ ] A documented context model propagates `trace_id`, `span_id`,
  `correlation_id`, and `causation_id` across HTTP requests, command handling,
  database transactions, outbox/inbox dispatch, and external-call seams.
- [ ] Structured logs use RFC3339 UTC timestamps and stable fields for service,
  module, operation, result, error code, retryability, and data classification.
- [ ] Material actions include pseudonymous actor and accounting-scope
  identifiers only when the owning authorization contract permits them.
- [ ] Aggregate type, identifier, and version are represented as bounded
  diagnostic references; unrestricted request, event, SQL, or response bodies
  are not logged.
- [ ] Tokens, credentials, bank details, payroll values, unrestricted tax
  identifiers, secrets, and verification credentials are rejected or redacted
  by deterministic negative tests.
- [ ] Required technical spans cover HTTP request, authorization decision seam,
  idempotency lookup seam, command handler, repository operation, PostgreSQL
  transaction, outbox claim/delivery, inbox handling, provider call, report
  job, and recovery action where the corresponding component exists.
- [ ] SQL statement text and external payloads are normalized or redacted; the
  telemetry implementation does not persist raw sensitive payloads.
- [ ] The approved metric catalogue is represented with bounded labels for
  latency, command outcomes, transactions, outbox/inbox backlog, and platform
  failure classes. Domain-specific metrics remain owned by later capabilities.
- [ ] Telemetry exporter failure, timeout, or shutdown does not create a
  successful business result, duplicate authoritative operation, or unbounded
  retry loop.
- [ ] Tests prove context propagation, redaction, bounded labels, exporter
  failure behavior, and clean shutdown without a remote telemetry service.

### 6.3 Contract and impact analysis

| Area | Planned impact |
|---|---|
| Application commands/queries | No new finance command or query. Technical context is attached at existing boundaries. |
| API | Preserve existing health semantics. If a support reference is exposed, it is a sanitized correlation reference, not a payload dump. |
| Database | No telemetry tables or migrations. Existing transaction and query seams may be instrumented. |
| Events/workers | Propagate technical context through existing outbox/inbox and worker lifecycle contracts without changing event ownership or payload meaning. |
| Authorization | No policy implementation. Authorization decisions may create spans with safe outcome categories. |
| Frontend | No new finance behavior. Existing client error handling may preserve a safe support reference when the API provides one. |
| Observability | This item owns the structured telemetry, context, redaction, instrumentation, and metric contracts. |

### 6.4 Suggested implementation steps

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

### 6.5 Required test evidence

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

### 7.1 Acceptance criteria

- [ ] A dashboard contract defines panels for API health/latency, PostgreSQL
  health, outbox/inbox backlog and age, error classes, capacity, and current
  operational exceptions.
- [ ] The dashboard contract identifies source metrics, bounded dimensions,
  freshness target, owner, and safe behavior when data is missing.
- [ ] Health views distinguish availability, latency, errors, pending work,
  aging, exceptions, and capacity; they do not collapse all failures into one
  generic error rate.
- [ ] Alert definitions use P1–P4 severity, owner, response target, runbook
  link, suppression/maintenance behavior, escalation path, and completion
  evidence.
- [ ] Alert rules cover integrity uncertainty, critical control failure,
  outbox/backlog age, database saturation, dependency failure, and capacity
  risk without asserting unsupported production thresholds.
- [ ] A runbook template captures owner, prerequisites, detection, decision
  points, safe commands, evidence to preserve, escalation, recovery checks,
  reconciliation requirements, and closure criteria.
- [ ] Initial platform runbooks cover failed migration, outbox backlog/poison
  item, database restore, telemetry/monitoring failure, and capacity
  saturation. Business-specific payment, close, filing, and audit runbooks are
  assigned to their owning future delivery items.
- [ ] Runbooks explicitly prohibit direct destructive financial edits,
  unreviewed portal changes, secret disclosure, and treating telemetry as
  authoritative financial evidence.
- [ ] A local verification fixture proves dashboard/alert/runbook contracts,
  missing-data behavior, and failure propagation without Azure or production
  credentials.
- [ ] Evidence identifies the commit, commands, tool versions, result, safe
  failure location, deferred qualification scope, and no raw telemetry
  payloads, secrets, connection strings, or sensitive values.

### 7.2 Contract and impact analysis

| Area | Planned impact |
|---|---|
| Application/API | No new finance route. Existing health and support-reference contracts may be documented for dashboard use. |
| Database | No dashboard or incident schema in this M0 item. Restore and migration runbooks reference approved operational procedures. |
| Infrastructure | Local dashboard/contract verification only. Azure Monitor and production wiring remain deferred qualification. |
| Authorization | Dashboard access is described as authorized-operations scope; enforcement remains with `EP-IAM-001`. |
| Frontend | No operational console UI is required for this item; dashboard format/provider is an implementation choice within the approved contract. |
| Documentation | Owns dashboard definitions, alert catalogue, runbook template, initial runbooks, and verification evidence. |

### 7.3 Suggested implementation steps

1. Translate the approved metric catalogue and NFR freshness requirements into
   a versioned dashboard/panel contract.
2. Define alert severity, ownership, suppression, escalation, and safe
   missing-data behavior.
3. Create the reusable runbook template and the initial platform runbooks.
4. Add contract fixtures and negative cases for unowned alerts, missing runbook
   links, unsupported thresholds, and unsafe commands.
5. Record local evidence and list the external monitoring and quarterly alert
   exercises that remain deferred.

### 7.4 Required test evidence

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
