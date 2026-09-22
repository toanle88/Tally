# RUN-001 — Failed migration

## Runbook metadata

| Field | Value |
|---|---|
| Runbook | `RUN-001` — Failed migration |
| Owner | Platform Operations |
| Version | `v1` |
| Effective date | 2026-09-22 |
| Status | M0 local operational baseline |
| Audience | Authorized operations users and the approved release owner |
| Data classification | Operational diagnostic data; no secrets or raw database values |
| Related dashboard | [dashboard.v1](../dashboard-contract-v1.md) — PostgreSQL health and operational exceptions |
| Related alerts | Critical control failure |

## Prerequisites

- Confirm the environment and release/build identifier.
- Obtain an authorized release or incident owner and a sanitized support
  reference.
- Stop an incompatible rollout before applying another migration.
- Do not request database passwords, connection strings, or raw table dumps in
  the incident record.

## Detection

Start this runbook when migration application fails, migration compatibility
is unknown, `/health/ready` reports a migration problem when that endpoint is
available, or the database status does not match the declared application
compatibility window. A failed migration is a control failure, not proof that
the database is safe to use.

## Decision points

1. If the application/database compatibility is unknown, block affected writes
   and preserve the failure evidence.
2. For a shared environment, use a reviewed forward-fix or the approved
   restore procedure. Do not use migration down operations as an ad hoc
   rollback.
3. For a disposable local environment, an isolated restore or rebuild may be
   considered after the required evidence is preserved.
4. If the failure could have changed authoritative facts or audit evidence,
   escalate to the owning capability and Incident Commander before resuming.

## Safe commands

These commands inspect or validate the repository migration contract. Run them
against the declared environment and preserve sanitized output only.

```bash
make db-migrate-status
make db-migrate-validate
make db-migrate-check
make db-verify
```

`make db-migrate` is an approved release action only after compatibility,
ownership, and the forward-fix have been reviewed. No direct database repair
command is part of this runbook.

## Evidence to preserve

- Environment, commit/build identifier, migration set, and migration version.
- UTC start/end times and the sanitized command result.
- Support reference, typed failure class, owner, approval, and decision.
- Whether the environment was shared or disposable.

Do not preserve passwords, connection strings, raw SQL, raw migration output
containing sensitive values, or unrestricted database records.

## Escalation

Escalate first to the release owner and Platform Operations. Escalate to the
database owner and Incident Commander when compatibility, data integrity,
audit evidence, or the recovery path is uncertain. A critical control failure
uses the `NFR-OBS-002` response target.

## Recovery checks

- Migration status is internally consistent and the checksum inventory passes.
- The application compatibility and health contract is restored.
- `make db-verify` passes for the applicable environment.
- No incompatible rollout remains active.

Recovery checks do not establish that finance-domain totals or audit evidence
are correct; those require the owning capability's reconciliation evidence.

## Reconciliation requirements

If the migration touched authoritative data, the owning capability must
reconcile control totals, lifecycle states, source watermarks, and audit
evidence before affected writes resume. Any difference remains an assigned
exception.

## Closure criteria

Close only after the migration decision, command results, recovery checks,
owner approval, residual exceptions, and communication are recorded. A green
health signal alone is insufficient.

## Deferred qualification

This local runbook does not qualify production migration timing, rollback,
Azure database restore, or quarterly recovery exercises. Production release
qualification remains governed by the applicable `NFR-REC-*`, `NFR-MNT-*`, and
`QG-08` evidence; production qualification remains deferred.

## Safety rules

Direct destructive financial edits are prohibited. Unreviewed portal changes
are prohibited. Secret disclosure is prohibited. Telemetry is diagnostic and
is not authoritative financial or audit evidence.
