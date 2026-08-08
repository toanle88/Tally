# DLV-PLAT-004 — OpenAPI-First REST Workflow User Stories

| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-004` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | Planned |
| Dependency position | Builds on `DLV-PLAT-001` and `DLV-PLAT-003`; no finance capability implementation is required. |
| Exit evidence | OpenAPI 3.1 validates and bundles; generated Go and TypeScript artifacts compile; focused CI detects contract or generated-artifact drift. |

## 1. Purpose and scope

**As the TALLY developer, I want one version-controlled OpenAPI contract to drive the HTTP boundary and generated client artifacts, so that API consumers and future finance modules share a reviewable, compatible contract.**

This item establishes the OpenAPI-first workflow. It may use a minimal technical health or shell operation to prove generation, but must not implement finance commands, repositories, authorization policy, business idempotency, or domain events.

## 2. Approved boundaries

- The contract format is OpenAPI 3.1 and REST/JSON.
- Contract sources live under `contracts/openapi/`, with shared components, capability paths, and examples separated according to the API specification.
- The contract owns transport schemas, operation IDs, required headers, problem-details errors, pagination conventions, and HTTP result rules.
- Generated Go server/types and TypeScript client/types are derived artifacts and are not hand-edited.
- Money uses the API specification's exact-decimal representation; examples and logs contain no secrets or real financial data.

## 3. Explicit exclusions

- Finance endpoint behavior, domain commands, aggregates, repositories, database wiring, or business events.
- Authentication, authorization, accounting-scope enforcement, audit evidence, and business idempotency (`DLV-PLAT-006`).
- Integration events, outbox/inbox, workers, retries, and replay (`DLV-PLAT-007`).
- Full pull-request quality orchestration (`DLV-CI-001`) beyond the focused contract drift check.
- Frontend routing, forms, design-system components, or capability screens (`EP-UX-001`).

## 4. User stories

### User Story 1 — Establish the contract layout and common schemas

**As the TALLY developer, I want a maintainable OpenAPI source layout and common schemas, so that future capabilities extend the API without duplicating transport rules.**

- [x] An OpenAPI 3.1 root document exists under `contracts/openapi/` and references committed component and path files.
- [x] Common headers, command requests, established results, pagination/query conventions, and RFC 9457-style problem details are represented consistently.
- [x] Examples are deterministic and contain no credentials or real personal/financial data.
- [x] Every operation has a unique, stable `operationId` and an identified owning capability boundary.

### User Story 2 — Validate and bundle the OpenAPI contract

**As the TALLY developer, I want pinned, repeatable contract validation, so that malformed references and incompatible API changes fail before review.**

- [x] The repository pins the selected OpenAPI linter/bundler and exposes a root validation command.
- [x] Validation covers all referenced files and rejects broken references, duplicate operation IDs, invalid schemas, and malformed examples.
- [x] A negative fixture proves an invalid contract fails with a non-zero result and no secret output.
- [x] Bundling or equivalent verification output is deterministic.

### User Story 3 — Generate and verify Go API artifacts

**As the Go API developer, I want generated server interfaces and transport types from the contract, so that future handlers conform to the reviewed HTTP boundary.**

- [x] A pinned generator configuration produces Go artifacts from the committed contract.
- [x] Generated Go code compiles under the normal Go verification command and has a generated-file marker.
- [x] Generation is deterministic and does not require a globally installed unpinned tool.
- [x] A minimal compile or shell integration references the generated boundary without introducing finance behavior.

### User Story 4 — Generate and verify the TypeScript client

**As the React developer, I want generated TypeScript API types/client functions, so that the frontend cannot silently diverge from the backend contract.**

- [x] A pinned generator configuration produces TypeScript client/types from the same OpenAPI source.
- [x] Generated TypeScript output compiles with the pnpm-managed frontend verification command.
- [x] The client preserves operation IDs, response/error types, exact-decimal transport values, and nullable/optional distinctions.
- [x] Generated output is clearly marked and no frontend screen is required to complete this item.

### User Story 5 — Detect contract and generated-artifact drift

**As the TALLY maintainer, I want local and focused CI checks to detect contract and generated-code drift, so that committed artifacts cannot become stale.**

- [ ] One documented root command validates the contract, regenerates both language artifacts, checks compilation, and fails when regeneration changes tracked output.
- [ ] Local negative verification proves the check fails for stale output and passes after regeneration.
- [ ] Focused CI invokes the same command on a clean checkout.
- [ ] The workflow clearly leaves full API compatibility, security, frontend, SQL, Terraform, and documentation gates to their owning items or `DLV-CI-001`.

## 5. Definition of Ready

- [ ] The OpenAPI 3.1 specification and exact endpoint catalog are available as the source of truth.
- [ ] Contract directory, generated output locations, and toolchain versions are agreed and documented.
- [ ] Boundaries with `DLV-PLAT-003`, `DLV-PLAT-005`–`007`, `EP-UX-001`, and `DLV-CI-001` are preserved.
- [ ] Five stories are small enough for one or a short chain of reviewable changes.

## 6. Definition of Done

- [ ] All five stories and acceptance criteria pass.
- [ ] OpenAPI validation and bundling are reproducible from a clean checkout.
- [ ] Generated Go and TypeScript artifacts compile and remain synchronized with the contract.
- [ ] Local and focused CI drift checks pass, including controlled negative checks.
- [ ] Documentation identifies commands, tool pins, source locations, generated locations, and ownership boundaries.
- [ ] No finance capability, protected action, business event, or adjacent delivery item is falsely marked complete.

## 7. Traceability

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-004` |
| Direct FR/GFR IDs | None; this item establishes transport tooling rather than finance behavior. |
| Quality contribution | API/integration-contract foundation; partial `QG-04` and clean-generation evidence for `QG-10`. |
| Exit evidence | OpenAPI validates; generated Go and TypeScript artifacts compile; focused CI rejects contract/generated-artifact drift. |

## 8. Source references

- `docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md`
- `docs/specs/finance_delivery_plan_v1.0.md`
- `docs/specs/technical_specifications/10_technical_traceability_decisions_v1.0.md`
- `docs/backlog/stories/DLV-PLAT-003_user_stories.md`
