# EP-WFA-001 — Workflow and Approvals User Stories

| Field | Value |
|---|---|
| Epic | `EP-WFA-001` — Workflow and approvals |
| Status | User Stories 1–5 implementation slices locally in place — focused evidence recorded on 2026-10-15; persistence, notification, accessibility, and release checks remain pending |
| Milestone | `M3` — Approval and period controls |
| Delivery items | `DLV-GFR-004`, `DLV-FR-WFA-001`–`DLV-FR-WFA-005` |
| Supporting workflows | `DLV-WF-6.1`, `DLV-WF-6.2`, `DLV-WF-7.12`; these workflows remain owned by Fiscal Period Management and GL |
| Owning bounded context | Workflow & Approvals — `internal/workflow`, `workflow` schema |
| Primary users | Approver; Delegator; Approval Administrator |
| Authoritative records | `ApprovalPolicy`, `ApprovalRequest`, `Delegation`; `ApprovalDecision` is an entity within `ApprovalRequest` |
| UX surfaces | `WFA-WS-01`, `WFA-SCR-01`–`WFA-SCR-04` |
| Exit evidence | Domain, persistence, API, authorization, audit, outbox, idempotency, concurrency, UI, accessibility, integration, and recovery evidence for all five stories |

This document is the planning and delivery-traceability artifact for
`EP-WFA-001`. User Stories 1–5 implementation and focused local verification
are recorded below; full-epic integration and release qualification remain
open.
The stories are grouped into coherent vertical slices while preserving every
approved WFA requirement, operation, route, permission, event, and ownership
boundary.

## 1. Outcome

Deliver the Workflow & Approvals bounded context so approval-bearing actions can
route through the applicable versioned policy, produce an immutable approval
request and decision history, support controlled delegation and escalation, and
expose the current owner, state, blockers, evidence, and next action.

When a consuming business context applies a decision, it must use the immutable
Workflow decision reference and revalidate its own current aggregate version and
business state before changing its lifecycle. Workflow does not post journals,
change fiscal periods, or mutate another context's aggregate.

## 2. Learning objective

Learn how to build a generic approval bounded context that remains independent
from business aggregates while preserving policy versioning, segregation of
duties, immutable decisions, cross-context revalidation, transactional outbox
publication, safe retries, and user-visible recovery states.

## 3. Scope

- Maintain versioned, effective-dated approval policies with applicability,
  scope, amount thresholds, role requirements, steps, delegation rules, and
  escalation rules.
- Create approval requests against a resolved policy version and an immutable
  subject reference/version/snapshot suitable for review.
- Record policy-compliant approval decisions with immutable evidence and publish
  the approved WFA decision event.
- Establish time-bounded, scoped delegation and policy-defined escalation while
  preserving the original request, subject, policy, and audit lineage.
- Provide the approved API routes, permissions, database ownership, outbox
  events, worklist/detail/decision/policy/delegation-escalation surfaces, and
  typed failure outcomes.
- Provide the contract that consuming contexts use to apply a decision after
  revalidating their current business state.
- Enforce authorization, segregation of duties, expected versions, idempotency,
  audit evidence, privacy minimization, observability, accessibility, and
  recovery behavior at each applicable boundary.

## 4. Explicit exclusions

- No ownership of GL journals, posting gates, fiscal periods, close runs,
  reopen requests, AP, AR, payment, payroll, tax, reporting, asset, revenue,
  or other business aggregates.
- No implementation of consuming `Apply...ApprovalDecision` commands inside
  Workflow. Those commands belong to the consuming bounded context; Workflow
  publishes and exposes the immutable decision reference.
- No direct access to another module's adapter, repository, table, or schema.
- No replacement of Identity & Access. IAM owns users, roles, permissions,
  access scopes, and segregation rules; Workflow calls approved authorization
  ports and retains decision references/evidence only.
- No separate `Escalation` aggregate unless the DDD baseline is explicitly
  revised. The DDD baseline names `EscalateApproval` and
  `ApprovalEscalated`, but lists only `ApprovalPolicy`, `ApprovalRequest`, and
  `Delegation` as Workflow authoritative aggregates.
- No new command, event, permission, route, lifecycle state, policy-precedence
  rule, or revocation operation not present in the approved sources.
- No destructive edit of a policy version, request, decision, delegation, or
  audit fact. Corrections and administrative changes retain lineage.
- No production, live Entra, full audit-chain, performance/capacity, disaster
  recovery, or release-qualification claim from local story work alone.

## 5. Ownership and domain rules

### 5.1 Owning component

`internal/workflow` owns the Workflow & Approvals domain, application handlers,
repositories, and `workflow` PostgreSQL schema. The approved root records are:

- `ApprovalPolicy`, containing `ApprovalRule` and `ApprovalStepDefinition`.
- `ApprovalRequest`, containing `ApprovalStep` and `ApprovalDecision`.
- `Delegation`.

The workflow module owns approval policy applicability, request lifecycle,
decision recording, delegation, and escalation behavior. Other modules hold
identifiers and immutable references or projections; they do not write the
`workflow` schema.

### 5.2 Approval and version invariants

- Approval policies are versioned and effective-dated. Historical requests and
  decisions retain the policy version used; later policy changes do not rewrite
  that history.
- An approval request identifies its subject, subject version, submitting
  actor/process, applicable policy version, scope, and reviewable snapshot or
  reference. The request does not become a second owner of the subject
  aggregate.
