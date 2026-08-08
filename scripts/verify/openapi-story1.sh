#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
path_directory="$root/contracts/openapi/paths"
source_file="$root/docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md"
source_count="$(grep -Ec '^\| FR-' "$source_file")"
mapfile -t path_files < <(find "$path_directory" -maxdepth 1 -type f -name '*.yaml' ! -name catalog.yaml | sort)
operation_count="$(grep -Ehc '^  (get|post|put|patch|delete):' "${path_files[@]}" | awk '{sum += $1} END {print sum + 0}')"
mapfile -t operation_ids < <(grep -Eh '^    operationId:' "${path_files[@]}" | sed 's/^[[:space:]]*operationId:[[:space:]]*//')
mapfile -t owners < <(grep -Eh '^    x-owner-capability:' "${path_files[@]}" | sed 's/^[[:space:]]*x-owner-capability:[[:space:]]*//')
state_changing_count="$(grep -Ehc '^  (post|put|patch|delete):' "${path_files[@]}" | awk '{sum += $1} END {print sum + 0}')"
required_idempotency_count="$(grep -Ehc 'Idempotency-Key-Required' "${path_files[@]}" | awk '{sum += $1} END {print sum + 0}')"

[[ "$source_count" -eq "$operation_count" ]] || { echo "Contract operation count $operation_count does not match authoritative catalog count $source_count." >&2; exit 1; }
[[ "$(printf '%s\n' "${operation_ids[@]}" | sort -u | wc -l)" -eq "${#operation_ids[@]}" ]] || { echo 'Duplicate operationId values found.' >&2; exit 1; }
[[ "${#owners[@]}" -eq "${#operation_ids[@]}" ]] || { echo 'Every operation must declare x-owner-capability.' >&2; exit 1; }
[[ "$required_idempotency_count" -eq "$state_changing_count" ]] || { echo "Every state-changing operation must reference the required idempotency header." >&2; exit 1; }
! grep -Eq "^      '201':" "${path_files[@]}" || { echo 'User Story 1 contract must not advertise unsupported blanket HTTP 201 responses.' >&2; exit 1; }

allowed='master-data general-ledger accounts-payable accounts-receivable payroll invoicing payments reporting intercompany revenue-recognition fixed-assets multi-currency fiscal-periods coa-segments bank-reconciliation tax approvals identity-access audit-integrity'
for owner in "${owners[@]}"; do
  [[ " $allowed " == *" $owner "* ]] || { echo "Unknown x-owner-capability value: $owner" >&2; exit 1; }
done

required_files=(contracts/openapi/openapi.yaml contracts/openapi/components/common.yaml contracts/openapi/paths/catalog.yaml contracts/openapi/examples/command-request.yaml contracts/openapi/examples/established-result.yaml contracts/openapi/examples/problem-details.yaml)
for file in "${required_files[@]}"; do
  [[ -f "$root/$file" ]] || { echo "Missing committed contract file: $file" >&2; exit 1; }
done
[[ "${#path_files[@]}" -eq 19 ]] || { echo "Expected 19 capability path files, found ${#path_files[@]}." >&2; exit 1; }

while IFS= read -r reference; do
  target_file="${reference%%#/*}"; pointer="${reference#*#/}"; target_path="$path_directory/$target_file"
  pointer="${pointer//~1//}"; pointer="${pointer//~0/~}"
  [[ -f "$target_path" ]] || { echo "Catalog reference target does not exist: $target_path" >&2; exit 1; }
  grep -Fq "$pointer:" "$target_path" || { echo "Catalog reference target does not resolve to $pointer in $target_path" >&2; exit 1; }
done < <(sed -n 's#^  \$ref: \./##p' "$path_directory/catalog.yaml")

! grep -Eiq '(password|secret|token|accountNumber)' "$root/contracts/openapi/examples/command-request.yaml" || { echo 'Command example contains a credential or sensitive field.' >&2; exit 1; }
owner_count="$(printf '%s\n' "${owners[@]}" | sort -u | wc -l)"
echo "OpenAPI User Story 1 structure verified: ${#operation_ids[@]} operations, $owner_count capability owners."
