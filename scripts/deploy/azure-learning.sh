#!/usr/bin/env bash

set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM_BIN:-terraform}"
az_bin="${AZ_BIN:-az}"
curl_bin="${CURL_BIN:-curl}"
python_bin="${PYTHON_BIN:-python3}"
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
  ENVIRONMENT=dev|demo CONFIRM_DESTROY=dev|demo make azure-learning-destroy

The command uses Azure CLI authentication, isolated Terraform state, and
TF_VAR_* environment variables. Destroy uses only the selected environment
state and never accepts prod-reference.
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
	[[ "$command_name" == "plan" || "$command_name" == "apply" || "$command_name" == "destroy" ]] || { usage >&2; exit 2; }
	[[ "$command_name" != "destroy" ]] || validate_destroy_inputs

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

validate_destroy_inputs() {
	[[ "${CONFIRM_DESTROY:-}" == "$environment" ]] || fail "set CONFIRM_DESTROY=$environment to authorize destroy"
}

terraform_root="${repo_root}/infra/terraform/environments/${environment}"
plan_file=""
foundation_plan=""
destroy_plan=""
destroy_plan_json=""
destroy_plan_log=""
destroy_command_log=""
destroy_state_file=""
post_destroy_plan=""
destroy_report=""
destroy_phase="not-started"
destroy_status="not-run"
backup_status="not-run"
backup_name=""
resource_group_id=""
resource_group_name=""
state_status="not-verified"
resource_group_status="not-verified"
lock_status="not-verified"
secret_status="not-verified"
cleanup() {
	local exit_status=$?
	[[ -z "$plan_file" ]] || rm -f -- "$plan_file"
	[[ -z "$foundation_plan" ]] || rm -f -- "$foundation_plan"
	[[ -z "$destroy_plan" ]] || rm -f -- "$destroy_plan"
	[[ -z "$destroy_plan_json" ]] || rm -f -- "$destroy_plan_json"
	[[ -z "$destroy_plan_log" ]] || rm -f -- "$destroy_plan_log"
	[[ -z "$destroy_command_log" ]] || rm -f -- "$destroy_command_log"
	[[ -z "$destroy_state_file" ]] || rm -f -- "$destroy_state_file"
	[[ -z "$post_destroy_plan" ]] || rm -f -- "$post_destroy_plan"
	if [[ "$command_name" == "destroy" && -n "$destroy_report" ]]; then
		write_destroy_report "$exit_status" || true
	fi
}
trap cleanup EXIT

write_destroy_report() {
	local exit_status="$1"
	local overall_outcome="failed"
	[[ "$exit_status" == "0" && "$destroy_status" == "succeeded" ]] && overall_outcome="pass"

	"$python_bin" - "$destroy_report" "$environment" "$overall_outcome" "$destroy_phase" "$backup_status" "$backup_name" \
		"$resource_group_id" "$resource_group_name" "$resource_group_status" "$state_status" "$lock_status" "$secret_status" <<'PY'
import json
import os
import sys

(
    report_file, environment, overall_outcome, phase, backup_status, backup_name,
    resource_group_id, resource_group_name, resource_group_status, state_status,
    lock_status, secret_status,
) = sys.argv[1:]

os.makedirs(os.path.dirname(report_file), exist_ok=True)
with open(report_file, "w", encoding="utf-8") as report:
    json.dump({
        "story_id": "DLV-IAC-002-us9",
        "environment": environment,
        "overall_outcome": overall_outcome,
        "failure_phase": phase,
        "backup_status": backup_status,
        "backup_name": backup_name or None,
        "resource_group_id": resource_group_id or None,
        "resource_group_name": resource_group_name or None,
        "resource_group_status": resource_group_status,
        "terraform_state_status": state_status,
        "state_lock_status": lock_status,
        "secret_hygiene_status": secret_status,
        "raw_plan_included": False,
        "raw_state_included": False,
        "raw_azure_errors_included": False,
    }, report, indent=2)
    report.write("\n")
PY
}

init_state() {
	local -a init_args=(
		-chdir="$terraform_root" init -reconfigure -input=false -no-color
		-backend-config="resource_group_name=${TF_STATE_RESOURCE_GROUP}" \
		-backend-config="storage_account_name=${TF_STATE_STORAGE_ACCOUNT}" \
		-backend-config="container_name=${environment}" \
		-backend-config="subscription_id=${ARM_SUBSCRIPTION_ID}" \
		-backend-config="tenant_id=${ARM_TENANT_ID}" \
		-backend-config="use_azuread_auth=true" \
		-lockfile=readonly
	)
	if [[ "$command_name" == "destroy" ]]; then
		"$terraform_bin" "${init_args[@]}" >"${destroy_command_log:-/dev/null}" 2>&1 || fail "Terraform remote-state initialization failed"
	else
		"$terraform_bin" "${init_args[@]}"
	fi
}

