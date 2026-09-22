#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
dashboard_contract="${DASHBOARD_CONTRACT_FILE:-${root}/docs/operations/dashboard-contract-v1.md}"
alert_contract="${ALERT_CONTRACT_FILE:-${root}/docs/operations/alert-contract-v1.md}"
runbook_template="${RUNBOOK_TEMPLATE_FILE:-${root}/docs/operations/runbook-template-v1.md}"
runbook_directory="${RUNBOOK_DIRECTORY:-${root}/docs/operations/runbooks}"

dashboard_validator="${root}/scripts/verify/dashboard-contract.sh"
alert_validator="${root}/scripts/verify/alert-contract.sh"
runbook_validator="${root}/scripts/verify/runbook-contract.sh"

runbook_files=(
  "${runbook_directory}/run-001-failed-migration.md"
  "${runbook_directory}/run-002-outbox-backlog-poison-item.md"
  "${runbook_directory}/run-003-database-restore.md"
  "${runbook_directory}/telemetry-monitoring-failure.md"
  "${runbook_directory}/run-010-capacity-saturation.md"
)

temporary_directory="$(mktemp -d)"
trap 'rm -rf -- "${temporary_directory}"' EXIT

overall_status=0

pass() {
  printf '[pass] %s\n' "$1"
}

fail() {
  printf '[fail] %s\n' "$1" >&2
}

require_file() {
  [[ -f "$1" ]]
}

require_text() {
  local file="$1"
  local expected="$2"

  rg -Fq -- "$expected" "$file"
}

execute_check() {
  local check_id="$1"
  local output_file
  shift

  output_file="${temporary_directory}/${check_id//[^[:alnum:]_.-]/_}.log"
  "$@" >"${output_file}" 2>&1
}

run_required_check() {
  local check_id="$1"
  shift

  if execute_check "$check_id" "$@"; then
    pass "$check_id"
    return 0
  fi

  fail "$check_id (failure localized to this check; command output suppressed)"
  return 1
}

expect_rejected() {
  local check_id="$1"
  shift

  if execute_check "$check_id" "$@"; then
    fail "$check_id (invalid synthetic fixture was accepted)"
    return 1
  fi

  pass "$check_id (invalid synthetic fixture rejected)"
  return 0
}

validate_cross_contract_references() {
  local runbook

  require_file "${dashboard_contract}" || return 1
  require_file "${alert_contract}" || return 1
  require_file "${runbook_template}" || return 1

  require_text "${alert_contract}" './dashboard-contract-v1.md' || return 1
  require_text "${alert_contract}" './runbooks/run-001-failed-migration.md' || return 1
  require_text "${alert_contract}" './runbooks/run-002-outbox-backlog-poison-item.md' || return 1
  require_text "${alert_contract}" './runbooks/run-010-capacity-saturation.md' || return 1

  for runbook in "${runbook_files[@]}"; do
    require_file "${runbook}" || return 1
    require_text "${runbook}" 'dashboard-contract-v1.md' || return 1
  done
}

validate_missing_data_policy() {
  local path

  require_text "${dashboard_contract}" 'Missing or stale telemetry is an operational condition' || return 1
  require_text "${dashboard_contract}" 'never as zero, healthy, successful, or' || return 1
  require_text "${alert_contract}" 'Missing or stale telemetry is an operational condition' || return 1
  require_text "${alert_contract}" '`unknown`, `stale`, or `degraded`' || return 1
  require_text "${runbook_template}" 'or stale telemetry is an operational condition' || return 1
  require_text "${runbook_template}" 'Distinguish unavailable, stale,' || return 1
  require_text "${runbook_template}" 'degraded, pending, failed, and integrity-uncertain states.' || return 1

  for path in "${runbook_files[@]}"; do
    if ! rg -qi 'unknown|stale|degraded|unavailable' "${path}"; then
      return 1
    fi
  done
}