- `ApprovalDecisionReference` contains the immutable
  `ApprovalRequestId`, `DecisionId`, `PolicyVersion`, `DecisionVersion`,
  `AppliedSubjectVersion`, `DecidedAt`, and `AppliedAt` values defined by the
  DDD baseline. A business aggregate in `PendingApproval` may retain only its
  request identifier; the consuming aggregate completes its reference when the
  decision is applied.
- Approval decisions are owned by Workflow through `DecideApprovalRequest`.
  A business context cannot originate or edit its own approval decision.
- Applying a decision is a separate consuming-context transition. The consumer
  revalidates the current subject/aggregate version, authorization, policy
  applicability, and business invariants; a stale or invalid decision cannot
  establish the business effect.
- Authorization is evaluated across the dimensions applicable to the action:
  legal entity, business unit or segment, account or account class, transaction
  type, amount threshold, currency, fiscal period, data sensitivity, and action.
- The minimum segregation rules from DDD §8.2 apply, including that a requester
  cannot approve the same fiscal-period reopen, and policy administration
  requires independent approval and audit evidence.
- State-changing retries use the approved platform idempotency and expected
  aggregate-version conventions. The same business identity and fingerprint
  return the established outcome; changed content returns a typed conflict with
  no second effect.
- Material actions and decisions produce audit evidence with actor,
  authentication subject, scope, action, authorization/approval reference,
  correlation/causation, before/after fingerprints where applicable, and
  policy/subject versions.

### 5.3 Published WFA contracts

The approved technical specifications identify these operation contracts:

| Requirement | Operation | API route | API operation ID | Application handler | Permission |
|---|---|---|---|---|---|
| `FR-WFA-001` | `CreateApprovalRequest` | `POST /api/v1/approvals/actions/create-approval-request` | `wfaCreateApprovalRequest` | `wfaCreateApprovalRequestHandler` | `finance.wfa.create.approval.request` |
| `FR-WFA-001` | `List approval requests` | `GET /api/v1/approvals/reference/requests` | `wfaListApprovalRequests` | `wfaListApprovalRequestsHandler` | `finance.wfa.read.approval.requests` |
| `FR-WFA-001` | `Get approval request` | `GET /api/v1/approvals/reference/requests/{approvalRequestId}` | `wfaGetApprovalRequest` | `wfaGetApprovalRequestHandler` | `finance.wfa.read.approval.requests` |
| `FR-WFA-002` | `DecideApprovalRequest` | `POST /api/v1/approvals/actions/decide-approval-request` | `wfaDecideApprovalRequest` | `wfaDecideApprovalRequestHandler` | `finance.wfa.decide.approval.request` |
| `FR-WFA-003` | `DelegateApproval` | `POST /api/v1/approvals/actions/delegate-approval` | `wfaDelegateApproval` | `wfaDelegateApprovalHandler` | `finance.wfa.delegate.approval` |
| `FR-WFA-004` | `EscalateApproval` | `POST /api/v1/approvals/actions/escalate-approval` | `wfaEscalateApproval` | `wfaEscalateApprovalHandler` | `finance.wfa.escalate.approval` |
| `FR-WFA-005` | `Maintain approval policies` | `PUT /api/v1/approvals/configuration/maintain-approval-policies` | `wfaMaintainApprovalPolicies` | `wfaMaintainApprovalPoliciesHandler` | `finance.wfa.maintain.approval.policies` |

The approved WFA event names and event subjects are:

| Operation | Event | Event contract |
|---|---|---|
| `CreateApprovalRequest` | `ApprovalRequested` | `approval_requested` |
| `DecideApprovalRequest` | `ApprovalDecisionRecorded` | `approval_decision_recorded` |
| `DelegateApproval` | `ApprovalDelegated` | `approval_delegated` |
| `EscalateApproval` | `ApprovalEscalated` | `approval_escalated` |

`Maintain approval policies` is a PRD functional action, not a DDD command or
named domain event in the current baseline. Its versioned administrative
outcome must still be authorized, independently approved where required,
audited, and published through the approved integration mechanism without
inventing a new event name in this story plan.

## 6. Traceability

| Story | Direct delivery and requirement coverage | Supporting sources |
|---|---|---|
| User Story 1 — Maintain approval policies | `DLV-FR-WFA-005`, `FR-WFA-005` | `DLV-GFR-004`, `GFR-002`–`GFR-004`, `NFR-MNT-003`, `NFR-MNT-006`, `NFR-SEC-012`, `NFR-AUD-001`–`NFR-AUD-004` |
| User Story 2 — Create approval requests | `DLV-FR-WFA-001`, `FR-WFA-001` | `DLV-GFR-004`, `GFR-006`–`GFR-009`, `NFR-PERF-002`, `NFR-SEC-002`–`NFR-SEC-005`, `NFR-REL-001`–`NFR-REL-004` |
| User Story 3 — Record and apply decisions | `DLV-FR-WFA-002`, `FR-WFA-002` | `DLV-GFR-004`, `GFR-003`, `GFR-006`–`GFR-014`, `NFR-PERF-007`, `NFR-PERF-013`, `NFR-INT-002`–`NFR-INT-004`, `NFR-AUD-001`–`NFR-AUD-010` |
| User Story 4 — Delegate approval responsibility | `DLV-FR-WFA-003`, `FR-WFA-003` | `GFR-002`–`GFR-004`, `GFR-009`, `NFR-CAP-007`, `NFR-SEC-004`–`NFR-SEC-006`, `NFR-SEC-012`, `NFR-ACC-008` |
| User Story 5 — Escalate approvals | `DLV-FR-WFA-004`, `FR-WFA-004` | `GFR-004`, `GFR-009`, `GFR-012`, `NFR-CAP-007`, `NFR-PERF-013`, `NFR-OBS-006`–`NFR-OBS-008`, `NFR-AUD-001`–`NFR-AUD-003` |

