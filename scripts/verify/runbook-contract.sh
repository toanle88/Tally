#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
template="${RUNBOOK_TEMPLATE_FILE:-${root}/docs/operations/runbook-template-v1.md}"
runbook_directory="${RUNBOOK_DIRECTORY:-${root}/docs/operations/runbooks}"
alert_contract="${ALERT_CONTRACT_FILE:-${root}/docs/operations/alert-contract-v1.md}"

runbook_files=(
  "${runbook_directory}/run-001-failed-migration.md"
  "${runbook_directory}/run-002-outbox-backlog-poison-item.md"
  "${runbook_directory}/run-003-database-restore.md"
  "${runbook_directory}/telemetry-monitoring-failure.md"
  "${runbook_directory}/run-010-capacity-saturation.md"
)

fail() {
  echo "runbook contract verification failed: $*" >&2
  return 1
}

require_text() {
  local file="$1"
  local expected="$2"

  if ! rg -Fq -- "$expected" "$file"; then
    fail "missing required text in ${file}: ${expected}"
    return 1
  fi
}

validate_template() {
  local file="$1"

  [[ -f "$file" ]] || { fail "template file does not exist: ${file}"; return 1; }

  for heading in \
    '## Runbook metadata' \
    '## Prerequisites' \
    '## Detection' \
    '## Decision points' \
    '## Safe commands' \
    '## Evidence to preserve' \
    '## Escalation' \
    '## Recovery checks' \
    '## Reconciliation requirements' \
    '## Closure criteria' \
    '## Deferred qualification'; do
    require_text "$file" "$heading" || return 1
  done

  for safety_rule in \
    'Direct destructive financial edits are prohibited' \
    'Unreviewed portal changes are prohibited' \
    'Secret disclosure is prohibited' \
    'Telemetry is diagnostic and is not authoritative financial'; do
    require_text "$file" "$safety_rule" || return 1
  done
}

validate_no_unsafe_content() {
  local file="$1"

  # Code blocks must not provide destructive SQL or an imperative unsafe
  # operator instruction. Safety warnings in prose are required and allowed.
  if awk '/^```/{in_code = !in_code; next} in_code {print}' "$file" |
    rg -n -i '\b(delete[[:space:]]+from|truncate([[:space:]]|$)|drop[[:space:]]+(table|schema)|update[[:space:]].*set|alter[[:space:]]+table)\b'; then
    fail "unsafe database mutation appears in a command block: ${file}"
    return 1
  fi

  if rg -n -i '^[[:space:]]*[-*][[:space:]]*(run|use|perform|execute|apply|edit|delete|update|drop|truncate|print|disclose|treat)[^\n]*(unreviewed portal|direct destructive|secret|credential|token|raw payload|telemetry as authoritative)' "$file"; then
    fail "unsafe imperative instruction appears in: ${file}"
    return 1
  fi
}

validate_runbook() {
  local file="$1"

  [[ -f "$file" ]] || { fail "runbook file does not exist: ${file}"; return 1; }

  for heading in \
    '## Runbook metadata' \
    '## Prerequisites' \
    '## Detection' \
    '## Decision points' \
    '## Safe commands' \
    '## Evidence to preserve' \
    '## Escalation' \
    '## Recovery checks' \
    '## Reconciliation requirements' \
    '## Closure criteria' \
    '## Deferred qualification' \
    '## Safety rules'; do
    require_text "$file" "$heading" || return 1
  done

  require_text "$file" '| Owner | Platform Operations' || return 1
  require_text "$file" '| Version | `v1` |' || return 1
  require_text "$file" '| Effective date |' || return 1
  require_text "$file" '| Status | M0 local operational baseline |' || return 1
  require_text "$file" 'Operational diagnostic data' || return 1
  require_text "$file" 'production' || return 1
  require_text "$file" 'deferred' || return 1
  require_text "$file" 'make ' || return 1
  require_text "$file" 'Direct destructive financial edits are prohibited' || return 1
  require_text "$file" 'Unreviewed portal changes' || return 1
  require_text "$file" 'are prohibited' || return 1
  require_text "$file" 'Secret disclosure is prohibited' || return 1
  require_text "$file" 'Telemetry is diagnostic and' || return 1

  validate_no_unsafe_content "$file" || return 1
}

validate_identifiers() {
  local path
  local identifier
  local identifiers

  identifiers="$(rg -o 'RUN-[0-9]{3}' "$template" "${runbook_files[@]}" "$alert_contract" 2>/dev/null | sed 's/.*RUN-/RUN-/' | sort -u || true)"
  while IFS= read -r identifier; do
    [[ -n "$identifier" ]] || continue
    case "$identifier" in
      RUN-001|RUN-002|RUN-003|RUN-004|RUN-005|RUN-006|RUN-007|RUN-008|RUN-009|RUN-010)
        ;;
      *)
        fail "unsupported runbook identifier: ${identifier}"
        return 1
        ;;
    esac
  done <<< "$identifiers"

  for path in "${runbook_files[@]}" "$alert_contract"; do
    if rg -n 'RUN-011|RUN-0(1[1-9]|[2-9][0-9])' "$path"; then
      fail "unsupported RUN-* identifier appears in: ${path}"
      return 1
    fi
  done
}

