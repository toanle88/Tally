# DLV-FR-FPM-004 / DLV-FR-FPM-005 / DLV-FR-FPM-007 / DLV-FR-FPM-008 — US3 verification evidence

Status: implementation delivered on branch `feat/dlv-fr-fpm-001-us3-close-run`.
This record covers hard-close task execution, Workflow decision application,
authoritative GL finalization/release recovery, seal state, the approved FPM
routes, and the focused close-run workspace. It does not claim completion of
EP-FPM-001, reclose/reopen stories, PostgreSQL/Testcontainers runtime
qualification, browser/accessibility qualification, or release checks.

## Traceability

- Epic: EP-FPM-001 — Fiscal period management
- Story: US3 — Execute, approve, resume, or abort a hard-close run
- Delivery: DLV-FR-FPM-004 / FR-FPM-004; DLV-FR-FPM-005 / FR-FPM-005; DLV-FR-FPM-007 / FR-FPM-007; DLV-FR-FPM-008 / FR-FPM-008
- Commands: `ResumeCloseRun`, `AbortCloseRun`, `ApplyCloseExceptionApprovalDecision`, `ApplyCloseApprovalDecision`
- Supporting boundaries: FPM close-task, Workflow decision, Audit seal, GL `FinalizePostingGate`, GL `ReleasePostingBarrier`, and internal `GetPostingGateStatus`
- Screens: `FPM-SCR-02` and `FPM-SCR-05`

## Implementation evidence

- `internal/fiscalperiod/close_run.go` retains one FPM-owned close-run identity, records mandatory task progress and immutable evidence, persists resumable task failures, validates current Workflow decision evidence and SoD, finalizes only through the GL-owned gate, and keeps audit sealing separate from accounting finalization.
- Barrier recovery reuses the persisted `BarrierCommandID` and `BarrierIdempotencyKey`. Finalization recovery accepts only an authoritative HardClosed gate for the same run and watermark. A finalized gate is never released.
- Initial abort checks GL ownership and zero admitted close postings. It records `Aborting` before the GL release, reconciles an acknowledged release without issuing a second release, and opens the next soft-close epoch only after the authoritative released gate is valid. Reclose close-runs are not releasable.
- Exception and final approval evidence is retained in `ApprovalDecisions`; changed duplicate decisions, stale subject versions, wrong run/request references, expired exceptions, and invalid SoD evidence are rejected.
- Seal requests retain a stable request identity. Seal failure leaves the period `HardClosed` and the close run `Completed` with `SealFailed`; a retry does not reopen accounting or replace established finalization evidence.
- `internal/fiscalperiod/postgres_repository.go` persists close-run details as FPM-owned JSON, keeps aborted run rows immutable when a later initial close creates a new process identity, selects the active/retryable run for recovery, and retains audit/outbox transactional boundaries.
- `internal/platform/httpapi/fiscal_period_handler.go` implements the approved four action routes and exposes close steps, exceptions, approval evidence, final/release gate evidence, final watermark, seal state, and failure state in the established result.
- `web/src/app/fiscal-period-hard-close-workspace.tsx` distinguishes task failure, unresolved/expired exception, invalidated approval, dependency unavailability, finalization pending, and seal pending/failed states while keeping the period restrictive.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Resume preserves the existing process identity and authoritative owner | barrier retry, task retry, version/owner checks, and focused domain tests | Implemented and focused-tested |
| Mandatory tasks retain progress, owner, blocker, attempts, and immutable evidence | `CloseStep`, `CloseTaskPort`, progress application, retryable failure state, and task tests | Implemented and focused-tested |
| Exception decisions are current, scoped, immutable, and Workflow-validated | `CloseException`, `CloseApprovalDecision`, policy/subject/request checks, duplicate-content conflict test | Implemented and focused-tested |
| Final approval cannot bypass tasks, exceptions, watermark, gate, authorization, or SoD | `validateFinalApprovalReadiness`, current GL gate check, final approval test | Implemented and focused-tested |
| GL finalization precedes FPM completion and retains final watermark | `FinalizePostingGate`, authoritative-result validation, finalization test | Implemented and focused-tested |
| Initial abort releases only an unadmitted barrier and recovers after release acknowledgement loss | release checks, `Aborting` state, authoritative release reconciliation, abort tests | Implemented and focused-tested |
| Seal failure does not reopen accounting and retries retain one request identity | `requestAuditSeal`, immutable finalization facts, seal tests | Implemented and focused-tested |
| API and UI expose recovery state and approved actions | handler, route, component, and TypeScript checks | Implemented and focused-tested |

## Verification executed

| Command | Result |
| --- | --- |
| `GOCACHE=/tmp/tally-go-cache-dlv-fr-fpm-001-us3 go test ./internal/fiscalperiod` | Pass; US3 domain and recovery tests |
| `GOCACHE=/tmp/tally-go-cache-dlv-fr-fpm-001-us3 go test ./internal/fiscalperiod ./internal/platform/httpapi ./cmd/api ./internal/gl ./internal/workflow` | Pass |
| `bash -n scripts/db/migrate.sh scripts/db/verify.sh` | Pass |
| `./scripts/db/migrate.sh validate` | Pass; all migration directories validated |
| `./scripts/db/migrate.sh check` | Pass; migration checksum inventory valid |
| `./node_modules/.bin/vitest run --project=dom src/app/fiscal-period-hard-close-workspace.test.tsx src/routes/router.test.tsx --pool=threads --maxWorkers=1 --fileParallelism=false --reporter=dot` from `web` | Pass; 2 files and 18 tests |
| `./node_modules/.bin/vitest run --project=unit src/routes/route-registry.test.ts --pool=threads --maxWorkers=1 --fileParallelism=false --reporter=dot` from `web` | Pass; 1 file and 4 tests |
| `./node_modules/.bin/tsc --noEmit` from `web` | Pass |
| `GOCACHE=/tmp/tally-go-cache-dlv-fr-fpm-001-us3 go test ./...` | Blocked by the sandbox: unrelated authentication/database tests could not bind local TCP listeners (`operation not permitted`); the changed fiscal-period/API packages passed separately |

## Qualification limits and deferred scope

- PostgreSQL/Testcontainers persistence checks, `make api-check`, slow/release
  checks, and browser/accessibility qualification were not run because the
  repository instructions make them explicit opt-in checks.
- The `pnpm -C web exec ...` wrapper attempted during verification could not
  fetch pnpm registry metadata in the restricted environment; the checked-in
  local Vitest and TypeScript binaries ran successfully instead. No dependency
  or lockfile was changed.
- `GetPostingGateStatus` remains an internal GL recovery seam. Public GL
  status API work and the US4/US5 reopen and reclose lifecycle remain outside
  this story.

## Source references

- `docs/backlog/stories/EP-FPM-001_user_stories.md` — US3 acceptance and traceability
- `docs/specs/finance_domain_model_ddd.md` — close-run lifecycle, GL finalization, release restrictions, and seal recovery
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` — FR-FPM-004, FR-FPM-005, FR-FPM-007, and FR-FPM-008
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — WF-6.1, FPM-SCR-02, and FPM-SCR-05
- `contracts/openapi/paths/fiscal-periods.yaml` — approved FPM action routes and generic result/problem contracts
