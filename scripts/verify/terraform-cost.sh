#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ "${1:-}" == "--self-test" ]]; then
  node "${root}/scripts/verify/terraform-cost-policy.js" --self-test

  self_test_root="$(mktemp -d "${TMPDIR:-/tmp}/tally-cost-self-test.XXXXXX")"
  trap 'rm -rf "${self_test_root}"' EXIT
  plan_fixture="${self_test_root}/plan.json"
  report_fixture="${self_test_root}/report.json"
  fake_infracost="${self_test_root}/infracost"
  printf '%s\n' '{}' >"${plan_fixture}"
  printf '%s\n' '{"currency":"USD","totalMonthlyCost":"100","diffTotalMonthlyCost":"1"}' >"${report_fixture}"
  cat >"${fake_infracost}" <<'FAKE_INFRACOST'
#!/usr/bin/env bash
set -euo pipefail
output_file=""
for argument in "$@"; do
  if [[ "${previous_argument:-}" == "--out-file" ]]; then output_file="$argument"; fi
  previous_argument="$argument"
done
[[ -n "$output_file" ]] || exit 1
cp "${FAKE_INFRACOST_REPORT}" "$output_file"
FAKE_INFRACOST
  chmod +x "${fake_infracost}"

  set +e
  PLAN_JSON="${plan_fixture}" ACTIVE_MONTH_COST=49 ACTIVE_MONTH_COST_LIMIT=50 \
    INFRACOST_BIN="${fake_infracost}" FAKE_INFRACOST_REPORT="${report_fixture}" COST_APPROVAL= \
    bash "$0" >/dev/null
  status=$?
  set -e
  [[ "$status" == 0 ]] || { echo "cost self-test wrapper smoke case failed" >&2; exit 1; }

  set +e
  PLAN_JSON="${plan_fixture}" ACTIVE_MONTH_COST=50 ACTIVE_MONTH_COST_LIMIT=50 \
    INFRACOST_BIN="${fake_infracost}" FAKE_INFRACOST_REPORT="${report_fixture}" COST_APPROVAL= \
    bash "$0" >/dev/null
  status=$?
  set -e
  [[ "$status" == 2 ]] || { echo "cost self-test did not require approval above the threshold" >&2; exit 1; }

  set +e
  PLAN_JSON="${plan_fixture}" ACTIVE_MONTH_COST=50 ACTIVE_MONTH_COST_LIMIT=50 \
    INFRACOST_BIN="${fake_infracost}" FAKE_INFRACOST_REPORT="${report_fixture}" COST_APPROVAL=approved \
    bash "$0" >/dev/null
  status=$?
  set -e
  [[ "$status" == 0 ]] || { echo "cost self-test rejected an explicitly approved over-threshold result" >&2; exit 1; }
  echo "Terraform cost self-test passed."
  exit 0
fi

plan_json="${PLAN_JSON:-}"
infracost_bin="${INFRACOST_BIN:-infracost}"
active_month_cost="${ACTIVE_MONTH_COST:-}"
active_month_limit="${ACTIVE_MONTH_COST_LIMIT:-50}"
[[ -n "${plan_json}" && -f "${plan_json}" ]] || { echo "PLAN_JSON must point to terraform show -json output" >&2; exit 1; }
[[ "${active_month_cost}" =~ ^[0-9]+([.][0-9]+)?$ ]] || { echo "ACTIVE_MONTH_COST must be the current active-month cost as a non-negative number" >&2; exit 1; }
[[ "${active_month_limit}" =~ ^[0-9]+([.][0-9]+)?$ ]] || { echo "ACTIVE_MONTH_COST_LIMIT must be a non-negative number" >&2; exit 1; }
command -v "${infracost_bin}" >/dev/null 2>&1 || { echo "Infracost is required; run terraform-tools-check first" >&2; exit 1; }

output_dir="$(mktemp -d "${TMPDIR:-/tmp}/tally-infracost.XXXXXX")"
trap 'rm -rf "${output_dir}"' EXIT
output_file="${output_dir}/breakdown.json"
"${infracost_bin}" breakdown --path "${plan_json}" --format json --out-file "${output_file}"

node "${root}/scripts/verify/terraform-cost-policy.js" "${output_file}" "${active_month_cost}" "${active_month_limit}"
