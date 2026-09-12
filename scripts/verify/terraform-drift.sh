#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ "${1:-}" == "--self-test" ]]; then
  self_test_root="$(mktemp -d "${TMPDIR:-/tmp}/tally-drift-self-test.XXXXXX")"
  trap 'rm -rf "${self_test_root}"' EXIT
  fake_log="${self_test_root}/terraform.log"
  fake_terraform="${self_test_root}/terraform"
  cat >"${fake_terraform}" <<'FAKE_TERRAFORM'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "${FAKE_TERRAFORM_LOG}"
if [[ "${1:-}" == "-chdir="* && "${2:-}" == "plan" ]]; then
  exit "${FAKE_TERRAFORM_STATUS:-0}"
fi
exit 0
FAKE_TERRAFORM
  chmod +x "${fake_terraform}"

  run_case() {
    local expected="$1"
    local fake_status="$2"
    set +e
    FAKE_TERRAFORM_LOG="$fake_log" FAKE_TERRAFORM_STATUS="$fake_status" \
      ENVIRONMENT=dev TF_STATE_RESOURCE_GROUP=rg TF_STATE_STORAGE_ACCOUNT=storage \
      TERRAFORM_BIN="$fake_terraform" bash "$0"
    local actual=$?
    set -e
    [[ "$actual" == "$expected" ]] || { echo "drift self-test expected ${expected}, got ${actual}" >&2; exit 1; }
  }

  : >"${fake_log}"
  run_case 0 0
  run_case 2 2
  run_case 1 1

  if ! grep -q -- '-reconfigure' "${fake_log}" || \
     ! grep -q -- '-lockfile=readonly' "${fake_log}" || \
     ! grep -q -- '-backend-config=resource_group_name=rg' "${fake_log}" || \
     ! grep -q -- '-backend-config=storage_account_name=storage' "${fake_log}" || \
     ! grep -q -- '-backend-config=container_name=dev' "${fake_log}" || \
     ! grep -q -- 'plan -refresh-only -input=false -lock-timeout=5m -detailed-exitcode' "${fake_log}" || \
     grep -q -- 'apply' "${fake_log}"; then
    echo "drift self-test did not verify the backend and refresh-only command contract" >&2
    exit 1
  fi

  set +e
  ENVIRONMENT=dev TERRAFORM_BIN="$fake_terraform" bash "$0" >/dev/null 2>&1
  missing_backend_status=$?
  set -e
  [[ "$missing_backend_status" == 1 ]] || { echo "drift self-test accepted missing backend configuration" >&2; exit 1; }
  echo "Terraform drift self-test passed."
  exit 0
fi

environment="${ENVIRONMENT:-}"
case "${environment}" in
  dev|demo|prod-reference) ;;
  *) echo "ENVIRONMENT must be dev, demo, or prod-reference" >&2; exit 1 ;;
esac

: "${TF_STATE_RESOURCE_GROUP:?TF_STATE_RESOURCE_GROUP is required for authenticated drift checks}"
: "${TF_STATE_STORAGE_ACCOUNT:?TF_STATE_STORAGE_ACCOUNT is required for authenticated drift checks}"

terraform_bin="${TERRAFORM_BIN:-terraform}"
terraform_root="${root}/infra/terraform/environments/${environment}"
plan_file="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-drift.XXXXXX.tfplan")"
trap 'rm -f "${plan_file}"' EXIT

"${terraform_bin}" -chdir="${terraform_root}" init -reconfigure -input=false -lockfile=readonly \
  -backend-config="resource_group_name=${TF_STATE_RESOURCE_GROUP}" \
  -backend-config="storage_account_name=${TF_STATE_STORAGE_ACCOUNT}" \
  -backend-config="container_name=${environment}"
set +e
"${terraform_bin}" -chdir="${terraform_root}" plan -refresh-only -input=false -lock-timeout=5m -detailed-exitcode -out="${plan_file}" "$@"
status=$?
set -e
case "${status}" in
  0) echo "Terraform drift check passed: ${environment} is in sync." ;;
  2) echo "Terraform drift detected for ${environment}; review the refresh-only plan." >&2; exit 2 ;;
  *) echo "Terraform drift check failed for ${environment} with exit code ${status}." >&2; exit 1 ;;
esac