verify_subscription() {
	local actual subscription_log="/dev/null"
	[[ "$command_name" == "destroy" ]] && subscription_log="${destroy_command_log:-/dev/null}"
	actual="$("$az_bin" account show --subscription "$ARM_SUBSCRIPTION_ID" --query id -o tsv --only-show-errors 2>"$subscription_log")" || fail "Azure CLI is not authenticated for the requested subscription"
	[[ "$(normalize_identifier "$actual")" == "$(normalize_identifier "$ARM_SUBSCRIPTION_ID")" ]] || fail "Azure CLI subscription does not match ARM_SUBSCRIPTION_ID"
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

normalize_identifier() {
	printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

terraform_output_raw() {
	"$terraform_bin" -chdir="$terraform_root" output -raw "$1" 2>/dev/null
}

verify_destroy_scope() {
	destroy_phase="scope-verification"
	local profile expected_resource_group normalized_resource_group_id normalized_expected_resource_group
	expected_resource_group="${TF_VAR_organization}-fin-${environment}-${TF_VAR_region_code}-rg-01"
	resource_group_name="$expected_resource_group"
	[[ -z "$destroy_state_file" ]] || rm -f -- "$destroy_state_file"
	destroy_state_file="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-destroy-state-XXXXXX")"
	"$terraform_bin" -chdir="$terraform_root" state list >"$destroy_state_file" 2>/dev/null || fail "Terraform state could not be inspected"
	if [[ ! -s "$destroy_state_file" ]]; then
		resource_group_id=""
		return
	fi

	profile="$(terraform_output_raw environment_profile)" || fail "selected Terraform state does not expose an environment profile"
	[[ "$profile" == "$environment" ]] || fail "selected Terraform state profile does not match ENVIRONMENT"
	resource_group_id="$(terraform_output_raw resource_group_id)" || fail "selected Terraform state does not expose a resource-group ID"
	[[ -n "$resource_group_id" ]] || fail "selected Terraform state resource-group ID is empty"
	if [[ -n "$resource_group_id" ]]; then
		normalized_resource_group_id="$(normalize_identifier "$resource_group_id")"
		normalized_expected_resource_group="$(normalize_identifier "/subscriptions/${ARM_SUBSCRIPTION_ID}/resourcegroups/${expected_resource_group}")"
		[[ "$normalized_resource_group_id" == "$normalized_expected_resource_group" ]] || fail "Terraform resource group is outside the selected environment scope"
	fi
}

state_contains_postgres_server() {
	local address
	while IFS= read -r address; do
		[[ "$address" == "module.postgresql.azurerm_postgresql_flexible_server.this" ]] && return 0
	done <"$destroy_state_file"
	return 1
}

verify_backup_cli_contract() {
	destroy_phase="backup-cli-contract"
	local create_help show_help
	create_help="$("$az_bin" postgres flexible-server backup create --help 2>/dev/null || true)"
	show_help="$("$az_bin" postgres flexible-server backup show --help 2>/dev/null || true)"
	[[ "$create_help" == *"--name"* && "$show_help" == *"--name"* ]] || fail "Azure CLI backup commands do not expose --name; refusing destroy"
}

create_and_verify_database_backup() {
	destroy_phase="database-backup"
	if ! state_contains_postgres_server; then
		backup_status="not-applicable"
		return
	fi

	local postgres_server_id postgres_server_name expected_postgres_server expected_prefix normalized_postgres_id backup_result
	postgres_server_id="$(terraform_output_raw postgres_server_id)" || fail "PostgreSQL server output is missing from selected state"
	postgres_server_name="${postgres_server_id##*/}"
	expected_postgres_server="${TF_VAR_organization}-fin-${environment}-${TF_VAR_region_code}-pg-01"
	expected_prefix="$(normalize_identifier "/subscriptions/${ARM_SUBSCRIPTION_ID}/resourcegroups/${resource_group_name}/providers/microsoft.dbforpostgresql/flexibleservers/")"
	normalized_postgres_id="$(normalize_identifier "$postgres_server_id")"
	[[ "$postgres_server_name" == "$expected_postgres_server" && "$normalized_postgres_id" == "$expected_prefix${expected_postgres_server}" ]] || fail "PostgreSQL server is outside the selected environment scope"

	backup_name="tally-${environment}-destroy-$(date -u +%Y%m%dT%H%M%SZ)-${RANDOM}"
	"$az_bin" postgres flexible-server backup create \
		--resource-group "$resource_group_name" \
		--server-name "$postgres_server_name" \
		--name "$backup_name" \
		--subscription "$ARM_SUBSCRIPTION_ID" \
		--only-show-errors --output none >"$destroy_plan_log" 2>&1 || fail "final PostgreSQL backup failed; destroy was not attempted"

	backup_result="$("$az_bin" postgres flexible-server backup show \
		--resource-group "$resource_group_name" \
		--server-name "$postgres_server_name" \
		--name "$backup_name" \
		--subscription "$ARM_SUBSCRIPTION_ID" \
		--query name --only-show-errors -o tsv 2>"$destroy_plan_log" || true)"
	[[ "$backup_result" == "$backup_name" ]] || fail "final PostgreSQL backup could not be verified; destroy was not attempted"
	backup_status="verified"
}

create_destroy_plan() {
	destroy_phase="destroy-plan"
	[[ -z "$destroy_plan" ]] || rm -f -- "$destroy_plan"
	[[ -z "$destroy_plan_json" ]] || rm -f -- "$destroy_plan_json"
	[[ -z "$destroy_plan_log" ]] || rm -f -- "$destroy_plan_log"
	destroy_plan="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-destroy-XXXXXX.tfplan")"
	destroy_plan_json="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-destroy-XXXXXX.json")"
	destroy_plan_log="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-destroy-XXXXXX.log")"
	"$terraform_bin" -chdir="$terraform_root" plan -destroy -input=false -no-color -lock-timeout=5m -out="$destroy_plan" >"$destroy_plan_log" 2>&1 || fail "Terraform destroy plan failed"
	"$terraform_bin" -chdir="$terraform_root" show -json "$destroy_plan" >"$destroy_plan_json" 2>"$destroy_plan_log" || fail "Terraform destroy plan could not be inspected"

	"$python_bin" - "$destroy_plan_json" 2>"$destroy_plan_log" <<'PY' || fail "Terraform destroy plan contains an unsafe action"
