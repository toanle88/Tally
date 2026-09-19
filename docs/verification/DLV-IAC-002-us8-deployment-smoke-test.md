# DLV-IAC-002 User Story 8 — Deployment smoke-test verification

Status: repository implementation complete; live Azure and migration evidence pending.

## Scope

The smoke runner distinguishes a successful Terraform apply from a usable
learning deployment. It accepts only `dev` or `demo`, reads the selected
Azure Blob-backed Terraform state, and uses read-only Azure CLI calls. It does
not run Terraform plan/apply/destroy, migrations, seed data, or finance writes.

Run it after an approved authenticated deployment:

```bash
ENVIRONMENT=dev make azure-learning-smoke
```

The redacted report is written to `artifacts/deployment-smoke/<environment>.json`.
It records owner/dependency for each check and cleanup as `not-run` because
destruction is User Story 9 scope.

## Evidence record

| Field | Value |
|---|---|
| Story | DLV-IAC-002 → User Story 8 |
| Commit | _populate from report/run_ |
| Environment | _dev or disposable demo_ |
| Operator / UTC time | _populate after witnessed run_ |
| Overall outcome | _pass, fail, or blocked_ |
| Migration status | `not-verified` unless separately owned evidence is supplied |
| Migration evidence reference | _none recorded_ |
| Cleanup status | `not-run` |
| Report | `artifacts/deployment-smoke/<environment>.json` |

## Required checks

The report covers Terraform output/environment matching; resource group,
Container Apps environment, API, worker, ACR, Static Web App, PostgreSQL, Key
Vault, identities, Log Analytics, and Application Insights; HTTPS/TLS and
`GET /health/live`; `/health/live` probe configuration; revisions and immutable
image digests; ACR manifests; scoped `AcrPull` and Key Vault roles; and
Container Apps/PostgreSQL diagnostics.

`/health/ready` is deliberately not required. It remains owned by the later
API/database-readiness delivery. Static Web App content availability is a
known limitation because Story 6 provisions the resource but does not deploy
frontend content.

## Migration boundary and exceptions

Infrastructure does not own Goose migrations or release qualification. User
Story 8 therefore records migration evidence as blocked/not verified unless a
separately owned controlled release step supplies a reference. The smoke test
does not modify schema, application startup, migrations, seed data, or
established finance state. Local `make db-migrate` is not Azure release
evidence. Keep the migration acceptance criterion open until that release
delivery provides one-shot migration evidence.

## Repository verification

Credential-free verification is:

```bash
make azure-learning-smoke-check
```

Live pass/fail evidence requires an approved authenticated `dev` or disposable
`demo` environment with synthetic data and must be recorded above.
