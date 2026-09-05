#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM_BIN:-terraform}"
terraform_root="${root}/infra/terraform"

roots=(
	"${terraform_root}/bootstrap"
	"${terraform_root}/environments/dev"
	"${terraform_root}/environments/demo"
	"${terraform_root}/environments/prod-reference"
)

fail() {
	echo "terraform verification failed: $*" >&2
	exit 1
}

command -v "${terraform_bin}" >/dev/null 2>&1 || fail "Terraform is required"

terraform_version="$(${terraform_bin} version -json | sed -n 's/.*"terraform_version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[[ -n "${terraform_version}" ]] || fail "could not determine Terraform version"
terraform_major="${terraform_version%%.*}"
terraform_minor="${terraform_version#*.}"
terraform_minor="${terraform_minor%%.*}"
(( terraform_major == 1 && terraform_minor >= 8 )) || fail "Terraform ${terraform_version} is outside >= 1.8, < 2.0"

test -d "${terraform_root}/modules" || fail "missing modules boundary"

require_contract() {
	local root_path="$1"
	local contract="${root_path}/versions.tf"
	test -f "${contract}" || fail "missing root contract: ${contract}"

	grep -Fq 'required_version = ">= 1.8, < 2.0"' "${contract}" || fail "invalid Terraform constraint: ${contract}"
	grep -Fq 'source  = "hashicorp/azurerm"' "${contract}" || fail "missing AzureRM source: ${contract}"
	grep -Fq 'version = "~> 4.0"' "${contract}" || fail "invalid AzureRM constraint: ${contract}"
	grep -Fq 'source  = "hashicorp/azuread"' "${contract}" || fail "missing AzureAD source: ${contract}"
	grep -Fq 'version = "~> 3.0"' "${contract}" || fail "invalid AzureAD constraint: ${contract}"
	grep -Fq 'source  = "hashicorp/random"' "${contract}" || fail "missing Random source: ${contract}"
	grep -Fq 'version = "~> 3.6"' "${contract}" || fail "invalid Random constraint: ${contract}"
	grep -Fq 'backend "azurerm"' "${contract}" || fail "missing AzureRM backend declaration: ${contract}"
	grep -Eq 'use_azuread_auth[[:space:]]*=[[:space:]]*true' "${contract}" || fail "AzureRM backend must use Entra authentication: ${contract}"
}

require_text() {
	local pattern="$1"
	local file="$2"
	local description="$3"
	if ! rg -q --glob '*.tf' --glob '!.terraform/**' "${pattern}" "${file}"; then
		fail "missing ${description}: ${file}"
	fi
}

check_backend_key() {
	local root_path="$1"
	local expected_key="$2"
	local contract="${root_path}/versions.tf"
	grep -Eq "key[[:space:]]*=[[:space:]]*\"${expected_key}\"" "${contract}" || fail "invalid backend state key in ${contract}; expected ${expected_key}"
}

check_exact_collection() {
	local file="$1"
	local collection_name="$2"
	shift 2
	local expected_count="$#"
	local actual_values
	local actual_count
	local expected

	actual_values="$(awk -v collection_name="${collection_name}" '
		$0 ~ "^[[:space:]]*" collection_name "[[:space:]]*=[[:space:]]*toset\\(\\[" { in_collection = 1; next }
		in_collection && /^[[:space:]]*\]/ { exit }
		in_collection && match($0, /"[^"]+"/) { print substr($0, RSTART + 1, RLENGTH - 2) }
	' "${file}")"
	actual_count="$(printf '%s\n' "${actual_values}" | sed '/^$/d' | wc -l | tr -d ' ')"
	[[ "${actual_count}" -eq "${expected_count}" ]] || fail "invalid ${collection_name} collection in ${file}"

	for expected in "$@"; do
		printf '%s\n' "${actual_values}" | grep -Fxq "${expected}" || fail "missing ${expected} from ${collection_name} collection in ${file}"
	done
}

