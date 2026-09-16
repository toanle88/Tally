#!/usr/bin/env bash

set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM_BIN:-terraform}"
az_bin="${AZ_BIN:-az}"
curl_bin="${CURL_BIN:-curl}"
environment="${ENVIRONMENT:-}"
command_name="${1:-}"

fail() {
	printf 'Azure learning deployment failed: %s\n' "$*" >&2
	exit 1
}

usage() {
	cat <<'EOF'
Usage:
  ENVIRONMENT=dev|demo make azure-learning-plan
  ENVIRONMENT=dev|demo CONFIRM_APPLY=dev|demo make azure-learning-apply

The command uses Azure CLI authentication, isolated Terraform state, and
TF_VAR_* environment variables. It never accepts prod-reference.
EOF
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is required"
}

require_value() {
	local name="$1"
	[[ -n "${!name:-}" ]] || fail "${name} is required"
}

valid_uuid() { [[ "$1" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; }
valid_digest() { [[ "$1" =~ ^sha256:[0-9a-f]{64}$ ]]; }
valid_image_source() { [[ "$1" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; }
valid_repository() { [[ "$1" =~ ^[a-z0-9]+([._/-][a-z0-9]+)*$ ]]; }

validate_common_inputs() {
	[[ "$environment" == "dev" || "$environment" == "demo" ]] || fail "ENVIRONMENT must be dev or demo"
	[[ "$command_name" == "plan" || "$command_name" == "apply" ]] || { usage >&2; exit 2; }

	for name in ARM_SUBSCRIPTION_ID ARM_TENANT_ID TF_STATE_RESOURCE_GROUP TF_STATE_STORAGE_ACCOUNT \
		TF_VAR_organization TF_VAR_location TF_VAR_region_code TF_VAR_owner TF_VAR_cost_center \
		TF_VAR_tenant_id TF_VAR_api_image_repository TF_VAR_api_image_digest \
		TF_VAR_worker_image_repository TF_VAR_worker_image_digest TF_VAR_worker_backlog_rule \
		TF_VAR_postgres_administrator_login TF_VAR_postgres_administrator_password_wo \
		TF_VAR_postgres_administrator_password_wo_version TF_VAR_postgres_database_name \
		TF_VAR_learning_firewall_rules TF_VAR_learning_key_vault_ip_rules; do
		require_value "$name"
	done

	valid_uuid "$ARM_SUBSCRIPTION_ID" || fail "ARM_SUBSCRIPTION_ID must be a UUID"
	valid_uuid "$ARM_TENANT_ID" || fail "ARM_TENANT_ID must be a UUID"
	[[ "$TF_VAR_tenant_id" == "$ARM_TENANT_ID" ]] || fail "TF_VAR_tenant_id must match ARM_TENANT_ID"
	valid_digest "$TF_VAR_api_image_digest" || fail "TF_VAR_api_image_digest must be a lowercase sha256 digest"
	valid_digest "$TF_VAR_worker_image_digest" || fail "TF_VAR_worker_image_digest must be a lowercase sha256 digest"
	valid_repository "$TF_VAR_api_image_repository" || fail "TF_VAR_api_image_repository is invalid"
	valid_repository "$TF_VAR_worker_image_repository" || fail "TF_VAR_worker_image_repository is invalid"
	if [[ "$environment" == "demo" ]]; then
		require_value TF_VAR_demo_expires_on
		[[ "$TF_VAR_demo_expires_on" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || fail "TF_VAR_demo_expires_on must use YYYY-MM-DD"
	fi
}

validate_apply_inputs() {
	[[ "${CONFIRM_APPLY:-}" == "$environment" ]] || fail "set CONFIRM_APPLY=$environment to authorize apply"
	require_value AZURE_LEARNING_ESTIMATED_MONTHLY_COST_USD
	[[ "$AZURE_LEARNING_ESTIMATED_MONTHLY_COST_USD" =~ ^[0-9]+([.][0-9]{1,2})?$ ]] || fail "AZURE_LEARNING_ESTIMATED_MONTHLY_COST_USD must be a non-negative USD amount"
	for name in AZURE_API_SOURCE_IMAGE AZURE_WORKER_SOURCE_IMAGE; do
		require_value "$name"
		valid_image_source "${!name}" || fail "${name} must be an immutable registry/repository@sha256 digest"
	done
	[[ "${AZURE_API_SOURCE_IMAGE##*@}" == "$TF_VAR_api_image_digest" ]] || fail "AZURE_API_SOURCE_IMAGE digest must match TF_VAR_api_image_digest"
	[[ "${AZURE_WORKER_SOURCE_IMAGE##*@}" == "$TF_VAR_worker_image_digest" ]] || fail "AZURE_WORKER_SOURCE_IMAGE digest must match TF_VAR_worker_image_digest"
}

terraform_root="${repo_root}/infra/terraform/environments/${environment}"
plan_file=""
foundation_plan=""
cleanup() {
	[[ -z "$plan_file" ]] || rm -f -- "$plan_file"
	[[ -z "$foundation_plan" ]] || rm -f -- "$foundation_plan"
}
trap cleanup EXIT

init_state() {
	"$terraform_bin" -chdir="$terraform_root" init -reconfigure -input=false -no-color \
		-backend-config="resource_group_name=${TF_STATE_RESOURCE_GROUP}" \
		-backend-config="storage_account_name=${TF_STATE_STORAGE_ACCOUNT}" \
		-backend-config="container_name=${environment}" \
		-backend-config="subscription_id=${ARM_SUBSCRIPTION_ID}" \
		-backend-config="tenant_id=${ARM_TENANT_ID}" \
		-backend-config="use_azuread_auth=true" \
		-lockfile=readonly
}

verify_subscription() {
	local actual
	actual="$("$az_bin" account show --subscription "$ARM_SUBSCRIPTION_ID" --query id -o tsv)" || fail "Azure CLI is not authenticated for the requested subscription"
	[[ "$actual" == "$ARM_SUBSCRIPTION_ID" ]] || fail "Azure CLI subscription does not match ARM_SUBSCRIPTION_ID"
}

plan() {
	plan_file="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-XXXXXX.tfplan")"
	"$terraform_bin" -chdir="$terraform_root" plan -input=false -no-color -lock-timeout=5m -out="$plan_file"
}

confirm_full_plan() {
	local confirmation
	printf 'Estimated monthly Azure cost: USD %s (operator supplied, not verified). Review the plan and type %s to apply: ' "$AZURE_LEARNING_ESTIMATED_MONTHLY_COST_USD" "$environment" >&2
	read -r confirmation || fail "full-plan confirmation was not provided"
	[[ "$confirmation" == "$environment" ]] || fail "full-plan confirmation did not match ENVIRONMENT"
}

confirm_foundation() {
	local confirmation
	printf 'The foundation phase will create the resource group and ACR. Type %s to authorize this first apply: ' "$environment" >&2
	read -r confirmation || fail "foundation confirmation was not provided"
	[[ "$confirmation" == "$environment" ]] || fail "foundation confirmation did not match ENVIRONMENT"
}

apply_foundation_and_import_images() {
	local acr_login_server acr_name
	foundation_plan="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-foundation-XXXXXX.tfplan")"
	"$terraform_bin" -chdir="$terraform_root" plan -input=false -no-color -lock-timeout=5m \
		-target=module.resource_group -target=module.container_registry -out="$foundation_plan"
	"$terraform_bin" -chdir="$terraform_root" apply -input=false -no-color -lock-timeout=5m "$foundation_plan"
	rm -f -- "$foundation_plan"
	foundation_plan=""

	acr_login_server="$("$terraform_bin" -chdir="$terraform_root" output -raw container_registry_login_server)"
	acr_name="${acr_login_server%%.*}"
	printf 'Importing immutable learning images into ACR %s\n' "$acr_name"
	"$az_bin" acr import --name "$acr_name" --source "$AZURE_API_SOURCE_IMAGE" --image "${TF_VAR_api_image_repository}:learning"
	"$az_bin" acr import --name "$acr_name" --source "$AZURE_WORKER_SOURCE_IMAGE" --image "${TF_VAR_worker_image_repository}:learning"

	for pair in \
		"$TF_VAR_api_image_repository:$TF_VAR_api_image_digest" \
		"$TF_VAR_worker_image_repository:$TF_VAR_worker_image_digest"; do
		local repository="${pair%%:*}" digest="${pair#*:}" found
		found="$("$az_bin" acr repository show-manifests --name "$acr_name" --repository "$repository" --query "[?digest=='${digest}'] | [0].digest" -o tsv)"
		[[ "$found" == "$digest" ]] || fail "ACR image digest verification failed for ${repository}"
	done
}

print_outputs() {
	local name value
	for name in environment_profile resource_group_id container_registry_name container_registry_login_server \
		static_web_app_hostname api_url worker_id postgres_server_fqdn database_id key_vault_uri \
		api_identity_principal_id worker_identity_principal_id; do
		value="$("$terraform_bin" -chdir="$terraform_root" output -raw "$name" 2>/dev/null || true)"
		[[ -n "$value" ]] && printf '%s=%s\n' "$name" "$value"
	done
}

health_check() {
	local api_url
	api_url="$("$terraform_bin" -chdir="$terraform_root" output -raw api_url)"
	printf 'Checking existing API liveness endpoint: %s/health/live\n' "$api_url"
	"$curl_bin" --fail --silent --show-error --max-time 10 --retry 12 --retry-delay 5 \
		"${api_url%/}/health/live" || fail "API liveness check failed"
	printf '\n'
}

self_test() {
	local old_environment="$environment" old_command="$command_name"
	AZURE_LEARNING_ESTIMATED_MONTHLY_COST_USD=0
	environment=prod-reference command_name=plan
	if (validate_common_inputs) 2>/dev/null; then fail "self-test accepted prod-reference"; fi
	environment=dev command_name=apply
	if (validate_apply_inputs) 2>/dev/null; then fail "self-test accepted missing confirmation"; fi
	valid_digest sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa || fail "digest validator failed"
	valid_image_source registry.example/api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa || fail "source validator failed"
	if valid_image_source registry.example/api:latest 2>/dev/null; then fail "source validator accepted mutable image"; fi
	if printf '%s\n' dev | (environment=dev; confirm_full_plan) 2>/dev/null; then :; else fail "self-test rejected matching full-plan confirmation"; fi
	if printf '%s\n' demo | (environment=dev; confirm_full_plan) 2>/dev/null; then fail "self-test accepted mismatched full-plan confirmation"; fi
	if (environment=dev; confirm_full_plan </dev/null) 2>/dev/null; then fail "self-test accepted missing full-plan confirmation"; fi
	environment="$old_environment" command_name="$old_command"
	printf 'Azure learning deployment self-tests passed.\n'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
	if [[ "$command_name" == "--self-test" ]]; then
		self_test
		exit 0
	fi

	require_command "$terraform_bin"
	validate_common_inputs
	[[ "$command_name" == "plan" ]] || { require_command "$az_bin"; require_command "$curl_bin"; }
	verify_subscription
	init_state

	if [[ "$command_name" == "plan" ]]; then
		plan
		printf 'Plan completed for %s; no resources were applied.\n' "$environment"
		exit 0
	fi

	validate_apply_inputs
	confirm_foundation
	apply_foundation_and_import_images
	plan
	confirm_full_plan
	"$terraform_bin" -chdir="$terraform_root" apply -input=false -no-color -lock-timeout=5m "$plan_file"
	print_outputs
	health_check
	printf 'Azure learning deployment completed for %s.\n' "$environment"
fi
