# DLV-PLAT-004 User Story 5 — Contract and Generated-Artifact Drift Verification

## Scope

This record verifies the focused OpenAPI contract and generated-artifact drift
workflow. It does not claim full API compatibility, security, frontend, SQL,
Terraform, documentation, finance behavior, or `DLV-CI-001` completion.

## Tool and source contract

- Go: `1.26.3` from `go.mod`
- Node.js: `24.x` CI baseline
- pnpm: `11.9.0`
- Redocly CLI: `1.34.4`
- ogen: `v1.23.0`
- `@hey-api/openapi-ts`: `0.99.0`
- OpenAPI source: `contracts/openapi/openapi.yaml`
- Generated Go output: `internal/platform/httpapi/generated/`
- Generated TypeScript output: `web/src/generated/api/`
- Focused workflow: `.github/workflows/openapi.yml`

## Commands executed

Commands were run from the repository root. The sandbox required a writable
temporary Go build cache, so Go verification used `GOCACHE=/tmp/tally-go-cache`.

```bash
bash -n scripts/openapi/api-check.sh \
  scripts/openapi/typescript-negative-check.sh \
  scripts/openapi/typescript-client-check.sh
GOCACHE=/tmp/tally-go-cache make api-check
GOCACHE=/tmp/tally-go-cache make api-negative-check
```

## Results

- OpenAPI linting passed and deterministic bundle verification passed.
- Go generation, artifact inventory comparison, `go test ./...`, and API build passed.
- TypeScript generation was deterministic, matched the committed artifact inventory, passed the complete recursive committed-output comparison, and compiled successfully.
- Go negative checks rejected manual edits, missing files, extra files, and invalid input.
- TypeScript negative checks rejected manual edits, missing files, and extra files.
- Freshly regenerated Go and TypeScript output passed their respective checks.
- The aggregate `make api-check` command passed without changing tracked generated artifacts.
- Focused CI is configured to install both lockfiles with frozen resolution and invoke the same `make api-check` command.
- Focused CI completed successfully for pull request 24.

## CI evidence

- Workflow: `.github/workflows/openapi.yml`
- Result: passed
- Pull request: `#24`
- Run: [GitHub Actions run 31252064661](https://github.com/toanle88/Tally/actions/runs/31252064661)
- Job: [Contract and generated-artifact drift, job 93089812424](https://github.com/toanle88/Tally/actions/runs/31252064661/job/93089812424)