check_lockfile() {
	local root_path="$1"
	local lockfile="${root_path}/.terraform.lock.hcl"
	test -s "${lockfile}" || fail "missing provider lockfile: ${lockfile}"

	for provider in azurerm azuread random; do
		grep -Fq "provider \"registry.terraform.io/hashicorp/${provider}\"" "${lockfile}" || fail "missing ${provider} lock entry: ${lockfile}"
	done
	grep -Fq 'constraints = "~> 4.0"' "${lockfile}" || fail "missing AzureRM lock constraint: ${lockfile}"
	[[ "$(grep -Ec '^[[:space:]]*constraints[[:space:]]*=' "${lockfile}")" -eq 3 ]] || fail "lockfile does not constrain all providers: ${lockfile}"
	[[ "$(grep -Ec '^[[:space:]]*version[[:space:]]*=[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"' "${lockfile}")" -eq 3 ]] || fail "lockfile does not resolve all providers: ${lockfile}"
	[[ "$(grep -Ec '^[[:space:]]*hashes[[:space:]]*=[[:space:]]*\[' "${lockfile}")" -eq 3 ]] || fail "lockfile does not contain checksums for all providers: ${lockfile}"
}

echo "== Terraform root contract verification =="
for root_path in "${roots[@]}"; do
	test -d "${root_path}" || fail "missing Terraform root: ${root_path}"
	require_contract "${root_path}"

	echo "-- ${root_path#${root}/}"
	"${terraform_bin}" -chdir="${root_path}" fmt -check
	"${terraform_bin}" -chdir="${root_path}" init -backend=false -input=false -lockfile=readonly -no-color
	"${terraform_bin}" -chdir="${root_path}" validate -no-color
	check_lockfile "${root_path}"
done

echo "== State bootstrap contract verification =="
bootstrap_root="${terraform_root}/bootstrap"
for resource_pattern in \
	'resource[[:space:]]+"azurerm_resource_group"[[:space:]]+"state"' \
	'resource[[:space:]]+"azurerm_storage_account"[[:space:]]+"state"' \
	'resource[[:space:]]+"azurerm_storage_container"[[:space:]]+"state"' \
	'resource[[:space:]]+"azurerm_user_assigned_identity"[[:space:]]+"state"' \
	'resource[[:space:]]+"azurerm_role_assignment"[[:space:]]+"bootstrap_operator"' \
	'resource[[:space:]]+"azurerm_role_assignment"[[:space:]]+"environment_state"'; do
	require_text "${resource_pattern}" "${bootstrap_root}" "bootstrap resource ${resource_pattern}"
done

for setting in \
	'https_traffic_only_enabled[[:space:]]*=[[:space:]]*true' \
	'min_tls_version[[:space:]]*=[[:space:]]*\"TLS1_2\"' \
	'allow_nested_items_to_be_public[[:space:]]*=[[:space:]]*false' \
	'shared_access_key_enabled[[:space:]]*=[[:space:]]*false' \
	'default_to_oauth_authentication[[:space:]]*=[[:space:]]*true' \
	'infrastructure_encryption_enabled[[:space:]]*=[[:space:]]*true' \
	'public_network_access_enabled[[:space:]]*=[[:space:]]*true' \
	'versioning_enabled[[:space:]]*=[[:space:]]*true' \
	'container_access_type[[:space:]]*=[[:space:]]*\"private\"' \
	'role_definition_name[[:space:]]*=[[:space:]]*\"Storage Blob Data Contributor\"'; do
	require_text "${setting}" "${bootstrap_root}" "state protection setting ${setting}"
done

for required_tag in application environment owner cost_center managed_by data_classification; do
	require_text "${required_tag}[[:space:]]*=" "${bootstrap_root}" "required tag ${required_tag}"
done
require_text 'expires_on[[:space:]]*=' "${bootstrap_root}" "demo expiry tag"

require_text 'state_containers[[:space:]]*=[[:space:]]*toset' "${bootstrap_root}" "state container inventory"
check_exact_collection "${bootstrap_root}/main.tf" "state_containers" bootstrap dev demo prod-reference
require_text 'workload_environments[[:space:]]*=[[:space:]]*toset' "${bootstrap_root}" "workload identity inventory"
check_exact_collection "${bootstrap_root}/main.tf" "workload_environments" dev demo prod-reference
require_text 'scope[[:space:]]*=[[:space:]]*azurerm_storage_container\.state\[each\.key\]\.id' "${bootstrap_root}" "environment container-scoped RBAC"
require_text 'principal_id[[:space:]]*=[[:space:]]*azurerm_user_assigned_identity\.state\[each\.key\]\.principal_id' "${bootstrap_root}" "environment identity-scoped RBAC"

if rg -n '^[[:space:]]*(module|data)[[:space:]]+"' "${bootstrap_root}" --glob '*.tf' --glob '!.terraform/**'; then
	fail "bootstrap must not depend on modules or data sources"
fi

check_backend_key "${terraform_root}/bootstrap" "bootstrap/terraform.tfstate"
check_backend_key "${terraform_root}/environments/dev" "dev/terraform.tfstate"
check_backend_key "${terraform_root}/environments/demo" "demo/terraform.tfstate"
check_backend_key "${terraform_root}/environments/prod-reference" "prod-reference/terraform.tfstate"

for environment_root in \
	"${terraform_root}/environments/dev" \
	"${terraform_root}/environments/demo" \
	"${terraform_root}/environments/prod-reference"; do
	if rg -n '^[[:space:]]*(resource|data|module)[[:space:]]+"' "${environment_root}" --glob '*.tf' --glob '!.terraform/**'; then
		fail "workload resources are out of scope for User Story 2: ${environment_root}"
	fi
done

echo "== Boundary and artifact verification =="
source_files="$(rg --files --hidden -g '!.terraform/**' "${terraform_root}" || true)"
while IFS= read -r file; do
	[[ -z "${file}" ]] && continue
	case "${file}" in
		*.tf|*.terraform.lock.hcl|*.gitkeep) ;;
		*) fail "unexpected infrastructure artifact: ${file}" ;;
	esac
done <<<"${source_files}"

if rg -n -i '(client_secret|access_key|secret_value|password)[[:space:]]*=[[:space:]]*"[^$]' "${terraform_root}" --glob '*.tf' --glob '!.terraform/**'; then
	fail "literal credential material found in Terraform source"
fi

if rg -n -i '(primary_access_key|connection_string|secret_value|client_secret|password)[[:space:]]*=' "${bootstrap_root}" --glob '*.tf' --glob '!.terraform/**'; then
	fail "secret or credential output/configuration found in bootstrap source"
fi

files="$(git ls-files --cached --others --exclude-standard -- "${terraform_root}" || true)"
while IFS= read -r file; do
	[[ -z "${file}" ]] && continue
	case "${file}" in
		*.tfstate|*.tfstate.*|*.tfplan|*.tfvars|*.tfvars.json|*/crash.log|*/crash.*.log)
			fail "forbidden tracked or unignored Terraform artifact: ${file}"
			;;
		*/internal/*|*/cmd/*|*/web/*)
			fail "application/domain path crossed into infrastructure: ${file}"
			;;
	esac
done <<<"${files}"

echo "Terraform repository boundary verification passed"
