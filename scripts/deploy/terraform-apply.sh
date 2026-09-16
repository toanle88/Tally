#!/usr/bin/env bash

set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM_BIN:-terraform}"
environment="${ENVIRONMENT:-}"

fail() {
	printf 'Terraform apply failed: %s\n' "$*" >&2
	exit 1
}

require_value() {
	[[ -n "${!1:-}" ]] || fail "${1} is required"
}

[[ "$environment" == "dev" || "$environment" == "demo" || "$environment" == "prod-reference" ]] || fail "ENVIRONMENT must be dev, demo, or prod-reference"
command -v "$terraform_bin" >/dev/null 2>&1 || fail "Terraform is required"

for name in ARM_CLIENT_ID ARM_TENANT_ID ARM_SUBSCRIPTION_ID TF_STATE_RESOURCE_GROUP TF_STATE_STORAGE_ACCOUNT; do
	require_value "$name"
done

terraform_root="${repo_root}/infra/terraform/environments/${environment}"
runner_temp="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
plan_file="$(mktemp "${runner_temp%/}/tally-${environment}-XXXXXX.tfplan")"
plan_json="$(mktemp "${runner_temp%/}/tally-${environment}-XXXXXX.json")"
plan_log="$(mktemp "${runner_temp%/}/tally-${environment}-XXXXXX.log")"
summary_file="${runner_temp%/}/tally-${environment}-plan-summary.json"

cleanup() {
	rm -f -- "$plan_file" "$plan_json" "$plan_log"
}
trap cleanup EXIT

run_quiet() {
	local phase="$1"
	shift
	if ! "$@" >"$plan_log" 2>&1; then
		printf 'Terraform %s failed; inspect the protected workflow result.\n' "$phase" >&2
		exit 1
	fi
}

export ARM_USE_OIDC=true
export ARM_USE_AZUREAD=true
export AZURE_CORE_OUTPUT=none

run_quiet init "$terraform_bin" -chdir="$terraform_root" init -reconfigure -input=false -no-color \
	-backend-config="resource_group_name=${TF_STATE_RESOURCE_GROUP}" \
	-backend-config="storage_account_name=${TF_STATE_STORAGE_ACCOUNT}" \
	-backend-config="container_name=${environment}" \
	-backend-config="subscription_id=${ARM_SUBSCRIPTION_ID}" \
	-backend-config="tenant_id=${ARM_TENANT_ID}" \
	-backend-config="use_azuread_auth=true" \
	-lockfile=readonly

run_quiet plan "$terraform_bin" -chdir="$terraform_root" plan -input=false -no-color -lock-timeout=5m -out="$plan_file"
if ! "$terraform_bin" -chdir="$terraform_root" show -json "$plan_file" >"$plan_json" 2>"$plan_log"; then
	printf 'Terraform show failed; inspect the protected workflow result.\n' >&2
	exit 1
fi
node "$repo_root/scripts/verify/terraform-plan-summary.js" "$plan_json" "$environment" "${GITHUB_SHA:-unknown}" >"$summary_file"
run_quiet apply "$terraform_bin" -chdir="$terraform_root" apply -input=false -no-color -lock-timeout=5m "$plan_file"

printf 'Terraform apply completed for %s; sanitized plan summary: %s\n' "$environment" "$summary_file"
