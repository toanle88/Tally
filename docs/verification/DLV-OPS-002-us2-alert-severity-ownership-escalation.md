# DLV-OPS-002 User Story 2 — Alert severity, ownership, and escalation

## Scope

This evidence covers `EP-OPS-001` / `DLV-OPS-002` User Story 2. It verifies
the provider-neutral alert contract for P1–P4 severity, Platform Operations
ownership, response targets, dashboard and approved runbook references,
maintenance suppression, escalation, and completion evidence across the six
M0 alert categories:

- Integrity uncertainty
- Critical control failure
- Outbox/backlog age
- Database saturation
- Dependency failure
- Capacity risk

This evidence does not claim a paging provider, dashboard UI, live Azure
monitoring, incident persistence, production thresholds, quarterly alert
exercises, or User Story 3 runbook content.

## Source state

| Field | Value |
|---|---|
| Branch | `codex/dlv-ops-002-us2-alert-severity-ownership-escalation` |
| Baseline | `main` after DLV-OPS-002 User Story 1 dashboard contract |
| Baseline commit | `3cb0099b301302a5d64d8ee949faff01bcd5d89d` (`main`) |
| Implementation state | Local contract and verification changes; no commit, staging, or push performed |
| Verification profile | Provider-neutral, synthetic contract copies only |
| Remote dependencies | None; no Azure credentials, production data, paging provider, or dashboard provider required |

## Implementation

- [`docs/operations/alert-contract-v1.md`](../operations/alert-contract-v1.md)
  defines the `alert.v1` contract, P1–P4 response policy, six alert
  definitions, Platform Operations ownership, safe suppression, escalation,
  completion evidence, traceability, and deferred qualification boundaries.
- [`scripts/verify/alert-contract.sh`](../../scripts/verify/alert-contract.sh)
  validates required metadata and runs temporary negative cases for missing
  owner, severity, dashboard/runbook links, suppression, escalation,
  completion evidence, unsafe missing-data behavior, unbounded identifiers,
  and unsupported numeric thresholds.
- The Makefile exposes the repository-native `alert-contract-check` target.
- `scripts/README.md` records the verification script in the repository
  inventory.
- No application/API, OpenAPI, database, event, worker, frontend,
  authorization, or finance-domain code was changed.

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Observability and operational foundation milestone. |
| `EP-OPS-001` / `DLV-OPS-002` | Parent epic and baseline dashboard/runbook delivery item. |
| `GFR-012` | Alert states preserve intermediate, exception, reconciliation, and terminal distinctions. |
| `ARC-OBS-002` | Operational telemetry remains separate from authoritative business metrics. |
| `ARC-PRV-001` / `NFR-SEC-010` | Sensitive values and credentials are excluded from alert contracts and evidence. |
| `ARC-CAP-001` | Capacity alerts use bounded configured policies and controlled unknown states. |
| `ARC-TST-001` | Contract and negative-case verification is layered and reproducible. |
| `NFR-OBS-002` | Critical failures have an accountable responder target. |
| `NFR-OBS-006` | Backlog and age targets are configurable by workflow and business calendar. |
| `NFR-OBS-007` | Every alert has severity, owner, response target, runbook, suppression, and escalation metadata. |
| `NFR-OBS-008` | Typed dependency, capacity, conflict, and internal outcomes remain distinct. |
| `NFR-OBS-009` | High-severity completion includes incident impact, evidence, recovery, and follow-up. |
| `NFR-OBS-012` | Alert policy changes are authorized, versioned, and auditable. |
| `NFR-MNT-007` | Critical alerts reference owned runbook coverage. |
| `NFR-TST-003` | The story maps to local verification evidence and explicit production deferrals. |
| `QG-01` / `QG-08` / `QG-10` | Traceability, operational readiness, and release evidence gates. |

## Verification results

| Command | Result | Evidence |
|---|---|---|
| `bash -n scripts/verify/alert-contract.sh` | Passed | Shell syntax is valid. |
| `make alert-contract-check` | Passed | Contract metadata, six alert rows, complete metadata, and all ten negative fixtures passed. |
| `make dashboard-contract-check` | Passed | Existing User Story 1 dashboard contract and its negative fixtures still pass. |
| `git diff --check` | Passed | No whitespace errors; Git emitted only CRLF normalization warnings for existing repository text files. |

## Deferred qualification and source boundaries

- P1–P4 semantics and response targets are contract metadata; no paging or
  production notification service is configured.
- Trigger conditions reference typed signals and approved workflow,
  business-calendar, NFR, dependency, or capacity policies. Unsupported
  numeric production thresholds are intentionally absent.
- Runbook links reference the approved `RUN-*` catalog. The reusable runbook
  template and initial platform runbook content remain User Story 3 scope.
- Future finance capabilities own business-specific metrics, alert thresholds,
  capability alerts, and runbooks.
- Authorization, accounting-scope enforcement, emergency access, and incident
  records remain owned by their approved capabilities.
