# DLV-FR-FPM-009 / DLV-FR-FPM-010 — US4 verification evidence

Status: implementation delivered on branch `feat/dlv-fr-fpm-009-010-us4-reopen`.
This record covers the FPM-owned reopen request lifecycle, Workflow decision
validation boundary, scoped/operational GL gate opening, expiry reconciliation,
approved routes, and the focused FPM-SCR-03 workspace. It does not claim
completion of EP-FPM-001, US5 closure/reclose, PostgreSQL/Testcontainers
runtime qualification, browser/accessibility qualification, or release checks.

## Traceability

- Epic: EP-FPM-001 — Fiscal period management
- Story: US4 — Request and authorize a scoped or operational reopen
- Delivery: DLV-FR-FPM-009 / FR-FPM-009; DLV-FR-FPM-010 / FR-FPM-010
- Commands: `RequestReopen`, `ApplyReopenApprovalDecision`
- Approved routes: `fpmRequestReopen`, `fpmApplyReopenApprovalDecision`
- Screens: `FPM-SCR-03`, with authoritative gate evidence remaining GL-owned and visible through the existing FPM result shape
- Supporting boundaries: Workflow immutable decision query/validation, GL `OpenScopedReopenGate`, GL `OpenOperationalReopenGate`, and internal `GetPostingGateStatus`

## Implementation evidence

- `internal/fiscalperiod/reopen.go` records a canonical request fingerprint,
  reason, correction/impact package, scope, maximum expiry, expected period and
  gate versions, authority epoch, requester, close lineage, correlation, and
  causation. A request is accepted only from a completed `HardClosed` period
  with an authoritative hard-closed GL gate.
- `RequestReopen` commits `PendingApproval` while leaving the period and GL gate
  unchanged. The request path never calls either GL reopen-opening method.
- `ApplyReopenApprovalDecision` validates immutable Workflow evidence through
  the composition-root `ReopenDecisionPort`, rejects self-approval and missing
  SoD evidence, rechecks current request/period/gate versions, and persists
  approval or rejection evidence. Rejection leaves `HardClosed` and does not
  open a gate.
- Approved scoped requests call only GL's scoped-open port; approved
  operational requests call only GL's operational-open port. FPM validates the
  authoritative returned mode, owner, scope identity, expiry, control epoch,
  and incremented gate version before changing the period to `Reopening` and
  the request to `Active`.
- `ReconcileReopenExpiry` is an internal, non-HTTP reconciliation seam. It
  checks current authoritative GL ownership and the service clock, records
  `ExpiredPendingClosure`, emits the approved operational expiry event for
  operational requests, retains the period in `Reopening`, and does not close
  or release GL. US5 owns the later no-change closure or mandatory reclose.
- PostgreSQL persistence stores the request as an FPM-owned row with immutable
  correction/impact and approval/gate evidence JSON, close lineage, versions,
  expiry, and active-owner/fingerprint indexes. FPM writes audit evidence and
  events through the existing transactional repository boundary; it does not
  access GL or Workflow tables.
- The HTTP adapter accepts the already-approved generic command envelope and
  returns typed authorization, conflict, rejection, dependency, and
  established-result responses. Pending results expose a null approval-request
  reference until Workflow supplies the immutable decision reference.
- `web/src/app/fiscal-period-reopen-workspace.tsx` exposes the request package,
  decision evidence, period/request/gate status, authorized scope, expiry, and
  restrictive recovery guidance. It does not present a request or rejected
  decision as posting authority.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Eligible HardClosed request records bounded mode, scope, correction/impact evidence, expiry, versions, epoch, requester, and correlation | `RequestReopen`, `ReopenRequest`, fingerprinting, repository row, request test | Implemented and focused-tested |
| Pending request leaves period and GL gate restrictive | request path and `TestRequestReopenCreatesPendingApprovalWithoutOpeningGate` | Implemented and focused-tested |
| Current immutable Workflow decision and SoD are required; rejection has no gate effect | composition adapter, decision validation, rejection/self-approval tests | Implemented and focused-tested |
| Scoped and operational approvals use their corresponding GL-owned gate and validate the authoritative result | `openReopenGate`, `validateReopenGateResult`, scoped/operational tests | Implemented and focused-tested |
| Successful opening records Active/Reopening state, gate version, expiry, owner, and scope | active transition, result projection, component test | Implemented and focused-tested |
| Expiry records `ExpiredPendingClosure` while retaining restrictive ownership | `ReconcileReopenExpiry`, authoritative status check, expiry test | Implemented and focused-tested; scheduler/event-consumer wiring remains a later operational integration concern |
| Same-fingerprint retry is harmless; changed evidence, stale versions, wrong scope, unauthorized actor, and self-approval are rejected | command fingerprints, optimistic versions, Workflow/SoD checks, domain/API tests | Implemented and focused-tested |
| Request and approval evidence is auditable and linked to the original close | FPM audit record, close-run lineage, PostgreSQL row, outbox mutation | Implemented in code; persistence runtime qualification remains deferred |

## Verification executed

| Command | Result |
| --- | --- |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/fiscalperiod ./internal/platform/httpapi ./cmd/api ./internal/gl ./internal/workflow` | Pass |
| `./scripts/db/migrate.sh validate` | Pass; all migration directories validated |
| `./scripts/db/migrate.sh check` | Pass; migration checksum inventory valid |
| `pnpm exec vitest run --pool=forks --maxWorkers=2 --fileParallelism=false src/app/fiscal-period-reopen-workspace.test.tsx src/routes/route-registry.test.ts src/routes/router.test.tsx` from `web` | Pass; 3 files and 23 tests |
| `pnpm run build` from `web` | Pass; TypeScript build and Vite production build |
| `git diff --check` | Pass; only repository line-ending warnings were reported by Git |
| `GOCACHE=/tmp/tally-go-cache-fast make check-fast` | Migration checks passed; full Go target was blocked by unrelated sandbox TCP-listener restrictions in authentication/database tests |

## Qualification limits and deferred scope

- PostgreSQL/Testcontainers persistence checks, `make api-check`, slow/release
  checks, and browser/accessibility qualification were not run because the
  repository instructions make them explicit opt-in checks.
- The internal expiry seam is intentionally not a new public route. GL remains
  authoritative for posting rejection after expiry; US5 owns gate closure,
  no-change completion, reclose handoff, and admission-summary application.
- The full fast Go suite also includes existing tests that create local HTTP or
  database listeners. The restricted sandbox rejected those binds; all changed
  FPM/API packages and the unrelated GL/Workflow packages passed separately.
- US1, US5, US6, remaining FPM acceptance criteria, and EP-FPM-001 are not
  marked complete by this evidence.

## Source references

- `docs/backlog/stories/EP-FPM-001_user_stories.md` — US4 acceptance and traceability
- `docs/specs/finance_domain_model_ddd.md` — Fiscal Period Reopen and Reclose, §6.2
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` — FR-FPM-009 and FR-FPM-010
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — WF-6.2 and FPM-SCR-03
- `contracts/openapi/paths/fiscal-periods.yaml` — approved FPM action routes and generic command/result contracts
