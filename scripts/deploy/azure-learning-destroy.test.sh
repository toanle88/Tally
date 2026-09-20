#!/usr/bin/env bash

set -Eeuo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
test_root="$(mktemp -d "${TMPDIR:-/tmp}/tally-azure-learning-destroy-test.XXXXXX")"
trap 'rm -rf -- "$test_root"' EXIT

log_file="$test_root/calls.log"
fake_terraform="$test_root/terraform"
fake_az="$test_root/az"
fake_state_marker="$test_root/applied"
fake_plan_json="$test_root/plan.json"

cat >"$fake_terraform" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'terraform %s\n' "$*" >>"$FAKE_LOG"

if [[ "$*" == *"state list"* ]]; then
  [[ -f "$FAKE_STATE_MARKER" ]] || printf '%s\n' 'module.postgresql.azurerm_postgresql_flexible_server.this'
  exit 0
fi

if [[ "$*" == *" plan -destroy "* && "${FAKE_FAIL:-}" == "plan" ]]; then
  exit 17
fi

if [[ "$*" == *"output -raw environment_profile"* ]]; then
  printf '%s\n' "$FAKE_PROFILE"
  exit 0
fi

if [[ "$*" == *"output -raw resource_group_id"* ]]; then
  if [[ "${FAKE_BAD_RESOURCE_GROUP:-}" == "1" ]]; then
    printf '%s\n' '/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/unrelated-rg'
  else
    printf '%s\n' "/subscriptions/$ARM_SUBSCRIPTION_ID/resourceGroups/tally-fin-demo-eus-rg-01"
  fi
  exit 0
fi

if [[ "$*" == *"output -raw postgres_server_id"* ]]; then
	if [[ "${FAKE_BAD_POSTGRES:-}" == "1" ]]; then
	  printf '%s\n' "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/unrelated-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/unrelated-pg"
	  exit 0
	fi
  printf '%s\n' "/subscriptions/$ARM_SUBSCRIPTION_ID/resourceGroups/tally-fin-demo-eus-rg-01/providers/Microsoft.DBforPostgreSQL/flexibleServers/tally-fin-demo-eus-pg-01"
  exit 0
fi

for argument in "$@"; do
  [[ "$argument" == -out=* ]] && : >"${argument#-out=}"
done

if [[ "$*" == *" show -json "* ]]; then
  cat "$FAKE_PLAN_JSON"
fi

if [[ "$*" == *" apply "* ]]; then
  : >"$FAKE_STATE_MARKER"
fi
EOF

cat >"$fake_az" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'az %s\n' "$*" >>"$FAKE_LOG"

if [[ "$*" == *"account show"* ]]; then
	[[ "${FAKE_BAD_SUBSCRIPTION:-}" != "1" ]] || printf '%s\n' '22222222-2222-2222-2222-222222222222'
	[[ "${FAKE_BAD_SUBSCRIPTION:-}" == "1" ]] && exit 0
  printf '%s\n' "$ARM_SUBSCRIPTION_ID"
  exit 0
fi

if [[ "$*" == *"backup create --help"* ]]; then
	[[ "${FAKE_NO_BACKUP_NAME:-}" != "1" ]] || exit 0
  printf '%s\n' '--name'
  exit 0
fi

if [[ "$*" == *"backup show --help"* ]]; then
	[[ "${FAKE_NO_BACKUP_NAME:-}" != "1" ]] || exit 0
  printf '%s\n' '--name'
  exit 0
fi

if [[ "$*" == *"backup create"* ]]; then
  [[ "${FAKE_FAIL:-}" != "backup" ]] || exit 18
  exit 0
fi

if [[ "$*" == *"backup show"* ]]; then
  [[ "${FAKE_FAIL:-}" != "backup-show" ]] || exit 19
  for argument in "$@"; do
    if [[ "${previous_argument:-}" == "--name" ]]; then
      printf '%s\n' "$argument"
      exit 0
    fi
    previous_argument="$argument"
  done
  exit 20
fi

if [[ "$*" == *"group exists"* ]]; then
  printf '%s\n' 'false'
  exit 0
fi
EOF

cat >"$fake_plan_json" <<'EOF'
{"resource_changes":[{"address":"module.resource_group.azurerm_resource_group.this","change":{"actions":["delete"]}},{"address":"module.postgresql.azurerm_postgresql_flexible_server.this","change":{"actions":["delete"]}}]}
EOF

chmod +x "$fake_terraform" "$fake_az"

source "$script_dir/azure-learning.sh"
environment=demo
command_name=destroy
terraform_root="$test_root"
terraform_bin="$fake_terraform"
az_bin="$fake_az"
destroy_report="$test_root/report.json"
destroy_command_log="$test_root/command.log"

