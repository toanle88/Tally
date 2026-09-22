# DLV-OPS-002 User Story 3 — Runbook template and initial platform runbooks

## Scope

This evidence covers `EP-OPS-001` / `DLV-OPS-002` User Story 3. It verifies
the provider-neutral runbook template and the initial platform procedures for
failed migration, outbox backlog/poison item, database restore, telemetry and
monitoring failure, and capacity saturation.

The telemetry/monitoring procedure is intentionally unnumbered. The approved
observability catalog defines `RUN-001` through `RUN-010`; this story does not
invent `RUN-011` or alter that catalog.

This evidence does not claim a dashboard UI, paging provider, live Azure
Monitor, production credentials, production restore, production capacity
qualification, quarterly exercises, finance-domain runbooks, or User Story 4
aggregate operational-readiness evidence.

## Source state

| Field | Value |
|---|---|
| Branch | `codex/dlv-ops-002-us3-runbook-foundation` |
| Baseline | `main` after DLV-OPS-002 User Story 2 alert contract |
| Baseline commit | `0533126` |
| Implementation state | Local documentation and contract-verification changes; no commit, staging, or push performed |
| Verification profile | Provider-neutral, synthetic contract checks and existing local platform gates |
| Remote dependencies | None; no Azure credentials, production data, paging provider, or live monitoring service required |

## Implementation

- [`runbook-template-v1.md`](../operations/runbook-template-v1.md) defines
  owner, prerequisites, detection, decisions, safe commands, evidence,
  escalation, recovery, reconciliation, closure, and deferred qualification.
- The initial runbooks are maintained as separate linked documents:
  [`RUN-001`](../operations/runbooks/run-001-failed-migration.md),
  [`RUN-002`](../operations/runbooks/run-002-outbox-backlog-poison-item.md),
  [`RUN-003`](../operations/runbooks/run-003-database-restore.md),
  [telemetry and monitoring failure](../operations/runbooks/telemetry-monitoring-failure.md),
  and [`RUN-010`](../operations/runbooks/run-010-capacity-saturation.md).
- The alert contract links delivered `RUN-001`, `RUN-002`, and `RUN-010`
  procedures directly while future capability-owned runbooks retain their
  approved technical-specification references.
- [`runbook-contract.sh`](../../scripts/verify/runbook-contract.sh) validates
  mandatory sections, safety boundaries, approved identifiers, direct alert
  links, and temporary negative cases.
- The Makefile exposes `runbook-contract-check`, and `scripts/README.md`
  records the new verification gate.
- No application/API, OpenAPI, database, event, worker, frontend,
  authorization, infrastructure, or finance-domain code was changed.

## Acceptance evidence

| Acceptance criterion | Evidence |
|---|---|
| Template captures owner, prerequisites, detection, decision points, safe commands, evidence, escalation, recovery, reconciliation, and closure | Template required-section validation in `make runbook-contract-check` |
| Initial platform runbooks cover the five approved story scenarios | Five runbook files and the contract verifier's exact file inventory |
| Unsafe direct edits, portal changes, secret disclosure, and telemetry-as-truth are prohibited | Required safety text and negative fixtures in `runbook-contract.sh` |

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Observability and operational foundation milestone. |
| `EP-OPS-001` / `DLV-OPS-002` | Parent epic and baseline dashboard/runbook delivery item. |
| `GFR-012` | Operational outcomes preserve intermediate, exception, reconciliation, and terminal states. |
| `ARC-OBS-001` / `ARC-OBS-002` | Correlation and operational telemetry remain separate from authoritative business evidence. |
| `ARC-PRV-001` | Diagnostics exclude sensitive values and credentials. |
| `ARC-CAP-001` | Capacity behavior uses bounded, controlled states. |
| `ARC-REC-001` | Recovery preserves authoritative records and reconciliation. |
| `ARC-TST-001` | Contract and negative-case verification is reproducible. |
| `NFR-OBS-001`–`NFR-OBS-012` | Health, detection, support references, safe telemetry, backlog, alert, incident, and change-control expectations. |
| `NFR-MNT-001`, `NFR-MNT-007`, `NFR-MNT-010` | Traceability, runbook coverage, and no unresolved critical operational defect. |
| Applicable `NFR-REC-*` | Restore, backlog, no-duplicate, authorization, and reconciliation boundaries. |
| `NFR-TST-003`, `NFR-TST-008`, `NFR-TST-009` | Verification method, recovery evidence, runbooks, and release evidence. |
| `QG-01` / `QG-08` / `QG-10` | Source integrity, operational readiness, and release evidence gates. |

## Verification results

| Command | Result | Evidence |
|---|---|---|
| `bash -n scripts/verify/runbook-contract.sh` | Passed | Shell syntax is valid. |
| `make runbook-contract-check` | Passed | Template, five runbooks, links, safety rules, identifiers, and all negative fixtures passed. |
| `make alert-contract-check` | Passed | Existing alert contract accepted the direct delivered-runbook links. |
| `make dashboard-contract-check` | Passed | Existing dashboard contract and negative fixtures passed. |
| `make telemetry-failure-sensitive-data-check` | Blocked | Linux environment has no `go`; invoking `/mnt/c/Program Files/Go/bin/go.exe` was blocked by the WSL vsock runtime error. |
| `make traces-metrics-check` | Blocked | Linux environment has no `go`. |
| `make outbox-worker-check` | Blocked | Linux environment has no `go`. |
| `make db-migrate-validate` | Blocked | Linux environment has no `go`. |
| `make db-migrate-check` | Passed | Migration checksum inventory is valid. |
| `make repository-integrity-check` | Passed | Repository integrity verification passed. |
| `git diff --check` | Passed | No whitespace errors; Git emitted existing CRLF normalization warnings. |

## Deferred qualification and source boundaries

- Runbook content is a local operational contract; it does not create an
  operational console, incident store, paging integration, or monitoring
  provider.
- Shared migration recovery remains forward-fix or isolated restore; direct
  destructive financial edits are prohibited.
- Outbox event facts remain immutable, poison work remains a durable managed
  exception, and replay requires established-result/reconciliation checks.
- Database restore requires isolated validation and reconciliation; the current
  M0 repository does not contain the future finance schemas needed to claim
  production financial control totals or audit-sequence evidence.
- Telemetry remains diagnostic and cannot establish financial truth or change a
  business result when export fails.
- Production Azure recovery, capacity, retention, quarterly runbook review,
  external-provider qualification, and User Story 4 aggregate verification
  remain deferred.
