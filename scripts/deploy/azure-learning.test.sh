#!/usr/bin/env bash

set -Eeuo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
test_root="$(mktemp -d "${TMPDIR:-/tmp}/tally-azure-learning-test.XXXXXX")"
trap 'rm -rf -- "$test_root"' EXIT

log_file="$test_root/calls.log"
fake_terraform="$test_root/terraform"
fake_az="$test_root/az"
fake_curl="$test_root/curl"

cat >"$fake_terraform" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'terraform %s\n' "$*" >>"$FAKE_LOG"
[[ "${FAKE_FAIL:-}" == terraform ]] && exit 17
for argument in "$@"; do
  [[ "$argument" == -out=* ]] && : >"${argument#-out=}"
done
if [[ "$*" == *" output -raw container_registry_login_server"* ]]; then printf 'example.azurecr.io\n'; fi
if [[ "$*" == *" output -raw api_url"* ]]; then printf 'https://api.example.test\n'; fi
EOF
cat >"$fake_az" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'az %s\n' "$*" >>"$FAKE_LOG"
if [[ "${FAKE_FAIL:-}" == import ]]; then exit 18; fi
if [[ "$*" == *"account show"* ]]; then printf '%s\n' "$ARM_SUBSCRIPTION_ID"; fi
if [[ "$*" == *"show-manifests"* ]]; then printf '%s\n' "$FAKE_DIGEST"; fi
EOF
cat >"$fake_curl" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'curl %s\n' "$*" >>"$FAKE_LOG"
if [[ -n "${FAKE_FAIL:-}" ]]; then exit 19; fi
EOF
chmod +x "$fake_terraform" "$fake_az" "$fake_curl"

source "$script_dir/azure-learning.sh"
environment=dev
command_name=apply
terraform_root="$test_root"
terraform_bin="$fake_terraform"
az_bin="$fake_az"
curl_bin="$fake_curl"
export FAKE_LOG="$log_file"
export ARM_SUBSCRIPTION_ID=11111111-1111-1111-1111-111111111111
export FAKE_DIGEST=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
export AZURE_API_SOURCE_IMAGE=source.example/api@${FAKE_DIGEST}
export AZURE_WORKER_SOURCE_IMAGE=source.example/worker@${FAKE_DIGEST}
export TF_VAR_api_image_repository=api
export TF_VAR_worker_image_repository=worker
export TF_VAR_api_image_digest="$FAKE_DIGEST"
export TF_VAR_worker_image_digest="$FAKE_DIGEST"
export AZURE_LEARNING_ESTIMATED_MONTHLY_COST_USD=1.00

printf 'dev\n' | confirm_foundation
apply_foundation_and_import_images
plan
printf 'dev\n' | confirm_full_plan
"$terraform_bin" apply "$plan_file"
health_check

first_call="$(sed -n '1p' "$log_file")"
second_call="$(sed -n '2p' "$log_file")"
if [[ "$first_call" != *' plan '* || "$second_call" != *' apply '* ]]; then
  printf 'foundation ordering failed: first=%s second=%s\n' "$first_call" "$second_call" >&2
  exit 1
fi
rg -q 'acr import' "$log_file" || { echo 'image import was not exercised' >&2; exit 1; }
rg -q 'curl .*health/live' "$log_file" || { echo 'health check was not exercised' >&2; exit 1; }

expect_failure() {
  local expected="$1"
  shift
  local status=0
  if (set -Eeuo pipefail; "$@") >/dev/null 2>&1; then status=0; else status=$?; fi
  [[ "$status" != 0 ]] || { echo "expected failure: $expected" >&2; exit 1; }
}

export FAKE_FAIL=terraform
expect_failure terraform-failure plan
export FAKE_FAIL=import
expect_failure import-failure apply_foundation_and_import_images
export FAKE_FAIL=health
expect_failure health-failure health_check
unset FAKE_FAIL

if sed -n '/^print_outputs()/,/^}/p' "$script_dir/azure-learning.sh" | rg -q 'password|secret|token'; then
  echo 'secret-like output was added to the allow-list' >&2
  exit 1
fi

echo 'Azure learning deployment failure-path tests passed.'