Cross-context acceptance also contributes to `DLV-WF-6.1` / `WF-6.1`,
`DLV-WF-6.2` / `WF-6.2`, and `DLV-WF-7.12` / `WF-7.12`. Those workflow
demonstrations remain owned by Fiscal Period Management and GL; this epic
provides the approval contract and evidence they consume.

## 7. User stories

### 7.1 User Story 1 — Maintain versioned approval policies

**As an Approval Administrator, I want to create and maintain versioned,
effective-dated approval policies, so that approval-bearing actions route to
the correct steps, scopes, thresholds, and eligible roles.**

Traceability: `DLV-FR-WFA-005`, `FR-WFA-005`, `DLV-GFR-004`, `NFR-MNT-003`,
`NFR-MNT-006`, `NFR-SEC-012`, `NFR-AUD-001`–`NFR-AUD-004`, `WFA-SCR-03`.

Acceptance criteria:

- [x] An authorized Approval Administrator can maintain a policy with its
  applicability, scope, amount thresholds, role requirements, approval steps,
  delegation rules, escalation rules, and effective-date range.
- [x] Policy identity, field validity, scope containment, effective-date
  validity, and policy-applicability conflicts are validated before the change
  can become active. The exact precedence/overlap rule must follow an approved
  domain decision rather than an implementation-specific default.
- [x] Administrative changes require the independent approval and audit
  evidence required by the approved policy. A policy cannot become active from
  an unauthorized or self-approved change.
- [x] An activated or historically applied policy version is immutable. A
  later change creates a new version and does not rewrite requests, decisions,
  or historical authorization evidence.
- [x] The operation result exposes policy version, scope, thresholds, role
  requirements, effective dates, approval status, activation outcome,
  dependent-workflow impact, and any typed rejection or conflict.
- [x] The API enforces `finance.wfa.maintain.approval.policies`, expected
  version, idempotency, scope containment, and safe authorization denial at the
  application/domain boundary; frontend visibility is advisory only.
- [x] `WFA-SCR-03` supports keyboard-accessible editing, validation summaries,
  effective-date review, approval status, and safe recovery from stale-version
  or authorization conflicts without exposing restricted data.

Likely implementation areas:

- `internal/workflow/` policy aggregate, validation, application handler,
  authorization/audit ports, repository, and focused tests.
- `db/migrations/workflow/`, `db/queries/workflow/`, and generated SQLC output
  for `workflow.approval_policy` using the repository's migration conventions.
- `contracts/openapi/paths/approvals.yaml` and generated Go/TypeScript
  artifacts for the approved policy route.
- `web/src/capabilities/approvals/` and `WFA-SCR-03` route wiring.

Required evidence:

- Domain tests for valid policy creation, invalid applicability, effective-date
  boundaries, independent approval, immutable historical versions, duplicate
  submission, changed-fingerprint conflict, and expected-version conflict.
- PostgreSQL tests for schema ownership, version retention, unique policy
  identity, transaction rollback, audit linkage, and durable idempotency.
- API, authorization, accessibility, and UI evidence for the policy editor and
  typed 400/403/409/422/503 outcomes as applicable.

#### User Story 1 implementation evidence (2026-10-13)

- Implemented the `internal/workflow` policy aggregate and generic approval
  request/decision path with exact-decimal thresholds, effective-dated scope
  overlap checks, immutable version history, independent approval, self-approval
  rejection, audit evidence, transactional outbox preparation, and durable
  idempotency.
- Added the owned `workflow` schema, migrations, SQLC queries/generated code,
  bootstrap authority, typed `PUT /api/v1/approvals/configuration/maintain-approval-policies`
  contract/handler, runtime wiring, and `WFA-SCR-03` at
  `/approvals/wfa-scr-03`.
- Passed locally: `go test ./...`, `make check-fast`, `make sqlc-check`,
  `make api-generate-check`, `make api-ts-check`, the focused WFA route test,
  and `pnpm -C web run build`.
- Not claimed as release evidence: PostgreSQL/Testcontainers persistence gates
  (opt-in by repository policy) and the full `pnpm -C web run test:fast` suite;
  the latter started Vitest but hung before discovering workers in this local
  environment.

### 7.2 User Story 2 — Create approval requests

**As a requester or approval-bearing domain process, I want to create an
approval request against the applicable policy version, so that the eligible
approver receives a reviewable and traceable task before the business effect is
established.**

Traceability: `DLV-FR-WFA-001`, `FR-WFA-001`, `DLV-GFR-004`,
`NFR-PERF-002`, `NFR-SEC-002`–`NFR-SEC-005`, `NFR-REL-001`–`NFR-REL-004`,
`WFA-WS-01`, `WFA-SCR-01`.

Acceptance criteria:

- [x] `CreateApprovalRequest` resolves the applicable effective policy using
  the action, scope, subject, amount/currency, fiscal-period, sensitivity, and
  other dimensions relevant to that policy before creating the request.
- [x] The request retains a stable `ApprovalRequestId`, submitting actor or
  process, subject reference, subject version, applicable policy version,
  scope, and an immutable subject snapshot/reference sufficient for approver
  review. The workflow record does not become the subject aggregate owner.
