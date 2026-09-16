# DLV-IAC-002 User Story 6 — Optional learning environment

## Scope and evidence boundary

This record covers the repository-side deployment workflow for the optional
Azure `dev` and disposable `demo` profiles. It does not claim production
qualification, CI/OIDC federation, smoke-test qualification, destroy safety,
or recovery evidence owned by later stories.

The deployment entry points are:

```bash
ENVIRONMENT=dev make azure-learning-plan
CONFIRM_APPLY=dev ENVIRONMENT=dev make azure-learning-apply
make azure-learning-deployment-check
```

The apply command uses Azure CLI authentication and the environment-specific
Azure Blob backend container. It creates the resource group and ACR first,
imports externally supplied immutable API and worker images, verifies the
requested digests, then plans the complete Terraform profile and pauses for a
confirmation before the foundation apply and a second interactive
environment-name confirmation before applying the complete profile.
Temporary plan files are removed on exit. Only an allow-list of non-sensitive
Terraform outputs is printed.

Apply requires `AZURE_LEARNING_ESTIMATED_MONTHLY_COST_USD`. This is an
operator-supplied, unverified learning estimate displayed before confirmation,
not a billing quotation or live cost qualification.

The wrapper uses the operator's Azure CLI identity for this learning workflow.
Bootstrap-provisioned matching state identities and OIDC federation remain
pending qualifications owned by Stories 2 and 7.

## Repository verification

The credential-free wrapper checks passed with:

```text
make azure-learning-deployment-check
terraform fmt -check -recursive infra/terraform
git diff --check
```

The existing environment contract and provider-backed checks remain required:

```text
make terraform-environments-check
make terraform-check
make terraform-modules-check
make terraform-tools-check
make terraform-lint-check
make terraform-security-check
```

They require the repository's Node.js and pinned Terraform verification tools.
No Azure credentials, state, plan, image, or application data is committed by
this delivery.

## Health and data boundary

The current API exposes only `GET /health/live`; the learning Container Apps
profile uses that endpoint for both liveness and readiness probes. The approved
TS07 specification names a distinct `/health/ready` endpoint, so this is an
explicit learning-profile exception and not production health evidence.

The workflow does not add API routes, migrations, seed data, finance
capabilities, authorization behavior, or Static Web Apps content deployment.
Only synthetic, non-sensitive deployment data is permitted.

## Live evidence template

To complete authenticated evidence, record:

```text
Environment:
Commit SHA:
Operator and timestamp:
Azure subscription/region:
Terraform plan review and cost decision:
API image digest:
Worker image digest:
Resource outputs (redacted, allow-listed):
GET /health/live result:
Synthetic-data confirmation:
Unresolved limitations:
```
