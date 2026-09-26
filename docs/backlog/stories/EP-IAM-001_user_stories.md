# EP-IAM-001 — Identity and Access User Stories

| Field | Value |
|---|---|
| Epic | `EP-IAM-001` — Identity and access |
| Status | User Story 1 locally implemented and verified; User Story 2 implementation added with verification pending; User Stories 3–7 remain planned |
| Milestone | `M1` — Capability foundation |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Delivery items | `DLV-GFR-002`, `DLV-GFR-003`, `DLV-GFR-015`, `DLV-FR-IAM-001`–`DLV-FR-IAM-006` |
| Owning module | `internal/identity` / `identity` schema |
| Primary users | Security Administrator; Access Approver; Auditor |
| Exit evidence | Entra authentication boundary, deny-by-default authorization, scoped access decisions, segregation controls, emergency-access lifecycle, privacy protection, and audit evidence |

## 1. Outcome

Provide the application-owned identity and authorization capability that maps
authenticated Entra subjects to TALLY users, roles, permissions, accounting
and business scopes, segregation-of-duties rules, and controlled emergency
access.

The capability must make authorization authoritative at the API and domain
boundary. Frontend visibility is advisory only. Every protected action is
evaluated against the applicable access dimensions, denied by default when a
required decision cannot be established, and recorded with enough evidence for
authorized review without exposing restricted values.

## 2. Learning objective

Learn how to add application-owned authorization to a modular monolith without
confusing authentication with permission, leaking sensitive policy or finance
data, bypassing bounded-context ownership, or allowing emergency access to
become an unbounded privilege.

## 3. Scope

- Validate Entra-issued API access tokens and establish a stable application
  actor identity.
- Maintain `User`, `Role`, `AccessPolicy`, and `SegregationRule` aggregates in
  the Identity & Access bounded context.
- Manage user lifecycle, role grants, scoped permissions, access-policy
  versions, segregation rules, and emergency-access grants/revocations.
- Evaluate the applicable dimensions for each protected action:
  legal entity, business unit or segment, account or account class,
  transaction type, amount, currency, fiscal period, data sensitivity, and
  requested action.
- Enforce minimum segregation rules before a protected business action is
  established and return a safe, typed denial reason.
- Enforce time-bound emergency access, automatic expiry, revocation, and
  post-use review.
- Minimize and mask sensitive payroll, tax, bank, personal, and security data
  in views, errors, logs, events, exports, and audit evidence.
- Provide authorization decision references and policy versions to material
  actions and the existing audit boundary.
- Verify the capability with synthetic identities and policy fixtures only.

## 4. Explicit exclusions

- No finance-domain aggregate, journal, payment, payroll, tax, approval, or
  audit-chain implementation owned by another bounded context.
- No replacement of Microsoft Entra ID, password storage, or enterprise MFA
  with application-managed credentials.
- No direct cross-module repository or schema access. Other modules consume
  authorization through approved application ports or decision contracts.
- No frontend capability screens beyond the IAM administration and decision
  explanation surfaces required by the approved UX contract.
- No production tenant, real personal data, payroll data, bank data, tokens,
  credentials, or live provider qualification in local verification.
- No assumption that every access dimension applies to every operation; the
  operation-specific policy determines which dimensions are relevant.
- No claim that production security, penetration, availability, or release
  qualification is complete from local tests alone.

## 5. Architecture and ownership

### 5.1 Owning component

`internal/identity` owns the domain objects, policy evaluation, lifecycle
transitions, and authorization decision contract for Identity & Access. Its
PostgreSQL ownership is the `identity` schema with the `user`, `role`,
`access_policy`, and `segregation_rule` aggregates.

`cmd/api` and the HTTP/authentication adapter validate the external token and
construct the application actor context. They do not own finance permission
rules. The API remains the authoritative enforcement point even when the SPA
hides unavailable actions.

