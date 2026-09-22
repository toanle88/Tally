# DLV-OPS-002 User Story 4 — Operational readiness evidence

## Scope and evidence boundary

This evidence covers `EP-OPS-001` / `DLV-OPS-002` User Story 4. It proves
that the local dashboard, alert, and runbook contracts can be reviewed together
with synthetic data and that invalid or unsafe missing-data behavior fails
safely.

This evidence does not claim a dashboard UI, live Azure Monitor, a paging
provider, production thresholds, production credentials or data, retention
qualification, quarterly operational exercises, or future finance-domain
runbooks.

## Source state

| Field | Value |
|---|---|
| Branch | `codex/dlv-ops-002-us4-operational-readiness-evidence` |
| Baseline commit | `79c2a2a2e38f08274a13ca83e1cc4806a8a964b1` (`main`) |
| Implementation state | Working-tree changes; no commit, staging, or push performed |
| Verification profile | Local provider-neutral contract checks with disposable synthetic fixtures |
| Remote dependencies | None; no Azure credentials, production data, database, telemetry service, or paging provider required |

## Implementation

- [`scripts/verify/operational-readiness.sh`](../../scripts/verify/operational-readiness.sh)
  runs the dashboard, alert, and runbook contract gates, validates their
  references and missing-data rules, and keeps child failure output out of the
  operator-facing result.
- The gate creates disposable dashboard, alert, and runbook-template fixtures
  with unsafe missing-data substitutions and expects each established contract
  validator to reject them.
- A failure-propagation fixture adds a synthetic raw-payload marker to an
  invalid dashboard copy. The child validator returns non-zero, the aggregate
  result remains non-zero, and the marker is absent from the captured safe
  failure output.
- `make operational-readiness-check` is the repository-native entry point.
- The existing runbook contract validator now explicitly protects the approved
  missing/unavailable-data vocabulary and rejects a runbook template that loses
  those rules.
- No application/API, database, event, worker, authorization, frontend,
  finance-domain, infrastructure, or external-monitoring behavior changed.

## Acceptance evidence

| Acceptance criterion | Evidence |
|---|---|
| A local fixture proves dashboard/alert/runbook contracts, missing-data behavior, and failure propagation without Azure or production credentials. | Positive child gates, cross-contract reference validation, missing-data policy validation, three rejected synthetic fixtures, and the sanitized failure-propagation check in `make operational-readiness-check`. |
| Evidence identifies commit, commands, tool versions, result, safe failure location, deferred qualification scope, and no raw sensitive values. | This record's source-state, toolchain, verification, failure-safety, and deferred-qualification sections. |

## Verification commands and results

| Command | Result | Evidence |
|---|---|---|
| `bash -n scripts/verify/operational-readiness.sh scripts/verify/runbook-contract.sh` | Passed | Both verification scripts parse successfully. |
| `make operational-readiness-check` | Passed | Dashboard, alert, runbook, cross-contract, missing-data, synthetic rejection, and failure-propagation checks passed. |
| `make dashboard-contract-check` | Passed | Existing dashboard contract and negative fixtures passed. |
| `make alert-contract-check` | Passed | Existing alert contract and negative fixtures passed. |
| `make runbook-contract-check` | Passed | Template, five runbooks, safety rules, identifiers, links, and missing-data negative fixture passed. |
| `make repository-integrity-check` | Passed | Repository path and workflow safety checks passed. |
| `git diff --check` | Passed | No whitespace errors. |

## Toolchain

| Tool | Version or source |
|---|---|
| Bash | GNU bash 5.2.21 |
| Make | GNU Make 4.3 |
| ripgrep | 15.2.0 |
| Git | 2.43.0 |
| Go | Repository target `1.26.3`; not required by this provider-neutral gate and unavailable in the current WSL shell |
| Node.js / pnpm | Node.js is not required by this gate; repository package manager is pnpm `11.9.0` |

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Engineering foundation milestone. |
| `EP-OPS-001` / `DLV-OPS-002` | Parent epic and baseline dashboard/runbook delivery item. |
| `GFR-012` | Cross-context operational states remain distinct and do not imply fabricated success. |
| `ARC-OBS-002` | Operational diagnostics remain separate from authoritative business evidence. |
| `ARC-PRV-001` / `NFR-SEC-010` | Synthetic verification excludes sensitive values and credentials. |
| `ARC-TST-001` | Contract, negative, failure, and evidence checks are layered and reproducible. |
| `NFR-OBS-001`–`NFR-OBS-012` | Health views, freshness, safe diagnostics, typed outcomes, alert ownership, and operational evidence boundaries. |
| `NFR-MNT-001`, `NFR-MNT-007`, `NFR-MNT-010` | Traceability, runbook coverage, and no unresolved critical operational defect. |
| `NFR-TST-003`, `NFR-TST-008`, `NFR-TST-009` | Verification method, recovery/runbook boundaries, and release-evidence readiness. |
| `QG-01` / `QG-08` / `QG-10` | Source integrity, operational readiness, and release evidence gates. |

## Failure safety and data boundaries

- Child command output is redirected to a disposable temporary directory and is
  never copied into the evidence record or emitted by the aggregate failure
  path.
- The aggregate reports fixed check identifiers and whether an expected
  synthetic fixture was rejected; it does not print fixture contents.
- The fixtures contain no production data, raw telemetry, credentials, secrets,
  connection strings, financial values, unrestricted identifiers, or external
  provider payloads.
- The gate performs no state-changing operation, retry loop, database access,
  network access, telemetry export, or remote monitoring call.
- A failing child contract remains a failing aggregate result; an invalid
  contract cannot be converted into a passing readiness result.

## Deferred qualification and source boundaries

- Azure Monitor export, live dashboards, paging, and production alert routing
  remain deferred.
- Production thresholds, SLO/error-budget reporting, retention, and quarterly
  alert/runbook exercises remain deferred.
- Dashboard authorization, accounting-scope enforcement, emergency access, and
  privileged operational actions remain owned by `EP-IAM-001`.
- Payment, period-close, filing, audit-integrity, and other finance-domain
  runbooks remain owned by their future capability deliveries.
- Local contract evidence is not production availability, capacity, recovery,
  financial-integrity, or release qualification evidence.
