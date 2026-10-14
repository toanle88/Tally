# Learning guide

TALLY is intentionally built as a learning project. The useful unit of
progress is a repeatable vertical slice: domain behavior, persistence, API,
interface, authorization, tests, observability, recovery evidence, and clear
limits on what was not verified.

This page explains the project at three levels: the business problem, the
technology used to explore it, and the architecture that keeps responsibility
clear. The current checkout is an implementation slice, not the complete
finance platform described by the target specifications.

## How to read the repository

Before following a feature, separate three kinds of truth:

1. **Approved target:** the DDD, PRD, UX, NFR, system-design, and technical
   specification documents define the intended finance platform.
2. **Repository reality:** source files, migrations, generated artifacts, and
   route wiring show what this checkout actually contains.
3. **Verification evidence:** a command record states what was tested, where,
   when, and with what limitation. A passing local check is not a production
   qualification claim.

When these sources appear to disagree, do not silently promote a planned
contract or a mock screen into an implemented capability. Start with
[current status and evidence](/current-status), inspect the source tree, then
read the relevant verification record and authoritative specification.

## Business: what TALLY models

### The business problem

TALLY is learning how to turn business activity into trustworthy financial
records. That is more demanding than storing rows such as `amount = 100` or
`status = approved`. A finance system must also answer:

- Which legal entity, book, and accounting scope does this activity belong to?
- Which accounts and master-data values are valid at the time of the action?
- Who requested, approved, or executed the action, and were they authorized in
  that scope?
- What happens if the request is retried, the policy changes, the period
  closes, or two people edit the same record?
- How can an established fact be corrected without erasing the history that
  explains it?

TALLY therefore treats business rules, ownership, authorization, persistence,
and evidence as one problem. The UI is only one way to observe or initiate a
business operation; it is not the source of truth.

### Core business concepts

| Concept | Meaning in the domain | Why it matters |
|---|---|---|
| Accounting scope | The legal-entity, organization, or book context in which an action is evaluated | The same actor can be allowed in one scope and denied in another |
| Master data | Parties, legal entities, fiscal calendars, and other governed reference data | Financial transactions need stable, valid context rather than free-form labels |
| Chart of accounts | The controlled vocabulary of accounts and segments used to classify activity | It prevents postings from using invalid or ungoverned combinations |
| General Ledger | The accounting boundary that validates, approves, posts, reverses, and controls journal facts | It is the authoritative producer of final journal effects |
| Approval policy | A versioned rule describing when an operation needs approval and which evidence is relevant | A request must be evaluated against the policy that was effective for it |
| Approval request | A durable request to perform a governed operation, with subject and policy evidence | A later decision can refer to what was requested without reconstructing the past |
| Established fact | A posted or otherwise committed financial result | It must not be silently edited; corrections become new linked facts |

The repository currently implements slices of these concepts. It does not
claim to be a complete accounting suite: invoices, payments, subledgers, tax,
payroll, reporting, and other bounded contexts remain target scope.

### Domain vocabulary

These terms recur throughout the code and specifications:

| Term | Practical meaning in TALLY |
|---|---|
| Bounded context | A business boundary with its own language, rules, persistence, and authoritative facts |
| Aggregate | A consistency boundary whose state and invariants are changed through an owning model or application command |
| Command | An intentional request to change state, such as `CreateApprovalRequest` or `PostJournal` |
| Event | A durable statement that a state change occurred and may be consumed by another owner |
| Subject | The business record or operation an approval request concerns; workflow references it but does not own it |
| Subject evidence | A bounded, immutable snapshot/reference retained so later review does not depend on mutable source data |
| Policy revision | A versioned, effective-dated approval rule set; an existing request remains bound to the version resolved at creation |
| Established fact | A posted, accepted, settled, reconciled, or otherwise committed result that must not be destructively edited |
| Correction lineage | The explicit links between an original fact and its reversal, adjustment, amendment, replacement, or compensation |

The glossary is intentionally business-oriented. A class or table name is not
automatically a domain owner: ownership comes from the bounded-context rules
and the authoritative command/event contracts.

### Example: from business request to accounting effect

Consider a user asking to perform an operation that may eventually create a
journal entry:

1. The user selects an accounting scope and submits the operation with the
   required business identity and request fingerprint.