`internal/audit` owns append-only audit-chain storage and integrity operations.
IAM supplies actor, authentication-subject, action, scope, authorization,
policy-version, reason, correlation, and before/after references through the
approved audit port; it does not write the audit schema directly.

The UX layer owns rendering and interaction. IAM owns the meaning of allowed,
denied, expired, revoked, masked, and review-required outcomes.

### 5.2 Domain objects and invariants

- `User` has `UserStatus`, an `AuthenticationSubject`, role assignments, and
  entity access scopes.
- `Role` has permission grants and scope constraints.
- `AccessPolicy` has a version, effective date range, subject scope, resource
  scope, action set, and access rules.
- `SegregationRule` has a conflicting permission set and enforcement mode.
- Authorization is default-deny and uses the current policy version and
  current subject/access state.
- Access removal, role removal, and emergency expiry take effect for
  interactive and noninteractive access within the approved revocation target.
- Emergency grants are time-bound, reason-coded, independently approved where
  possible, limited to four hours by default, and reviewed after use.
- Policy, role, segregation, emergency, and other privileged changes are
  independently attributable and auditable.
- Mutable IAM operations use the existing idempotency and expected-version
  conventions. Established audit and authorization evidence is not destructively
  edited.

## 6. Traceability

| Identifier | Relationship |
|---|---|
| `M1` | Capability-foundation milestone. |
| `EP-IAM-001` | Identity and Access epic. |
| `EP-PLAT-001` | Supplies the API, database, idempotency, audit-context, and verification foundation. |
| `DLV-GFR-002` | Every action is evaluated against applicable authorization dimensions. |
| `DLV-GFR-003` | Prohibited segregation combinations are prevented with a safe denial reason. |
| `DLV-GFR-015` | Sensitive information is shown only to authorized users and minimized in shared evidence/views. |
| `DLV-FR-IAM-001`–`DLV-FR-IAM-006` | User, role, access-policy, segregation-rule, and emergency-access functional actions. |
| `DDD §§2.18, 8, 11` | IAM aggregates, authorization dimensions, segregation, emergency access, and evidence. |
| `UX §§3, 5, 7.18, 10.2` | Role context, scope/action behavior, IAM screens, denial explanation, and sensitive-data presentation. |
| `NFR-SEC-001`–`NFR-SEC-018` | Identity, MFA, default-deny authorization, revocation, sensitive data, emergency access, and security verification. |
| `NFR-PRV-001`–`NFR-PRV-003`, `NFR-PRV-008`–`NFR-PRV-009` | Purpose, minimization, field-level access, synthetic nonproduction data, and notification privacy. |
| `NFR-AUD-001`, `NFR-AUD-003`, `NFR-AUD-010` | Audit evidence for access changes and audit-access activity. |
| `NFR-MNT-001`, `NFR-MNT-003`, `NFR-MNT-006`, `NFR-MNT-008`, `NFR-MNT-010` | Traceability, versioned policy/configuration, authorized administrative change, documentation, and release safety. |
| `NFR-TST-003`, `NFR-TST-006`, `NFR-TST-009`, `NFR-TST-010` | Verification coverage, security testing, release evidence, and post-release checks. |
| `QG-01`, `QG-02`, `QG-03`, `QG-04`, `QG-05`, `QG-06`, `QG-08`, `QG-10` | M1 traceability, correctness, persistence, API, UX, security, observability, and release evidence gates. |

## 7. User stories

### 7.1 User Story 1 — Establish authenticated application identity

**As a TALLY user, I want my enterprise identity to be validated and mapped
to an application user, so that every protected action has an accountable
actor without storing a password in TALLY.**

Acceptance criteria:

- [x] The SPA uses authorization-code flow with PKCE, and the API validates
  issuer, tenant, audience, signature, algorithm, expiry, and not-before with
  the approved clock-skew rule.
- [x] The validated actor includes the Entra `oid`, `tid`, and `sub` values as
  appropriate, plus the application user identity used for finance and audit
  decisions.
- [x] Invalid, expired, wrong-tenant, wrong-audience, malformed, or otherwise
  unverifiable tokens fail closed without revealing token or policy details.