- [x] The request contains the policy-defined approval steps and becomes an
  actionable request state visible to the responsible owner. If policy
  applicability, authorization, or a required dependency cannot be established,
  the operation returns a typed rejection/unavailable outcome and creates no
  partial request.
- [x] A request remains tied to the policy version resolved at creation. A
  later policy change does not silently rebind an existing request.
- [ ] The result exposes the request identity, current lifecycle state,
  validation/approval status, responsible owner, policy/subject versions, and
  exception or recovery path. `WFA-WS-01` can filter assigned, due, blocked,
  delegated, escalated, and completed work as authorized.
- [x] Repeating the same request identity and fingerprint returns the
  established request result. Reusing that identity with changed functional
  content returns an idempotency conflict and creates no second request.
- [ ] Concurrent creation against changed source/policy/aggregate versions
  returns a typed version or domain conflict rather than silently creating a
  request from stale facts.
- [x] The transaction records the request and its `ApprovalRequested` outbox
  message atomically. The published payload contains only the minimum safe
  identifiers, versions, scope, correlation/causation, and classification
  required by approved consumers.
- [ ] `WFA-SCR-01` shows the subject snapshot/reference, policy/version, steps,
  current owner, state, blockers, lineage, and evidence without becoming a
  second mutation surface for the source record.

Likely implementation areas:

- `internal/workflow/` request aggregate, policy-resolution port, command
  handler, idempotency/concurrency behavior, audit port, and outbox mapping.
- `db/migrations/workflow/` and `db/queries/workflow/` for
  `workflow.approval_request` and its aggregate-owned child records.
- `contracts/openapi/paths/approvals.yaml` for the create-request envelope and
  generated artifacts.
- `internal/platform/httpapi/workflow_approval_request_handler.go` for the
  typed create/list/detail boundary, and
  `web/src/app/workflow-approval-workspace.tsx` for the approval worklist and
  read-only request detail.

Required evidence:

- Domain tests for policy selection, immutable subject version/snapshot,
  unauthorized/missing-policy outcomes, duplicate delivery, changed-fingerprint
  conflict, stale source version, and outbox failure rollback.
- Integration tests proving one request identity produces one local effect and
  duplicate `ApprovalRequested` delivery is safe for registered consumers.
- Worklist/detail, permission-filtering, accessibility, and error-recovery
  evidence for `WFA-WS-01` and `WFA-SCR-01`.

#### User Story 2 implementation evidence (2026-10-14)

- Added exact-decimal effective-policy resolution requiring exactly one active
  match across applicability, dimensions, scope, amount, currency, and
  effective interval. Policy identity/version, copied steps, responsible roles,
  due evidence, actor, safe subject snapshot, and snapshot fingerprint are
  retained on the request.
- Added workflow-owned evidence columns and indexes, SQLC regeneration, and
  PostgreSQL request reads/writes. Request creation records the request and
  `approval_requested` outbox event through the existing transactional
  repository boundary; policy binding is copied and is not re-resolved on
  replay.
- Added typed `wfaCreateApprovalRequest`, `wfaListApprovalRequests`, and
  `wfaGetApprovalRequest` contracts/handlers with the new
  `finance.wfa.read.approval.requests` permission, typed policy/idempotency/
  authorization conflicts, and safe authorized filtering/pagination.
- Added `WFA-WS-01` and read-only `WFA-SCR-01` routes using generated API
  clients, exact-decimal form inputs, bounded subject evidence, and no source
  record mutation surface.
- The current read slice exposes role-based responsible roles, lifecycle state,
  due filtering, subject/policy evidence, copied steps, and safe authorized
  pagination. Concrete user assignment, decision/delegation/escalation
  lineage, blocker panels, and exception/recovery actions remain deferred to
  the later decision, delegation, and escalation stories.
- Focused verification passed: `GOCACHE=/tmp/tally-go-cache go test
  ./internal/workflow ./internal/platform/httpapi ./cmd/api`,
  `make sqlc-check`, `GOCACHE=/tmp/tally-go-cache make api-generate-check`,
  `make api-ts-check`, `pnpm --dir web build`, and the web unit project
  (`4` files, `11` tests). The DOM project stalls before reporting tests in
  this local runner.
- Not claimed in this story slice: source-context adapters that independently
  revalidate a source aggregate immediately before request creation, concrete
  user-to-role assignment resolution, and PostgreSQL/Testcontainers release
  gates (opt-in repository checks).

### 7.3 User Story 3 — Record and expose approval decisions

**As an eligible Approver, I want to approve or reject an assigned request with
the required evidence, so that the owning business context can apply one
immutable decision after revalidating its current state.**

Traceability: `DLV-FR-WFA-002`, `FR-WFA-002`, `DLV-GFR-004`,
`NFR-PERF-007`, `NFR-PERF-013`, `NFR-INT-002`–`NFR-INT-004`,
`NFR-AUD-001`–`NFR-AUD-010`, `WFA-SCR-01`, `WFA-SCR-02`.

Acceptance criteria:

- [x] `DecideApprovalRequest` permits only an actor eligible for the current
  policy step, scope, effective interval, and delegation state. IAM
  authorization and segregation checks are evaluated at decision time, not
  trusted from a stale worklist result.
- [x] A requester cannot approve the same approval request when the applicable
  segregation rule prohibits it. Denial exposes a safe reason and permitted
  recovery path without leaking policy-sensitive values.
- [ ] The decision workspace displays the immutable subject snapshot/reference,
  policy/version, current step, prior decisions, delegation/escalation history,
  required evidence, and current application status available through approved
  contracts.
