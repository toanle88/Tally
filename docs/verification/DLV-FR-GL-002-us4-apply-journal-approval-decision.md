# DLV-FR-GL-002 — User Story 4 verification evidence

Status: implementation added on branch `feat/gl-us4-apply-journal-approval-decision`.
The GL-owned approval-decision application path is implemented and locally
verified across the domain, generic HTTP operation, development runtime,
frontend result workflow, migration, and PostgreSQL repository. Production
Workflow/Fiscal Period Management/SOD/reference adapter composition and
browser qualification remain explicit boundaries.

## Traceability

- Epic: EP-GL-001 — General Ledger
- Story: User Story 4 — Apply a journal approval decision
- Delivery: DLV-FR-GL-002 / FR-GL-002
- Existing operation: `glApplyJournalApprovalDecision`
- Primary result surface: `GL-SCR-02`; no approval-policy editor was added to GL
- Branch: `feat/gl-us4-apply-journal-approval-decision`

## Implementation evidence

- `internal/gl/posting_approval.go` defines the GL-owned command, immutable
  decision-reference boundary, authorization port, current-state
  revalidation, approved/rejected lifecycle rules, and deterministic
  application fingerprint.
- `internal/gl/posting_memory.go` implements the explicit development/test
  adapter. Approval establishes `Posted` and a ledger position; rejection
  establishes terminal `Cancelled` with no ledger effect. Application and
  decision identities replay safely and changed content conflicts.
- `internal/gl/postgres_posting_approval_repository.go` locks the posting
  gate before the pending journal, reloads the full submitted request, checks
  current versions and decision evidence, and writes the terminal journal,
  approval evidence, audit record, posting attempt, gate position, and
  transactional outbox publication together.
- `db/migrations/gl/00004_apply_journal_approval_decision.sql` adds terminal
  cancellation, approval application/decision identity columns, operation
  evidence, and immutability protection for posted and cancelled facts.
- `internal/platform/httpapi/gl_posting_approval_handler.go` implements the
  existing generic command route, common scope/correlation/idempotency headers,
  typed problem results, and an established result containing decision,
  subject/version, gate, audit, replay, owner/action evidence.
- `cmd/api/gl_posting_runtime.go` composes the shared in-memory posting
  repository with the approval-decision service for the development runtime.
  The Workflow verifier and authorization adapter are explicitly named
  memory adapters; no local approval policy is created.
- `web/src/app/gl-posting-workspace.tsx` adds the pending-result apply panel.
  It displays current journal/approval evidence, captures immutable decision
  evidence, keeps command identity stable across ambiguous retries, and shows
  the applied result without exposing policy-edit controls.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Immutable Workflow decision reference matches journal/request/scope/policy | `PostingApprovalDecisionVerifier`, subject-version/candidate-fingerprint checks, domain tests, HTTP test | Implemented locally; production Workflow adapter remains a composition boundary |
| Current journal, reference, currency/balance, authorization, period, and gate revalidation | `PostingApprovalDecisionService.Execute`, `PostingRequest.Validate`, gate/reference ports, Postgres gate lock/reload, stale-version tests | Implemented locally; live cross-context reference adapters remain a qualification boundary |
| Approved decision posts atomically with approval, gate, audit, and outbox evidence | Memory lifecycle test, Postgres repository integration test, transactional audit/event writes | Implemented and tested |
| Rejected decision has no ledger effect and becomes `Cancelled` | Domain test and Postgres apply integration test path; rejected memory test asserts zero ledger position | Implemented locally; cancellation is the selected current policy lifecycle |
| Stale/mismatched/already-applied decisions are typed conflicts | Domain and HTTP mapping tests for reference, version, scope, duplicate, and idempotency conflicts | Implemented and tested |
| Same identity/fingerprint replays; changed content cannot apply a second decision | Memory and Postgres replay/conflict tests | Implemented and tested |
| UI exposes decision reference, subject/version, current/applied state, revalidation notice, owner, and next action | `gl-posting-workspace.test.tsx` approval-panel test and result/evidence panels | Implemented locally; browser and assistive-technology qualification remains open |

## Verification executed

| Command | Result |
| --- | --- |
| `pnpm contract:lint` | Pass |
| `pnpm contract:check` | Pass; deterministic bundle and invalid fixtures |
| `GOCACHE=/tmp/tally-go-cache make api-generate-check` | Pass |
| `make api-ts-check` | Pass |
| `make db-migrate-validate` | Pass |
| `make db-migrate-inventory` | Pass; checksum inventory refreshed |
| `make db-migrate-check` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test ./...` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test -tags=integration ./internal/platform/database -run '^TestGLPostingRepositoryAppliesApprovalDecisionAtomicallyAndReplays$' -count=1 -v` | Pass with PostgreSQL 18 Testcontainers and Docker access |
| `GOCACHE=/tmp/tally-go-cache go test -tags=integration ./internal/platform/database -run '^TestGLPostingRepositoryPostsAtomicallyAndSerializesGateAdmission$' -count=1 -v` | Pass with PostgreSQL 18 Testcontainers and Docker access |
| `pnpm --dir web exec tsc --noEmit` | Pass |
| `pnpm --dir web exec vitest run src/app/gl-posting-workspace.test.tsx --reporter=dot` | Pass; 4 tests including approval application |
| `pnpm --dir web test` | Pass; 17 files and 93 tests |
| `pnpm --dir web run build` | Pass; Vite production build |
| `git diff --check` | Pass |

## Qualification limits

- The production identity runtime still leaves the GL posting mutation
  services unset until approved Workflow/SOD, Fiscal Period Management, audit,
  and cross-context reference adapters are composed; the HTTP route therefore
  fails closed with a typed service-unavailable result in that composition.
- The in-memory verifier is an explicit development/test adapter. GL does not
  evaluate or edit Workflow approval policy and does not read another module's
  schema or adapter.
- PostgreSQL repository and migration behavior is integration-tested, but no
  production runtime composition currently injects those adapters.
- Playwright, manual keyboard/focus/zoom/reflow, assistive-technology review,
  and live external Workflow/FPM/SOD qualification were not run.