- [x] Local tests use a clearly marked signed fixture issuer or fixture
  identities that cannot be enabled in shared Azure environments; no real
  tenant credentials are required.
- [x] General and privileged session-expiry behavior, warning behavior, and
  high-risk step-up requirements are represented at the approved API/UX
  boundaries.

#### User Story 1 implementation evidence

- [x] Go authentication validation, JWKS/discovery boundary, actor port, local
  fixture resolver, fail-closed middleware, anonymous health route, and safe
  diagnostics are implemented and covered by focused tests.
- [x] The OpenAPI contract declares bearer security and explicit
  `AUTHENTICATION_REQUIRED` 401 responses; committed Go/TypeScript artifacts
  were regenerated from the contract.
- [x] The React boundary uses MSAL authorization-code PKCE when explicitly
  configured, session-scoped browser caching, expiry/step-up states, the
  accessible two-minute warning, and one automatic 401 refresh/retry.
- [x] Evidence record: `docs/verification/DLV-IAM-001-us1-authenticated-application-identity.md`.
- [ ] Live Entra tenant behavior, production security controls, penetration
  testing, and production qualification remain unverified and are not claimed.

### 7.2 User Story 2 — Manage users and access assignments

**As a Security Administrator, I want to create and maintain application users
and their assignments, so that access can be granted, changed, suspended, and
terminated with evidence.**

Traceability: `DLV-FR-IAM-001`, `DLV-GFR-015`, `NFR-SEC-006`,
`NFR-SEC-012`, `NFR-PRV-001`–`NFR-PRV-003`.

Acceptance criteria:

- [x] Authorized administrators can create, update, activate, suspend, and
  terminate users while preserving the authentication-subject reference and
  explicit `UserStatus`.
- [x] User access records show roles, scopes, status, review evidence, and
  validation conflicts without exposing restricted personal or security data.
- [ ] Suspension, termination, role removal, and access-policy changes become
  effective for interactive and noninteractive access within 15 minutes.
- [x] State-changing user operations enforce authorization, idempotency,
  expected-version/concurrency behavior, and audit evidence.
- [x] A user cannot grant or retain access outside the approved scope of the
  administering actor, and denied changes return a typed safe reason.

#### User Story 2 implementation evidence

- [x] The identity domain and application service implement explicit lifecycle transitions, immutable authentication subjects, atomic complete-set assignment replacement, idempotency fingerprints, expected-version checks, authorization containment, and audit-record ports.
- [x] The identity-owned migration and SQL query sources define user state and unique assignment joins; generated SQLC output and a transaction-scoped PostgreSQL repository adapter are included, and PostgreSQL integration fixtures include the identity migration set.
- [x] The manage-users OpenAPI source defines a typed action envelope and If-Match header; checked-in TypeScript types reflect the new request model.
- [x] IAM-WS-01 and IAM-SCR-01 render masked identity data, status, roles, scopes, review evidence, validation conflicts, safe duplicate/denial results, and version-conflict recovery.
- [x] PostgreSQL integration covers clean migration, identity-schema ownership, uniqueness, atomic assignment replacement, concurrent optimistic locking, transactional audit references, and durable idempotency replay.
- [x] API tests cover typed action decoding, required idempotency headers, masked identity data, authorization denial, lifecycle activation, and version conflicts.
- [ ] pnpm wrapper execution, TypeScript artifact regeneration, and the full API-check wrapper remain pending; direct frontend and Playwright checks passed and are recorded in the evidence document.
- Evidence record: docs/verification/DLV-IAM-001-us2-user-access-assignments.md.

### 7.3 User Story 3 — Manage roles and permission grants

**As an Access Approver, I want versioned roles and explicit permission grants,
so that least-privilege access can be reviewed without hidden privilege.**

Traceability: `DLV-FR-IAM-002`, `NFR-SEC-004`, `NFR-SEC-012`,
`NFR-MNT-003`, `NFR-MNT-006`.

