# DLV-CI-001 — Pull-request CI Quality Pipeline Verification

## Scope

This record covers the repository-owned pull-request quality orchestration for
`M0 / EP-PLAT-001`. It does not claim Azure authentication, remote Terraform
state access, apply/destroy, smoke testing, production qualification, release
qualification, or external GitHub branch-protection configuration.

## Implementation evidence

- Aggregate workflow: `.github/workflows/pull-request-quality.yml`
- Required status check: `Pull-request quality / required`
- Repository CI contract: `scripts/verify/ci-contract.js`
- Repository integrity gate: `scripts/verify/repository-integrity.sh`
- Terraform PR plan gate: `scripts/verify/terraform-pr-plan.sh`
- Focused workflow coordination: `.github/workflows/openapi.yml`,
  `.github/workflows/persistence.yml`, `.github/workflows/terraform.yml`, and
  `.github/workflows/docs-site.yml`

The aggregate workflow uses read-only repository permissions, commit-pinned
Dependency Review and Gitleaks actions, committed pnpm lockfiles, temporary
synthetic Terraform variables, backend-disabled and refresh-disabled plans,
and no raw plan/state artifact publication.

## Revision and declared tool versions

- Verification base commit: `cabaaafc5b3c58f14c2e648bb80cf73696cf6b2d`.
- Implementation branch: `codex/dlv-ci-001-pr-quality-pipeline`; the working
  tree changes are intentionally uncommitted.
- Workflow-declared runtimes/tools: Go `1.26.3` from `go.mod`, Node `24.x`,
  pnpm `11.9.0`, Terraform `1.15.2`, TFLint `0.64.0`, Checkov `3.3.8`, and
  Infracost `0.10.45`.
- Reviewed scanner action pins: Dependency Review
  `a1d282b36b6f3519aa1f3fc636f609c47dddb294`; Gitleaks
  `e0c47f4f8be36e29cdc102c57e68cb5cbf0e8d1e`.
- Locally observed: Terraform `1.16.1`; Linux Go, Node.js, pnpm runtime, and
  actionlint were unavailable.

## Verification executed in the current environment

| Command | Result |
|---|---|
| `bash -n scripts/verify/repository-integrity.sh scripts/verify/terraform-pr-plan.sh scripts/deploy/terraform-apply.sh` | Passed |
| `bash scripts/verify/repository-integrity.sh --self-test` | Passed |
| `bash scripts/verify/repository-integrity.sh` | Passed |
| Python YAML parse and aggregate/focused-workflow contract assertions | Passed |
| `make -n ci-check` | Passed dry-run inspection |
| `make -n terraform-pr-check` | Passed dry-run inspection |
| Aggregate fan-in simulation with success, skipped, failed, and cancelled dependency results | Passed; only all-success input is accepted |
| `git diff --check` | Passed |

## Hosted diagnostic run

The hosted run for the earlier pull-request merge revision
`3406824a968de594f7c85109aae8818638a495a5` (head branch revision
`949b3be2827ebe928d28371fb86e3e5cb430286f`) was inspected with
`gh run view 35487515660 --json jobs` and its job logs. Contracts, persistence,
and documentation passed. Application quality failed because the frontend
install did not populate `web/node_modules`, while infrastructure and security
failed because `actions/setup-node@v5` implicitly enabled pnpm caching before
pnpm was installed. The working-tree correction disables that implicit cache in
every setup-node step and installs the standalone frontend with
`pnpm --dir web --ignore-workspace install --frozen-lockfile`. A corrected
hosted rerun remains pending because this working tree is uncommitted.

## Verification not completed locally

| Evidence | Result | Limitation |
|---|---|---|
| `make ci-check` | Blocked | Linux Node.js is unavailable in the current workspace. |
| `make terraform-pr-check` | Blocked | Go/Node tooling is unavailable and Terraform provider registry access is blocked locally. |
| `make persistence-check` | Blocked | The command ran but stopped because Linux Go was unavailable before Docker/Testcontainers execution. |
| `pnpm docs:check` and frozen-install commands | Blocked | The commands reached the local pnpm shim but Linux Node.js was unavailable; frontend tests, builds, and accessibility tests were not started. |
| Actionlint and pinned scanner action execution | Not run | Requires the hosted GitHub Actions environment. |
| Clean positive, drift-negative, failure-propagation, and skipped-job runs | Pending | Requires hosted pull-request runs or equivalent workflow execution. |
| External branch-protection activation | Not verified | GitHub repository settings are outside this repository change. |

## Safety and deferred scope

The workflow does not use Azure or production credentials, authenticated remote
state, Terraform apply/destroy, deployment smoke checks, raw Terraform plans or
state, production data, or finance data. A green aggregate workflow is not a
claim that external branch protection is configured or that release quality,
recovery, performance, or full-system qualification is complete.
