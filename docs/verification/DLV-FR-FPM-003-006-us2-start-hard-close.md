# DLV-FR-FPM-003 / DLV-FR-FPM-006 — US2 verification evidence

Status: implementation delivered on branch `feat/dlv-fr-fpm-001-us2-start-hard-close`.
This record covers the FPM hard-close candidate, authoritative GL posting-barrier
handoff, approved HTTP commands, FPM persistence boundary, and the focused UI
workflow. It does not claim completion of EP-FPM-001, the remaining FPM stories,
PostgreSQL/Testcontainers runtime qualification, or browser/accessibility
qualification.

## Traceability

- Epic: EP-FPM-001 — Fiscal period management
- Story: US2 — Start hard close and acquire the authoritative posting barrier
- Delivery: DLV-FR-FPM-003 / FR-FPM-003; DLV-FR-FPM-006 / FR-FPM-006
- Commands: `StartHardClose`, `ApplyPostingGateResult`
- Approved routes: `fpmStartHardClose`, `fpmApplyPostingGateResult`
- Screens: `FPM-WS-01`, `FPM-SCR-02`, with GL evidence remaining visible in `GL-SCR-03`
- GL dependency: `AcquirePostingBarrier` and the internal `GetPostingGateStatus` recovery seam

## Implementation evidence

- `internal/fiscalperiod/hard_close.go` owns the FPM period, soft-close run,
  control epoch, and close-run transition. It creates a candidate close run in
  `Initiating` without granting it posting ownership, requires the expected
  scope/run/epoch/policy/version, and applies only an authoritative GL
  `CloseOnly` result for the matching close run.
- The successful handoff records frozen admission summary, barrier ledger
  position, GL decision/policy/audit references, and the FPM application
  fingerprint. The period changes to `Closing`, the soft-close run and epoch
  become `HandoffPending`, and the close run becomes `BarrierAcquired`.
- GL business rejections are mapped separately from dependency unavailability;
  a failed acquisition leaves the period `SoftClosed` and the existing
  soft-close owner restrictive.
- `internal/fiscalperiod/postgres_repository.go` locks the FPM period/control
  rows, checks aggregate versions, writes audit evidence through the existing
  audit owner, and publishes FPM events through the transactional outbox. It
  does not read or write GL tables.
- `internal/platform/httpapi/fiscal_period_handler.go` maps the approved
  generic command envelope to typed FPM commands and returns established
  results or typed authorization, conflict, rejection, and dependency
  problems.
- `web/src/app/fiscal-period-hard-close-workspace.tsx` exposes the restrictive
  pending state, authoritative `CloseOnly` state, frozen summary, owner, gate
  version, barrier ledger position, and dependency/business-rejection guidance.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Eligible `SoftClosed` period and expected soft-close scope, epoch, policy, period version, and gate version are required | FPM precondition validation; domain tests; HTTP request mapping | Implemented and focused-tested |
| Candidate close run is non-owning until GL confirms the barrier | `CloseRunInitiating`; GL application port; candidate/failed-state tests | Implemented and focused-tested |
| Matching GL result transfers ownership to `CloseOnly`, freezes the epoch, and records barrier evidence | `validatePostingGateResult`; `BarrierEvidence`; three FPM events; successful handoff test | Implemented and focused-tested |
| Duplicate result is harmless; changed result, scope, process, and version are rejected without partial mutation | idempotency/evidence fingerprint checks and conflict tests | Implemented and focused-tested |
| GL failure keeps the period restrictive and distinguishes dependency outage from business rejection | error classification, failed close-run state, domain/UI tests | Implemented and focused-tested |
| API and authorization boundaries are exposed through approved routes | FPM handler and handler tests | Implemented and focused-tested |
| Hard-close UI exposes barrier state and recovery guidance | FPM workspace, route tests, component tests | Implemented; frontend runtime test execution remains an environment qualification item |
| PostgreSQL row locking, audit, and transactional outbox behavior | FPM PostgreSQL repository and migration set | Implemented; Docker/Testcontainers runtime remains opt-in and not claimed here |

## Verification executed

| Command | Result |
| --- | --- |
| `bash -n scripts/db/migrate.sh scripts/db/verify.sh` | Pass |
| `diff --unified db/migrations/checksums.sha256 /tmp/tally-migration-checksums` | Pass |
| `make check-fast` | Pass; migration validation/checksum verification and all Go tests |
| `go test ./internal/fiscalperiod ./internal/platform/httpapi ./cmd/api` | Pass |
| `./node_modules/.bin/tsc --noEmit` from `web` | Pass |
| `./node_modules/.bin/vitest run --project=dom src/app/fiscal-period-hard-close-workspace.test.tsx src/routes/router.test.tsx --pool=threads --maxWorkers=1 --fileParallelism=false --reporter=dot` from `web` | Pass; 2 files and 17 tests |
| `./node_modules/.bin/vitest run --project=unit src/routes/route-registry.test.ts --pool=threads --maxWorkers=1 --fileParallelism=false --reporter=verbose` from `web` | Pass; 1 file and 4 tests |

## Qualification limits and deferred scope

- The repository instructions make PostgreSQL/Testcontainers persistence checks,
  `make api-check`, and slow/release checks opt-in; they were not run.
- Browser keyboard/focus/reflow and assistive-technology qualification were not
  run. The focused component and route tests pass, but the full web suite and
  browser qualification remain outside this verification record.
- `GetPostingGateStatus` remains an internal GL recovery seam; the public GL
  status operation is later GL scope. A retry first reconciles a candidate
  against the authoritative GL gate status; when that status is not the matching
  hard-close owner, the retry calls the GL barrier command with the original
  command identity so GL can return its retained durable outcome.
- US1 soft-close creation/end, US3 close execution/finalization, US4 reopen,
  US5 reclose, and US6 period-control takeover/exception extension remain
  planned. EP-FPM-001 and the roadmap are not marked complete.

## Source references

- `docs/backlog/stories/EP-FPM-001_user_stories.md` — US2 acceptance and traceability
- `docs/specs/finance_domain_model_ddd.md` — FPM/GL ownership, close handoff, and posting-gate invariants
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` — FR-FPM-003 and FR-FPM-006
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — WF-6.1, FPM-WS-01, FPM-SCR-02, GL-SCR-03
- `contracts/openapi/paths/fiscal-periods.yaml` — approved FPM command routes and generic result/problem contracts
