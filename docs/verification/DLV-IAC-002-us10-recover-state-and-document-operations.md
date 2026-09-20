# DLV-IAC-002 User Story 10 — Recover state and document operations

Status: repository runbook and evidence documentation implemented; authenticated
Azure backup/restore and recovery exercise evidence remains pending.

## Scope and evidence boundary

This record covers the repeatable recovery guidance for the optional Azure
`dev` and disposable `demo` Terraform environments. It also documents the
`prod-reference` recovery requirements without claiming that the reference
profile has been deployed or qualified.

The runbooks are operator guidance. They do not execute Azure, Terraform,
backup, restore, identity, or resource-group mutations during repository
verification. Live exercises require an approved subscription, synthetic data,
an identified operator, and redacted evidence.

Traceability:

`M0` → `EP-IAC-001` → `DLV-IAC-002` → User Story 10

Controls: `RUN-008`, `RUN-009`, `GFR-001`, `GFR-012`, `GFR-014`, `GFR-015`,
`NFR-SEC-010`, `NFR-PRV-004`, `NFR-PRV-005`, `NFR-PRV-008`, `NFR-REC-006`–
`NFR-REC-012`, `NFR-OBS-003`, `NFR-OBS-004`, `NFR-OBS-007`, `NFR-OBS-009`,
`NFR-MNT-001`, `NFR-MNT-007`, `NFR-MNT-008`, `NFR-TST-008`, and
`NFR-TST-009`; quality gates `QG-01`, `QG-06`, `QG-08`, `QG-09`, and `QG-10`.

## Ownership and recovery invariants

- Infrastructure owns Terraform resources, backend configuration, environment
  profiles, state identities, and deployment evidence. It does not own finance
  records, application migrations, application authorization, or finance
  reconciliation semantics.
- Bootstrap state and workload state are separate. A `dev` or `demo` recovery
  never targets the bootstrap state or another environment container.
- Terraform code and reviewed plans remain the normal ownership path. Portal
  edits are emergency evidence only and must be reconciled through Terraform
  before the environment is considered recovered.
- `dev` and `demo` are the only learning-profile mutation targets. The
  `prod-reference` profile is not a destroy target and has no live qualification
  claim in this record.
- State, plans, credentials, connection strings, raw Azure errors, and
  sensitive backup contents are confidential. Evidence contains identifiers,
  outcomes, and redacted references only.
- Recovery is incomplete until resource/state consistency, backup or restore
  outcome, required smoke checks, reconciliation, and unresolved exceptions are
  recorded.

## Supported operator surfaces

Use the existing wrappers and selected environment consistently:

```text
ENVIRONMENT=dev make terraform-drift-check
ENVIRONMENT=dev make azure-learning-plan
CONFIRM_APPLY=dev ENVIRONMENT=dev make azure-learning-apply
CONFIRM_DESTROY=demo ENVIRONMENT=demo make azure-learning-destroy
ENVIRONMENT=demo make azure-learning-smoke
make azure-learning-deployment-check
make azure-learning-destroy-check
```

Authenticated commands require the environment-specific Azure Blob backend
configuration and Azure identity inputs described in the deployment and OIDC
verification records. Never copy those values into this document or an
evidence report.

## Runbook control checklist

Both `RUN-008` and `RUN-009` use the following control fields:

- **Owner:** infrastructure/deployment operator; the escalation owner is the
  project or cloud operations owner for the selected environment.
- **Prerequisites:** approved environment and scope, current commit, required
  Azure/Terraform identity, reviewed plan or recovery approval, synthetic data
  for any database exercise, and a redacted evidence location.
- **Decision points:** active versus stale lock, in-sync versus drifted state,
  recoverable versus foreign resource, safe retry versus escalation, and
  learning-profile exercise versus production-reference requirement.
- **Evidence:** owner, timestamps, environment, command, result, sanitized
  state/backup references, reconciliation outcome, and unresolved exception.
- **Escalation:** stop on active locks, subscription or resource identity
  mismatch, foreign resources, uncertain external outcomes, failed backup, or
  any inability to establish ownership and consistency.
- **Completion checks:** lock released, state and resources reconciled, smoke
  checks completed where applicable, temporary artifacts removed, and every
  residual exception assigned to an owner and next action.

## RUN-008 — Terraform state recovery and drift

### State-lock recovery

1. Stop duplicate apply, plan, destroy, or drift actions for the selected
   environment. Record the operator, environment, UTC time, command, and
   correlation/reference value.
2. Confirm the selected subscription, Terraform root, backend storage account,
   state container, and current operation owner. An active operation is resolved
   by its owner or by waiting for its bounded operation to finish; it is not
   unlocked concurrently.
3. If the lock is proven stale after the active-operation check, obtain the
   required recovery approval and use Terraform's supported `force-unlock`
   operation against the exact environment root and lock ID. Record the lock
   ID, reason, approver, and command outcome. Never delete a Blob lock artifact
   manually.
4. Reinitialize the exact backend with the reviewed environment configuration,
   run a refresh-only drift check, and create a new reviewed plan before any
   apply or destroy action.

`force-unlock` is not part of the normal destroy workflow. The Story 9 destroy
wrapper deliberately verifies lock release and does not perform manual lock
deletion or force-unlock.

### Drift recovery

`make terraform-drift-check` is read-only and uses Terraform's refresh-only
plan. Interpret its results as follows:

- Exit `0`: the selected environment is in sync; retain the command result.
- Exit `2`: drift exists; preserve the redacted result, classify each change,
  and remediate through Terraform code, an approved variable change, or a
  reviewed import/state correction.
