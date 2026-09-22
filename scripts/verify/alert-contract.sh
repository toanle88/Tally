#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
contract="${ALERT_CONTRACT_FILE:-${root}/docs/operations/alert-contract-v1.md}"

fail() {
  echo "alert contract verification failed: $*" >&2
  return 1
}

require_text() {
  local file="$1"
  local expected="$2"

  if ! rg -Fq -- "$expected" "$file"; then
    fail "missing required text: ${expected}"
    return 1
  fi
}

validate_contract() {
  local file="$1"
  local alert_rows
  local alert_count
  local signal

  if [[ ! -f "$file" ]]; then
    fail "contract file does not exist: ${file}"
    return 1
  fi

  require_text "$file" '| Contract ID | `DLV-OPS-002-US2` |' || return 1
  require_text "$file" '| Version | `alert.v1` |' || return 1
  require_text "$file" '| Owner | Platform Operations |' || return 1
  require_text "$file" 'Provider-neutral contract' || return 1
  require_text "$file" 'not authoritative financial or audit evidence' || return 1
  require_text "$file" 'NFR-OBS-002' || return 1
  require_text "$file" 'NFR-OBS-007' || return 1
  require_text "$file" 'NFR-OBS-012' || return 1
  require_text "$file" 'unknown' || return 1
  require_text "$file" 'stale' || return 1
  require_text "$file" 'never becomes zero, healthy' || return 1
  require_text "$file" 'unsupported production trigger threshold' || return 1

  for severity in P1 P2 P3 P4; do
    require_text "$file" "**${severity}**" || return 1
  done

  for condition in \
    'Integrity uncertainty' \
    'Critical control failure' \
    'Outbox/backlog age' \
    'Database saturation' \
    'Dependency failure' \
    'Capacity risk'; do
    require_text "$file" "| **${condition}** |" || return 1
  done

  alert_rows="$(rg '^\| \*\*(Integrity uncertainty|Critical control failure|Outbox/backlog age|Database saturation|Dependency failure|Capacity risk)\*\* \|' "$file" || true)"
  alert_count="$(printf '%s\n' "$alert_rows" | sed '/^$/d' | wc -l | tr -d ' ')"
  if [[ "$alert_count" != "6" ]]; then
    fail "expected 6 alert definition rows, found ${alert_count}"
    return 1
  fi

  while IFS= read -r alert_row; do
    [[ -n "$alert_row" ]] || continue

    if [[ "$(awk -F '|' '{print NF - 1}' <<<"$alert_row")" != "10" ]]; then
      fail "alert row does not contain the ten contract delimiters: ${alert_row}"
      return 1
    fi
    if [[ "$alert_row" != *'Platform Operations'* ]]; then
      fail "alert row has no Platform Operations owner: ${alert_row}"
      return 1
    fi
    if [[ ! "$alert_row" =~ \|[[:space:]]P[1-4][[:space:]]\| ]]; then
      fail "alert row has no P1-P4 severity: ${alert_row}"
      return 1
    fi
    if [[ "$alert_row" != *'dashboard-contract-v1.md'* ]]; then
      fail "alert row has no dashboard contract link: ${alert_row}"
      return 1
    fi
    if [[ ! "$alert_row" =~ RUN-0(0[1-9]|10) ]]; then
      fail "alert row has no approved RUN-001..RUN-010 reference: ${alert_row}"
      return 1
    fi
    if [[ "$alert_row" != *'Maintenance'* && "$alert_row" != *'maintenance'* ]]; then
      fail "alert row has no maintenance/suppression behavior: ${alert_row}"
      return 1
    fi
    for suppression_field in owner reason expiry; do
      if [[ "$alert_row" != *"$suppression_field"* ]]; then
        fail "alert row has no suppression ${suppression_field}: ${alert_row}"
        return 1
      fi
    done
    if [[ "$alert_row" != *'Escalat'* && "$alert_row" != *'escalat'* ]]; then
      fail "alert row has no escalation path: ${alert_row}"
      return 1
    fi
    if [[ "$alert_row" != *'Completion evidence:'* ]]; then
      fail "alert row has no completion evidence: ${alert_row}"
      return 1
    fi
    if [[ "$alert_row" != *'unknown'* && "$alert_row" != *'stale'* ]]; then
      fail "alert row has no safe missing-data behavior: ${alert_row}"
      return 1
    fi
    if [[ "$alert_row" =~ (aggregate_id|customer_id|message_id|SQL[[:space:]]+text|secret|credential|token|raw[[:space:]]+payload|bank[[:space:]]+details|payroll[[:space:]]+values|tax[[:space:]]+identifiers) ]]; then
      fail "alert row contains a prohibited identifier or payload reference: ${alert_row}"
      return 1
    fi
    signal="$(awk -F '|' '{print $4}' <<<"$alert_row")"
    if [[ "$signal" =~ (>=|<=|>|<|=)[[:space:]]*[0-9] || "$signal" =~ [0-9]+% || "$signal" =~ [0-9]+[[:space:]]+percent || "$signal" =~ threshold[[:space:]]+[0-9] ]]; then
      fail "alert row asserts an unsupported numeric production threshold: ${alert_row}"
      return 1
    fi
  done <<< "$alert_rows"
}