Acceptance criteria:

- [x] Authorized administrators can create and maintain roles with explicit
  permission grants and applicable scope constraints.
- [x] Role changes validate permission, scope, effective-date, duplicate, and
  segregation conflicts before they can become active.
- [x] Administrative policy and role changes require the independent approval
  and audit evidence required by the approved policy.
- [x] Historical decisions retain the role/policy version used at the time;
  later changes do not rewrite prior authorization evidence.
- [x] Tests prove least privilege, deny-by-default behavior, role removal,
  stale-version conflict, and safe repeat behavior.

#### User Story 3 implementation evidence

- [x] internal/identity implements a versioned Role aggregate with explicit
  permission grants, opaque scopes, effective dates, complete grant-set
  replacement, non-destructive retirement, approved-catalogue validation,
  actor-scope containment, independent approval evidence, segregation and audit
  ports, idempotency, and optimistic concurrency.
- [x] Identity-owned role/revision/grant tables, SQLC queries/generated output,
  role-assignment foreign-key enforcement, PostgreSQL atomic revision commits,
  historical revision retention, audit linkage, retired/unknown assignment
  rejection, and durable role-command replay are covered.
- [x] The manage-roles API is typed in OpenAPI and generated Go/TypeScript
  artifacts, including If-Match consistency and safe 400/403/409/422/503
  outcomes. HTTP tests cover typed role creation/update, denial, validation,
  and stale-version recovery.
- [x] IAM-WS-01 and IAM-SCR-02 render synthetic role worklist/detail states
  with grants, scopes, effective dates, approval/validation states, retirement,
  masked evidence, duplicate handling, and accessible version-conflict
  recovery.
- [x] Focused domain, security, idempotency, concurrency, PostgreSQL, API, and
  frontend tests pass through the available host toolchains. The Workflow
  approval and segregation adapters remain explicit ports with synthetic
  local doubles; production workflow/segregation qualification remains future
  scope.
- [ ] The exact pnpm check web step and make api-check wrapper remain
  environment-limited: Windows pnpm cannot execute the POSIX Vitest shim, and
  the WSL-to-Windows Redocly wrapper maps the workspace path incorrectly.
  Direct Windows Node/Vitest, TypeScript build, Redocly lint, typed artifact
  generation, Go tests, and SQLC checks pass.
- Evidence record: docs/verification/DLV-IAM-002-us3-role-permission-grants.md.

### 7.4 User Story 4 — Evaluate scoped access policies

Status: implementation complete for authorization evaluation and durable policy
reads. Full policy administration UI/API remains a follow-up and is not marked
complete by this story.

Evidence: `docs/verification/DLV-IAM-003-us4-scoped-access-policies.md`.

**As an application module, I want one authoritative authorization decision
contract, so that every protected action evaluates the dimensions that apply to
that operation.**

Traceability: `DLV-FR-IAM-003`, `DLV-GFR-002`, `NFR-SEC-004`,
`NFR-SEC-006`, `NFR-SEC-011`, `NFR-OBS-008`.

Acceptance criteria:

- [x] Policy evaluation supports applicable legal-entity, business-unit or
  segment, account or account-class, transaction-type, amount, currency,
  fiscal-period, sensitivity, and action dimensions.
- [x] The decision is default-deny, identifies the permission/policy version,
  and returns an auditable decision reference without exposing restricted policy
  implementation details.
- [x] The API rejects unauthorized actions even when the SPA displays the
  action; authorization is re-evaluated at the authoritative mutation boundary.
- [x] Field-level sensitive-data restrictions are enforced independently from
  record-level access, including filters, counts, comparisons, errors, and
  exports where those surfaces exist.
- [x] Allowed, denied, expired, unavailable-policy, and stale-policy results
  are distinct and have safe next-action guidance.

### 7.5 User Story 5 — Enforce segregation-of-duties rules

**As an Access Approver, I want prohibited duty combinations rejected before a
protected action is established, so that authorization cannot be bypassed by a
conflicting role or actor.**