- [x] A decision records an immutable `DecisionId`, `DecisionVersion`, request
  identity, policy version, subject version, approver/authentication subject,
  decision time, policy-required reason/evidence, and delegation reference when
  applicable. It cannot be destructively edited or deleted.
- [x] The command validates the current request version and policy-defined
  actionable state. A terminal, stale, already-decided, expired, or changed
  subject outcome returns a typed conflict/rejection and establishes no second
  decision.
- [x] Repeating the same decision identity and fingerprint returns the
  established result. Reusing the identity with different decision content
  returns an idempotency conflict with no second decision.
- [x] The transaction records the decision and its `ApprovalDecisionRecorded`
  outbox message atomically. Event delivery contains the immutable decision
  reference needed by approved consuming contexts.
- [ ] The consuming business context applies the decision through its own
  `Apply...ApprovalDecision` command, revalidates current aggregate/business
  state and policy applicability, and records the complete immutable decision
  reference only when application succeeds. Workflow does not perform that
  business mutation.
- [ ] `WFA-SCR-02` shows approved/rejected or other policy-defined outcome,
  current state, responsible owner, conflict/rejection reason, and the next
  permitted action. User-visible approval/decision notifications meet the
  applicable NFR timing target.

Likely implementation areas:

- `internal/workflow/` decision entity/transition, authorization and
  segregation ports, immutable evidence model, handler, audit integration, and
  event publication.
- `internal/platform/integration` or the approved outbox/inbox adapter for
  `ApprovalDecisionRecorded`; no direct consumer-schema writes.
- `contracts/openapi/paths/approvals.yaml` and generated API artifacts for the
  decision route.
- `web/src/capabilities/approvals/` and shared `approval-panel`/
  `evidence-drawer` components.

Required evidence:

- Domain and API tests for eligible/ineligible approvers, self-approval,
  delegation, expired policy, stale subject, terminal request, duplicate
  decision, changed fingerprint, and concurrent decision races.
- Persistence/outbox tests for atomic decision recording, immutable history,
  event retry, duplicate event delivery, and recovery from an ambiguous
  consumer application.
- A consumer-contract test showing a stale or changed business aggregate is
  rejected on application and that a valid decision reference is applied once.
- UI, authorization, accessibility, notification, and safe-sensitive-data
  evidence for `WFA-SCR-02`.

#### User Story 3 implementation evidence (2026-10-14)

- Added immutable decision recording in `internal/workflow`, including current
  step, outcome, reason, policy/subject versions, approver identity, decision
  time, delegation reference, audit evidence, and the
  `approval_decision_recorded` outbox payload. Decision-time authorization
  rechecks IAM, active role/scope assignment, delegation policy, effective
  policy interval, current actionable step, and applicable segregation rules.
- Added durable schema/query support for decision step and delegation evidence
  in migration `workflow/00003_add_approval_decision_evidence.sql`, with SQLC
  output regenerated. Existing decision rows remain readable through nullable
  step evidence for migration compatibility.
- Added the typed `wfaDecideApprovalRequest` API operation with strict command
  payload validation, typed stale/expired/step/idempotency/authorization
  outcomes, immutable decision history in request projections, and the locked
  `not-reported` application-status contract until an approved consuming
  context is wired.
- Added `WFA-SCR-02` at `/approvals/wfa-scr-02`. It loads the authoritative
  request detail, shows subject/policy/current-step/prior-decision evidence,
  submits stable retry-safe command identities, and preserves the command
  identity when a retryable conflict is returned.
- Added domain, API, and runtime authorization tests for eligible/ineligible
  roles, scope, delegation evidence, self-approval segregation, current-step
  enforcement, expired policy, immutable decision history, duplicate replay,
  changed-fingerprint conflict, stale subject rejection, and a synthetic
  consumer that applies valid evidence once while rejecting stale evidence.
- Verified with `GOCACHE=/tmp/tally-go-cache go test
  ./internal/identity ./internal/workflow ./internal/platform/httpapi ./cmd/api`,
  `make sqlc-check`, `GOCACHE=/tmp/tally-go-cache make api-generate-check`,
  `make api-ts-check`, `pnpm --dir web exec tsc --noEmit`, the focused route
  test, `pnpm --dir web build`, and `git diff --check`. OpenAPI and TypeScript
  generated artifacts were regenerated and verified from the canonical
  contracts.
- Not claimed in this story slice: a production consuming business-context
  adapter, applied-status acknowledgements, escalation workflows, notification
  delivery, PostgreSQL/Testcontainers release gates, or the full frontend
  Vitest suite.

### 7.4 User Story 4 — Delegate approval responsibility

**As an authorized Delegator, I want to delegate approval responsibility to an
eligible actor for a defined scope and time range, so that assigned work can
continue without weakening policy or segregation controls.**

Traceability: `DLV-FR-WFA-003`, `FR-WFA-003`, `GFR-002`–`GFR-004`,
`GFR-009`, `NFR-CAP-007`, `NFR-SEC-004`–`NFR-SEC-006`, `NFR-SEC-012`,
`NFR-ACC-008`, `WFA-WS-01`, `WFA-SCR-04`.

Acceptance criteria:

- [x] `DelegateApproval` validates the delegator's authorization, the
  delegate's eligibility, the requested scope, the effective date range, the
  applicable policy version, and all relevant segregation rules before
  establishing a delegation.
- [x] A delegation has stable `DelegationId`, delegator, delegate, scope, and
  effective-date values. Its history is retained and independently attributable
  for audit and later review.
