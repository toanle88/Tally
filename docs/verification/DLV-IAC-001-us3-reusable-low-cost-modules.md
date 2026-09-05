# DLV-IAC-001 User Story 3 Verification

Status: implemented on `feat/dlv-iac-001-us3-reusable-low-cost-modules`; live Azure deployment remains out of scope.

The module contract adds these TS07-named modules: resource group, Log Analytics/Application Insights, Container Registry, Container Apps environment, Container App, Static Web App, PostgreSQL Flexible Server, Key Vault, user-assigned managed identity, budget, and GitHub federation.

Every module has `main.tf`, `variables.tf`, `outputs.tf`, and provider-source `versions.tf`; the Container App and PostgreSQL modules also have focused `tests/contract.tftest.hcl` fixtures. Providers, backends, roots, state, and child-module composition remain root-owned. Inputs use caller-provided names and tags; no secret values are output. Container Apps require a lowercase immutable `sha256` digest and use a user-assigned identity for ACR and Key Vault references. ACR administration is disabled. PostgreSQL requires major version 18, write-only administrator password input, explicit backup/HA/network inputs, diagnostics, `require_secure_transport=ON`, and TLS 1.2 minimum. Budgets expose subscription/resource-group scope and threshold notifications. GitHub federation uses the fixed GitHub issuer/audience and a caller-provided non-wildcard subject.

The checked-in contract oracle is `scripts/verify/terraform-modules-contract.json`; `scripts/verify/terraform-modules-contract.js` checks module shape, provider/backend boundaries, required inputs, validation blocks, secret hygiene, security-critical literals, and the focused Terraform test fixtures. `scripts/verify/terraform-modules.sh` initializes temporary copies of the tested modules and executes the focused Terraform tests, so generated provider lockfiles remain outside the source tree. Provider-backed `init`/`validate` and these tests require a networked CI environment because this workspace cannot resolve the Terraform Registry or start its cached provider binaries. These checks are credential-free; Azure plan/apply and environment composition remain Story 4/5 scope.

Source basis: `docs/specs/technical_specifications/07_terraform_azure_deployment_specifications_v1.0.md`, `docs/specs/system_design/04_security_deployment_operations_v1.0.md`, `docs/specs/technical_specifications/06_security_identity_authorization_specifications_v1.0.md`, and Story 3 in `docs/backlog/stories/EP-IAC-001_user_stories.md`.

Verification commands:

```text
bash -n scripts/verify/terraform-modules.sh
node scripts/verify/terraform-modules-contract.js
make terraform-modules-check
terraform -chdir=infra/terraform/modules/container-app test
terraform -chdir=infra/terraform/modules/postgresql test
make terraform-check
make check
pnpm docs:check
git diff --check
```
