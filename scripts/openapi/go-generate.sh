#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
output_directory="${1:-internal/platform/httpapi/generated}"
input_contract="${2:-contracts/openapi/openapi.yaml}"

cd "${root}"

[[ -f .ogen.yml ]] || { echo "Missing ogen configuration: .ogen.yml" >&2; exit 1; }
[[ -f "${input_contract}" ]] || { echo "Missing OpenAPI input: ${input_contract}" >&2; exit 1; }

bundle_directory="$(mktemp -d)"
trap 'rm -rf "${bundle_directory}"' EXIT
bundle_file="${bundle_directory}/openapi.bundle.yaml"

bash scripts/openapi/redocly-run.sh \
  bundle "${input_contract}" \
  --output "${bundle_file}"

go tool ogen \
  --config .ogen.yml \
  --target "${output_directory}" \
  -package generated \
  --clean \
  "${bundle_file}"
