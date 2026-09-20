#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM_BIN:-terraform}"
plan_root="$(mktemp -d "${TMPDIR:-/tmp}/tally-terraform-pr-plan.XXXXXX")"
trap 'rm -rf "${plan_root}"' EXIT

fail() {
	printf 'Terraform PR plan verification failed: %s\n' "$1" >&2
	exit 1
}

command -v "${terraform_bin}" >/dev/null 2>&1 || fail "Terraform is required"
command -v node >/dev/null 2>&1 || fail "Node.js is required for plan-policy verification"

write_bootstrap_vars() {
	local file="$1"
	printf '%s\n' \
		'location = "eastus"' \
		'subscription_id = "00000000-0000-0000-0000-000000000001"' \
		'resource_group_name = "tally-ci-state"' \
		'storage_account_name = "tallycistate001"' \
		'region_code = "eus"' \
		'owner = "ci"' \
		'cost_center = "learning"' \
		'bootstrap_principal_object_id = "00000000-0000-0000-0000-000000000002"' \
		'demo_expires_on = "2099-12-31"' \
		'github_repository = "toanle88/Tally"' \
		'budget_contacts = { contact_emails = ["ci@example.invalid"], contact_roles = [], contact_groups = [] }' \
		>"${file}"
}

write_environment_vars() {
	local file="$1"
	local environment="$2"

	printf '%s\n' \
		'organization = "tally"' \
		'location = "eastus"' \
		'region_code = "eus"' \
		'owner = "ci"' \
		'cost_center = "learning"' \
		'ci_deployment_principal_id = null' \
		'tenant_id = "00000000-0000-0000-0000-000000000003"' \
		'api_image_repository = "example.invalid/tally/api"' \
		'api_image_digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"' \
		'worker_image_repository = "example.invalid/tally/worker"' \
		'worker_image_digest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"' \
		'worker_backlog_rule = { name = "ci", custom_rule_type = "prometheus", metadata = { serverAddress = "http://example.invalid", query = "up", threshold = "1" } }' \
		'postgres_administrator_login = "tallyci"' \
		'postgres_administrator_password_wo = "ci-only-placeholder-value"' \
		'postgres_administrator_password_wo_version = 1' \
		'postgres_database_name = "tally"' \
		'learning_firewall_rules = [{ name = "ci", start_ip_address = "192.0.2.10", end_ip_address = "192.0.2.10" }]' \
		'learning_key_vault_ip_rules = ["192.0.2.10"]' \
		>"${file}"

	if [[ "${environment}" == "demo" ]]; then
		printf '%s\n' 'demo_expires_on = "2099-12-31"' >>"${file}"
	fi

	if [[ "${environment}" == "prod-reference" ]]; then
		printf '%s\n' \
			'prod_container_apps_subnet_id = "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/ci/providers/Microsoft.Network/virtualNetworks/ci/subnets/apps"' \
			'prod_postgres_subnet_id = "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/ci/providers/Microsoft.Network/virtualNetworks/ci/subnets/postgres"' \
			'prod_postgres_private_dns_zone_id = "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/ci/providers/Microsoft.Network/privateDnsZones/privatelink.postgres.database.azure.com"' \
			'prod_key_vault_subnet_ids = ["/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/ci/providers/Microsoft.Network/virtualNetworks/ci/subnets/keyvault"]' \
			'prod_postgres_sku_name = "GP_Standard_D2s_v5"' \
			'prod_postgres_storage_mb = 32768' \
			'prod_api_max_replicas = 2' \
			'prod_worker_max_replicas = 1' \
			'prod_primary_availability_zone = "1"' \
			'prod_standby_availability_zone = "2"' \
			>>"${file}"
	fi
}

prepare_backend_disabled_root() {
	local source_root="$1"
	local destination_root="$2"

	mkdir -p "${destination_root}"
	cp -a "${source_root}/." "${destination_root}/"
	find "${destination_root}" -type d -name '.terraform' -prune -exec rm -rf -- {} +

	while IFS= read -r terraform_file; do
		local filtered_file="${terraform_file}.tmp"
		awk '
			function brace_delta(line, opens, closes) {
				opens = gsub(/\{/, "", line)
				closes = gsub(/\}/, "", line)
				return opens - closes
			}

			/^[[:space:]]*backend[[:space:]]+"azurerm"[[:space:]]*\{/ {
				skip_depth = brace_delta($0)
				next
			}

			skip_depth > 0 {
				skip_depth += brace_delta($0)
				next
			}

			{ print }
		' "${terraform_file}" >"${filtered_file}"
		mv "${filtered_file}" "${terraform_file}"
	done < <(find "${destination_root}" -type f -name '*.tf' -print)

	if grep -R -l -E '^[[:space:]]*backend[[:space:]]+"azurerm"[[:space:]]*\{' "${destination_root}" --include='*.tf' >/dev/null 2>&1; then
		fail "temporary Terraform root still contains an azurerm backend"
	fi
}

run_plan() {
	local environment="$1"
	local terraform_root="${plan_root}/terraform-config/${environment}"
	local vars_file="${plan_root}/${environment}.tfvars"
	local plan_file="${plan_root}/${environment}.tfplan"
	local json_file="${plan_root}/${environment}.json"

	if [[ "${environment}" == "bootstrap" ]]; then
		write_bootstrap_vars "${vars_file}"
	else
		write_environment_vars "${vars_file}" "${environment}"
	fi

	printf 'Running credential-free Terraform plan for %s.\n' "${environment}"
	TF_DATA_DIR="${plan_root}/data-${environment}" "${terraform_bin}" -chdir="${terraform_root}" init -backend=false -input=false -lockfile=readonly -no-color >/dev/null
	TF_DATA_DIR="${plan_root}/data-${environment}" \
		ARM_USE_CLI=false \
		ARM_USE_MSI=false \
			ARM_USE_OIDC=false \
			ARM_USE_AZUREAD=false \
			ARM_SUBSCRIPTION_ID=00000000-0000-0000-0000-000000000001 \
			ARM_TENANT_ID=00000000-0000-0000-0000-000000000003 \
			ARM_SKIP_CREDENTIALS_VALIDATION=true \
			ARM_SKIP_PROVIDER_REGISTRATION=true \
		"${terraform_bin}" -chdir="${terraform_root}" plan \
			-refresh=false \
			-input=false \
			-lock=false \
			-no-color \
			-var-file="${vars_file}" \
			-out="${plan_file}" >/dev/null
	"${terraform_bin}" -chdir="${terraform_root}" show -json "${plan_file}" >"${json_file}"
	node "${root}/scripts/verify/terraform-plan-policy.js" --environment "${environment}" --plan-json "${json_file}" >/dev/null
}

prepare_backend_disabled_root "${root}/infra/terraform" "${plan_root}/terraform-config"

for environment in bootstrap dev demo prod-reference; do
	run_plan "${environment}"
done

printf 'Terraform credential-free PR plan verification passed.\n'
