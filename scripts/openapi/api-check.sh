#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "${root}"
bash scripts/openapi/contract-check.sh
bash scripts/openapi/api-generate-check.sh
bash scripts/openapi/typescript-client-check.sh

echo 'OpenAPI contract and generated-artifact drift check passed.'
