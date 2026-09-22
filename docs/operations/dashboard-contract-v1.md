# Baseline Operational Health Dashboard Contract

| Field | Value |
|---|---|
| Contract ID | `DLV-OPS-002-US1` |
| Version | `dashboard.v1` |
| Status | M0 local operational baseline |
| Owner | Platform Operations |
| Audience | Authorized operations users; authorization enforcement remains with `EP-IAM-001` |
| Data classification | Operational diagnostic data; not authoritative financial evidence |
| Freshness | Class A: no older than 2 minutes; other classes: no older than 5 minutes |
| Provider boundary | Provider-neutral contract; live Azure Monitor or another remote monitoring service is deferred |

## Purpose and contract boundary

This contract defines the minimum health views an authorized operator needs to
identify the kind of platform problem before choosing a recovery action. Each
view must preserve the distinction between availability, latency, errors,
pending work, aging, exceptions, and capacity.

The contract describes source signals and safe display behavior. It does not
create a dashboard UI, introduce an API route, persist dashboard data, define
an alert threshold, or replace an authoritative finance record, audit record,
event envelope, or recovery operation.

## Panel contract

| Panel | Purpose | Source metrics and signals | Bounded dimensions | Freshness target | Owner | Missing or stale behavior |
|---|---|---|---|---|---|---|
| **API availability and latency** | Separate process availability from request latency and HTTP outcome class. | `/health/live`; approved future `/health/ready`; `finance_http_request_duration_seconds`. | `route`, `method`, `status_class`; canonical route templates only. | M0 baseline: no older than 5 minutes; Class A: no older than 2 minutes. | Platform Operations | Display `unknown`, `stale`, or `degraded`; never treat missing samples as zero, healthy, or successful. |
| **PostgreSQL health** | Show database transaction behavior and dependency readiness separately from API availability. | `finance_db_transaction_duration_seconds`; approved future `/health/ready`; provider/database signals for connections, locks, transaction age, replication, backup, and slow queries. | `module`, `operation`, `result`; provider dimensions must remain bounded and must not expose topology or query text. | M0 baseline: no older than 5 minutes; Class A: no older than 2 minutes. | Platform Operations | Display `unknown`, `stale`, or `degraded`; do not infer database health from absent provider data. |
| **Outbox/inbox pending work and age** | Show pending work, oldest age, attempts, and poison or failed work as separate operational conditions. | `finance_outbox_pending_total`; `finance_outbox_oldest_age_seconds`; `finance_inbox_failure_total`; existing outbox/inbox backlog and managed-exception state. | `event_type`, `consumer`, `error_code`; bounded registries only. | M0 baseline: no older than 5 minutes; Class A: no older than 2 minutes. | Platform Operations | Display `unknown`, `stale`, or `degraded`; preserve the last observation only with an explicit stale marker. |
| **Error classes** | Distinguish domain rejection, authorization denial, concurrency conflict, dependency unavailability, capacity control, and internal failure. | Telemetry `result`, `error_code`, `failure_class`, and HTTP `status_class`; redacted structured logs. | `module`, `operation`, `result`, `error_code`, `failure_class`, `status_class`; no free-form error text. | M0 baseline: no older than 5 minutes; Class A: no older than 2 minutes. | Platform Operations | Display `unknown`, `stale`, or `degraded`; never collapse all classes into one generic error rate or claim success from missing data. |
| **Capacity** | Show configured and observed limits for API, worker admission, concurrency, database pools, and provider capacity without asserting unsupported production limits. | Existing worker admission and concurrency boundaries; database pool/provider capacity signals; bounded capacity-control outcomes. | `worker`, `module`, `capacity_state`, `result`; no instance IDs, customer IDs, or raw resource names. | M0 baseline: no older than 5 minutes; Class A: no older than 2 minutes. | Platform Operations | Display `unknown`, `stale`, or `degraded`; unavailable capacity signals remain unqualified and must not produce a green state. |
| **Operational exceptions** | Show current technical exceptions and their age without presenting them as financial truth. | `finance_inbox_failure_total`; outbox managed exceptions; worker lifecycle failures; typed failure classes; stale-data markers. | `event_type`, `consumer`, `error_code`, `failure_class`, `result`; bounded values only. | M0 baseline: no older than 5 minutes; Class A: no older than 2 minutes. | Platform Operations | Display `unknown`, `stale`, or `degraded`; do not hide an exception because its detail source is unavailable. |

## Outcome and state vocabulary

The dashboard keeps these categories distinct wherever the source supports
them:

- Availability: `available`, `unavailable`, `unknown`, `stale`.
- Latency: observed duration and declared freshness; latency is not inferred
  from availability alone.
- Pending work: pending count, oldest age, attempt state, and poison/managed
  exception state.
- Error classes: `domain_rejection`, `authorization_denial`,
  `concurrency_conflict`, `dependency_unavailability`, `capacity_control`,
  and `internal_failure`.
- Exceptions: active, aging, resolved, unknown, or stale according to the
  source record; unresolved source data is never silently cleared.
- Capacity: configured limit, observed use, admission control, and unavailable
  measurement where applicable.

Business-result metrics such as journal posting, payment settlement,
reconciliation, approval aging, period close, and audit-integrity outcomes are
owned by their future capability deliveries. Their absence from this baseline
does not indicate that the business capability is healthy or complete.

## Safety and missing-data rules

- Dashboard data is diagnostic and must not be used as authoritative financial,
  audit, posting, settlement, or reconciliation evidence.
- Sources, dimensions, and displays must never include secrets, credentials,
  tokens, request/event/response payloads, SQL text, complete bank details,
  payroll values, unrestricted tax identifiers, aggregate IDs, customer IDs,
  message IDs, or free-form error messages.
- Missing or stale telemetry is an operational condition. It is represented as
  `unknown`, `stale`, or `degraded`, never as zero, healthy, successful, or
  resolved.
- Provider-specific production thresholds, SLO claims, and alert severities
  are outside this contract and belong to later alert and qualification work.
- Dashboard access is described for authorized operations users only;
  enforcement, accounting-scope policy, emergency access, and audit evidence
  remain owned by their approved bounded contexts.

## Current implementation boundary

The repository currently provides `/health/live` and the platform telemetry
metrics named above. `/health/ready` is required by the approved technical
specification but is not yet implemented; this contract records it as an
approved future source rather than adding the route in this story.

Remote monitoring, live Azure export, production alerting, provider-specific
dashboard configuration, and quarterly operational exercises remain deferred.