export FAKE_LOG="$log_file"
export FAKE_STATE_MARKER="$fake_state_marker"
export FAKE_PLAN_JSON="$fake_plan_json"
export FAKE_PROFILE=demo
export ARM_SUBSCRIPTION_ID=11111111-1111-1111-1111-111111111111
export ARM_TENANT_ID=11111111-1111-1111-1111-111111111111
export TF_STATE_RESOURCE_GROUP=tally-state
export TF_STATE_STORAGE_ACCOUNT=tallystate
export TF_VAR_organization=tally
export TF_VAR_location=eastus
export TF_VAR_region_code=eus
export TF_VAR_owner=learning-owner
export TF_VAR_cost_center=learning
export TF_VAR_tenant_id="$ARM_TENANT_ID"
export TF_VAR_api_image_repository=tally-api
export TF_VAR_api_image_digest=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
export TF_VAR_worker_image_repository=tally-worker
export TF_VAR_worker_image_digest=sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
export TF_VAR_worker_backlog_rule='{"name":"outbox-backlog"}'
export TF_VAR_postgres_administrator_login=tallyadmin
export TF_VAR_postgres_administrator_password_wo=test-only-write-only-input
export TF_VAR_postgres_administrator_password_wo_version=1
export TF_VAR_postgres_database_name=tally
export TF_VAR_learning_firewall_rules='test-host'
export TF_VAR_learning_key_vault_ip_rules='203.0.113.10'
export TF_VAR_demo_expires_on=2099-12-31
export CONFIRM_DESTROY=demo

expect_failure() {
  local label="$1"
  shift
  if ( "$@" ) >/dev/null 2>&1; then
    printf 'expected failure did not occur: %s\n' "$label" >&2
    exit 1
  fi
}

expect_failure unsupported-environment env ENVIRONMENT=prod-reference bash "$script_dir/azure-learning.sh" destroy
unset CONFIRM_DESTROY
expect_failure missing-destroy-confirmation validate_destroy_inputs
CONFIRM_DESTROY=dev
export CONFIRM_DESTROY
expect_failure mismatched-destroy-confirmation validate_destroy_inputs
export CONFIRM_DESTROY=demo

FAKE_BAD_RESOURCE_GROUP=1
export FAKE_BAD_RESOURCE_GROUP
expect_failure unrelated-resource-group verify_destroy_scope
unset FAKE_BAD_RESOURCE_GROUP

FAKE_BAD_POSTGRES=1
export FAKE_BAD_POSTGRES
verify_destroy_scope
expect_failure unrelated-postgres-server create_and_verify_database_backup
unset FAKE_BAD_POSTGRES

verify_subscription
init_state
grep -q 'container_name=demo' "$log_file"
! grep -q 'container_name=dev' "$log_file"
verify_destroy_scope
FAKE_BAD_SUBSCRIPTION=1
export FAKE_BAD_SUBSCRIPTION
expect_failure subscription-mismatch verify_subscription
unset FAKE_BAD_SUBSCRIPTION
verify_backup_cli_contract
create_destroy_plan
printf 'demo\n' | confirm_destroy
create_and_verify_database_backup
apply_destroy_plan
verify_destroy_result

grep -q 'backup create' "$log_file"
grep -q 'plan -destroy' "$log_file"
grep -q 'plan -destroy -refresh=false' "$log_file"
grep -q ' apply ' "$log_file"
! grep -Eq 'group delete|force-unlock|docker' "$log_file"

if grep -Eiq 'password|secret|token|connection_string' "$log_file"; then
  echo 'secret-like output was sent to the fake command log' >&2
  exit 1
fi

if [[ "$backup_status" != "verified" || "$resource_group_status" != "gone" || "$state_status" != "empty" || "$lock_status" != "released" ]]; then
  echo 'destroy verification statuses were incomplete' >&2
  exit 1
fi

write_destroy_report 0
grep -q '"raw_plan_included": false' "$destroy_report"
grep -q '"raw_state_included": false' "$destroy_report"
grep -q '"raw_azure_errors_included": false' "$destroy_report"
! grep -Fq 'test-only-write-only-input' "$destroy_report"

verify_destroy_scope
create_and_verify_database_backup
[[ "$backup_status" == "not-applicable" ]] || { echo 'partial rerun did not skip absent PostgreSQL backup' >&2; exit 1; }

FAKE_FAIL=backup
export FAKE_FAIL
rm -f "$fake_state_marker"
verify_destroy_scope
expect_failure backup-failure create_and_verify_database_backup
unset FAKE_FAIL

cat >"$fake_plan_json" <<'EOF'
{"resource_changes":[{"address":"module.resource_group.azurerm_resource_group.this","change":{"actions":["update"]}}]}
EOF
FAKE_FAIL=plan
export FAKE_FAIL
expect_failure destroy-plan-failure create_destroy_plan
unset FAKE_FAIL
FAKE_NO_BACKUP_NAME=1
export FAKE_NO_BACKUP_NAME
expect_failure unsupported-backup-cli verify_backup_cli_contract
unset FAKE_NO_BACKUP_NAME
expect_failure unsafe-plan create_destroy_plan

echo 'Azure learning destroy contract tests passed.'