2. The owning module checks the operation's business rules, the applicable
   accounting context, and the user's authorization.
3. If approval is required, the workflow module creates an approval request.
   The current User Story 2 slice retains the subject reference, a bounded
   subject-evidence snapshot, and the effective policy version. This makes the
   request understandable even if source data or policy changes later.
4. In the target workflow, an authorized decision would move the request
   toward the owning business effect. Decision, delegation, and escalation are
   not claimed by the current request slice.
5. When the General Ledger establishes a journal fact, that fact is not
   overwritten to correct a mistake. A reversal, adjustment, amendment,
   replacement, or other explicit correction creates a traceable successor.

This example demonstrates an important separation: workflow governs whether
an operation may proceed, while the owning finance module remains responsible
for the resulting business fact.

### What is implemented and what is planned

The current repository contains platform, identity/access, organization,
chart-of-accounts, General Ledger, and initial workflow policy/request slices.
The approval-request slice does not claim decision, delegation, or escalation
behavior. Invoices, payments, subledgers, tax, payroll, reporting, and full
release qualification remain planned delivery scope. See [current status and
evidence](/current-status) and [roadmap and progress](/roadmap) for the
verified boundary.

## Technology: how the system is built

The approved technology baseline is deliberately conventional so the domain
rules remain visible:

| Concern | Technology and learning purpose |
|---|---|
| API and services | Go 1.26 with `net/http` and `chi`, organized as a modular monolith |
| Web interface | React, TypeScript, Vite, Tailwind, and shared finance interaction components |
| Persistence | PostgreSQL with Goose migrations, `pgx` connections, and `sqlc` typed queries |
| API boundary | OpenAPI 3.1 with generated Go server artifacts and a generated TypeScript client |
| Finance values | Exact decimal arithmetic and PostgreSQL `NUMERIC`; binary floating point is not used for money |
| Reliability | Scoped idempotency, optimistic concurrency, transactional outbox/inbox, retries, and result lookup |
| Authentication | Entra/OIDC boundaries with application-owned authorization and audit evidence |
| Local delivery | Docker Compose, pnpm, Go tests, Vitest, Playwright, and Testcontainers where the relevant gate is enabled |

The table combines current implementation choices with the approved target
baseline. A technology appearing here may be represented by a local adapter,
fixture, contract, or focused test rather than a production-qualified
integration.

### Generated code: source versus artifact

The repository contains a lot of generated code because it has two contracts
that are easy to get out of sync when maintained by hand: the HTTP API and the
database query boundary. Generated code is a checked-in artifact, not a second
source of truth.

```text
OpenAPI contract ──┬─ Redocly bundle ── ogen ────────────────> Go HTTP package
                   └─ Redocly bundle ── openapi-ts ──────────> TypeScript client

SQL migrations + queries + sqlc.yaml ── sqlc ───────────────> typed Go DB packages
```

The important inputs and outputs are:

| Authoritative input | Generated output | What the output provides |
|---|---|---|
| `contracts/openapi/openapi.yaml` and its components, paths, and examples | `internal/platform/httpapi/generated/` | Go request/response types, validators, routers, encoders, and handler interfaces |
| The same OpenAPI contract | `web/src/generated/api/` | TypeScript schemas, Fetch client, SDK functions, and serialization helpers |
| `db/migrations/**`, `db/queries/**`, and `sqlc.yaml` | `internal/**/**/*db/` packages such as `workflowdb` and `identitydb` | Typed Go models, query methods, and database argument/result mappings |

Generated files normally begin with a marker such as `Code generated by
ogen, DO NOT EDIT` or `This file is auto-generated by
@hey-api/openapi-ts`. That instruction is important: edit the OpenAPI or SQL
input, then regenerate. Handwritten application services, domain types,
repositories, HTTP wiring, and React screens consume the generated types and
add business behavior around them. Finance rules do not belong in generated
routers, SDKs, or query models.

The artifacts are committed deliberately. A fresh checkout can compile and
review the exact API/database boundary without asking every developer to run a
generator first, while CI can compare the committed files with deterministic
fresh output. This makes a stale artifact, deleted file, unexpected file, or
manual edit visible as a reviewable drift problem rather than a runtime
surprise.

### How to work with generated code

Use the source-to-artifact workflow below:

