# RUN-010 — Capacity saturation

## Runbook metadata

| Field | Value |
|---|---|
| Runbook | `RUN-010` — Capacity saturation |
| Owner | Platform Operations with the database, worker, and capacity owners |
| Version | `v1` |
| Effective date | 2026-09-22 |
| Status | M0 local operational baseline |
| Audience | Authorized operations users and service owners |
| Data classification | Operational diagnostic data; bounded capacity signals only |
| Related dashboard | [dashboard.v1](../dashboard-contract-v1.md) — Capacity and PostgreSQL health |
| Related alerts | Database saturation and capacity risk |

## Prerequisites

- Confirm the environment, support reference, owner, and bounded capacity
  signal involved.
- Identify whether the condition is observed saturation, admission control,
  queue/backlog growth, dependency unavailability, or an unknown/stale
  measurement.
- Confirm the approved capacity or maintenance decision before changing
  runtime configuration.

## Detection

Start this runbook when worker/API/database/provider capacity enters a typed
warning or control state, admission is rejecting or deferring work, queue age
is growing, or the dashboard cannot obtain a fresh capacity observation.
Unsupported provider percentages are not a trigger policy.

## Decision points

1. If capacity is unknown or stale, preserve the uncertainty and do not claim
   a green state.
2. Protect authoritative writes and critical controls before accepting more
   work.
3. Apply bounded admission, concurrency, batch, or approved scaling changes
   only with an owner and rollback path.
4. If the cause is a dependency or database failure, transfer to the relevant
   dependency or migration/recovery runbook.
5. Never silently drop accepted work; preserve pending or exceptional state
   and escalate when the recovery objective is at risk.

## Safe commands

These commands validate the local contract and synthetic worker/capacity
behavior. Provider-specific scaling and portal commands require a separately
approved environment procedure and are not introduced here.

```bash
make dashboard-contract-check
make alert-contract-check
make outbox-worker-check
```

The worker's bounded settings (`DB_MAX_CONNS`, `OUTBOX_BATCH_SIZE`, and
`OUTBOX_POLL_INTERVAL`) may be reviewed as configuration evidence. Changing
them requires authorization, validation, and a rollback path.

## Evidence to preserve

- UTC observation window, environment, support reference, commit/build
  identifier, capacity state, bounded metric values, and owner.
- Admission/rejection/defer state, pending age/count, dependency state,
  action approval, and measured result.
- No instance identifiers, customer identifiers, raw resource names, secrets,
  credentials, or provider payloads.

## Escalation

Escalate to the capacity owner and affected database/worker/service owner.
Escalate to the Incident Commander when integrity, availability, backlog
recovery, or a critical response objective is at risk.

## Recovery checks

- Capacity state is observed and bounded rather than inferred from absence.
- Admission and concurrency remain within approved limits.
- Critical writes are protected and no work was silently dropped.
- Backlog age, error classes, and dependency state move toward their approved
  targets.
- Any temporary configuration has an expiry or rollback decision.

## Reconciliation requirements

Reconcile accepted, pending, rejected, and exceptional work; outbox/inbox
state; source watermarks; and owning-capability results. Capacity recovery
must not duplicate an acknowledged operation or convert an uncertain external
outcome into a second business obligation.

## Closure criteria

Close only after capacity action, rollback/expiry, backlog behavior, no-drop
evidence, owner acceptance, residual exceptions, and communication are
recorded.

## Deferred qualification

Production capacity thresholds, load profiles, Azure scaling, cost impact,
headroom, and NFR performance/recovery qualification remain deferred. This
runbook is not a production capacity result.

## Safety rules

Direct destructive financial edits are prohibited. Unreviewed portal changes
are prohibited. Secret disclosure is prohibited. Telemetry is diagnostic and
is not authoritative financial, audit, or recovery evidence.
