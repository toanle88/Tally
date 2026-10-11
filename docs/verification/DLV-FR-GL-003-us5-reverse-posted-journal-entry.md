# DLV-FR-GL-003 — User Story 5 verification evidence

Status: implementation added on branch `feat/gl-us5-reverse-posted-journal-entry`.
The GL-owned posted-journal reversal path is implemented and locally verified
across the domain, in-memory development adapter, PostgreSQL repository and
migration, generic HTTP operation, frontend reversal preview/result workflow,
audit evidence, and transactional outbox publication. Production
Workflow/Fiscal Period Management/SOD/reference adapter composition and browser
qualification remain explicit boundaries.

## Traceability

- Epic: EP-GL-001 — General Ledger
- Story: User Story 5 — Reverse a posted journal entry
- Delivery: DLV-FR-GL-003 / FR-GL-003
- Existing operation: `glReverseJournalEntry`
- Permission: `finance.gl.reverse.journal.entry`
- Primary result surfaces: `GL-SCR-01`, `GL-SCR-02`, and `GL-WS-01`
- Branch: `feat/gl-us5-reverse-posted-journal-entry`

## Implementation evidence

- `internal/gl/journal_reversal.go` defines the reversal command, required
  scope/version/date/reason/authorization/source evidence, immutable-original
  eligibility, exact line inversion, separate `PolicyAdjustment` correction
  request, duplicate/source/idempotency protection, and typed rejection paths.
- `internal/gl/posting_memory.go` implements the explicit development/test
  adapter. It keeps the original journal and request unchanged, appends one
  linked posted reversal, records audit/attempt/event evidence, and replays
  the established result for the same identity and fingerprint.
- `internal/gl/postgres_journal_reversal_repository.go` locks and reloads the
  scope, gate, original journal, and source references, then writes the
  reversal journal/lines, audit record, posting attempt, and outbox event in
  one transaction. The composite self-reference and partial unique index
  enforce same-scope linkage and one reversal per original at the database
  boundary.
- `db/migrations/gl/00005_reverse_posted_journal_entry.sql` stores gate mode,
  reversal lineage, and correction reason, extends reversal outcomes and
  operation constraints, and adds the database invariants.
- `internal/platform/httpapi/gl_posting_reversal_handler.go` maps the existing
  generic command route to the GL service and returns typed validation,
  authorization, conflict, and service-unavailable results with original and
  linked reversal evidence.
- `web/src/app/gl-posting-workspace.tsx` adds an established-journal reversal
  panel with exact original lines, equal-and-opposite preview, reason/date,
  authorization and source evidence, stable retry identity, and linked-result
  evidence. The original remains presented as established/read-only.
- `contracts/openapi/paths/general-ledger.yaml` and generated API artifacts
  retain the existing operation while documenting the deterministic reversal
  command and exact-decimal line preview shape.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Command requires original reference, reason, scope, date, authorization, expected version, and new idempotency identity | `ReverseJournalEntryCommand.Validate`, HTTP handler mapping, domain/HTTP tests | Implemented and tested |
| Only a posted eligible journal is reversed; original fact remains immutable | `validateReversalOriginal`, memory test, PostgreSQL integration test | Implemented and tested |
| Separate equal-and-opposite linked journal is appended with current evidence | `buildReversalRequest`, repository commit, migration FK/index, domain and integration tests | Implemented and tested |
| Duplicate, stale-version, and closed/restricted gate requests return typed outcomes; identical retry replays | memory/PostgreSQL replay and conflict checks, stale/gate tests, HTTP mappings | Implemented and tested for the selected Open/SoftClose gate boundary |
| Result/UI expose lineage, reason/evidence, lifecycle, owner, and next action | reversal handler result mapping, workspace panel, focused UI test | Implemented locally; browser and assistive-technology qualification remains open |
| Reversal is attributable, audited, and published through transactional outbox | PostgreSQL transaction, audit/attempt/outbox integration assertions, migration constraints | Implemented and tested |

## Verification executed

| Command | Result |
| --- | --- |
| `make db-migrate-inventory` | Pass; checksum inventory refreshed |
| `make db-migrate-check` | Pass |
| `make db-migrate-validate` | Pass; GL migration directory validated |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/gl` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/platform/httpapi` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test ./cmd/api` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test ./...` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test -tags integration ./internal/platform/database -run TestGLPostingRepositoryReversesPostedJournalAtomicallyAndReplays -count=1` | Pass with PostgreSQL 18 Testcontainers and Docker access |
| `GOCACHE=/tmp/tally-go-cache go test -tags integration ./internal/platform/database -run 'TestGLPostingRepository' -count=1` | Pass; existing GL posting/approval integration coverage also passed |
| `./node_modules/.bin/tsc -b --pretty false` from `web` | Pass |
| `./node_modules/.bin/vitest run src/app/gl-posting-workspace.test.tsx` from `web` | Pass; reversal preview/result coverage passed |
| `make api-generate api-ts-generate` | Pass; generated Go and TypeScript artifacts refreshed |
| `GOCACHE=/tmp/tally-go-cache OPENAPI_CHECK_TIMEOUT_SECONDS=90 make api-check` | Interrupted by the user after contract lint and repeated bundle generation; not counted as a pass |
| `git diff --check` | Pass; Git emitted existing CRLF-normalization warnings for unrelated tracked files |

## Qualification limits

- The production identity runtime still leaves GL posting mutation services
  unset until approved Workflow/SOD, Fiscal Period Management, audit, and
  cross-context reference adapters are composed; the HTTP route therefore
  fails closed with a typed service-unavailable result in that composition.
- The in-memory authorizer, gate validator, and reference validator are
  explicit development/test adapters. GL does not evaluate or edit Workflow
  policy and does not read another module's schema or adapter.
- The selected reversal workflow admits `Open` and `SoftClosePolicy` gate
  evidence and rejects close/restricted modes. Separate reopen workflows are
  not introduced by this story.
- PostgreSQL repository and migration behavior is integration-tested, but no
  production runtime composition currently injects those adapters.
- Playwright, manual keyboard/focus/zoom/reflow, assistive-technology review,
  and live external Workflow/FPM/SOD qualification were not run.
- The repository-wide frontend `vitest run` was attempted: the parallel run
  reached 16 files and 83 tests before a worker-start timeout, and the
  single-worker rerun was stopped at the user's request. The changed
  `gl-posting-workspace` test file passed separately.
- The final review pass aligned PostgreSQL first-result authorization evidence
  with replay behavior, authorized replay responses before returning them, and
  cloned the reversal link in the memory adapter; tests were not rerun after
  those small fixes because the user requested that test processes stop.