1. Change the authoritative input: the OpenAPI contract for an API shape,
   migrations or SQL for persistence, or handwritten application code for
   business behavior.
2. Run the matching pinned generator and inspect the complete diff. A large
   generated diff can be legitimate when a shared schema changes, but it
   should be explainable from the source change.
3. Run the deterministic check before considering the slice complete. The
   check regenerates into a temporary directory and compares inventories and
   contents, so it also detects missing or extra artifacts.

| Task | Generate | Verify |
|---|---|---|
| Go HTTP server from OpenAPI | `make api-generate` | `make api-generate-check` |
| TypeScript API client from OpenAPI | `make api-ts-generate` | `make api-ts-check` |
| Go database packages from SQL | `make db-sqlc-generate` | `make sqlc-check` |

The repository pins the generator versions and wraps them in scripts so local
and CI generation use the same inputs and options. For example, `api-generate`
bundles the OpenAPI document before invoking `ogen`, while `api-ts-generate`
uses the bundled document with the Fetch client. The SQL workflow uses
`sqlc.yaml` to map each owned schema and query directory to its generated Go
package.

### Repository map: where to look next

Once the generated boundary is clear, use this map to find the handwritten
behavior around it:

| Learning question | Start here | What to look for |
|---|---|---|
| What does the API promise? | `contracts/openapi/` | Paths, schemas, security, errors, examples, and operation identifiers |
| Where is HTTP behavior wired? | `internal/platform/httpapi/` | Handwritten handlers, adapters, authorization context, and generated interfaces |
| Where is workflow behavior owned? | `internal/workflow/` | Policy/request models, application services, resolution ports, repositories, and tests |
| Where is GL behavior owned? | `internal/gl/` | Ledger, book, account, posting, reversal, and close-control rules |
| How is the database shaped? | `db/migrations/` and `sqlc.yaml` | Owned schemas, constraints, indexes, and schema-to-package mappings |
| Which SQL is executed? | `db/queries/` | Named queries, expected-version predicates, idempotency lookups, and result reads |
| How does the user experience the slice? | `web/src/app/` and `web/src/lib/` | Feature workspaces, shared state/evidence components, auth, and error recovery |
| Where are API calls typed? | `web/src/generated/api/` | Generated client functions and request/response types; do not hand-edit them |
| What proves the behavior? | `*_test.go`, `web/src/**/*.test.*`, and `docs/verification/` | Unit, component, contract, persistence, accessibility, and operational evidence |
| How is the workflow automated? | `Makefile`, `scripts/`, and `.github/workflows/` | Pinned tools, deterministic checks, CI boundaries, and release gates |

The directory map is a navigation aid, not a promise that every target domain
has an implementation. A route in the OpenAPI catalog, a fixture in the UI,
or a generated type can exist before its authoritative business aggregate does.

### Why the pieces fit together

The stack is designed to make a vertical slice easy to inspect:

- Go keeps HTTP handlers, application services, ports, and domain decisions
  explicit. The modular-monolith process is easy to run locally while still
  making ownership visible in the package layout.
- PostgreSQL provides transactions, constraints, locking, and durable state.
  Goose makes schema evolution reviewable, while explicit SQL and `sqlc` keep
  query behavior visible and type-checked instead of hiding it behind a large
  runtime ORM.
- OpenAPI is the shared boundary between the Go API and the React application.
  Generated artifacts make contract drift detectable during checks rather than
  waiting for a browser flow to fail.
- React and the shared finance components make lifecycle, evidence, conflict,
  privacy, and accessibility states visible instead of rendering every result
  as a generic success or error.

### Finance-specific technology choices

Money is not represented by a binary floating-point number. Decimal values are
kept exact, travel with their currency, and are stored in PostgreSQL `NUMERIC`
columns with the relevant precision and rounding rules. This prevents values
that look correct on screen from becoming different amounts when accumulated,
rounded, or serialized.

State-changing operations also need more than an HTTP status code:

- An idempotency key and canonical request fingerprint distinguish a retry of
  the same command from a different command accidentally reusing an identity.
- An aggregate version or expected version exposes a stale write as a conflict
  instead of silently overwriting another user's work.
- An audit reference records material authorization and state-change evidence;
  logs and traces help operate the system but are not the authoritative audit
  record.