Traceability: `DLV-FR-IAM-004`, `DLV-GFR-003`, DDD §8.2,
`NFR-SEC-005`, `NFR-SEC-012`.

Acceptance criteria:

- [ ] Authorized administrators can maintain versioned segregation rules with
  conflicting permission sets and enforcement modes.
- [ ] The minimum rules are enforced: payment-batch preparation/approval,
  fiscal-period reopen request/approval, vendor-bank-detail change/payment
  release during cooling-off, self-approval of high-threshold manual journals,
  payroll-detail versus summary-ledger access, and independent policy approval.
- [ ] A prohibited combination is rejected before the protected business action
  is established and returns a non-sensitive conflict reason and permitted
  resolution path.
- [ ] The rule is evaluated against current roles, actor history, scope, and
  policy version; stale or changed policy cannot silently authorize the action.
- [ ] Synthetic negative tests cover each minimum rule, a permitted control
  case, an expired exception, and an attempted client-side bypass.

### 7.6 User Story 6 — Grant, revoke, and review emergency access

**As an Access Approver, I want controlled emergency access with automatic
expiry and post-use review, so that urgent support is possible without creating
permanent privileged access.**

Traceability: `DLV-FR-IAM-005`, `DLV-FR-IAM-006`, DDD §8.3,
`NFR-SEC-003`, `NFR-SEC-006`, `NFR-SEC-013`, `NFR-AUD-001`, `NFR-AUD-003`.

Acceptance criteria:

- [ ] A grant records the actor, permissions/scopes, reason, approver, start,
  expiry, policy version, and required post-use review state.
- [ ] A grant is time-bound to no more than four hours by default, cannot be
  used after expiry even when cleanup is delayed, and requires step-up or the
  approved authentication assurance for high-risk use.
- [ ] Authorized actors can revoke a grant, and revocation takes effect within
  the approved 15-minute target across interactive and noninteractive access.
- [ ] Every action performed under a grant carries the grant reference and is
  included in the audit boundary without exposing sensitive action payloads.
- [ ] Revocation, expiry, failed approval, duplicate submission, and post-use
  review outcomes are visible as distinct states and are safe to retry.

### 7.7 User Story 7 — Protect sensitive access evidence and explain decisions

**As an Auditor, I want access decisions and sensitive-data handling to be
reviewable without revealing protected values, so that security evidence is
useful and privacy-preserving.**

Traceability: `DLV-GFR-015`, `NFR-SEC-009`–`NFR-SEC-012`,
`NFR-SEC-017`–`NFR-SEC-018`, `NFR-PRV-001`–`NFR-PRV-003`,
`NFR-AUD-001`, `NFR-AUD-003`, `NFR-AUD-010`, `NFR-OBS-003`–`NFR-OBS-004`.

Acceptance criteria:

- [ ] IAM views, errors, logs, events, notifications, and evidence use stable
  identifiers, classifications, decision references, and masked values rather
  than secrets, full bank details, payroll detail, tax identifiers, or raw
  tokens.
- [ ] An authorized decision explanation identifies the applicable scope/action
  dimension, duty conflict, policy version, or next permitted action without
  disclosing restricted policy or another user’s sensitive data.
- [ ] Access changes, policy changes, segregation changes, emergency grants,
  revocations, reveals, and privileged searches produce attributable audit
  evidence through the approved audit boundary.
- [ ] Exports and bulk views enforce the same row and field permissions as
  interactive views and record actor, scope, filters, purpose, and sensitivity
  classification where those capabilities exist.
- [ ] Automated negative tests prove synthetic sensitive values and credential
  markers cannot appear in diagnostics or evidence output.

## 8. Cross-cutting behavior

### Application/API

- API authorization is authoritative and uses the documented IAM permission
  names, including `finance.iam.manage.users`,
  `finance.iam.manage.roles`, `finance.iam.manage.access.policies`,
  `finance.iam.manage.segregation.rules`,
  `finance.iam.grant.emergency.access`, and
  `finance.iam.revoke.emergency.access`.
