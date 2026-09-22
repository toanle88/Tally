# RUN-003 — Database restore

## Runbook metadata

| Field | Value |
|---|---|
| Runbook | `RUN-003` — Database restore |
| Owner | Platform Operations with the database and owning-capability owners |
| Version | `v1` |
| Effective date | 2026-09-22 |
| Status | M0 local operational baseline |
| Audience | Authorized recovery operators |
| Data classification | Operational diagnostic data; backup contents remain restricted |
| Related dashboard | [dashboard.v1](../dashboard-contract-v1.md) — PostgreSQL health and operational exceptions |
| Related alerts | Database saturation or critical control failure, as applicable |

## Prerequisites

- Declare the incident, environment, restore target, owner, and approved
  recovery decision.
- Confirm the restore target is isolated from the failed source until
  validation is complete.
- Confirm the backup/restore reference without copying credentials, connection
  strings, or backup contents into operational evidence.
- Define the expected RTO/RPO measurement window and reconciliation owners.

## Detection

Start this runbook when the database is unavailable or corrupted, a recovery
decision requires restoration, or an approved backup/point-in-time recovery
exercise begins. A process restart is not a database restore.

## Decision points

1. Preserve the source state and incident evidence before restoring.
2. Restore only into an isolated, approved target first.
3. Apply the approved migration set and verify compatibility before exposing
   the target to application work.
4. If authoritative records, audit evidence, or source watermarks cannot be
   reconciled, keep affected writes blocked and escalate.
5. Do not use a local disposable rebuild as evidence of production restore
   capability.

## Safe commands

These commands validate the local migration and database contract after an
isolated target is available. Provider-specific backup commands are external
operator procedures and are not invented here.

```bash
make db-migrate-validate
make db-migrate-check
make db-migrate-status
make db-verify
```

For a disposable local target only, the approved database lifecycle commands
may be used after explicit confirmation:

```bash
make db-up
make db-migrate
make db-verify
```

No command in this runbook authorizes direct data repair or destructive edits.

## Evidence to preserve

- Sanitized environment, source/target class, backup reference, restore start
  and end times, commit/build identifier, migration version, and command
  results.
- Measured RTO/RPO, validation result, reconciliation result, residual
  exceptions, owner, and approval.
- Redacted backup/restore provider status only; never backup contents or
  credentials.

## Escalation

Escalate to the database owner and Incident Commander for any production or
shared-environment restore. Escalate to security and the owning capability if
access controls, legal hold, audit evidence, or financial reconciliation is
uncertain.

## Recovery checks

- The isolated target is healthy and migration-compatible.
- `make db-verify` passes for the applicable repository baseline.
- Access, encryption, retention, and legal-hold controls are verified by the
  responsible environment owner.
- Application work is enabled only after reconciliation approval.

## Reconciliation requirements

Reconcile authoritative records, audit/evidence sequence, control totals,
pending work, source watermarks, idempotency identities, correction lineage,
and dependent outcomes. Current M0 repository state does not contain the
future finance schemas needed to claim those checks; record them as deferred
or not applicable rather than passing them by inspection.

## Closure criteria

Close only after restore validation, RTO/RPO measurement, reconciliation
sign-off, access-control checks, residual-exception ownership, and recovery
communication are recorded.

## Deferred qualification

Live Azure/PostgreSQL backup restore, point-in-time recovery, quarterly restore
testing, annual disaster recovery, production RTO/RPO, and finance-domain
reconciliation are deferred qualification work. This document is not
production recovery evidence.

## Safety rules

Direct destructive financial edits are prohibited. Unreviewed portal changes
are prohibited. Secret disclosure is prohibited. Telemetry is diagnostic and
is not authoritative financial, audit, posting, or recovery evidence.
