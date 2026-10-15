# DLV-FR-WFA-004 — User Story 5 verification evidence

Status: local implementation slice verified on 2026-10-15 from branch
`feat/dlv-fr-wfa-004-us5-escalate-approval`. No commit, staging, push, or
destructive worktree operation was performed.

This record covers `EP-WFA-001` User Story 5 — Escalate overdue or blocked
approvals. It records focused implementation evidence, not release completion.

## Traceability

- Delivery: `DLV-FR-WFA-004`
- Requirement: `FR-WFA-004`
- Supporting requirements: `GFR-004`, `GFR-009`, `GFR-012`, `NFR-CAP-007`,
  `NFR-PERF-013`, `NFR-OBS-006`–`NFR-OBS-008`, `NFR-AUD-001`–`NFR-AUD-003`
- UX: `WFA-WS-01`, `WFA-SCR-04`
- Owning bounded context: Workflow & Approvals — `internal/workflow`, schema
  `workflow`
- Approved event: `ApprovalEscalated` / `approval_escalated`
- Approved permission: `finance.wfa.escalate.approval`
- Approved worker contract: `approval-escalation`, one-minute cadence, batch
  size 100, due/expired approval-step selection

The delivery sources contain a milestone discrepancy: the WFA epic/backlog
maps `DLV-FR-WFA-004` to M3, while the delivery-plan row for `DLV-GFR-004`
lists M2. This implementation preserves both source values and does not
resolve that planning conflict.

## Implemented surfaces

- `internal/workflow/approval_request.go` keeps escalation inside the existing
  `ApprovalRequest` aggregate, accepts `Escalated` as an actionable step state,
  retains immutable escalation evidence, and returns the established audit
  reference.
- `internal/workflow/approval_escalation.go` validates policy/version/subject
  evidence, overdue or explicit blocked eligibility, actor/process
  authorization, expected version, idempotency, and policy-derived target
  roles. It appends evidence without a decision or subject mutation.
- `internal/workflow/approval_escalation_worker.go` provides the approved
  process path with exact process identity, bounded candidate selection,
  deterministic command identities, and safe race handling.
- `internal/workflow/postgres_approval_request_repository.go` writes the
  escalation child after the transaction-bound audit reference is established,
  before the outbox event and durable idempotency finalization.
- `db/migrations/workflow/00005_create_approval_request_escalation_schema.sql`
  defines the immutable child table, policy-version link, unique request/step
  identity, actor/process evidence checks, target-role checks, audit-reference
  check, and indexes. SQLC output includes the new model.
- `contracts/openapi/paths/approvals.yaml` and
  `contracts/openapi/components/common.yaml` define the typed command,
  escalation projection, request state projections, and established result.
  Generated Go/TypeScript artifacts were regenerated from those contracts.
- `internal/platform/httpapi/workflow_approval_request_handler.go` maps the
  typed route, actor subject, escalation evidence, projection fields, and
  typed failure outcomes.
- `cmd/api/workflow_runtime.go` maps escalation audit lineage and a minimized
  event payload. Actor escalation uses the IAM evaluator and step eligibility;
  the approved worker process is accepted only under the exact process
  reference.
- `web/src/app/workflow-approval-workspace.tsx` exposes escalated/due/blocked
  worklist and detail evidence. `web/src/app/workflow-delegation-workspace.tsx`
  adds the approved WFA-SCR-04 escalation command form and replay-safe result.

## Focused test evidence

The following passed:

```text
GOCACHE=/tmp/tally-go-cache-us5 go test ./internal/workflow
GOCACHE=/tmp/tally-go-cache-us5 go test ./internal/platform/httpapi
GOCACHE=/tmp/tally-go-cache-us5 go test ./cmd/api
GOCACHE=/tmp/tally-go-cache-us5 go test ./cmd/worker
pnpm --dir web exec tsc --noEmit
make db-migrate-validate
make db-migrate-check
make sqlc-check
GOCACHE=/tmp/tally-go-cache-us5 make api-generate-check
make api-ts-check
git diff --check
```

`api-generate-check` emitted the repository's existing ogen warnings about
missing default responses and problem-status enforcement, then passed its
OpenAPI generation and Go test stages. Its listener-based tests required the
approved elevated execution because the restricted sandbox denies local TCP
listeners. SQLC regeneration required the repository-pinned `v1.31.1` tool;
the generated workflow model is included in the working tree.

The focused domain/API tests cover:

- eligible overdue actor escalation and explicit blocked escalation;
- disabled, early, missing-blocker, stale-subject, and authorization-denied
  paths without audit/event/request mutation;
- immutable policy/subject/owner/target/timing evidence and no decision side
  effect;
- same-fingerprint replay and changed-fingerprint idempotency conflict;
- typed API establishment/replay with pending root status and actor evidence;
- blocked worklist state projection; and
- deterministic one-minute worker identity and candidate processing.

## Deferred qualification

The following remain open and are intentionally not represented as passed
acceptance evidence:

- PostgreSQL/Testcontainers migration, transaction rollback, immutable-child,
  outbox/inbox duplicate/out-of-order, and durable idempotency execution;
- production `cmd/worker` composition, because the current worker executable
  has no approved audit-writer/IAM process dependency injection point;
- notification delivery, oldest-item/age metrics, dashboards, bounded
  telemetry labels, alerting, and operational runbook qualification;
- live IAM/Entra and production segregation qualification;
- browser-level accessibility, performance/capacity, screen-reader, zoom,
  and notification timing evidence; and
- release gates such as `make persistence-check`,
  `make persistence-integration-test`, `make api-check`, `pnpm test:slow`, and
  `make check-slow`, which were not run under the repository's iterative-work
  guidance.