- [x] A delegate can act only while the delegation is effective and within its
  scope. Expired, out-of-scope, or unauthorized delegation does not permit a
  decision, even if a stale worklist still displays the task. This delivery
  slice has no revoke/end command; expiry is the only termination mechanism
  until that contract is approved.
- [x] The active delegation is visible in the assigned/delegated worklist and
  request detail, with owner, scope, effective interval, and evidence shown to
  authorized users.
- [x] A delegated decision records the actual approver and the applicable
  delegation evidence; delegation does not erase the original request owner or
  broaden the policy's required approval level.
- [x] Repeating the same delegation identity and fingerprint returns its
  established result; conflicting reuse or concurrent updates return typed
  idempotency/version conflicts without overwriting an established delegation.
- [x] The transaction records the delegation and its `ApprovalDelegated`
  outbox message atomically, with minimized payloads and correlation/causation
  evidence.
- [x] `WFA-SCR-04` supports the approved create behavior with accessible
  validation, a finite-interval expiry warning, audit evidence in the
  established result, and safe conflict recovery. End/revoke behavior remains
  deferred because the approved operation list contains only `DelegateApproval`.

Likely implementation areas:

- `internal/workflow/` `Delegation` aggregate, eligibility/scope validation,
  handler, audit port, and event mapping.
- `db/migrations/workflow/` and `db/queries/workflow/` for
  `workflow.delegation` and its effective-dated history.
- `contracts/openapi/paths/approvals.yaml` and generated artifacts for the
  delegation action.
- `web/src/app/workflow-delegation-workspace.tsx` and route wiring for
  delegated filters and `WFA-SCR-04`.

Required evidence:

- Domain tests for valid scope/time windows, self-delegation, prohibited
  segregation combinations, expiry, out-of-scope action, duplicate submission,
  changed fingerprint, and concurrent delegation changes.
- Persistence and outbox tests proving immutable delegation history and safe
  replay of `ApprovalDelegated`.
- UI and authorization evidence showing delegated work without exposing
  records outside the delegate's scope.

#### User Story 4 implementation evidence (2026-10-15)

- Added the immutable `Delegation` aggregate and finite `[effectiveFrom,
  effectiveTo)` interval validation in `internal/workflow`, including
  self-delegation, policy-version, scope, compatible-step, and expiry rules.
  Delegation decision-time resolution is bound to the live aggregate, so a
  stale worklist cannot authorize an expired, mismatched, or out-of-scope
  decision.
- Added `workflow.delegation` migration and generated SQLC model, PostgreSQL
  transaction wiring for audit, aggregate, outbox publication, and durable
  idempotency finalization, plus memory adapters used by the local runtime.
- Added typed `DelegateApproval` at
  `POST /api/v1/approvals/actions/delegate-approval`, the approved
  `finance.wfa.delegate.approval` permission, `approval_delegated` event
  payload, replay/conflict outcomes, and generated Go/TypeScript artifacts.
- Added IAM evaluator wiring for the existing permission boundary, active
  delegate user/role/scope checks, policy-defined delegation eligibility, and
  decision-time delegated approver evidence. No new segregation identifier was
  invented; the existing evaluator and domain self-delegation guard are used.
- Added WFA-SCR-04 at `/approvals/wfa-scr-04`, active-delegation projections in
  the worklist/detail/decision surfaces, typed delegation-ID decision input,
  finite UTC form validation, and retry-safe command identity retention.
- Added domain, API-handler, decision-time expiry, route, and replay/conflict
  tests. Verified with `GOCACHE=/tmp/tally-go-cache-story4 go test
  ./internal/workflow ./internal/platform/httpapi ./cmd/api`, `make
  db-migrate-validate`, `make db-migrate-check`, `make sqlc-check`, `make
  api-generate-check`, `make api-ts-check`, `pnpm --dir web exec tsc --noEmit`,
  `pnpm --dir web build`, the focused route tests, and `git diff --check`.
- Not claimed in this story slice: revoke/end behavior, PostgreSQL/Testcontainers
  release qualification, live Entra or production SoD qualification, witnessed
  screen-reader/400% zoom review, or User Story 5 escalation behavior.

### 7.5 User Story 5 — Escalate overdue or blocked approvals

**As an authorized approval owner or domain process, I want to escalate an
overdue or blocked approval according to policy, so that responsibility is
visible and recoverable without bypassing the required decision controls.**

Traceability: `DLV-FR-WFA-004`, `FR-WFA-004`, `GFR-004`, `GFR-009`,
`GFR-012`, `NFR-CAP-007`, `NFR-PERF-013`, `NFR-OBS-006`–`NFR-OBS-008`,
`NFR-AUD-001`–`NFR-AUD-003`, `WFA-WS-01`, `WFA-SCR-04`.

Acceptance criteria:

- [x] `EscalateApproval` can be invoked only for a request/step eligible for
  escalation under its effective policy, such as the policy-defined overdue or
  blocked condition, and only by an authorized actor or approved domain
  process.
- [x] Escalation records the request/step identity, current policy and subject
  versions, escalation reason, responsible owner/target, actor or process, and
  timing/evidence required by the policy. It does not alter the subject fact or
  record an approval decision.
- [x] Escalation does not skip a required step, broaden authorization, remove
  segregation controls, or make a downstream business effect authoritative.
- [x] The request and worklist expose escalated, due, blocked, owner, age,
  blocker, next action, and recovery information. The original request and
  prior decision/delegation history remain visible.
- [x] Repeating the same escalation identity and fingerprint returns the
  established result. Concurrent escalation, decision, delegation, or request
  changes return a typed conflict and do not silently replace the established
  outcome.