- A transactional outbox records publication intent in the same transaction as
  the local business change. An inbox and result lookup make duplicate or
  delayed delivery observable and safe to retry.

### Verification layers

The repository deliberately has several levels of evidence:

1. Unit tests exercise domain rules and technical primitives quickly.
2. Generated-artifact and OpenAPI checks catch contract or SQL drift.
3. PostgreSQL/Testcontainers checks exercise migrations, constraints, and
   transaction behavior.
4. Vitest, Playwright, and accessibility checks exercise the browser boundary.
5. Operational and release checks cover workers, telemetry, infrastructure,
   recovery, and qualification limits.

Not every layer is run by every local command. A passing focused test or docs
build is useful evidence for that layer only; it is not evidence that live
Entra, production infrastructure, capacity, disaster recovery, or the full
finance domain has been qualified.

### Verification matrix

Use the smallest check that answers the question you are asking, then add the
broader gate when the change requires it:

| Command | What it proves | What it does not prove |
|---|---|---|
| `go test ./internal/workflow ./internal/platform/httpapi ./cmd/api` | Focused workflow and API behavior compiles and passes unit/handler tests | PostgreSQL runtime behavior, browser behavior, or release qualification |
| `make sqlc-compile` | SQL schema/query inputs parse and type-check | That committed generated output is current |
| `make sqlc-check` | Committed SQLC artifacts match deterministic regeneration | PostgreSQL constraints or transaction behavior under a live database |
| `make api-generate-check` | OpenAPI produces deterministic Go artifacts with the expected inventory | The handler implements every generated operation |
| `make api-ts-check` | OpenAPI produces deterministic TypeScript artifacts and the client compiles | Browser authentication or end-to-end UI behavior |
| `pnpm --dir web build` | The frontend TypeScript/build boundary compiles | A user can complete every route or accessibility qualification |
| `pnpm docs:build` | VitePress pages and internal links render | Application code or hosted GitHub Pages deployment |
| `make persistence-check` | The opt-in persistence drift/release gate, including database checks, passes | Production capacity, recovery, security, or live cloud qualification |

The repository labels some persistence, browser, and release checks as opt-in
because they are slower and require external services. Do not describe an
unexecuted gate as passed.

## Architecture: how ownership works

TALLY is a modular monolith: one deployable Go application contains modules
with explicit bounded-context ownership. Deployment is shared; business
authority and database ownership are not.

### Modular monolith, not shared-everything

The application is deployed together, but it is not one undifferentiated
business package. Each bounded context owns the language, invariants, state,
repositories, migrations, and adapters for its area. This gives a solo learner
one process and one database to operate while preserving the boundaries that
make later integration or service extraction understandable.

The key rule is ownership, not physical deployment. A module must not reach
into another module's adapter or write directly to another module's tables just
because both happen to run in the same process. Shared code belongs in
`internal/platform` only when it is a technical primitive, not when it hides a
finance decision.

| Module | Owns the business boundary |
|---|---|
| `internal/identity` | Users, roles, policies, authorization, and access evidence |
| `internal/organization` | Legal entities, parties, calendars, and master data |
| `internal/coa` | Account segments, values, combinations, and COA governance |
| `internal/gl` | Ledgers, books, accounts, posting, reversals, and close controls |
| `internal/workflow` | Approval policies, request evidence, and workflow request lifecycle |
| `internal/platform` | Technical primitives such as database, HTTP, idempotency, events, and telemetry |

The owning module is the sole producer of its business facts. Other modules
use published interfaces or integration messages rather than reaching into
another module's adapter or tables. Shared packages provide technical
primitives; they do not become a second owner of finance rules.

For example, workflow can retain a subject reference and bounded evidence for
an approval request without becoming the owner of the subject's master-data
record. General Ledger can own the final journal effect without making
identity, organization, or workflow tables its source of truth. These
boundaries prevent a convenient join today from becoming an invisible
cross-context dependency tomorrow.

### Persistence and integration flow

For a state-changing capability, the path from source to durable effect is:

```text
Migration + SQL query + sqlc.yaml
              ↓
       generated DB package
              ↓
     handwritten repository adapter
              ↓
      application service/handler
              ↓
  local transaction: state + audit + idempotency + outbox
              ↓
      worker dispatches after commit
              ↓
   consumer inbox + local effect + result
```

