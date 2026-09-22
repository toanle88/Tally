# RUN-002 — Outbox backlog or poison item

## Runbook metadata

| Field | Value |
|---|---|
| Runbook | `RUN-002` — Outbox backlog/poison item |
| Owner | Platform Operations with the integration/worker owner |
| Version | `v1` |
| Effective date | 2026-09-22 |
| Status | M0 local operational baseline |
| Audience | Authorized operations users and the integration/worker owner |
| Data classification | Operational diagnostic data; payloads and unrestricted identifiers are excluded |
| Related dashboard | [dashboard.v1](../dashboard-contract-v1.md) — Outbox/inbox pending work and age |
| Related alerts | Outbox/backlog age |

## Prerequisites

- Confirm the environment, support reference, event type, consumer, and typed
  failure class using bounded diagnostic fields only.
- Identify the integration/worker owner and whether the work is pending,
  retryable, leased, established, or a managed exception.
- Preserve the current state before changing retry or delivery behavior.
- Do not request event bodies, message identifiers, credentials, or raw
  database records.

## Detection

Start this runbook when `finance_outbox_pending_total` or
`finance_outbox_oldest_age_seconds` violates its configured workflow/NFR
policy, when a managed exception is present, or when delivery failure and
inbox failure metrics show a typed operational exception.

## Decision points

1. If the dependency is unavailable, keep work pending or in its existing
   managed state and use controlled degradation.
2. If the item is retryable, verify the established-result and lease state
   before allowing controlled retry.
3. If the item is poison or non-transient, keep the durable managed exception;
   do not delete the item or silently retry it.
4. If an identity-content conflict, duplicate effect, or uncertain result is
   reported, stop the affected path and escalate to the integration and
   owning-capability owners.
5. Use the existing dispatcher/replay operation only when an authorized
   operator interface exists. This repository does not invent a manual replay
   CLI or permit raw database repair.

## Safe commands

The following commands verify the local contract and existing worker/replay
behavior with synthetic data. They are not a substitute for an authorized
production replay action.

```bash
make dashboard-contract-check
make alert-contract-check
make outbox-worker-check
```

No direct update, delete, payload replacement, lease override, or unreviewed
manual replay command is approved by this runbook.

## Evidence to preserve

- UTC observation window, environment, commit/build identifier, support
  reference, event type, consumer, bounded error code, and failure class.
- Pending count, oldest age, attempt state, managed-exception state, and
  sanitized worker result.
- Owner, decision, retry/replay approval, and final result.

Do not preserve event bodies, raw SQL, message identifiers, credentials,
tokens, or unrestricted financial identifiers.

## Escalation

Escalate from Platform Operations to the integration/worker owner. Escalate to
the Incident Commander when the configured backlog response objective is at
risk, an integrity-uncertain state appears, or duplicate delivery cannot be
ruled out.

## Recovery checks

- The cause or dependency state is understood and bounded.
- Controlled delivery resumes without bypassing lease ownership or inbox
  establishment.
- Pending age and count move toward the configured target.
- Managed exceptions are resolved only through an approved operation.
- Duplicate effects, changed fingerprints, and stale claims are not observed.

## Reconciliation requirements

Reconcile the established local result before retry or replay. Verify durable
outbox/inbox state, source watermarks, dependent outcomes, and any owning
capability evidence. A replay must preserve the original event facts and must
not publish a new integration effect merely because the original result is
uncertain.

## Closure criteria

Close only when the backlog condition is resolved or assigned, residual
managed exceptions have owners, controlled delivery evidence is preserved,
and the integration/owning-capability owner accepts the result.

## Deferred qualification

This runbook does not qualify a production backlog catch-up rate, external
broker, finance-domain consumer, or live alerting provider. Those require the
owning capability and recovery qualification evidence; those qualifications
remain deferred.

## Safety rules

Direct destructive financial edits are prohibited. Unreviewed portal changes
are prohibited. Secret disclosure is prohibited. Telemetry is diagnostic and
is not authoritative financial, audit, or reconciliation evidence.
