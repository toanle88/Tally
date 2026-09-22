#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
contract="${DASHBOARD_CONTRACT_FILE:-${root}/docs/operations/dashboard-contract-v1.md}"

fail() {
  echo "dashboard contract verification failed: $*" >&2
  return 1
}

require_text() {
  local file="$1"
  local text="$2"

  if ! rg -Fq -- "$text" "$file"; then
    fail "missing required text: ${text}"
    return 1
  fi
}

validate_contract() {
  local file="$1"
  local panel_rows
  local panel_count
  local error_panel

  if [[ ! -f "$file" ]]; then
    fail "contract file does not exist: ${file}"
    return 1
  fi

  require_text "$file" '| Contract ID | `DLV-OPS-002-US1` |' || return 1
  require_text "$file" '| Version | `dashboard.v1` |' || return 1
  require_text "$file" '| Owner | Platform Operations |' || return 1
  require_text "$file" 'Class A: no older than 2 minutes; other classes: no older than 5 minutes' || return 1
  require_text "$file" 'M0 baseline: no older than 5 minutes; Class A: no older than 2 minutes.' || return 1
  require_text "$file" 'authorized operations users' || return 1
  require_text "$file" 'diagnostic and must not be used as authoritative financial' || return 1
  require_text "$file" 'Missing or stale telemetry is an operational condition' || return 1
  require_text "$file" 'never as zero, healthy, successful, or' || return 1
  require_text "$file" 'resolved.' || return 1
  require_text "$file" 'Provider-specific production thresholds' || return 1

  for metric in \
    'finance_http_request_duration_seconds' \
    'finance_db_transaction_duration_seconds' \
    'finance_outbox_pending_total' \
    'finance_outbox_oldest_age_seconds' \
    'finance_inbox_failure_total' \
    'result' \
    'error_code' \
    'failure_class' \
    'status_class'; do
    require_text "$file" "$metric" || return 1
  done

  for panel in \
    'API availability and latency' \
    'PostgreSQL health' \
    'Outbox/inbox pending work and age' \
    'Error classes' \
    'Capacity' \
    'Operational exceptions'; do
    require_text "$file" "**${panel}**" || return 1
  done

  for category in \
    'domain_rejection' \
    'authorization_denial' \
    'concurrency_conflict' \
    'dependency_unavailability' \
    'capacity_control' \
    'internal_failure'; do
    require_text "$file" "$category" || return 1
  done

  panel_rows="$(rg '^\| \*\*[^*]+\*\* \|' "$file" || true)"
  panel_count="$(printf '%s\n' "$panel_rows" | sed '/^$/d' | wc -l | tr -d ' ')"
  if [[ "$panel_count" != "6" ]]; then
    fail "expected 6 panel rows, found ${panel_count}"
    return 1
  fi

  while IFS= read -r panel_row; do
    [[ -n "$panel_row" ]] || continue

    if [[ "$(awk -F '|' '{print NF - 1}' <<<"$panel_row")" != "8" ]]; then
      fail "panel row does not contain the seven contract columns: ${panel_row}"
      return 1
    fi
    if [[ "$panel_row" != *'Platform Operations'* ]]; then
      fail "panel row has no owner: ${panel_row}"
      return 1
    fi
    if [[ "$panel_row" != *'unknown'* || "$panel_row" != *'stale'* || "$panel_row" != *'degraded'* ]]; then
      fail "panel row has unsafe missing/stale behavior: ${panel_row}"
      return 1
    fi
    if [[ "$panel_row" =~ (aggregate_id|customer_id|message_id|SQL[[:space:]]+text|secret|credential|token) ]]; then
      fail "panel row contains a prohibited identifier or payload reference: ${panel_row}"
      return 1
    fi
    if [[ "$panel_row" =~ (P1|P2|P3|P4|production[[:space:]]+threshold) ]]; then
      fail "panel row contains alert or unsupported production threshold data: ${panel_row}"
      return 1
    fi
  done <<< "$panel_rows"

  error_panel="$(rg '^\| \*\*Error classes\*\* \|' "$file")"
  [[ "$error_panel" == *'domain rejection'* && \
     "$error_panel" == *'authorization denial'* && \
     "$error_panel" == *'concurrency conflict'* && \
     "$error_panel" == *'dependency unavailability'* && \
     "$error_panel" == *'capacity control'* && \
     "$error_panel" == *'internal failure'* ]] || \
    { fail 'error classes are not kept distinct in the panel contract'; return 1; }
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

sed 's/Platform Operations/Unowned Operations/g' "$contract" >"${temporary_dir}/missing-owner.md"
expect_rejected 'missing panel owner' "${temporary_dir}/missing-owner.md"

sed '0,/`route`, `method`, `status_class`/s//`aggregate_id`/' "$contract" >"${temporary_dir}/unbounded-dimension.md"
expect_rejected 'unbounded dimension' "${temporary_dir}/unbounded-dimension.md"

sed 's/domain rejection, authorization denial, concurrency conflict, dependency unavailability, capacity control, and internal failure/generic error rate/' "$contract" >"${temporary_dir}/collapsed-errors.md"
expect_rejected 'collapsed error classes' "${temporary_dir}/collapsed-errors.md"

sed '0,/`unknown`, `stale`, or `degraded`/s//zero/' "$contract" >"${temporary_dir}/unsafe-missing-data.md"
expect_rejected 'unsafe missing-data behavior' "${temporary_dir}/unsafe-missing-data.md"

echo "dashboard contract verification passed"