import json
import sys

with open(sys.argv[1], encoding="utf-8") as plan_file:
    plan = json.load(plan_file)

changes = []
unsafe = []
for resource in plan.get("resource_changes", []):
    actions = resource.get("change", {}).get("actions", [])
    if actions == ["no-op"]:
        continue
    changes.append(resource)
    if actions != ["delete"]:
        unsafe.append(resource)

if unsafe:
    print(f"unsafe Terraform action for {len(unsafe)} resource(s)", file=sys.stderr)
    sys.exit(1)

print(f"Value-free destroy plan: {len(changes)} resource(s) scheduled for deletion.")
for resource in changes:
    print(f"delete {resource['address']}")
PY
}

confirm_destroy() {
	local confirmation
	printf 'The exact value-free destroy plan for %s is shown above. Type %s to authorize destruction: ' "$environment" "$environment" >&2
	read -r confirmation || fail "destroy confirmation was not provided"
	[[ "$confirmation" == "$environment" ]] || fail "destroy confirmation did not match ENVIRONMENT"
}

apply_destroy_plan() {
	destroy_phase="destroy-apply"
	"$terraform_bin" -chdir="$terraform_root" apply -input=false -no-color -lock-timeout=5m "$destroy_plan" >"$destroy_plan_log" 2>&1 || fail "Terraform destroy failed; state was preserved for recovery"
}

verify_destroy_result() {
	destroy_phase="destroy-verification"
	local group_exists state_count post_status
	group_exists="$("$az_bin" group exists --name "$resource_group_name" --subscription "$ARM_SUBSCRIPTION_ID" --only-show-errors -o tsv 2>"$destroy_plan_log" || true)"
	[[ "$group_exists" == "false" ]] || fail "selected resource group still exists after destroy"
	resource_group_status="gone"

	: >"$destroy_state_file"
	"$terraform_bin" -chdir="$terraform_root" state list >"$destroy_state_file" 2>/dev/null || fail "Terraform state could not be rechecked after destroy"
	state_count="$(wc -l <"$destroy_state_file" | tr -d '[:space:]')"
	[[ "$state_count" == "0" ]] || fail "Terraform state still contains resources after destroy"
	state_status="empty"

	post_destroy_plan="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-post-destroy-XXXXXX.tfplan")"
	set +e
	"$terraform_bin" -chdir="$terraform_root" plan -destroy -refresh=false -input=false -no-color -lock-timeout=5m -out="$post_destroy_plan" >"$destroy_plan_log" 2>&1
	post_status=$?
	set -e
	[[ "$post_status" == "0" ]] || fail "post-destroy state-lock verification failed"
	lock_status="released"
	secret_status="verified-no-secret-output"
	destroy_status="succeeded"
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
	environment=dev command_name=destroy
	if (validate_destroy_inputs) 2>/dev/null; then fail "self-test accepted missing destroy confirmation"; fi
	if printf '%s\n' demo | (environment=dev; confirm_destroy) 2>/dev/null; then fail "self-test accepted mismatched destroy confirmation"; fi
	if printf '%s\n' dev | (environment=dev; confirm_destroy) 2>/dev/null; then :; else fail "self-test rejected matching destroy confirmation"; fi
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
	if [[ "$command_name" == "destroy" ]]; then
		require_command "$az_bin"
		require_command "$python_bin"
		destroy_report="$repo_root/artifacts/deployment-destroy/${environment}.json"
		destroy_command_log="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-destroy-command-XXXXXX.log")"
	else
		[[ "$command_name" == "plan" ]] || { require_command "$az_bin"; require_command "$curl_bin"; }
	fi
	verify_subscription
	init_state

	if [[ "$command_name" == "plan" ]]; then
		plan
		printf 'Plan completed for %s; no resources were applied.\n' "$environment"
		exit 0
	fi

	if [[ "$command_name" == "destroy" ]]; then
		verify_destroy_scope
		verify_backup_cli_contract
		create_destroy_plan
		confirm_destroy
		create_and_verify_database_backup
		apply_destroy_plan
		verify_destroy_result
		printf 'Azure learning destroy completed for %s; redacted report: %s\n' "$environment" "$destroy_report"
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
