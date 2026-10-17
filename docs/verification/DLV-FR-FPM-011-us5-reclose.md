# DLV-FR-FPM-011 — US5 verification evidence

Status: implementation delivered on branch `feat/dlv-fr-fpm-011-us5-reclose`.
This record covers closing an approved scoped or operational reopen, the
authoritative no-change versus mandatory-reclose branch, reclose gate
ownership transfer, finalization, the approved route, and the FPM workspace.
It does not claim completion of EP-FPM-001, PostgreSQL/Testcontainers runtime
qualification, browser/accessibility qualification, or release checks.

## Traceability

- Epic: EP-FPM-001 — Fiscal period management
- Story: US5 — Close an approved reopen and reclose safely
- Delivery: DLV-FR-FPM-011 / FR-FPM-011
- Command: `StartReclose`
- Supporting commands: `ApplyPostingGateResult`, GL
  `CloseScopedReopenGate`, `CloseOperationalReopenGate`, and `BeginRecloseGate`
- Approved route: `fpmStartReclose`
- Supporting events: `ReopenCompletedNoChange`, `RecloseHandoffStarted`,
  `GateAdmissionSummaryRecorded`, `PeriodStateChanged`, and `PeriodReclosed`

## Implementation evidence

- FPM validates the request identity, period/request/gate versions, active or
  expired-closure state, and `StartReclose` authorization before asking GL to
  close the reopen gate.
- The shared `ApplyPostingGateResult` seam accepts the typed scoped or
  operational close result and stores the immutable frozen admission summary
  on the reopen request. A hard-closed GL result completes the request as
  `CompletedNoChange`, restores the period to `HardClosed`, and leaves the
  original close-run watermark, seal, and identity untouched.
- A positive authoritative summary cannot be selected away by the controller.
  FPM creates a fresh `Reclose` close-run identity, records prior reopen/close
  lineage, requests `BeginRecloseGate`, and requires the GL result to show
  exclusive reclose ownership before moving the request to
  `RecloseInProgress` and the period to `Closing`.
- A retry after a persisted `GateClosing` or `RecloseHandoff` state accepts only
  the current persisted version tuple or the exact version progression of the
  interrupted `StartReclose` command. GL status reconciliation prevents a
  second close or begin-reclose effect after an external handoff succeeded.
- Existing close-run task, approval, finalization, watermark, and seal
  controls apply to the reclose run. Reclose finalization completes the
  request, returns the period to `HardClosed`, retains the prior evidence, and
  emits `PeriodReclosed`; the initial-close abort/release path rejects reclose
  runs.
- GL preserves the scoped or operational reopen owner type and prior gate mode
  in the reclose result. The FPM migration permits `close_type = 'Reclose'`
  while retaining the existing active-run and status constraints.
- The HTTP adapter uses the already-generated `/api/v1/fiscal-periods/actions/
  start-reclose` route and returns the same established result shape used by
  the other FPM commands. The UI presents one closure action and explains
  that zero admissions mean no-change while any admitted correction makes
  reclose mandatory.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Apply the authoritative close-gate result to the matching reopen request | `applyReopenCloseGateResult`, typed operation/mode/owner/version validation | Implemented and focused-tested |
| Zero admissions complete as no-change without a new run | `completeReopenNoChange`, `TestStartRecloseCompletesAuthoritativeNoChangeWithoutNewCloseRun` | Implemented and focused-tested |
| Positive admission creates a new reclose identity and transfers ownership | `newRecloseCandidate`, `beginRecloseGate`, scoped and operational `StartReclose` tests, GL scoped-owner test | Implemented and focused-tested |
| Reclose is non-releasable and finalization requires existing close controls | GL `BeginRecloseGate`/release rejection, reclose finalization test | Implemented and focused-tested |
| Finalization returns HardClosed and retains prior lineage | `applyAuthoritativeFinalization`, `TestRecloseFinalizationCompletesRequestAndEmitsPeriodReclosed` | Implemented and focused-tested |
| Dependency failure, stale/changed results, duplicate, concurrency, and persistence runtime qualification | Error/recovery contracts, interruption recovery tests, replay identity/version tests, route validation, durable seam, migration and repository implementation | Focused domain/API coverage implemented; broader persistence/concurrency runtime qualification remains deferred |
| UI does not allow a less restrictive outcome | `FiscalPeriodReopenWorkspace`, component test for authoritative decision path | Implemented and focused-tested |

## Verification executed

| Command | Result |
| --- | --- |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/fiscalperiod ./internal/gl ./internal/platform/httpapi` | Pass |
| `GOCACHE=/tmp/tally-go-cache go vet ./internal/fiscalperiod ./internal/gl` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test ./...` | Blocked by restricted sandbox TCP listener tests in `internal/platform/authentication` and `internal/platform/database`; changed packages passed before those failures |
| `pnpm -C web exec vitest run src/app/fiscal-period-reopen-workspace.test.tsx --pool=forks --maxWorkers=2 --fileParallelism=false` | Pass; 1 file and 3 tests |
| `git diff --check` | Pass; Git reported only existing line-ending warnings for unrelated files |
| PostgreSQL/Testcontainers, browser/accessibility, slow/release checks | Not run; opt-in per repository instructions |

## Remaining limits

- Persistence execution against PostgreSQL/Testcontainers and full API/browser
  qualification still need an environment that permits local listener or
  container networking.
- The implementation keeps the existing generated contract and does not add
  future US6 takeover/exception behavior.
