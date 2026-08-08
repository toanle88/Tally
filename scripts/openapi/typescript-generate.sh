#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
output_directory="${1:-${root}/web/src/generated/api}"
input_contract="${2:-}"

cd "${root}"

if [[ -z "${input_contract}" ]]; then
  bundle_directory="$(mktemp -d)"
  trap 'rm -rf "${bundle_directory}"' EXIT
  input_contract="${bundle_directory}/openapi.bundle.yaml"

  bash scripts/openapi/redocly-run.sh \
    bundle contracts/openapi/openapi.yaml \
    --output "${input_contract}"
fi

node_modules/.bin/openapi-ts \
  -f openapi-ts.config.mjs \
  -i "${input_contract}" \
  -o "${output_directory}" \
  --no-log-file
