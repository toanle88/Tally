# Telemetry and monitoring failure

## Runbook metadata

| Field | Value |
|---|---|
| Runbook | Named platform scenario: Telemetry and monitoring failure |
| Owner | Platform Operations |
| Version | `v1` |
| Effective date | 2026-09-22 |
| Status | M0 local operational baseline |
| Audience | Authorized operations users and service owners |
| Data classification | Operational diagnostic data; no telemetry payloads or secrets |
| Related dashboard | [dashboard.v1](../dashboard-contract-v1.md) — stale/unknown behavior |
| Related alerts | Dependency failure, critical control failure, or capacity risk as applicable |

This is intentionally unnumbered. The approved technical runbook catalog
defines `RUN-001` through `RUN-010`; this story does not invent an additional
catalog identifier.

## Prerequisites

- Identify the affected signal, service, environment, time window, and
  sanitized support reference.
- Determine whether the application result is independent of the diagnostic
  exporter or monitoring provider.
- Preserve the last known safe state without requesting raw telemetry,
  credentials, tokens, or provider payloads.

## Detection

Start this runbook when metrics, traces, logs, dashboard data, or exporter
shutdown reports are missing, stale, malformed, or unavailable. Treat missing
data as `unknown`, `stale`, or `degraded`; never infer healthy behavior from
absence of observations.

## Decision points

1. If only telemetry is unavailable and the application control state remains
   known, keep business behavior independent of telemetry failure and escalate
   the diagnostic outage.
2. If a critical control or integrity state cannot be established because the
   source is unavailable, block affected writes according to the owning
   control policy and escalate.
3. Do not create an unbounded exporter retry loop or turn exporter failure into
   a successful business result.
4. Use the existing local failure tests when diagnosing repository behavior;
   do not add live provider configuration to the incident path.

## Safe commands

These commands use synthetic local exporters and existing health/contract
checks. They do not require Azure credentials or production data.

```bash
curl --fail --silent http://localhost:8080/health/live
make telemetry-failure-sensitive-data-check
make traces-metrics-check
make dashboard-contract-check
make alert-contract-check
```

The liveness response is not a readiness, financial, audit, or monitoring
qualification result.

## Evidence to preserve

- UTC observation window, service/environment, commit/build identifier,
  support reference, signal state, exporter result, and bounded error code.
- Whether business processing continued, was degraded, or was blocked, with
  the owner and decision recorded.
- Sanitized contract/test output only.

Do not preserve raw request/event/response bodies, SQL, credentials, tokens,
connection strings, or unrestricted identifiers.

## Escalation

Escalate to Platform Operations and the affected service owner. Escalate to
the Incident Commander when a critical control, integrity state, or required
support reference cannot be established.

## Recovery checks

- Exporter shutdown and failure behavior is bounded and at-most-once.
- Logs, traces, metrics, and dashboard freshness return to an observed state.
- Sensitive-data and redaction checks remain passing.
- Application outcomes remain independent of diagnostic export success.

## Reconciliation requirements

Telemetry is not authoritative financial, audit, posting, settlement,
reconciliation, or recovery evidence. Reconcile any affected business or
control state through the owning database, outbox/inbox, audit, or domain
operation; do not reconstruct financial truth from logs or metrics.

## Closure criteria

Close only when signal freshness is restored or an approved residual
exception has an owner and expiry, application impact is recorded, sensitive
data checks pass, and affected operators have been notified.

## Deferred qualification

Live Azure Monitor export, paging integration, retention qualification,
quarterly alert exercises, and production monitoring failure exercises remain
deferred. The local telemetry checks do not establish production readiness.

## Safety rules

Direct destructive financial edits are prohibited. Unreviewed portal changes
are prohibited. Secret disclosure is prohibited. Telemetry is diagnostic and
is not authoritative financial, audit, or recovery evidence.