- [ ] The transaction records the escalation and its `ApprovalEscalated`
  outbox message atomically. Duplicate/out-of-order delivery creates at most
  one local effect and remains explainable to operations. The PostgreSQL
  transaction path is implemented, but persistence/outbox integration evidence
  remains pending.
- [ ] Notification and operational telemetry expose approval age, oldest items,
  owner, escalation path, and typed domain/authorization/concurrency/
  dependency/internal outcomes without unbounded metric labels or sensitive
  payloads.
- [x] `WFA-SCR-04` shows escalation evidence and safe recovery. Automatic
  escalation workers are included only where the approved policy/technical
  contract explicitly requires them; this story does not invent a scheduler
  rule.

Likely implementation areas:

- `internal/workflow/` escalation command validation, request/step transition,
  outbox mapping, observability fields, and audit integration. Escalation must
  remain within the approved aggregate/event model rather than becoming a new
  root.
- `contracts/openapi/paths/approvals.yaml` and generated artifacts for the
  escalation action.
- `web/src/capabilities/approvals/` for blocked/due/escalated worklist states
  and `WFA-SCR-04`.
- `internal/platform/worker` only if a separately approved process contract
  requires automated escalation.

Required evidence:

- Domain/API tests for eligible and ineligible escalation, overdue/blocked
  boundaries, no-decision side effects, duplicate delivery, concurrent changes,
  and dependency-unavailable outcomes.
- Outbox/inbox and recovery tests proving escalation is durable and replay-safe.
- Worklist, telemetry, notification, accessibility, authorization, and
  sensitive-data evidence for due/blocked/escalated states.

#### User Story 5 implementation evidence (2026-10-15)

- Added policy-bound `EscalateApproval` behavior to the existing
  `ApprovalRequest` aggregate. An escalation preserves the pending request and
  immutable subject/policy evidence, transitions only the actionable step to
  `Escalated`, derives responsible/target roles from the copied policy, and
  records actor or approved-process, timing, reason, blocker, correlation,
  causation, and audit references. It never records a decision or downstream
  business effect.
- Added the immutable `workflow.approval_request_escalation` child table and
  SQLC model. PostgreSQL save ordering writes the audit fact first, then the
  escalation child, outbox event, and durable idempotency finalization in the
  same transaction; the migration includes an immutability trigger and
  evidence checks.
- Added typed `wfaEscalateApproval` API/OpenAPI contracts, generated Go and
  TypeScript artifacts, `finance.wfa.escalate.approval` authorization, safe
  `approval_escalated` outbox payload fields, and typed conflict, authorization,
  validation, not-found, and dependency outcomes.
- Added the policy-derived worklist/detail projections and WFA-SCR-04 form for
  escalated, overdue, blocked, age, owner, target, blocker, next-action,
  recovery, actor/process, and audit evidence. Retry identity remains stable
  after transport failure and established results identify replay.
- Added the approved one-minute/100-item overdue worker in `internal/workflow`
  with exact process identity, bounded candidate selection, deterministic
  escalation/idempotency identities, and race-safe skip behavior.
- Focused evidence covers eligible/ineligible actor escalation, blocked and
  overdue boundaries, authorization denial without mutation, immutable
  evidence, no-decision side effects, replay/idempotency conflict, typed API
  response/replay, blocked worklist projection, and deterministic worker
  processing.
- Verified locally with the focused Go tests, migration validation/checksum
  checks, SQLC drift check, OpenAPI Go generation check, TypeScript generation
  check, TypeScript compilation, and `git diff --check`. The exact commands
  and outcomes are recorded in
  [`DLV-FR-WFA-004-us5-escalate-approval.md`](../../verification/DLV-FR-WFA-004-us5-escalate-approval.md).
- Not claimed in this story slice: PostgreSQL/Testcontainers execution,
  production `cmd/worker` composition with the repository's external audit
  writer, duplicate/out-of-order inbox delivery qualification, notification
  delivery, operational metrics/dashboard qualification, live IAM/SoD,
  witnessed accessibility/performance review, or full release gates.

## 8. Cross-story definition of done

The epic is ready for delivery review only when all applicable criteria below
have evidence. A checked item means verified implementation evidence, not merely
that the story was planned.

- [ ] All five approved WFA operation routes, handlers, permissions, and
  OpenAPI schemas match the technical specifications.
- [ ] `internal/workflow` is the only owner of the `workflow` schema's
  `ApprovalPolicy`, `ApprovalRequest`, and `Delegation` records. No consuming
  module writes those tables directly.
- [ ] Domain transitions enforce authorization, applicable policy version,
  segregation, expected version, idempotency, immutable history, and typed
  rejection/conflict/unavailable outcomes.
- [ ] `ApprovalRequested`, `ApprovalDecisionRecorded`, `ApprovalDelegated`,
  and `ApprovalEscalated` are written through the transactional outbox with
  safe payloads and durable delivery/retry behavior.
- [ ] Consuming-context tests prove that a decision is applied only after
  current-state revalidation and that duplicate or stale decisions do not
  create a second business effect.
- [ ] Audit evidence is attributable, append-only, access-controlled, and
  queryable with the policy/request/decision/delegation/escalation lineage.
- [ ] `WFA-WS-01` and `WFA-SCR-01`–`WFA-SCR-04` expose current state, allowed
  and blocked actions, owner, blocker, age, evidence, and recovery path.
- [ ] Approval inbox, subject, and decision views meet `NFR-PERF-007`; open
  approval capacity meets `NFR-CAP-007`; notifications meet `NFR-PERF-013`.