- Any other nonzero exit: treat backend, identity, provider, or Terraform
  failure as unresolved; do not apply a guessed correction.

After remediation, rerun refresh-only drift detection and a normal plan. A
recovery is not complete while unmanaged or foreign resources remain in scope.

### Failed apply

- Stop and preserve the sanitized workflow or command result; do not paste raw
  plans, state, credentials, or provider payloads into the evidence record.
- Confirm the selected environment and inspect the current state before retrying.
  The existing apply path applies the exact reviewed plan and uses the backend
  lock timeout; it must not be replaced by an ad-hoc resource command.
- If resources were partially created, reconcile the state and remote resource
  identities through Terraform. Use import only as a separately reviewed
  remediation when an existing resource is intentionally Terraform-owned.
- Re-run plan, review scope and cost, then apply only the approved result. Any
  unresolved provider or ownership mismatch remains an exception.

### Failed destroy

- Use only the guarded Story 9 destroy workflow for `dev` or `demo`; preserve
  the pre-destroy backup result and redacted destroy report.
- Confirm whether PostgreSQL, the resource group, and the state entries still
  exist. A missing PostgreSQL server makes a new database backup not applicable;
  it does not justify deleting unrelated resources.
- Resolve the failure cause, verify the selected state lock is released, and
  rerun only after a new reviewed destroy plan and backup decision.
- Verify resource-group absence, empty workload state, lock release, cleanup,
  and unresolved exceptions. Never use `az group delete`, bootstrap state,
  another environment root, or manual lock deletion.

### Resource-group recovery

- Verify subscription, environment profile, resource-group ID/name, and the
  expected Terraform root before changing anything.
- If the selected learning resource group is absent, use the reviewed
  environment plan/apply path to recreate it. Do not create a replacement group
  through the portal and continue as if it were Terraform-owned.
- If the group exists but its identity, tags, ownership, or contents do not
  match the selected state, stop and escalate. Foreign or unmanaged resources
  must be classified before import, correction, or destroy.
- Re-run plan, drift, deployment smoke checks, and evidence capture after
  recreation or reconciliation.

## RUN-009 — Credential and identity rotation

1. Identify the credential or identity, environment, affected scope, current
   owner, expiry/rotation reason, and rollback or overlap window. Do not record
   secret values.
2. Provision or select the replacement through the approved Terraform, Key
   Vault, managed-identity, or GitHub OIDC path. Environment identities remain
   scoped to their own state container and resource group.
3. Keep the old identity or credential available during the approved overlap;
   update references and reviewed Terraform inputs, then run a plan before
   applying the change.
4. Verify backend state access, ACR pull, Key Vault access, deployment identity,
   and the selected environment's smoke checks. For write-only PostgreSQL
   credentials, update the secret through the external secret path and advance
   its version without echoing the value.
5. Retire the old identity only after verification. Record the old/new
   identifier references, owner, timestamps, permission scope, verification
   result, retirement result, and any exception. Preserve lineage without
   retaining secret material.

## Backup, restore, and recovery exercise obligations

| Area | Learning `dev`/`demo` | `prod-reference` requirement |
|---|---|---|
| Terraform state | Azure Blob versioning and deletion retention are part of bootstrap; access is scoped to the selected state container. Restore only through an approved state-recovery procedure and never commit downloaded state. | Private, controlled backend access with tested version recovery and ownership review. No live qualification is claimed here. |
| Database data | Synthetic data only. Current learning PostgreSQL profiles use seven-day backup retention and no production RTO/RPO claim. Restore into an isolated disposable target. | The reference profile specifies 35-day retention, geo-redundant backup, HA/private topology, residency, encryption, access, retention, and legal-hold controls. |
| Access | Use the approved Azure identity and least-privilege scope. Do not place tokens, passwords, connection strings, or raw backup contents in evidence. | Validate managed identity, private access, separation of duties, recovery authorization, and audit evidence. |
| Reconciliation | Verify resource identity, schema/migration compatibility, record counts/control totals where applicable, source watermarks, pending work, and smoke outcomes. | Also verify authoritative finance balances, dependent outcomes, audit sequence/proof, legal holds, and owned residual exceptions. |
| Evidence | Record owner, UTC start/end, environment, command, result, backup/restore reference, reconciliation result, and unresolved exception. | Record measured RTO/RPO and business, operations, and security sign-off before making a production claim. |

The repository does not claim that an authenticated restore, finance-data
reconciliation, RTO/RPO measurement, or production disaster-recovery exercise
has occurred. Those results must be added only from a witnessed, approved run.

## Operational evidence template

```text
Story: User Story 10 / DLV-IAC-002
Runbook: RUN-008 or RUN-009 / scenario:
Commit:
Environment: dev, demo, or prod-reference reference-only
Operator / owner:
Approver / escalation owner:
UTC start/end:
Subscription and region: sanitized reference only
Command: redacted command and wrapper target
Result: pass, fail, blocked, or not-run
State/backup/restore reference: sanitized identifier only
Reconciliation checks and result:
Cleanup / lock-release result:
Unresolved exception and next action:
Evidence location:
```

## Repository verification

The credential-free evidence boundary is:

```text
git diff --check
make terraform-check
make terraform-environments-check
make terraform-modules-check
make terraform-ci-check
make azure-learning-deployment-check
make azure-learning-destroy-check
```

These checks validate the existing Terraform, deployment, drift, identity,
destroy, and secret-hygiene contracts. They do not contact Azure or establish
live backup, restore, recovery, or production qualification evidence.

## Acceptance status

The repository runbook, evidence template, traceability, and learning versus
production boundary are documented here. The Story 10 acceptance remains open
for authenticated recovery exercises and their redacted evidence.