The generated database package knows how to encode a query and map a row. The
repository adapter decides how that query participates in the owning module's
transaction. The application service decides whether the business operation
is allowed and valid. Keeping those decisions separate makes it possible to
review both the SQL mechanics and the business invariant.

### The consistency tools and the problem each solves

| Tool | Problem it addresses |
|---|---|
| Local transaction | The owning module must commit its related state atomically |
| Authorization evidence | A material change must explain who was allowed to do it and under which policy |
| Idempotency | A client or worker retry must not create a second business effect |
| Optimistic concurrency | A stale screen must not silently overwrite a newer change |
| Transactional outbox | A committed state change must not lose its integration publication intent |
| Inbox and result lookup | Duplicate or delayed messages must be handled deterministically |
| Immutable correction lineage | A correction must preserve the original established fact and its history |

### A state-changing request, end to end

When studying the current approval-request slice, follow this concrete path:

1. The React UI sends an OpenAPI-shaped command with the selected scope,
   request identity, and required subject reference.
2. HTTP middleware authenticates the actor and the workflow application layer
   validates the request shape, resolves the effective policy, and checks the
   operation's authorization boundary.
3. The workflow transaction persists the request, immutable bounded subject
   evidence, idempotency result, audit evidence, and any outbox publication
   intent that belongs to the command.
4. A repeated command returns the established result or a typed conflict; it
   does not create a second request for the same business identity and
   fingerprint.
5. Authorized worklist and detail reads show the request lifecycle and
   evidence without bypassing the workflow owner. A later workflow decision is
   intentionally outside this slice.

For any other feature, use the same questions: who owns the fact, where is the
invariant enforced, what transaction establishes it, how is a retry recognized,
what evidence is retained, and what does the UI show when the operation is
denied, stale, unavailable, or already completed?

### Worked example: User Story 2 file map

The current approval-request slice is a useful first feature to trace because
it crosses the complete boundary without claiming the later approval decision.

| Slice concern | Repository location |
|---|---|
| Functional intent and acceptance | `docs/backlog/stories/EP-WFA-001_user_stories.md`, User Story 2 |
| API source contract | `contracts/openapi/paths/approvals.yaml` |
| HTTP boundary | `internal/platform/httpapi/workflow_approval_request_handler.go` |
| Request/policy behavior | `internal/workflow/approval_request.go`, `approval_request_resolver.go`, and related services/tests |
| Database schema and SQL | `db/migrations/workflow/` and `db/queries/workflow/` |
| Generated database types | `internal/workflow/workflowdb/` |
| Generated API types | `internal/platform/httpapi/generated/` and `web/src/generated/api/` |
| Browser worklist/detail | `web/src/app/workflow-approval-workspace.tsx` |
| Focused verification | `internal/workflow/*_test.go`, `internal/platform/httpapi/*workflow*_test.go`, and the web route tests |

The runtime flow is then:

1. `POST /api/v1/approvals/actions/create-approval-request` receives a
   command with scope, subject, operation, and request identity.
2. The workflow resolver selects exactly one effective policy using the
   applicable dimensions, amount/currency, scope, and effective interval.
3. The request stores the selected policy version, copied steps, subject
   version, bounded snapshot/fingerprint, responsible roles, and lifecycle
   evidence.
4. The repository commits the request and `approval_requested` outbox intent
   atomically. A repeated identity/fingerprint returns the established result;
   changed functional content returns an idempotency conflict.
5. Authorized reads use `GET /api/v1/approvals/reference/requests` and
   `GET /api/v1/approvals/reference/requests/{approvalRequestId}` to populate
   the worklist and read-only detail view.

This is a delivered create/read slice. The decision, delegation, escalation,
assignment, and full exception-recovery paths are intentionally later scope.

This is the main architectural lesson: a shared deployment can remain simple
to run while still preserving boundaries that would matter in a future
distributed integration model.

## Failure, recovery, and evidence

Finance software must make unsuccessful and repeated operations understandable.
The useful question is not only “did the request return 200?” but also “what
state exists now, what evidence explains it, and what can be retried safely?”

