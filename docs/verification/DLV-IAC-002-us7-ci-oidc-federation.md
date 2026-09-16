# DLV-IAC-002 User Story 7 — Federated CI/CD

## Scope and evidence boundary

This record covers the repository-side implementation for `EP-IAC-001` User
Story 7. It establishes GitHub Actions OIDC trust for `dev`, `demo`, and
`prod-reference`, keeps pull-request checks credential-free, and provides a
manual, protected apply workflow. Live Azure authentication, GitHub
environment approval configuration, and bootstrap execution remain external
operator prerequisites and are not claimed here.

The repository workflows are:

- `.github/workflows/terraform.yml` — pull-request and main-branch
  credential-free Terraform verification.
- `.github/workflows/terraform-apply.yml` — manual `main`-only OIDC apply.

## Federation and role boundaries

The bootstrap root creates one federated service principal per environment with
these exact subjects:

```text
repo:toanle88/Tally:environment:dev
repo:toanle88/Tally:environment:demo
repo:toanle88/Tally:environment:prod-reference
```

The issuer is `https://token.actions.githubusercontent.com` and the audience
is `api://AzureADTokenExchange`. Wildcard subjects and long-lived Azure client
secrets are prohibited.

Each principal receives `Storage Blob Data Contributor` only on its matching
Terraform state container. The matching environment root can grant that
principal `Contributor` only on its own resource group. No subscription-wide
deployment role is introduced.

Initial setup is deliberately two-phase: an operator applies bootstrap,
creates or updates the environment resource group with the matching principal
ID, and then the protected workflow can perform later changes. Recreating a
destroyed learning environment requires repeating the reviewed bootstrap and
role-assignment sequence.

## GitHub setup checklist

For each protected GitHub environment (`dev`, `demo`, and
`prod-reference`):

1. Restrict deployments to `main` and configure required reviewers, including
   prevention of self-review where supported.
2. Store `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, and `AZURE_SUBSCRIPTION_ID` as
   environment secrets. These are identifiers, not client credentials; do not
   create `AZURE_CLIENT_SECRET` or `AZURE_CREDENTIALS`.
3. Store non-sensitive Terraform inputs as environment variables and the
   PostgreSQL write-only password only as an environment secret.
4. Set `TF_STATE_RESOURCE_GROUP` and `TF_STATE_STORAGE_ACCOUNT` to the
   bootstrap outputs.
5. Set `TF_VAR_ci_deployment_principal_id` to the matching bootstrap
   `github_federation_principal_ids` output.
6. Confirm the repository is using the exact subject for that environment and
   that the Azure role assignments are scoped only to the matching state
   container and resource group.

The workflow does not use `pull_request_target`; fork pull requests receive no
Azure credentials. Raw state and `.tfplan` files remain in the runner’s
temporary directory. Only a value-free plan summary is uploaded, with a
seven-day retention setting.

## Verification commands

```text
make terraform-ci-check
make terraform-check
make terraform-environments-check
make terraform-modules-check
terraform fmt -check -recursive infra/terraform
bash -n scripts/deploy/terraform-apply.sh
git diff --check
```

Repository evidence recorded on 2026-09-16:

- Passed: Terraform validation for bootstrap and all three environment roots.
- Passed: mock-provider tests for `dev`, `demo`, `prod-reference`, and the
  resource-group module.
- Passed: environment, module, and CI/OIDC contract self-tests; sanitized
  plan-summary generation; Terraform formatting; shell syntax; and diff
  whitespace checks.
- Not executed successfully on the review host: aggregate Make targets that
  require Linux Node, TFLint, or Checkov, because those tools were unavailable.
- Not executed successfully on the review host: `pnpm docs:check`, because
  pnpm stopped at its non-interactive dependency-directory purge guard.

The local contract is credential-free and does not prove Azure token exchange,
remote-state access, environment approval, or a successful live apply.

## Traceability

`EP-IAC-001` → `DLV-IAC-002` → User Story 7; M0; `ADR-015`, `ARC-IAC-001`,
`ARC-CICD-001`, `ARC-SEC-003`; `GFR-001`, `GFR-015`; `NFR-SEC-010`,
`NFR-SEC-015`, `NFR-MNT-001`, `NFR-MNT-008`; QG-01, QG-06, QG-08, QG-10.
