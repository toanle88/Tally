# DLV-IAC-001 User Story 2 — Protected Remote-State Verification

## Scope

This record verifies the protected remote-state bootstrap for:

- `infra/terraform/bootstrap`
- `infra/terraform/environments/dev`
- `infra/terraform/environments/demo`
- `infra/terraform/environments/prod-reference`

The bootstrap root owns the state resource group, StorageV2 account, private
Blob containers, state identities, and container-scoped RBAC. Workload modules,
GitHub federation, Azure application resources, and live deployment are outside
this story.

## State design

| Root | Container | Key | State identity |
|---|---|---|---|
| bootstrap | `bootstrap` | `bootstrap/terraform.tfstate` | Bootstrap operator principal |
| dev | `dev` | `dev/terraform.tfstate` | `dev` user-assigned identity |
| demo | `demo` | `demo/terraform.tfstate` | `demo` user-assigned identity |
| prod-reference | `prod-reference` | `prod-reference/terraform.tfstate` | `prod-reference` user-assigned identity |

Containers are private. The account requires HTTPS/TLS 1.2, disables Shared Key
authentication and anonymous nested-item access, defaults to OAuth, and enables
Blob versioning with seven-day deletion retention. Azure Blob leases provide
Terraform state locking.

## Two-phase bootstrap

Prerequisites:

- Terraform `>= 1.8, < 2.0`, Azure CLI, and the reviewed provider lockfile.
- An Azure subscription and region.
- A globally unique lowercase storage-account name.
- A `demo_expires_on` value in `YYYY-MM-DD` format for the disposable demo
  state identity.
- An Entra principal with resource creation and
  `Microsoft.Authorization/roleAssignments/write` permissions.
- Required values supplied through environment variables or a temporary
  external tfvars file. The required variable names are `location`,
  `resource_group_name`, `storage_account_name`, `region_code`, `owner`,
  `cost_center`, `bootstrap_principal_object_id`, and `demo_expires_on`.
- `data_classification` is optional and defaults to `confidential`; `tags` is
  optional and defaults to an empty map.

The bootstrap state storage does not exist before the first apply. Run the
first apply with backend initialization disabled and keep the local state file
outside source control:

```bash
az login
az account set --subscription "$AZURE_SUBSCRIPTION_ID"
export ARM_SUBSCRIPTION_ID="$AZURE_SUBSCRIPTION_ID"
export ARM_USE_CLI=true
export TF_VAR_location="$AZURE_LOCATION"
export TF_VAR_resource_group_name="$TF_STATE_RESOURCE_GROUP"
export TF_VAR_storage_account_name="$TF_STATE_STORAGE_ACCOUNT"
export TF_VAR_region_code="$AZURE_REGION_CODE"
export TF_VAR_owner="$TALLY_OWNER"
export TF_VAR_cost_center="$TALLY_COST_CENTER"
export TF_VAR_bootstrap_principal_object_id="$BOOTSTRAP_PRINCIPAL_OBJECT_ID"
export TF_VAR_demo_expires_on="$DEMO_EXPIRES_ON"

terraform -chdir=infra/terraform/bootstrap init \
  -backend=false -input=false -lockfile=readonly
terraform -chdir=infra/terraform/bootstrap plan \
  -input=false -out=/tmp/tally-bootstrap.tfplan
terraform -chdir=infra/terraform/bootstrap apply \
  -input=false /tmp/tally-bootstrap.tfplan
```

After the storage account and containers exist, migrate bootstrap state to the
protected Blob backend. Authentication remains through Entra/Azure CLI; never
use an access key or put credentials in backend configuration:

```bash
export ARM_USE_AZUREAD=true
terraform -chdir=infra/terraform/bootstrap init -reconfigure \
  -input=false -lockfile=readonly \
  -backend-config="resource_group_name=$TF_STATE_RESOURCE_GROUP" \
  -backend-config="storage_account_name=$TF_STATE_STORAGE_ACCOUNT" \
  -backend-config="container_name=bootstrap"
```

Environment roots receive the same resource-group and storage-account backend
values, use their fixed source key, and must use the matching identity. Actual
environment initialization waits until that identity has usable authentication
through User Story 7 or an attached Azure runner. The bootstrap outputs
`state_identity_client_ids` and `state_identity_principal_ids`; the runner must
select the corresponding identity for each root rather than using the bootstrap
operator identity. This repository gate does not claim that authenticated
identity selection has been exercised.

For example, an attached Azure runner can initialize `dev` with the `dev`
identity output from bootstrap. Repeat the same procedure with the matching
identity and key for `demo` and `prod-reference`:

```bash
export ARM_SUBSCRIPTION_ID="$AZURE_SUBSCRIPTION_ID"
export ARM_USE_MSI=true
export ARM_CLIENT_ID="$DEV_STATE_IDENTITY_CLIENT_ID"
export ARM_USE_AZUREAD=true

terraform -chdir=infra/terraform/environments/dev init -reconfigure \
  -input=false -lockfile=readonly \
  -backend-config="resource_group_name=$TF_STATE_RESOURCE_GROUP" \
  -backend-config="storage_account_name=$TF_STATE_STORAGE_ACCOUNT" \
  -backend-config="container_name=dev" \
  -backend-config="client_id=$DEV_STATE_IDENTITY_CLIENT_ID"
```

The runner must obtain `DEV_STATE_IDENTITY_CLIENT_ID` from the bootstrap
`state_identity_client_ids` output and authenticate as that identity; setting a
client ID alone does not grant access or replace the required Azure role
assignment.

## Recovery and ownership

- The bootstrap root owns the state resource group, storage account, containers,
  identities, and RBAC assignments. Workload roots own only future workload
  resources.
- Before `force-unlock`, confirm no active writer exists and use only the exact
  lock ID reported by Terraform.
- For state corruption or accidental deletion, list Blob versions, promote an
  approved prior version, then run refresh and plan before any apply. Record the
  operator, timestamp, environment, selected version, result, and exception.
- Failed applies are recovered by preserving the state and rerunning a reviewed
  refresh/plan; portal edits are not the normal recovery path.
- If Azure AD has not yet replicated a newly created managed identity when a
  role assignment fails, wait for propagation, rerun the reviewed plan, and
  apply only that reviewed plan. Do not broaden the role scope or substitute a
  personal principal.
- State, plans, backend configuration files, and outputs are confidential and
  must not be committed or printed into CI logs.

## Verification evidence

The credential-free `make terraform-check` gate verifies:

- Terraform and provider constraints, lockfiles, formatting, initialization,
  and validation for all four roots.
- Bootstrap resource inventory, required tags, protected storage settings,
  private containers, scoped Blob Data Contributor assignments, and all state
  keys.
- Absence of workload resources in environment roots, credentials, state,
  plans, variable files, and generated artifacts.

No Azure resource creation or live Azure plan is claimed by this local record.
The per-environment identity-authentication acceptance and the delivery-plan
requirement for a live plan remain pending until an Azure-authenticated runner
records that evidence.
