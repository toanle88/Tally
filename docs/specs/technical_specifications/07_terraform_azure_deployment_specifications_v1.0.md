# Finance Platform Terraform and Azure Deployment Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Implementation-ready infrastructure baseline |
| IaC | Terraform 1.x with AzureRM and AzureAD providers |
| Azure runtime | Azure Static Web Apps and Azure Container Apps |

## 1. Repository layout

```text
infra/terraform/bootstrap/
infra/terraform/modules/resource-group/
infra/terraform/modules/log-analytics/
infra/terraform/modules/container-registry/
infra/terraform/modules/container-app-environment/
infra/terraform/modules/container-app/
infra/terraform/modules/static-web-app/
infra/terraform/modules/postgresql/
infra/terraform/modules/key-vault/
infra/terraform/modules/managed-identity/
infra/terraform/modules/budget/
infra/terraform/modules/github-federation/
infra/terraform/environments/dev/
infra/terraform/environments/demo/
infra/terraform/environments/prod-reference/
```

## 2. Provider and state baseline

```hcl
terraform {
  required_version = ">= 1.8, < 2.0"
  required_providers {
    azurerm = { source = "hashicorp/azurerm", version = "~> 4.0" }
    azuread = { source = "hashicorp/azuread", version = "~> 3.0" }
    random  = { source = "hashicorp/random", version = "~> 3.6" }
  }
  backend "azurerm" {}
}

provider "azurerm" { features {} }
```

Shared state uses Azure Blob with state locking, encryption and restricted RBAC. Each environment has a separate state key and workload identity. Terraform workspaces are not the primary isolation mechanism.

## 3. Naming and tags

Resource names follow `<org>-fin-<env>-<region>-<resource>-<nn>`. Required tags: `application`, `environment`, `owner`, `cost_center`, `managed_by=terraform`, `data_classification`, and `expires_on` for disposable demo resources.

## 4. Module contracts

| Module | Required inputs | Outputs | Key resources |
|---|---|---|---|
| resource-group | name, location, tags | id, name | `azurerm_resource_group` |
| log-analytics | resource group, retention | workspace ID | Log Analytics workspace, Application Insights |
| container-registry | name, SKU | login server, ID | ACR with admin disabled |
| container-app-environment | network, logs, zone option | environment ID | Container Apps environment |
| container-app | image digest, identity, CPU/memory, scaling, secrets refs | URL, identity | API or worker Container App |
| static-web-app | repository/build config | hostname | Static Web App |
| postgresql | version, SKU, storage, backup, HA, network | server FQDN, database | Flexible Server, database, diagnostics |
| key-vault | RBAC, network, retention | vault URI | Key Vault, role assignments |
| managed-identity | name, roles | principal/client IDs | User-assigned identity |
| budget | amount, thresholds, contacts | budget ID | Subscription/resource-group budget |
| github-federation | repository, environment, subject | client ID | Entra application/federated credential |

## 5. Environment profiles

| Setting | dev | demo | prod-reference |
|---|---|---|---|
| API min/max replicas | 0 / 1 | 0 / 1 | 2 / measured maximum |
| Worker | combined or 0/1 | 0/1 | independent replicas/jobs |
| PostgreSQL | smallest burstable | burstable, disposable | HA, private access, measured SKU |
| Network | public access restricted by firewall | restricted public | private endpoints/VNet |
| Availability claim | none | none | qualified against NFRs |
| Destroy policy | allowed | expected | prohibited without change approval |

## 6. Container Apps baseline

- API listens on 8080 and exposes `/health/live`, `/health/ready`, `/metrics` only on approved ingress paths.
- Learning CPU/memory starts at 0.5 vCPU/1 GiB and is adjusted by measurement.
- Images are deployed by immutable digest from ACR.
- Managed identity reads Key Vault and pulls images.
- Scale rules use HTTP concurrency for API and backlog metrics/jobs for worker; scale-to-zero is disabled during performance qualification.

## 7. PostgreSQL baseline

- PostgreSQL major version is 18.
- TLS is required; public network is limited to explicit learning addresses and not used by prod-reference.
- Backups, retention and geo/zone options are set per environment profile.
- Database parameters are changed only through Terraform or reviewed migration/runbook steps.
- Destruction of nonlocal database resources requires a final backup/export check.

## 8. GitHub Actions federation

Each protected environment has a separate federated identity subject. Pull requests run `fmt`, `validate`, security checks and a plan. Apply runs only from protected branches after approval. State and plan artifacts are access-controlled and retained according to policy.

## 9. Cost controls

- Budgets alert at 50%, 80% and 100% of the monthly learning budget.
- Demo resources carry expiry tags and a scheduled review.
- AKS, Redis, Kafka, API Management and premium data grids are not provisioned by this baseline.
- PostgreSQL is the principal recurring learning cost; local-first development and destroy/stop procedures are documented.

## 10. Terraform tests

- `terraform fmt -check`, `validate` and provider lockfile checks.
- TFLint and security policy scanning.
- Plan assertions for no public database in prod-reference, minimum replicas, managed identity and diagnostics.
- Ephemeral environment apply/smoke/destroy test on scheduled or release cadence.
- Drift detection at least daily for shared environments.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `43cf7ea3c10f9b1b3b8030a95deeb0d274308adef7e8d82a7cbc9e57729ce594` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
