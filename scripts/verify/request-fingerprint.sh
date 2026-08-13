#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

echo "== Request fingerprint unit tests =="
go test ./internal/platform/idempotency
echo "== Request fingerprint vet =="
go vet ./internal/platform/idempotency
echo "== Package ownership audit =="
if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|database|fixedassets|fiscalperiod|gl|httpapi|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' internal/platform/idempotency --glob '*.go'; then
  echo "idempotency must not import finance bounded-context or infrastructure packages" >&2
  exit 1
fi
if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "request fingerprint verification changed OpenAPI source or generated artifacts" >&2
  exit 1
fi
if ! git diff --cached --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "request fingerprint verification has staged OpenAPI source or generated artifacts" >&2
  exit 1
fi
echo "request fingerprint verification passed"