copy_and_replace() {
  local source="$1"
  local destination="$2"
  shift 2

  cp "${source}" "${destination}"
  while (( "$#" > 0 )); do
    local expression="$1"
    shift
    sed -E -i "${expression}" "${destination}"
  done
}

dashboard_fixture="${temporary_directory}/dashboard-missing-data.md"
alert_fixture="${temporary_directory}/alert-missing-data.md"
runbook_template_fixture="${temporary_directory}/runbook-template-missing-data.md"
runbook_directory_fixture="${temporary_directory}/runbooks-missing-data"
failure_propagation_fixture="${temporary_directory}/dashboard-failure-propagation.md"

if ! run_required_check \
  'dashboard-contract' \
  bash "${dashboard_validator}"; then
  overall_status=1
fi

if ! run_required_check \
  'alert-contract' \
  bash "${alert_validator}"; then
  overall_status=1
fi

if ! run_required_check \
  'runbook-contract' \
  bash "${runbook_validator}"; then
  overall_status=1
fi

if ! run_required_check \
  'cross-contract-references' \
  validate_cross_contract_references; then
  overall_status=1
fi

if ! run_required_check \
  'missing-data-policy' \
  validate_missing_data_policy; then
  overall_status=1
fi

copy_and_replace \
  "${dashboard_contract}" \
  "${dashboard_fixture}" \
  's/unknown/zero/g' \
  's/stale/healthy/g' \
  's/degraded/resolved/g'

copy_and_replace \
  "${alert_contract}" \
  "${alert_fixture}" \
  's/unknown/healthy/g' \
  's/stale/resolved/g' \
  's/degraded/successful/g'

copy_and_replace \
  "${runbook_template}" \
  "${runbook_template_fixture}" \
  's/or stale telemetry is an operational condition/or telemetry is a condition/' \
  's/Distinguish unavailable, stale,/Distinguish failure states/' \
  's/degraded, pending, failed, and integrity-uncertain states\./states\./'
cp -R "${runbook_directory}" "${runbook_directory_fixture}"

if ! expect_rejected \
  'dashboard-missing-data-fixture' \
  env DASHBOARD_CONTRACT_FILE="${dashboard_fixture}" bash "${dashboard_validator}"; then
  overall_status=1
fi

if ! expect_rejected \
  'alert-missing-data-fixture' \
  env ALERT_CONTRACT_FILE="${alert_fixture}" bash "${alert_validator}"; then
  overall_status=1
fi

if ! expect_rejected \
  'runbook-missing-data-fixture' \
  RUNBOOK_TEMPLATE_FILE="${runbook_template_fixture}" \
  RUNBOOK_DIRECTORY="${runbook_directory_fixture}" \
  ALERT_CONTRACT_FILE="${alert_contract}" \
  bash "${runbook_validator}"; then
  overall_status=1
fi

copy_and_replace \
  "${dashboard_contract}" \
  "${failure_propagation_fixture}" \
  's/unknown/zero/g' \
  's/stale/healthy/g' \
  's/degraded/resolved/g'
printf '%s\n' 'synthetic-raw-payload-marker' >>"${failure_propagation_fixture}"

propagation_output="${temporary_directory}/failure-propagation.log"
set +e
(
  run_required_check \
    'failure-propagation-child' \
    env DASHBOARD_CONTRACT_FILE="${failure_propagation_fixture}" bash "${dashboard_validator}"
) >"${propagation_output}" 2>&1
propagation_status=$?
set -e

if [[ "${propagation_status}" -eq 0 ]]; then
  fail 'failure-propagation (invalid child result was not propagated)'
  overall_status=1
elif rg -Fq -- 'synthetic-raw-payload-marker' "${propagation_output}"; then
  fail 'failure-propagation (raw fixture marker escaped into failure output)'
  overall_status=1
else
  pass 'failure-propagation (invalid child result returned non-zero with sanitized output)'
fi

if [[ "${overall_status}" -ne 0 ]]; then
  printf '%s\n' 'operational readiness verification failed' >&2
  exit 1
fi

printf '%s\n' 'operational readiness verification passed'
