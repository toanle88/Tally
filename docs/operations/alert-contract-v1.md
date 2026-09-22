# Baseline Operational Alert Contract

| Field | Value |
|---|---|
| Contract ID | `DLV-OPS-002-US2` |
| Version | `alert.v1` |
| Status | M0 local operational baseline |
| Owner | Platform Operations |
| Audience | Authorized operations users; authorization enforcement remains with `EP-IAM-001` |
| Data classification | Operational diagnostic data; not authoritative financial or audit evidence |
| Provider boundary | Provider-neutral contract; live Azure Monitor, paging, and production alert wiring are deferred |
| Runbook source | Approved `RUN-*` catalog in the observability technical specification; runbook content is User Story 3 scope |

## Purpose and contract boundary

This contract defines the minimum metadata and safe response behavior for the
M0 operational alert catalogue. It gives an authorized operator a severity,
accountable owner, response target, dashboard context, approved runbook
reference, suppression policy, escalation path, and completion evidence for
each alert condition.

The contract does not create an alerting provider, dashboard UI, API route,
incident database, finance-domain metric, production threshold, notification
integration, or authoritative financial or audit record. Alert conditions are
evaluated by a future provider against typed signals and approved workflow,
business-calendar, NFR, or capacity policies.

## Severity and response policy

| Severity | Operational meaning | Response target | Baseline escalation |
|---|---|---|---|
| **P1** | Possible integrity loss, critical control failure, or broad Class A outage. | Detect within the NFR-OBS-002 target of 5 minutes, page immediately, and engage an accountable responder within 10 minutes. | Incident Commander plus the owning integrity, security, or control owner; block affected writes when correctness is uncertain. |
| **P2** | Backlog, saturation, or recovery risk that threatens an NFR or controlled workflow. | Urgent response and controlled degradation using the configured workflow or NFR target. | Platform Operations lead, affected dependency owner, and Incident Commander when the response objective is at risk. |
| **P3** | Isolated dependency failure, report delay, or elevated conflict with a safe user-visible state. | Business-hours response using the configured dependency or workflow target. | Platform Operations to the affected capability or dependency owner; escalate to the Incident Commander if impact broadens. |
| **P4** | Capacity forecast or noncritical warning requiring planned remediation. | Planned remediation within the approved capacity or operational review cycle. | Platform Operations to the engineering or capacity owner; escalate when the forecast enters an approved risk state. |

The five-minute detection and ten-minute responder targets are the approved
critical-alert targets from `NFR-OBS-002`. Other targets remain configurable by
workflow, dependency, business calendar, and approved operational policy. This
contract contains no unsupported production trigger threshold.

## Alert definition contract

Every alert definition has the following fields: severity, signal and trigger
policy, owner, response target, dashboard link, runbook link, suppression or
maintenance behavior, escalation path, and completion evidence.