validate_alert_links() {
  [[ -f "$alert_contract" ]] || { fail "alert contract does not exist: ${alert_contract}"; return 1; }

  require_text "$alert_contract" './runbooks/run-001-failed-migration.md' || return 1
  require_text "$alert_contract" './runbooks/run-002-outbox-backlog-poison-item.md' || return 1
  require_text "$alert_contract" './runbooks/run-010-capacity-saturation.md' || return 1
}

validate_contract() {
  local template_file="$1"
  local runbook_dir="$2"
  local alert_file="$3"

  template="$template_file"
  runbook_directory="$runbook_dir"
  alert_contract="$alert_file"
  runbook_files=(
    "${runbook_directory}/run-001-failed-migration.md"
    "${runbook_directory}/run-002-outbox-backlog-poison-item.md"
    "${runbook_directory}/run-003-database-restore.md"
    "${runbook_directory}/telemetry-monitoring-failure.md"
    "${runbook_directory}/run-010-capacity-saturation.md"
  )

  validate_template "$template" || return 1
  for path in "${runbook_files[@]}"; do
    validate_runbook "$path" || return 1
  done
  validate_identifiers || return 1
  validate_alert_links || return 1
}

expect_rejected() {
  local label="$1"
  local temporary_template="$2"
  local temporary_directory="$3"
  local temporary_alert="$4"

  if validate_contract "$temporary_template" "$temporary_directory" "$temporary_alert" >/dev/null 2>&1; then
    fail "negative case was accepted: ${label}"
    return 1
  fi
}

validate_contract "$template" "$runbook_directory" "$alert_contract"

temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT

cp "$template" "${temporary_directory}/template.md"
cp "$alert_contract" "${temporary_directory}/alert.md"
cp -R "$runbook_directory" "${temporary_directory}/runbooks"

sed -i '0,/| Owner | Platform Operations |/s//| Owner | Missing owner |/' \
  "${temporary_directory}/runbooks/run-001-failed-migration.md"
expect_rejected 'missing owner' "${temporary_directory}/template.md" \
  "${temporary_directory}/runbooks" "${temporary_directory}/alert.md"

cp -R "$runbook_directory" "${temporary_directory}/missing-section-runbooks"
sed -i '0,/^## Escalation$/s//## Escalation removed/' \
  "${temporary_directory}/missing-section-runbooks/run-002-outbox-backlog-poison-item.md"
expect_rejected 'missing escalation section' "${temporary_directory}/template.md" \
  "${temporary_directory}/missing-section-runbooks" "${temporary_directory}/alert.md"

cp -R "$runbook_directory" "${temporary_directory}/unsafe-portal-runbooks"
printf '%s\n' '- Run unreviewed portal changes to bypass review.' >> \
  "${temporary_directory}/unsafe-portal-runbooks/run-003-database-restore.md"
expect_rejected 'unsafe portal instruction' "${temporary_directory}/template.md" \
  "${temporary_directory}/unsafe-portal-runbooks" "${temporary_directory}/alert.md"

cp -R "$runbook_directory" "${temporary_directory}/secret-runbooks"
printf '%s\n' '- Print credentials and tokens in the incident record.' >> \
  "${temporary_directory}/secret-runbooks/run-010-capacity-saturation.md"
expect_rejected 'secret disclosure instruction' "${temporary_directory}/template.md" \
  "${temporary_directory}/secret-runbooks" "${temporary_directory}/alert.md"

cp -R "$runbook_directory" "${temporary_directory}/authority-runbooks"
printf '%s\n' '- Treat telemetry as authoritative financial evidence.' >> \
  "${temporary_directory}/authority-runbooks/telemetry-monitoring-failure.md"
expect_rejected 'telemetry treated as authoritative' "${temporary_directory}/template.md" \
  "${temporary_directory}/authority-runbooks" "${temporary_directory}/alert.md"

cp "$alert_contract" "${temporary_directory}/invalid-link-alert.md"
sed -i 's#./runbooks/run-001-failed-migration.md#./runbooks/missing.md#' \
  "${temporary_directory}/invalid-link-alert.md"
expect_rejected 'invalid alert runbook link' "${temporary_directory}/template.md" \
  "$runbook_directory" "${temporary_directory}/invalid-link-alert.md"

cp -R "$runbook_directory" "${temporary_directory}/unsupported-id-runbooks"
printf '%s\n' '- RUN-011 — unsupported scenario.' >> \
  "${temporary_directory}/unsupported-id-runbooks/telemetry-monitoring-failure.md"
expect_rejected 'unsupported runbook identifier' "${temporary_directory}/template.md" \
  "${temporary_directory}/unsupported-id-runbooks" "${temporary_directory}/alert.md"

echo "runbook contract verification passed"