- Protected failures use the approved response classes: authentication failure,
  authorization denial, validation failure, conflict, dependency unavailable,
  and sensitive-data masking are not collapsed into one generic error.
- State-changing operations use the existing idempotency and correlation
  contracts. IAM must not add a second retry identity or mutate another
  bounded-context record directly.

### Database and persistence

- The `identity` schema owns IAM tables, migrations, indexes, and generated
  access code. Cross-schema write paths and business-data joins are forbidden.
- Policy, role, segregation, user, and emergency state transitions preserve
  versions, effective dates, status, actor, and audit references.
- No credential or raw token is persisted as an application user attribute.

### Integration and audit

- Material IAM changes use the existing transactional boundary and approved
  audit port. If an integration event is required by a later approved contract,
  it uses the existing outbox/inbox boundary; this plan invents no event name.
- Authorization context carries actor, authentication subject, correlation,
  causation, scope, permission, policy version, decision, and grant reference
  as safe metadata only.

### Frontend and UX

- The IAM administration worklist and screens follow `IAM-WS-01` and
  `IAM-SCR-01`–`IAM-SCR-05`.
- Scope, current state, permitted/blocked action, reason, owner, effective
  date, and next action remain visible where applicable.
- Sensitive values are masked by default and never exposed merely because a
  user can see the containing record.

### Security and privacy

- Local verification uses synthetic users, policy values, scopes, and
  sensitive-data markers. It never requires real Entra tokens or production
  records.
- Logs and traces contain safe classification and outcome fields only. Raw
  authorization headers, token claims beyond approved identifiers, policy
  payloads, bank/payroll/tax values, and unrestricted errors are excluded.

## 9. Ordered implementation plan

1. Confirm the IAM permission catalogue, user/role/policy/rule state models,
   policy-version/effective-date rules, and the adapter/application ports.
2. Implement the local authentication boundary and application actor mapping
   with signed fixture identities, then add invalid-token and wrong-context
   negative tests.
3. Implement `User` and `Role` lifecycle behavior with optimistic versioning,
   idempotency, permission validation, access revocation, and audit-port calls.
4. Implement `AccessPolicy` evaluation with the operation-specific dimension
   set, default-deny behavior, decision references, policy versions, and safe
   authorization errors.
5. Implement `SegregationRule` management and the minimum DDD §8.2 rules;
   verify checks occur before authoritative business state is established.
6. Implement emergency grant/revoke/expiry/review with a bounded duration,
   step-up boundary, grant references, and safe recovery for duplicate or
   uncertain requests.
7. Add the IAM API/OpenAPI and frontend administration/decision surfaces only
   through the approved contract and existing module boundaries.
8. Add focused, persistence, integration, security, privacy, accessibility,
   concurrency, idempotency, and evidence verification; update this document
   only for criteria supported by successful results.

## 10. Likely files and packages

- `internal/identity/` — aggregates, value objects, policy evaluator, ports,
  tests, and safe decision results.
- `db/migrations/` and `db/queries/identity/` — identity-owned schema and
  queries, with generated output under the existing platform conventions.
- `cmd/api/` and authentication middleware — Entra token validation and actor
  context composition only.
- `api/openapi/` and generated API artifacts — IAM routes, schemas, errors,
  and permission metadata if the approved contract requires them.
- `web/` — IAM administration, access-decision explanation, masking, and
  expiry/review states using the existing design system.
- `internal/audit/` or its approved port — audit evidence integration without
  direct schema ownership.
- `scripts/verify/`, `Makefile`, and `docs/verification/` — focused IAM
  contract/security/evidence checks.

## 11. Dependencies and handoffs

- `EP-PLAT-001` supplies Go/React/API/database, idempotency, concurrency,
  correlation, and CI foundations.
- `EP-UX-001` supplies the shell, access-aware components, masking, state, and
  accessibility patterns.
