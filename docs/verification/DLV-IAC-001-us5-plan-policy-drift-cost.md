# DLV-IAC-001 User Story 5 — Plan, policy, drift, and cost controls

## Scope and evidence boundary

This record covers the credential-free repository gate for `EP-IAC-001` User
Story 5. It does not claim an Azure subscription plan, budget deployment,
Infracost API result, or authenticated drift result. Those require operator
credentials and an external backend and remain explicit follow-up evidence.

The local and pull-request gate is:

```bash
make terraform-check
make terraform-modules-check
```

The pinned external-tool gate is:

```bash
make terraform-tools-check
make terraform-lint-check
make terraform-security-check
```

Tool versions are recorded in
[`scripts/verify/terraform-tools.env`](../../scripts/verify/terraform-tools.env).
The CI workflow installs those versions without Azure login, OIDC, apply, or
scheduled state access: [terraform.yml](../../.github/workflows/terraform.yml).

## Controls

| Control | Implementation | Verification boundary |
| --- | --- | --- |
| Format, validation, provider locks | `scripts/verify/terraform.sh` | All roots use backend-disabled init, format, validate, and lock checks. |
| Lint | `scripts/verify/terraform-lint.sh` | TFLint runs against bootstrap, environments, and modules. |
| Security scan | `scripts/verify/terraform-security.sh` | Checkov scans each root; only observed exact rule/path exceptions from the rationale, owner, and expiry manifest are filtered. Production-reference cannot be excepted. |
| Plan policy | `scripts/verify/terraform-plan-policy.js` | Standard `terraform show -json` input; versioned valid/invalid fixtures run in every local gate. |
| Shared budget | `infra/terraform/bootstrap` and `modules/budget` | One subscription budget, default monthly amount 20, thresholds 50/80/100, at least one recipient. |
| Drift | `scripts/verify/terraform-drift.sh` | `ENVIRONMENT=dev|demo|prod-reference`; authenticated refresh-only plan, exit 0 clean / 2 drift / 1 error. |
| Cost | `scripts/verify/terraform-cost.sh` | `PLAN_JSON` plus `ACTIVE_MONTH_COST`; Infracost reports plan delta and requires recorded approval when the combined active-month total exceeds USD50. |

## Plan review record

Each external plan review must record:

- environment and commit SHA;
- Terraform plan artifact reference and resource additions, changes, and destroys;
- estimated monthly cost and delta against the active-month total;
- data classification and whether synthetic or confidential inputs are permitted;
- drift result and any time-bound policy exception;
- reviewer, approval decision, and approval timestamp.

The review must not approve an apply expected to make the active-month total
exceed USD50 without explicit review, consistent with `COST-003`.

## Deferred external evidence

An operator with the matching Azure subscription and state-backend access must
run the environment plan, budget deployment, Infracost report, and daily
refresh-only drift procedure. Store only redacted plan summaries and review
records; never commit state, plan binaries, credentials, or sensitive variable
files. The story remains open until that evidence is captured or a time-bound
deferral is approved.
