# DLV-IAM-006 User Story 7 — Sensitive access evidence verification

## Scope and status

This record covers the IAM-owned local implementation for `EP-IAM-001` User
Story 7: safe decision projections, metadata-only access observations,
fail-closed reveal/export audit dependencies, masked sensitive-data guards,
and permission-filtered worklist export context.

Status: implementation added on `codex/iam-us7-sensitive-access-evidence`.
The focused Go verification passed. Frontend execution is pending because the
current WSL session has `pnpm` but no runnable Linux Node binary; the failure
occurs before Vitest or TypeScript starts.

This is synthetic local evidence only. It does not claim ownership of the
future audit bounded context, production security qualification, live Entra
qualification, or finance-module authorization enforcement.

## Traceability

| Source | Relationship |
|---|---|
| `EP-IAM-001`, `M1` | IAM access evidence and privacy boundary |
| `DLV-GFR-015` | Protected access evidence and explanation behavior |
| `NFR-SEC-009`–`NFR-SEC-012`, `NFR-SEC-017`–`NFR-SEC-018` | Default deny, masking, authorization and safe failure behavior |
| `NFR-PRV-001`–`NFR-PRV-003` | Restricted-value minimization and no indirect disclosure |
| `NFR-AUD-001`, `NFR-AUD-003`, `NFR-AUD-010` | Attributable access metadata through an audit port |
| `NFR-OBS-003`–`NFR-OBS-004` | Bounded operational classification and correlation references |
| DDD §§2.18, 8, 11; UX §§5.1, 6, 7.18, 9, 10.2 | Evidence, privacy, IAM-SCR-05 and explanation behavior |
| `ARC-PRV-001`, `ARC-SEC-002`, `ARC-AUD-001`, `ARC-OBS-001`, `TADR-012` | Privacy, security, audit/telemetry separation and ownership |

## Implementation evidence

- `internal/identity/access_evidence.go` adds the metadata-only
  `AccessObservation` contract, bounded operation/classification/outcome
  values, safe authorization and segregation explanation projections, and an
  `AccessObservationRecorder` port. Actor authentication is represented by the
  existing subject fingerprint; raw subjects, tokens, credentials, values and
  policy payloads are rejected.
- `AccessEvidenceService.RecordAccess` validates before recording and fails
  closed on recorder errors or a missing audit reference. Existing mutation
  audit ports and transaction paths are unchanged.
- `web/src/components/sensitive-data-guard` requires explicit safe metadata
  and an audit callback for every reveal/export attempt. A failed callback
  leaves the value masked and does not invoke export.
- `web/src/components/worklist-and-saved-filters` requires an audited,
  permission-filtered row and column projection before export. Client filters
  are applied after that projection, and export context includes actor,
  purpose, classification and source-version metadata.
- Existing IAM-SCR-05 and record-detail fixtures now pass synthetic decision
  and audit metadata without adding a route, OpenAPI schema, database table,
  cross-schema query, or event name.
- `web/tests/a11y/semantic.spec.ts` now exercises the enabled export control
  and confirms the synthetic filtered-result announcement.

## Verification commands and results

| Command or check | Result | Evidence or limitation |
|---|---|---|
| `GOCACHE=/tmp/tally-go-cache /usr/local/go/bin/go test ./internal/identity` | Passed | Access observation validation, fail-closed recorder behavior, safe explanation projections, and existing IAM tests. |
| `GOCACHE=/tmp/tally-go-cache /usr/local/go/bin/go test ./internal/identity ./internal/platform/httpapi ./internal/platform/telemetry` | Passed | Focused IAM, typed HTTP mapping, and bounded telemetry tests. |
| `GOCACHE=/tmp/tally-go-cache /usr/local/go/bin/go test -race ./internal/identity ./internal/platform/httpapi ./internal/platform/telemetry` | Passed | Focused race checks passed. |
| `GOCACHE=/tmp/tally-go-cache /usr/local/go/bin/go vet ./internal/identity ./internal/platform/httpapi ./internal/platform/telemetry` | Passed | Focused vet checks passed. |
| `GO_BIN=/usr/local/go/bin/go GOCACHE=/tmp/tally-go-cache bash scripts/verify/telemetry-failure-sensitive-data.sh` | Passed | Existing telemetry failure, race, vet, and sensitive-data checks passed. |
| `git diff --check` | Passed | No whitespace errors. |
| `GOCACHE=/tmp/tally-go-cache /usr/local/go/bin/go test ./...` | Blocked by environment | Unchanged authentication/database tests cannot bind local TCP/IPv6 listeners in this restricted WSL session. Other repository packages reached passing results. |
| `GOCACHE=/tmp/tally-go-cache /usr/local/go/bin/go vet ./...` | Passed | Repository-wide vet completed without diagnostics. |
| `PATH=/usr/local/go/bin:$PATH GOCACHE=/tmp/tally-go-cache make iam-sensitive-evidence-check` | Blocked by environment | The Go portions passed; the target stopped at its frontend step because `pnpm` could not find `node`. |
| `pnpm --dir web exec vitest run src/components/record-context/record-context.test.tsx src/components/operational-components.test.tsx src/app/identity-access-workspace.test.tsx` | Blocked by environment | `pnpm` started, then reported `node: not found`; no frontend test process started. |
| `pnpm --dir web exec tsc -b` | Blocked by environment | Same missing Linux Node runtime. |
| Playwright accessibility/semantic checks | Not run | Requires the unavailable frontend runtime. |

## Synthetic-data boundary

All fixture values remain presentation-only and are not production identity,
bank, payroll, tax, credential, token, or policy data. The access observation
contract carries stable references, classifications, decision/policy/grant
references, correlation/causation references, and fingerprints only.

## Limitations and deferred qualification

- The repository currently exposes no public decision explanation or evidence
  export API for Story 7; none was introduced.
- No production audit adapter was invented. The recorder is an IAM port and a
  memory double for local verification; the audit bounded context must provide
  the transactional implementation.
- Operational telemetry remains separate from authoritative audit evidence.
- No live Entra tenant, MFA/conditional-access configuration, penetration
  test, production data scan, or finance-module enforcement is claimed.
- Full repository Go/frontend checks and Playwright checks remain subject to a
  host with the repository's Node runtime available. The focused Go scope
  passed in this environment.
