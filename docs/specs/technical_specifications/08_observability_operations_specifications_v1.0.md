# Finance Platform Observability and Operations Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Implementation-ready operations baseline |
| Telemetry standard | OpenTelemetry with Go `slog` and Azure Monitor export |

## 1. Correlation model

Every HTTP request, application command, database transaction, outbox message and external call propagates `trace_id`, `span_id`, `correlation_id` and `causation_id`. Business identities are attributes only when classification permits. Audit evidence is separate from operational telemetry.

## 2. Structured log schema

| Field | Required | Notes |
|---|---|---|
| `timestamp`, `level`, `message` | Yes | RFC3339 UTC and structured severity |
| `service`, `module`, `operation` | Yes | Stable low-cardinality names |
| `trace_id`, `span_id`, `correlation_id` | Yes for requests/work | Propagation fields |
| `actor_id`, `accounting_scope_id` | Material actions | Pseudonymous IDs, not names |
| `aggregate_type`, `aggregate_id`, `aggregate_version` | Aggregate operations | No unrestricted payload |
| `result`, `error_code`, `retryable` | Yes | Stable taxonomy |
| `data_classification` | Yes | Redaction policy input |

Logs shall not include tokens, credentials, complete bank details, payroll values, unrestricted tax identifiers or full event/request bodies.

## 3. Metrics catalog

| Metric | Type | Labels | Purpose |
|---|---|---|---|
| finance_http_request_duration_seconds | histogram | route, method, status_class | Interactive latency |
| finance_command_total | counter | module, operation, result | Command outcomes |
| finance_db_transaction_duration_seconds | histogram | module, operation, result | Database transactions |
| finance_outbox_pending_total | gauge | event_type | Backlog |
| finance_outbox_oldest_age_seconds | gauge | event_type | Delivery age |
| finance_inbox_failure_total | counter | consumer, error_code | Consumer failures |
| finance_journal_posting_total | counter | purpose, result | Posting outcomes |
| finance_receipt_application_conflict_total | counter | reason | AR conflicts |
| finance_payment_instruction_exception_total | counter | reason | Payment exceptions |
| finance_unreconciled_settlement_total | gauge | source_context | Settlement backlog |
| finance_approval_request_age_seconds | histogram | policy, state | Approval aging |
| finance_period_close_stage_duration_seconds | histogram | stage, result | Close progress |
| finance_audit_integrity_incident_total | counter | type, severity | Integrity incidents |

All labels use bounded cardinality. Aggregate IDs, customer IDs and free-form error messages are prohibited metric labels.

## 4. Trace spans

Required spans: HTTP request, authorization decision, idempotency lookup, command handler, repository operation, PostgreSQL transaction, outbox claim/delivery, inbox handling, provider call, report job and recovery action. SQL statement text is normalized/redacted.

## 5. Dashboards

1. API health and latency by route class.
2. PostgreSQL connections, locks, transaction age, replication/backup health and slow queries.
3. Outbox/inbox backlog, age, attempts and poison items.
4. Journal posting and rejection reasons.
5. Payment settlement/return exceptions and reconciliation.
6. Approval backlog and aging.
7. Period close/reopen stages and blockers.
8. Audit integrity and evidence status.
9. Azure cost against learning budget.

## 6. Alert rules

| Severity | Trigger | Required response |
|---|---|---|
| P1 | Possible duplicate financial effect, audit mismatch, lost posting evidence, or broad Class A outage | Immediate page; block affected writes if integrity is uncertain |
| P2 | Outbox age or close/payment backlog exceeds NFR, DB saturation, recovery objective at risk | Urgent response and controlled degradation |
| P3 | Single provider failure, report delay, elevated conflicts | Business-hours response with user-visible status |
| P4 | Capacity forecast or noncritical job warning | Planned remediation |

Every alert links to a runbook, dashboard and owning team. Alerts based on absence of data include deployment/maintenance suppression logic.

## 7. Health endpoints

- `/health/live`: process event loop and fatal-state check only.
- `/health/ready`: database connectivity, migration compatibility and ability to accept declared work; optional dependencies do not make the whole API unready unless required for the route class.
- Health responses expose no secrets or internal topology.

## 8. Runbook catalog

| ID | Scenario | Required actions |
|---|---|---|
| RUN-001 | Failed migration | Stop incompatible rollout, preserve evidence, forward-fix or restore disposable environment |
| RUN-002 | Outbox backlog/poison item | Identify oldest/failed type, repair prerequisite or payload, controlled replay |
| RUN-003 | Database restore | Isolated restore, migrate, verify audit sequence, reconcile finance totals, measure RTO/RPO |
| RUN-004 | Payment provider outage | Stop unsafe retries, result lookup, reconcile provider evidence, resume safely |
| RUN-005 | Period-control interruption | Inspect authoritative gate/owner/version, take over only through DDD operation |
| RUN-006 | Audit mismatch | Block affected evidence-dependent actions, preserve artifacts, escalate incident |
| RUN-007 | Entra/authorization outage | Deny new privileged actions; preserve safe read access only where policy allows |
| RUN-008 | Terraform state recovery | Lock changes, restore backend/version, refresh/plan, reconcile drift |
| RUN-009 | Credential rotation | Overlap, deploy references, verify, retire old credential, retain lineage |
| RUN-010 | Capacity saturation | Apply admission limits, protect writes, scale approved workloads, diagnose cause |

## 9. Telemetry retention

Operational logs and traces follow data-classification and cost policy and are not the record of financial truth. Audit evidence and legally required records follow their separate retention and hold rules.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `39c6ab147df81494142db7b41b01b1a12dc3034cd85f348fe3bd8ef39c3b37d4` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