- [ ] Duplicate, out-of-order, stale, failed, and ambiguous outcomes are
  recoverable without deleting or rewriting an established approval fact.
- [ ] Keyboard, screen-reader, contrast, zoom, focus, validation, and fixed
  expiry behavior satisfy the applicable `NFR-ACC-*` requirements.
- [ ] Logs, traces, metrics, notifications, events, and exports minimize and
  mask sensitive data and distinguish domain rejection, authorization denial,
  concurrency conflict, dependency unavailability, capacity control, and
  internal failure.
- [ ] Verification records identify what was locally tested and what remains
  deferred to PostgreSQL runtime, downstream cross-context, live identity,
  production audit, performance/capacity, recovery, or release qualification.

## 9. Ordered implementation plan

1. Resolve the open policy semantics listed in Section 10 before freezing domain
   states or persistence constraints.
2. Establish the `internal/workflow` domain/application boundary and ports for
   IAM authorization/segregation, audit evidence, policy lookup, outbox, and
   idempotency/concurrency using existing platform primitives.
3. Deliver User Story 1 first so request creation has a versioned policy source
   and an explicit independent-approval path for policy administration.
4. Deliver User Story 2 with the `ApprovalRequested` contract, immutable subject
   reference/snapshot, request worklist, and durable retry behavior.
5. Deliver User Story 3 with immutable decision recording and the consumer
   application contract; verify the revalidation boundary with a synthetic
   consuming aggregate before wiring additional finance contexts.
6. Deliver User Story 4 with effective-dated delegation, scope/SoD checks, and
   delegated worklist behavior.
7. Deliver User Story 5 with policy-driven escalation, typed owner/blocker
   states, operational telemetry, and replay-safe publication.
8. Complete API-generated artifacts, PostgreSQL adapters, UI routes, shared
   components, audit evidence, notifications, and observability as one
   vertical slice rather than treating WFA as backend-only work.
9. Run focused domain, API, persistence, outbox/inbox, authorization,
   accessibility, and frontend tests; then record release/qualification gaps
   without marking roadmap delivery items complete.

## 10. Risks and decisions required before implementation

- **Policy administration bootstrap:** The PRD requires independent approval
  for policy changes, but the current WFA operation list has no separate
  policy-activation decision command. Confirm the bootstrap/independent
  approval path before implementing activation; do not allow self-approval or
  silently bypass the requirement.
- **Policy overlap and precedence:** The sources require valid applicability and
  effective dates but do not define the complete conflict/precedence algorithm.
  Confirm whether overlapping policies are prohibited, ordered, or resolved by
  another approved rule.
- **Delegation ending/revocation:** UX `WFA-SCR-04` names create/revoke
  delegation, while the approved operation list names only `DelegateApproval`.
  Confirm whether ending/revoking is a mode of that contract or a future
  requirement before adding an operation identifier.
- **Delegation retroactivity:** Confirm whether a delegation affects only new
  requests or also already-created actionable steps. Do not infer retroactive
  behavior from the worklist requirement.
- **Escalation trigger and target:** Confirm the policy source for due/blocked
  thresholds, target owner selection, and whether automation is required.
  `EscalateApproval` must not gain an unapproved scheduler or target model.
- **Decision application acknowledgement:** Confirm how consuming contexts
  publish or expose `AppliedAt` and application status back to authorized WFA
  views while preserving the consuming context as the lifecycle owner.
- **Milestone mapping conflict:** `EP-WFA-001` and `DLV-FR-WFA-001`–`005` are
  assigned to M3 in the delivery plan and roadmap, but the delivery-plan row
  for `DLV-GFR-004` lists M2. This story map retains both source values and
  does not resolve the planning baseline.

## 11. Dependencies

The delivery plan identifies these prerequisites:

- `EP-PLAT-001` — repository, OpenAPI, PostgreSQL, idempotency, outbox/inbox,
  testing, and shared technical primitives.
- `EP-IAM-001` — authenticated actor, scoped authorization, and segregation
  decisions.
- `EP-UX-001` — shared routing, worklists, forms, tables, dialogs, and
  accessibility surfaces.
- `EP-OPS-001` — structured logs, traces, metrics, dashboards, and runbook
  foundations.

Downstream consumers include `EP-FPM-001`, `EP-AR-001`, `EP-AP-001`,
`EP-PCM-001`, `EP-FA-001`, `EP-REV-001`, `EP-TAX-001`, `EP-PAYR-001`, and
other approval-bearing capabilities. Their application commands remain in
their owning modules. `EP-AUD-001` later provides full audit-integrity
qualification; WFA still requires the approved audit port and material-action
evidence at its first applicable slice.

## 12. Planning verification

- [x] Epic, delivery, operation, API, permission, event, module, schema, and
  UX identifiers were checked against the repository specifications.
- [x] The five stories cover every `FR-WFA-*` delivery item without adding a
  new WFA requirement identifier.
- [x] Bounded-context ownership and the decision-application boundary are
  explicit.
- [x] Open semantics that would materially affect implementation are recorded
  as risks instead of being invented.
- [x] User Story 4 implementation, generated artifacts, focused tests, runtime
  wiring, and delivery evidence are recorded on the implementation branch.
- [x] User Story 5 implementation, generated artifacts, focused tests, and
  delivery evidence are recorded on the implementation branch.
- [ ] Full-epic integration and release/qualification evidence remain open.

No commit or roadmap completion checkbox is implied by this planning artifact.
