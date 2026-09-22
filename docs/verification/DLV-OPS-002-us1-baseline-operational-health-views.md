# DLV-OPS-002 User Story 1 — Baseline Operational Health Views

## Scope

This evidence covers `EP-OPS-001` / `DLV-OPS-002` User Story 1. It verifies
the provider-neutral dashboard contract for API availability and latency,
PostgreSQL health, outbox/inbox pending work and age, typed error classes,
capacity, and operational exceptions.

This evidence does not claim a dashboard UI, live Azure monitoring, production
alerting, `/health/ready` implementation, finance-domain metrics, or completion
of `DLV-OPS-002` as a whole.

## Source state

| Field | Value |
|---|---|
| Branch | `codex/dlv-ops-002-us1-baseline-operational-health-views` |
| Baseline commit | `5fcd5d6e30b624f9149928d9a6dee724c7b03fc4` (`main`) |
| Implementation state | Working-tree changes; no commit, staging, or push performed |
| Verification profile | Local synthetic contract copies only |
| Remote dependencies | None; no Azure credentials, production data, or dashboard provider required |

## Implementation

- [`docs/operations/dashboard-contract-v1.md`](../operations/dashboard-contract-v1.md)
  defines the `dashboard.v1` contract, Platform Operations ownership, NFR
  freshness targets, six baseline panels, bounded dimensions, typed outcome
  categories, and safe missing/stale-data behavior.
- [`scripts/verify/dashboard-contract.sh`](../../scripts/verify/dashboard-contract.sh)
  validates the contract and runs temporary negative cases for missing panel
  ownership, unbounded dimensions, collapsed error classes, and unsafe
  missing-data behavior.
- The Makefile exposes the repository-native `dashboard-contract-check` target.
- No application/API, OpenAPI, database, event, frontend, authorization, or
  finance-domain code was changed.

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Engineering foundation milestone. |
| `EP-OPS-001` / `DLV-OPS-002` | Parent epic and baseline dashboard/runbook delivery item. |
| `GFR-012` | Operational views preserve intermediate, exception, reconciliation, and terminal distinctions. |
| `ARC-OBS-002` | Operational telemetry remains separate from authoritative business metrics. |
| `ARC-CAP-001` | Capacity is represented with measured, bounded signals and controlled unknown states. |
| `ARC-PRV-001` / `NFR-SEC-010` | Sensitive values and credentials are excluded from diagnostics. |
| `NFR-OBS-001` | Current health views and freshness targets. |
| `NFR-OBS-005` | Latency and operational measurement vocabulary. |
| `NFR-OBS-006` | Pending-work age and oldest-item visibility. |
| `NFR-OBS-008` | Typed operational outcomes are not collapsed into one error rate. |
| `NFR-MNT-001` / `NFR-TST-003` | Source-to-evidence traceability and verification record. |
| `QG-01` / `QG-08` / `QG-10` | Traceability, observability readiness, and release evidence gates. |

## Verification results

| Command | Result | Evidence |
|---|---|---|
| `bash -n scripts/verify/dashboard-contract.sh` | Passed | Validator syntax is valid. |
| `make dashboard-contract-check` | Passed | Six panels, source metrics, bounded dimensions, freshness/ownership metadata, typed outcomes, safe missing-data rules, and all four negative cases passed. |
| `git diff --check` | Passed | No whitespace errors; Git emitted only existing CRLF normalization warnings for repository text files. |

## Deferred qualification and source boundaries

- `/health/live` is the current availability source. `/health/ready` remains
  an approved future technical source and was documented without adding the
  route.
- Remote exporter availability, Azure Monitor delivery, provider-specific
  dashboards, production thresholds, alert exercises, and retention
  qualification remain deferred.
- Business-result panels for posting, payment, reconciliation, approval, close,
  and audit integrity remain owned by future capability deliveries.
- The repository-level `EP-OPS-001` status and `ROADMAP.md` entries remain
  open; this evidence closes only User Story 1's contract definition criteria.
