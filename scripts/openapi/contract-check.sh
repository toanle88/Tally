#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
runner="$root/scripts/openapi/redocly-run.sh"
bundle="$root/contracts/openapi/dist/openapi.bundle.yaml"
fixtures=(
  broken-reference.yaml
  duplicate-operation-id.yaml
  invalid-schema.yaml
  malformed-example.yaml
)
declare -A expected_errors=(
  [broken-reference.yaml]='unresolved|reference|resolve'
  [duplicate-operation-id.yaml]='operation.?id|duplicat'
  [invalid-schema.yaml]='schema|type'
  [malformed-example.yaml]='example|pattern'
)

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

bash "$runner" lint "$root/contracts/openapi/openapi.yaml"
rm -f "$bundle"
mkdir -p "$(dirname "$bundle")"

bash "$runner" bundle "$root/contracts/openapi/openapi.yaml" --output "$bundle"
first_hash="$(sha256sum "$bundle" | cut -d' ' -f1)"
bash "$runner" bundle "$root/contracts/openapi/openapi.yaml" --output "$bundle"
second_hash="$(sha256sum "$bundle" | cut -d' ' -f1)"
if [[ "$first_hash" != "$second_hash" ]]; then
  echo "OpenAPI bundle output is not deterministic" >&2
  exit 1
fi

for fixture in "${fixtures[@]}"; do
  output="$work_dir/$fixture.out"
  if bash "$runner" lint "$root/contracts/openapi/fixtures/invalid/$fixture" >"$output" 2>&1; then
    echo "Invalid OpenAPI fixture unexpectedly passed: $fixture" >&2
    exit 1
  fi
  if ! grep -Eiq "${expected_errors[$fixture]}" "$output"; then
    echo "Invalid OpenAPI fixture failed for an unexpected reason: $fixture" >&2
    exit 1
  fi
  if grep -Eiq 'password|secret|token|api[_-]?key|credential' "$output"; then
    echo "OpenAPI validation output contains secret-like text: $fixture" >&2
    exit 1
  fi
done

echo "OpenAPI contract validated; deterministic bundle sha256=$first_hash"
echo "Rejected ${#fixtures[@]} invalid fixtures without secret-like output"
