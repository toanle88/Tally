# DLV-FR-FPM-012/013 — US6 verification evidence

Status: focused local implementation evidence on branch
`feat/dlv-fr-fpm-012-013-us6-period-control-recovery`, recorded 2026-10-17.
This record covers period-control takeover, close-exception extension, the
approved generated API boundary, FPM-SCR-04/FPM-SCR-05 UI behavior, and the
RUN-005 operational contract. It does not claim completion of EP-FPM-001,
PostgreSQL/Testcontainers runtime qualification, browser/accessibility
qualification, or release checks.

## Traceability

- Epic: EP-FPM-001 — Fiscal period management
- Story: US6 — Recover period control and extend close exceptions
- Delivery: DLV-FR-FPM-012, DLV-FR-FPM-013, WF-7.12
- Commands: `TakeOverPeriodControl`, `ExtendCloseException`
- Approved routes: `fpmTakeOverPeriodControl`, `fpmExtendCloseException`
- Permissions: `finance.fpm.take.over.period.control`,
  `finance.fpm.extend.close.exception`
- Events retained: `SoftCloseResumed`, `CloseRunResumed`, and
  `PeriodStateChanged` only when the existing GL reconciliation changes the
  lifecycle state. Exception extension uses the existing transactional
  mutation/audit path and adds no new event type.
- Screens: `FPM-SCR-04` at `/fiscal-periods/fpm-scr-04` and `FPM-SCR-05` at
  `/fiscal-periods/fpm-scr-05`
- Runbook: [RUN-005 — Period-control interruption and recovery](../operations/runbooks/run-005-period-control-interruption.md)

## Implementation evidence

- FPM owns `ControlAuthority`, append-only ownership lineage, dedicated
  takeover/extension approval evidence, command fingerprints, expected
  period/process/gate/authority versions, and durable operation identities.
- Takeover requires valid persisted authority metadata, expired eligibility,
  independent approval, and a matching `GetPostingGateStatus` result. The
  close-run path reuses the existing authoritative posting-gate application
  seam when GL proves an interrupted CloseOnly transition. No GL open, release,
  finalize, or gate-weakening call is made.
- Exception extension records owner, impact, due/expiry, justification,
  evidence, timestamps, and append-only extension history. Unresolved remains
  unresolved; an approved-but-expired exception can become approved again with
  a new expiry; rejected, completed, finalized, and finalizing states remain
  blocked. Existing final approval and GL gate checks remain in force.
- PostgreSQL readers/writers preserve nullable object-validated `control_authority`
  JSONB and close-run detail/exception extension history. Legacy rows without
  authority metadata remain ineligible for takeover; no owner is synthesized.
- Production and memory composition roots bind the narrow Workflow validator,
  which checks request identity, candidate fingerprint, subject/policy version,
  decision outcome, approver separation, timestamp, and policy authorization.
- Existing generated API artifacts are reused. Established results expose
  owner lineage, authority epoch/version/expiry, cutoff, authoritative gate
  status, pending reconciliation, extension history, next action, and replay
  status. Recovery telemetry uses bounded operation/outcome and age buckets;
  correlation IDs remain the sanitized support reference.
- The frontend preserves text/announcement status semantics, shows owner and
  authority/gate evidence, and provides safe takeover and extension controls
  without color-only state meaning.

## Acceptance traceability

| Requirement | Evidence | Result |
|---|---|---|
| Soft-close takeover preserves identity and does not mutate GL | `TestTakeOverSoftClosePreservesIdentityAndNeverMutatesGL`, `TestConcurrentRecoveryCommandsHaveOneWinner` | Implemented and focused-tested |
| Close-run takeover reconciles an existing authoritative GL result | `TestTakeOverCloseRunReconcilesExistingGLBarrierWithoutMutation` | Implemented and focused-tested |
| Expiry, missing legacy authority, scope/version/gate mismatch, and dependency failure remain restrictive | `TestTakeOverRejectsActiveOrMissingAuthorityAndGateMismatch`, `TestRecoveryApprovalAndDependencyFailuresRemainRetryable` | Implemented and focused-tested |
| Independent approval, duplicate identity, changed duplicate, self-approval, rejected/stale approval, and changed candidate are enforced | recovery command validation, candidate fingerprints, `TestTakeoverRejectsInvalidRecoveryApprovalEvidence`, `TestTakeOverSoftClosePreservesIdentityAndNeverMutatesGL` | Implemented and focused-tested |
| Unresolved and approved-expired exception extension preserves original semantics and history | `TestExtendCloseExceptionPreservesUnresolvedStateAndHistory`, `TestExtendApprovedExpiredExceptionAndRejectTerminalCases` | Implemented and focused-tested |
| Rejected/finalizing exception states and invalid windows remain blocked | extension target validation, `TestExtendApprovedExpiredExceptionAndRejectTerminalCases`, `TestExtendCloseExceptionRejectsInvalidFutureWindow` | Implemented and focused-tested |
| API handler decoding, typed problem mapping, runtime binding, and established-result fields | `fiscal_period_recovery_handler.go`, runtime composition roots, focused Go package tests | Implemented; handler behavior is compile/focused-tested; broader API qualification remains deferred |
| FPM-SCR-04 route/recovery status and FPM-SCR-05 extension controls | recovery workspace/component tests, route registry/router tests | Implemented and focused-tested |
| PostgreSQL round-trip, transactional outbox/audit failure, concurrent runtime qualification, and accessibility release checks | migration/repository implementation plus deferred integration/browser checks | Deferred; not claimed here |

## Verification executed

| Command | Result |
|---|---|
| `GOCACHE=/tmp/tally-go-cache go test ./internal/fiscalperiod ./internal/platform/httpapi ./cmd/api` | Pass |
| `pnpm -C web exec vitest run src/app/fiscal-period-control-recovery-workspace.test.tsx src/app/fiscal-period-hard-close-workspace.test.tsx src/routes/route-registry.test.ts src/routes/router.test.tsx` | Pass; 4 files and 27 tests |
| `make db-migrate-inventory` | Pass; migration checksum inventory updated for `00005_add_period_control_recovery.sql` |
| `make db-migrate-validate` | Pass |
| `make db-migrate-check` | Pass |
| `GOCACHE=/tmp/tally-go-cache make check-fast` | Pass; migration validation, checksum validation, and all Go packages passed (elevated local-listener test environment) |
| PostgreSQL/Testcontainers, `make api-check`, browser/accessibility, slow/release checks | Not run; opt-in per repository instructions |

## Scope and status limits

- `ROADMAP.md` completion markers remain unchanged. The US6 backlog acceptance
  boxes remain evidence gates rather than being inferred complete from local
  implementation.
- This record does not claim `StartSoftClose`, `EndSoftClose`, operational
  reopen takeover, GL gate mutation, new read APIs, Workflow approval creation,
  production worker composition, hosted CI, or release qualification.
- Local focused results do not establish production availability, capacity,
  recovery-point objectives, audit-chain qualification, or downstream
  cross-context behavior.