| Outcome | Meaning | Learning point |
|---|---|---|
| Validation rejection | The command is malformed or violates a business rule | Reject before a partial business effect is created |
| Authorization denied | The actor is authenticated but cannot perform this operation in this scope | Authentication and authorization are different checks; denial is a deliberate result |
| Dependency unavailable | A required policy, subject, or source decision cannot be established | Fail closed and preserve a typed recovery path rather than guessing |
| Idempotent duplicate | The same business identity and fingerprint was already established | Return the existing result instead of creating a second effect |
| Idempotency conflict | The identity was reused with different functional content | Refuse the changed command and require a new business identity |
| Version conflict | The screen or source fact is stale | Refresh and retry deliberately; never silently overwrite newer state |
| Outbox retry | Local state committed but publication still needs dispatch | Durable publication intent survives worker failure and can be retried |
| Correction | An established fact needs to change | Create linked reversal/adjustment/replacement evidence instead of editing history |

The current approval-request slice demonstrates typed rejection/unavailable
outcomes, duplicate/result behavior, changed-fingerprint conflict, and atomic
request/outbox persistence. Source-context revalidation immediately before
creation, user assignment, decision, delegation, escalation, and full exception
recovery remain later qualification boundaries.

## Security, privacy, and evidence

The same operation has several security questions, and each has a different
owner:

| Concern | Question | Evidence or boundary |
|---|---|---|
| Authentication | Who is the external subject? | Entra/OIDC validation or the explicitly configured local fixture identity |
| Authorization | May that actor perform this operation in this scope? | Application-owned grants, policies, segregation checks, and typed allow/deny outcomes |
| Audit | What material change was authorized and recorded? | Actor, scope, policy/authorization references, correlation/causation, fingerprints, and audit reference |
| Privacy | Which fields may be displayed or exported? | Masking, permission-filtered views, data classification, and bounded subject evidence |
| Operations | What happened while the system processed the request? | Structured redacted logs, traces, metrics, health, backlog, and worker state |

Logs and traces are operational telemetry. They help diagnose timing, retries,
and failures, but they do not replace the authoritative audit record. Evidence
must not contain raw access tokens, credentials, policy secrets, or protected
business values merely to make a screen convenient. The UI should show the
least information needed for the authorized task and make the source,
classification, and recovery path understandable.

## Observability and operations

The platform treats observability as part of the feature rather than a final
dashboard exercise:

- Structured logs carry bounded request, command, database, and outbox
  correlation without leaking sensitive fields.
- OpenTelemetry traces connect an HTTP request to application work, database
  operations, and worker dispatch where that context exists.
- Metrics and health views expose service state, outbox backlog, retries,
  failures, and business-result categories rather than only process uptime.
- Runbooks describe how an operator distinguishes a denied command, a stale
  version, a blocked outbox row, an unavailable dependency, and a duplicate
  delivery.
- Recovery evidence explains what was replayed, reconciled, corrected, or
  intentionally left unverified.

The local and Azure learning profiles are useful for practicing these signals.
They are not evidence of production availability, retention, capacity, backup,
restore, or disaster-recovery qualification.

## Architecture trade-offs

Every major choice is a learning trade-off, not a universal rule:

| Decision | Benefit | Cost or limitation |
|---|---|---|
| Modular monolith | One process and deployment are affordable for a solo project while ownership remains visible | Module boundaries require discipline even though direct in-process calls are easy |
| PostgreSQL outbox/inbox | Transactions, durable state, retries, and integration intent can be learned without operating a broker | Database-backed dispatch has throughput and operational limits that a later broker/architecture may address |
| OpenAPI and generated clients | Go and TypeScript share a reviewable contract and drift is mechanically detected | Generated artifacts add files and require disciplined regeneration |
| Explicit SQL and `sqlc` | Queries, constraints, and ownership remain visible and type-checked | Developers must understand SQL and migration compatibility instead of relying on ORM conventions |
| Exact decimal values | Monetary calculations remain deterministic and auditable | Rounding, currency precision, serialization, and display require deliberate handling |
| Local-first infrastructure | Fast, low-cost feedback and disposable Azure practice | Local success does not establish cloud availability, security, or recovery targets |
| Application-owned authorization | Finance rules can use scope, sensitivity, history, and policy evidence beyond identity-provider claims | The application must maintain and qualify its own policy, audit, and revocation behavior |

