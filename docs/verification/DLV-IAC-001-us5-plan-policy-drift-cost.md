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

`terraform-check` also runs credential-free self-tests for the drift and cost
wrapper contracts. These self-tests use temporary fake tools and do not require
Azure credentials, a subscription, or the remote state backend.

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
| Security scan | `scripts/verify/terraform-security.sh` | Checkov scans each root; learning-profile findings use observed exact environment/rule/path exceptions. The non-deployable `prod-reference` topology explicitly skips nine deferred controls that require external network or replication resources; it is not treated as a production qualification. |
| Plan policy | `scripts/verify/terraform-plan-policy.js` | Standard `terraform show -json` input; versioned valid/invalid fixtures run in every local gate. |
| Shared budget | `infra/terraform/bootstrap` and `modules/budget` | One subscription budget, default monthly amount 20, thresholds 50/80/100, at least one recipient. |
| Drift | `scripts/verify/terraform-drift.sh` | `ENVIRONMENT=dev|demo|prod-reference` plus `TF_STATE_RESOURCE_GROUP` and `TF_STATE_STORAGE_ACCOUNT`; authenticated refresh-only plan, exit 0 clean / 2 drift / 1 error. |
| Cost | `scripts/verify/terraform-cost.sh` | `PLAN_JSON` plus `ACTIVE_MONTH_COST`; the Infracost `diffTotalMonthlyCost` delta is added to the active-month total using exact decimal arithmetic and requires recorded approval above USD50. |

## Plan review record

Use this copyable record for each external plan review:

```text
Environment:
Commit SHA:
Terraform plan artifact reference:
Resource additions:
Resource changes:
Resource destroys:
Estimated planned monthly cost:
Estimated monthly delta:
Active-month total:
Data classification:
Synthetic/confidential inputs permitted:
Drift result:
Time-bound policy exception and expiry:
Reviewer:
Approval decision:
Approval timestamp:
```

The review must not approve an apply expected to make the active-month total
exceed USD50 without explicit review, consistent with `COST-003`.

The cost gate accepts USD estimates. The USD 20 learning budget may use the
learner-selected local-currency equivalent under `COST-002`; no implicit
currency conversion is performed by the repository cost script.

The project owner/repository maintainer owns the repository gate, CI, and
backlog evidence. An Azure operator with matching subscription and state-backend
access owns the external checks, and the named reviewer owns the approval
decision. For every shared or instantiated environment, run the authenticated
drift command at least once every 24 hours:

```bash
ENVIRONMENT=dev \
TF_STATE_RESOURCE_GROUP=<state-resource-group> \
TF_STATE_STORAGE_ACCOUNT=<state-storage-account> \
make terraform-drift-check
```

Exit `0` records an in-sync result, exit `2` requires drift review, and exit
`1` requires operational escalation. Store only redacted summaries and review
records. `COST-010` monthly charge, orphan-resource, and budget reconciliation
remains an operations follow-up while Azure resources exist.

## Deferred external evidence

An operator with the matching Azure subscription and state-backend access must
run the environment plan, budget deployment, Infracost report, and daily
refresh-only drift procedure. Store only redacted plan summaries and review
records; never commit state, plan binaries, credentials, or sensitive variable
files. The story remains open until that evidence is captured or a time-bound
deferral is approved.