| Alert condition | Severity | Signal and trigger policy | Owner | Response target | Dashboard / runbook links | Suppression / maintenance behavior | Escalation path | Completion evidence |
|---|---|---|---|---|---|---|---|---|
| **Integrity uncertainty** | P1 | Audit-integrity incident, unexplained reconciliation difference, possible duplicate financial effect, or lost posting evidence. Fire on a typed integrity-uncertain state; do not infer a trigger from a guessed numeric threshold. | Platform Operations | Immediate page; accountable responder within 10 minutes; block affected writes when correctness is uncertain. | Dashboard: [dashboard.v1](./dashboard-contract-v1.md) — Operational exceptions. Runbook: [RUN-006 — Audit mismatch](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog). | Integrity uncertainty is never suppressed. A maintenance window with owner, reason, start/end, and expiry may deduplicate notifications only; the condition remains visible as active, unknown, or stale. | Escalate to the Incident Commander and Audit Integrity/Security owner; notify the affected capability owner. | Completion evidence: preserve redacted alert context, verify authoritative evidence and reconciliation, record owner/action/result/closure time, and complete the required incident review. |
| **Critical control failure** | P1 | Migration compatibility, posting-gate, authorization-control, audit-control, or other critical-control signal enters a typed control-failed state or an approved control policy. | Platform Operations | Immediate page; accountable responder within 10 minutes; prevent unsafe affected work until control status is established. | Dashboard: [dashboard.v1](./dashboard-contract-v1.md) — Operational exceptions and PostgreSQL health. Runbooks: [RUN-001 — Failed migration](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog), [RUN-005 — Period-control interruption](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog), or [RUN-006 — Audit mismatch](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog) as applicable. | Control failure is never suppressed. A maintenance window with owner, reason, start/end, and expiry may suppress only absence-of-data notifications; the control condition remains unknown or stale when its source is unavailable. | Escalate to the Incident Commander plus the owner of the affected control, security, or audit boundary. | Completion evidence: preserve redacted control evidence, verify the control is restored, record owner/action/result/closure time, and reconcile or review affected work before closure. |
| **Outbox/backlog age** | P2 | `finance_outbox_oldest_age_seconds`, `finance_outbox_pending_total`, managed exceptions, and delivery failure states exceed the configured workflow, business-calendar, or NFR target. | Platform Operations | Urgent response and controlled degradation using the configured backlog-age target. | Dashboard: [dashboard.v1](./dashboard-contract-v1.md) — Outbox/inbox pending work and age. Runbook: [RUN-002 — Outbox backlog/poison item](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog). | A scheduled deployment or maintenance window with owner, reason, start/end, and expiry may suppress notifications only for the declared window. Pending work, stale data, and managed exceptions remain visible; suppression expires automatically. | Escalate from Platform Operations to the integration/worker owner and then the Incident Commander if the configured response objective is at risk. | Completion evidence: preserve redacted backlog and failure-class evidence, verify controlled replay or repair without duplicate delivery, record owner/action/result/closure time, and reconcile residual work. |
| **Database saturation** | P2 | Database connection or pool saturation, lock or transaction-age risk, backup/replication risk, or readiness failure reaches the configured capacity or dependency policy. No production percentage or provider-specific limit is asserted here. | Platform Operations | Urgent response and controlled degradation using the configured database capacity target. | Dashboard: [dashboard.v1](./dashboard-contract-v1.md) — PostgreSQL health and Capacity. Runbook: [RUN-010 — Capacity saturation](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog). | A scheduled deployment or maintenance window with owner, reason, start/end, and expiry may suppress absence-of-data notifications only for the declared window. Saturation and stale/unknown state remain visible; suppression expires automatically. | Escalate to the database/dependency owner and Incident Commander when recovery or data-integrity risk is increasing. | Completion evidence: preserve redacted health and capacity evidence, verify saturation clears without data loss, record owner/action/result/closure time, and complete recovery checks. |
| **Dependency failure** | P3 | A typed `dependency_unavailability` state, provider failure, authorization dependency outage, or service-health loss reaches the configured dependency or workflow policy. | Platform Operations | Business-hours response with visible service status using the configured dependency target. | Dashboard: [dashboard.v1](./dashboard-contract-v1.md) — Error classes and Operational exceptions. Runbooks: [RUN-007 — Entra/authorization outage](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog) or [RUN-004 — Payment provider outage](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog) as applicable. | A scheduled deployment or maintenance window with owner, reason, start/end, and expiry may suppress absence-of-data notifications only for the declared window. Dependency failure remains visible as unavailable, unknown, or stale; suppression expires automatically. | Escalate to the affected dependency or capability owner; involve the Incident Commander if the failure broadens or threatens a Class A workflow. | Completion evidence: preserve redacted dependency and support-reference evidence, verify the dependency outcome and any reconciliation requirement, record owner/action/result/closure time, and update service status. |
| **Capacity risk** | P4 | Worker/API/database/provider forecast, admission-control, queue-depth, or bounded-capacity state enters the configured warning or risk policy. No production limit is asserted here. | Platform Operations | Planned remediation within the approved capacity review cycle; retain the warning until accepted or resolved. | Dashboard: [dashboard.v1](./dashboard-contract-v1.md) — Capacity. Runbook: [RUN-010 — Capacity saturation](../specs/technical_specifications/08_observability_operations_specifications_v1.0.md#8-runbook-catalog). | A scheduled deployment, load exercise, or maintenance window with owner, reason, start/end, and expiry may suppress notifications only for the declared window. Capacity risk remains visible as stale/unknown when measurement is unavailable; suppression expires automatically. | Escalate to the engineering or capacity owner; involve the Incident Commander if the risk enters an approved control-failure state. | Completion evidence: preserve redacted capacity evidence, approve and record the capacity action or exception, verify no work was silently dropped, and record owner/action/result/closure time. |

## Safe alert-state rules

- Alert data is diagnostic. It is not authoritative financial, audit, posting,
  settlement, reconciliation, or recovery evidence.
- Missing or stale telemetry is an operational condition represented as
  `unknown`, `stale`, or `degraded`; it never becomes zero, healthy,
  successful, resolved, or silently cleared.
- Suppression is an explicitly bounded maintenance or deployment behavior. It
  requires an owner, reason, start/end window, and expiry. Expired suppression
  returns the alert to normal evaluation.
- P1 integrity uncertainty and critical control failures are never suppressed.
  Absence-of-data suppression may hide a notification during declared
  maintenance, but it cannot hide the underlying unknown/stale condition.
- Trigger thresholds are configured by approved workflow, business-calendar,
  dependency, NFR, or capacity policy. Changes are authorized, versioned, and
  auditable under `NFR-OBS-012`; change evidence records before/after policy,
  owner, approval, effective time, and rollback path.
- P1 and P2 completion evidence records the affected capability/workflow,
  scope, duration, financial or evidence impact, containment, recovery,
  residual exceptions, and follow-up review. High-severity incident review is
  completed within the approved NFR-OBS-009 period, and alert/runbook metadata
  is reviewed at least quarterly under `NFR-MNT-007`.
- Alert contracts and evidence contain no secrets, credentials, tokens, raw
  request/event/response payloads, unrestricted SQL, aggregate IDs, customer
  IDs, message IDs, bank details, payroll values, or unrestricted tax
  identifiers.

## Current implementation boundary and deferrals

This is a local contract and verification artifact. The repository does not
create a paging provider, live Azure Monitor rule, dashboard UI, incident
record, or production alert threshold. Existing platform telemetry is used by
name where available; future audit-integrity, finance-domain, dependency, and
provider signals remain owned by their later capability deliveries.

Runbook links reference the approved catalog only. The reusable runbook
template and initial platform runbook content are User Story 3 scope.
Production alert exercises, live monitoring delivery, retention qualification,
and quarterly operational review remain deferred qualification work.

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Observability and operational foundation milestone. |
| `EP-OPS-001` / `DLV-OPS-002` | Parent epic and baseline dashboard/runbook delivery item. |
| `GFR-012` | Preserves intermediate, exception, reconciliation, and terminal operational outcomes. |
| `ARC-OBS-002` | Operational telemetry remains separate from authoritative business metrics. |
| `ARC-PRV-001` | Diagnostic data is minimized and sensitive values are excluded. |
| `ARC-CAP-001` | Capacity signals use measured, bounded, controlled states. |
| `ARC-TST-001` | Contract, negative, failure, and evidence checks are layered verification. |
| `NFR-OBS-002` | Critical failures are detected and assigned to an accountable responder. |
| `NFR-OBS-006` | Pending work and age targets are configurable by workflow and business calendar. |
| `NFR-OBS-007` | Every alert has severity, owner, response target, runbook, suppression, and escalation metadata. |
| `NFR-OBS-008` | Typed dependency, capacity, conflict, and internal failure outcomes remain distinct. |
| `NFR-OBS-009` | High-severity completion includes incident impact, evidence, recovery, and follow-up. |
| `NFR-OBS-012` | Alert policy changes are authorized, versioned, and auditable. |
| `NFR-MNT-007` | Critical alerts have owned runbook coverage and review expectations. |
| `NFR-SEC-010` | Secrets and provider credentials remain outside diagnostics. |
| `NFR-TST-003` | Requirements map to verification evidence or an explicit deferral. |
| `QG-01` / `QG-08` / `QG-10` | Source integrity, operational readiness, and release evidence gates. |