These trade-offs are recorded in the [architecture page](/architecture), the
[finance integrity page](/finance-integrity), and the [system design
documents](/sources).

## Hands-on learning path

### Start a local learning session

For the complete environment setup, use [local development](/development). A
typical sequence is:

```bash
pnpm --dir web --ignore-workspace install --frozen-lockfile
cp .env.example .env
make db-up
make db-prepare
pnpm dev-api
pnpm dev-web
```

Run the API and web commands in separate terminals. If you are only studying
the documentation, `pnpm docs:dev` starts the VitePress site without requiring
the finance runtime.

### Exercises

1. Find `CreateApprovalRequest` in the WFA story, then locate its OpenAPI path,
   HTTP handler, workflow service, migration, SQL query, generated database
   model, and React workspace.
2. Run `make api-generate-check` and `make api-ts-check`. Inspect a generated
   type, then identify the OpenAPI source that produced it; do not edit the
   generated file.
3. Run `make sqlc-compile` and `make sqlc-check`. Find the expected-version and
   idempotency predicates in the handwritten SQL.
4. Follow an authorization permission from the API route into the identity
   policy decision and then into the UI's denied or masked state.
5. Read the request repository transaction and identify where request state,
   subject evidence, audit reference, idempotency result, and outbox intent
   become one local effect.
6. Compare a normal result, duplicate result, changed-fingerprint conflict,
   unavailable dependency, and stale-version outcome. Record which layer owns
   the decision and what a user can do next.
7. Run `pnpm docs:build` after changing a documentation page and confirm that
   no generated documentation output is accidentally added to Git.

The goal of each exercise is not code volume. It is being able to explain the
business rule, owner, source input, generated artifact, transaction boundary,
verification command, and known limitation.

## Suggested reading path

1. Read [current status and evidence](/current-status) to separate implemented
   behavior, contracts, fixtures, and unverified claims.
2. Read [finance integrity](/finance-integrity) for exact money, immutable
   facts, correction lineage, idempotency, and audit boundaries.
3. Read [architecture](/architecture) for module ownership, persistence,
   integration, and deployment profiles.
4. Read [identity and access](/identity-access) for authentication,
   authorization, segregation, emergency access, and privacy evidence.
5. Read [local development](/development), then trace one vertical slice from
   domain rule to migration/query, API contract, generated client, UI, and
   tests.
6. Read the [approval-request user story](https://github.com/toanle88/Tally/blob/main/docs/backlog/stories/EP-WFA-001_user_stories.md)
   for the concrete current slice, acceptance criteria, implementation map,
   and deferred behavior.
7. Use the [authoritative sources](/sources) to go deeper into the domain,
   product requirements, UX, nonfunctional requirements, and technical
   specifications.

## Lessons so far

- A modular monolith keeps ownership visible while avoiding premature
  distributed-systems cost. A shared deployment does not justify shared table
  writes.
- Exact money, explicit accounting scope, immutable corrections, and
  optimistic concurrency are domain decisions—not late infrastructure polish.
- Authentication answers “who is this subject?” Authorization answers “may
  this operation happen in this scope, with this sensitivity and history?”
  TALLY keeps those boundaries separate and fails closed when a decision cannot
  be established.
- Durable outbox/inbox, idempotency, fingerprints, and result lookup are one
  coordination problem. Retries must return evidence of the established result
  rather than guessing whether a financial effect happened.
- Sensitive-access evidence should carry bounded metadata and references, not
  raw tokens, policy payloads, credentials, or protected values. Operational
  telemetry and authoritative audit evidence have different owners and
  retention rules.
- Shared abstractions should follow repeated concrete use. Foundational
  identity, money, scope, idempotency, evidence, and accessible interaction
  patterns are the deliberate exceptions.
- A local Azure learning profile is useful for infrastructure practice, but
  scale-to-zero and disposable resources do not satisfy production availability
  or recovery targets.

## Decision trail

The main architectural decisions are: modular monolith; PostgreSQL schema
ownership; OpenAPI-first REST; exact-decimal finance values; transactionally
coordinated outbox/inbox; Entra authentication with application-owned
authorization; and local-first, low-cost deployment profiles.

Architecture decisions and their rationale remain in the [system design
documents](https://github.com/toanle88/Tally/tree/main/docs/specs/system_design).
