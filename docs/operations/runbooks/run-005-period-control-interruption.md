# RUN-005 — Period-control interruption and recovery

## Runbook metadata

| Field | Value |
|---|---|
| Runbook | `RUN-005` — Period-control interruption/recovery |
| Owner | Fiscal Period Management with Platform Operations and the GL owner |
| Version | `v1` |
| Effective date | 2026-10-17 |
| Status | US6 local operational baseline |
| Audience | Authorized recovery operators and Close Coordinators |
| Data classification | Operational diagnostic data; raw financial payloads and unrestricted identifiers are excluded |
| Related screens | FPM-SCR-04 period-control recovery; FPM-SCR-05 close exception review |
| Related requirements | DLV-FR-FPM-012, DLV-FR-FPM-013, WF-7.12 |

## Prerequisites

- Confirm the accounting scope, fiscal period, interrupted process identity,
  support/correlation reference, and the bounded dependency outcome.
- Confirm that the operator is authorized for period-control recovery and that
  independent Workflow approval evidence exists for the exact candidate
  fingerprint, subject version, policy version, approver, decision, and
  separation-of-duties reference.
- Preserve the current owner, authority epoch/version, immutable cutoff, gate
  version, close-run/exception version, and audit evidence before retrying.
- Do not request or retain raw GL payloads, approval payloads, credentials,
  unrestricted screenshots, or direct database records.

## Detection

Start this runbook when a SoftCloseRun or CloseRun is interrupted, its control
authority is expired, an exception is approaching or beyond its due/expiry
window, or a recovery command returns `DEPENDENCY_UNAVAILABLE`, a stale-version
conflict, or an ambiguous authoritative gate result.

## Decision points

1. Read the authoritative GL state through `GetPostingGateStatus` for the
   selected scope and period. A timeout or unavailable response remains
   pending/restrictive; it is never treated as success.
2. If the process has no valid persisted owner/authority/cutoff metadata,
   stop. Legacy data is reconciliation-needed and is not eligible for a
   synthesized takeover.
3. If the authority is active, the process/gate/epoch/scope does not match, or
   the expected version is stale, stop and refresh the evidence. Do not replace
   the process identity.
4. Submit `TakeOverPeriodControl` only through FPM-SCR-04 with the same command
   identity on retry. The operation changes FPM ownership only; it does not
   open, release, finalize, or weaken a GL gate.
5. Submit `ExtendCloseException` only for the current exception and close-run
   version through FPM-SCR-05. The new due time and expiry must both be future
   values, with due no later than expiry, and the extension remains separately
   approved.
6. If the exception is rejected, finalized, completed, or the close run is
   finalizing, do not extend it. Resolve or escalate through the owning
   workflow.

## Safe commands

The following commands verify local contracts. They do not authorize manual
production recovery or direct data repair:

```bash
GOCACHE=/tmp/tally-go-cache go test ./internal/fiscalperiod ./internal/platform/httpapi ./cmd/api
make db-migrate-validate
make db-migrate-check
```

No direct SQL update, GL gate mutation, owner synthesis, approval creation,
approval replacement, or manual event publication is approved by this runbook.

## Evidence to preserve

- UTC observation time, environment, branch/build identifier, correlation or
  sanitized support reference, bounded outcome code, and dependency state.
- Fiscal period and process type, current owner, authority epoch/version,
  cutoff/expiry, expected and authoritative gate versions, and pending action.
- Workflow request/decision references, candidate fingerprint, subject/policy
  versions, approver and SoD evidence references, without copying approval
  payloads.
- For exception extension: exception code, owner, impact classification, due
  and expiry windows, blocker age, justification, and append-only extension
  history reference.
- Established result, audit reference, and whether the result was replayed.

## Escalation

Escalate dependency-unavailable or ambiguous GL status to Platform Operations
and the GL owner. Escalate invalid approval, owner, cutoff, or exception
evidence to the Close Coordinator and Workflow owner. Escalate integrity
uncertainty, duplicate effects, or unexplained lineage differences to the
Incident Commander and audit owner.

## Recovery checks

- The authoritative GL status matches the recorded scope, period, process,
  gate mode, gate version, prior owner lineage, and control epoch.
- A successful takeover preserves the existing process identity and appends one
  ownership change with authorization, approval, reason, and evidence links.
- A successful extension preserves the exception's unresolved or approved
  state according to its original decision and appends one immutable extension
  record. It does not establish final approval.
- Duplicate identical commands replay one established result; changed content,
  stale versions, wrong process, and concurrent losers remain typed conflicts.
- User-facing state remains restrictive and identifies the next safe action
  until reconciliation is established.

## Closure criteria

Close only when the established result, audit evidence, owner/exception
lineage, authoritative gate status, and sanitized support reference are
preserved; the next owner is acknowledged; and no unresolved dependency or
integrity-uncertain state is being reported as success.

## Deferred qualification

This runbook is a local operational contract. PostgreSQL/Testcontainers,
browser/accessibility, slow/release, hosted CI, production GL, Workflow, and
disaster-recovery qualification remain separate evidence activities.

## Safety rules

Financial facts and established process history are immutable. Correct through
the approved FPM command and Workflow evidence paths. Never edit FPM or GL
tables directly, weaken a gate to clear a blocker, or infer success from a
timeout. Telemetry and support references are diagnostic and are not
authoritative financial or audit evidence.