- `EP-OPS-001` supplies safe telemetry, alert, runbook, and operational
  evidence contracts.
- `EP-AUD-001` later owns the full audit-chain aggregate, sealing, proof, and
  integrity-incident behavior; IAM must use its contract boundary.
- `EP-OMD-001` and `EP-COA-001` own legal-entity, party, segment, and account
  reference data. IAM evaluates references through approved ports and does not
  own those records.
- `EP-WFA-001` owns generic approval policies, requests, decisions,
  delegation, and escalation. IAM consumes approved decision references for
  operations that require independent approval.

## 12. Definition of done

- All seven user stories have acceptance evidence for the implemented local
  scope and exact source traceability.
- Entra authentication is validated at the API boundary without application
  password storage or production credentials in local tests.
- User, role, policy, segregation, and emergency-access state is versioned,
  scoped, authorized, idempotent, auditable, and bounded by its owning schema.
- Authorization is default-deny, evaluates applicable dimensions, rechecks at
  the authoritative mutation boundary, and returns safe typed outcomes.
- Minimum segregation rules and emergency-access expiry/revocation/review are
  covered by positive, negative, concurrency, duplicate, and failure tests.
- Sensitive information is minimized and masked across views, errors, logs,
  events, exports, and evidence; no secret or raw token is emitted.
- IAM screens and API contracts preserve scope, state, denial reason, owner,
  effective date, and next-action meaning.
- `QG-01`, `QG-02`, `QG-03`, `QG-04`, `QG-05`, `QG-06`, `QG-08`, and `QG-10`
  evidence is recorded, with production security and release qualification
  explicitly separated from local acceptance.
- No finance capability, accounting effect, cross-schema write, or future
  workflow is marked complete by IAM planning alone.

## 13. Required verification evidence

- Authentication tests for valid, invalid, expired, wrong-tenant, wrong-
  audience, malformed, and fixture-only local identities.
- Authorization matrix tests for every IAM permission, default-deny behavior,
  allowed/denied scope dimensions, field-level restrictions, and safe error
  mapping.
- User/role/policy/rule/emergency lifecycle tests for idempotency, stale
  versions, duplicate requests, expiry, revocation, and recovery.
- Minimum segregation tests for payment preparation/release, reopen request/
  approval, vendor bank-detail cooling-off, high-threshold journal
  self-approval, payroll detail, and independent policy approval.
- Persistence tests for identity-schema ownership, migration/sqlc drift,
  effective dates, version transitions, and audit references.
- API and UI tests for scope selection, blocked-action explanation, masking,
  session expiry warning, step-up boundary, export filtering, and accessibility.
- Negative scans proving tokens, credentials, full sensitive values, raw policy
  payloads, and unrestricted errors cannot enter logs, traces, responses,
  notifications, or evidence.
- Evidence record containing source versions, commands, tool versions, synthetic
  data boundary, results, limitations, and deferred production qualification.

## 14. Risks and open decisions

- The current repository contains the approved IAM contracts and architecture,
  but no IAM bounded-context implementation is claimed by this plan.
- Entra tenant configuration, MFA, conditional access, and production identity
  administration require external qualification; local fixtures cannot prove
  those hosted controls.
- The exact approval handoff between IAM policy administration and
  `EP-WFA-001` must use a versioned decision reference without duplicating the
  workflow aggregate.
- The exact audit-port contract must be confirmed with `EP-AUD-001`; IAM must
  not create an alternate audit history.
- Scope reference data is owned by OMD/COA/GL and must be consumed through
  stable identifiers or ports rather than copied into IAM-owned business facts.
- Security testing, penetration testing, and production release qualification
  remain separate from local user-story acceptance.

## 15. Planning status

User Story 1 is implemented on the delivery branch and its local evidence is
recorded separately. This document does not mark the broader IAM epic, the
identity schema, user lifecycle, roles, policies, segregation rules, emergency
access, or any finance authorization behavior complete.

Implementation branch: `codex/iam-us2-user-access-assignments`