expect_rejected() {
  local label="$1"
  local file="$2"

  if validate_contract "$file" >/dev/null 2>&1; then
    fail "negative case was accepted: ${label}"
  fi
}

validate_contract "$contract"

temporary_dir="$(mktemp -d)"
trap 'rm -rf "$temporary_dir"' EXIT

sed '/^| \*\*Integrity uncertainty\*\* |/s/Platform Operations/Unowned Operations/' "$contract" >"${temporary_dir}/missing-owner.md"
expect_rejected 'missing owner' "${temporary_dir}/missing-owner.md"

sed '0,/| \*\*Integrity uncertainty\*\* | P1 |/s//| **Integrity uncertainty** | UNDEFINED |/' "$contract" >"${temporary_dir}/missing-severity.md"
expect_rejected 'missing severity' "${temporary_dir}/missing-severity.md"

sed '0,/RUN-006/s//RUN-UNKNOWN/' "$contract" >"${temporary_dir}/missing-runbook.md"
expect_rejected 'missing runbook' "${temporary_dir}/missing-runbook.md"

sed '0,/dashboard-contract-v1.md/s//dashboard-contract-missing.md/' "$contract" >"${temporary_dir}/missing-dashboard.md"
expect_rejected 'missing dashboard link' "${temporary_dir}/missing-dashboard.md"

sed '0,/maintenance window with owner, reason, start\/end, and expiry/s//maintenance window metadata removed/' "$contract" >"${temporary_dir}/missing-suppression.md"
expect_rejected 'missing suppression metadata' "${temporary_dir}/missing-suppression.md"

sed '0,/Escalate to/s//Notify/' "$contract" >"${temporary_dir}/missing-escalation.md"
expect_rejected 'missing escalation path' "${temporary_dir}/missing-escalation.md"

sed '0,/Completion evidence:/s//Completion record:/' "$contract" >"${temporary_dir}/missing-completion.md"
expect_rejected 'missing completion evidence' "${temporary_dir}/missing-completion.md"

sed '0,/active, unknown, or stale/s//active, healthy, or resolved/' "$contract" >"${temporary_dir}/unsafe-missing-data.md"
expect_rejected 'unsafe missing-data behavior' "${temporary_dir}/unsafe-missing-data.md"

sed '0,/typed integrity-uncertain state/s//customer_id/' "$contract" >"${temporary_dir}/unbounded-identifier.md"
expect_rejected 'unbounded identifier' "${temporary_dir}/unbounded-identifier.md"

sed '0,/configured workflow, business-calendar, or NFR target/s//80 percent production threshold/' "$contract" >"${temporary_dir}/unsupported-threshold.md"
expect_rejected 'unsupported production threshold' "${temporary_dir}/unsupported-threshold.md"

echo "alert contract verification passed"
