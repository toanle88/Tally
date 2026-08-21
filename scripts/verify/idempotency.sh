#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

export GOCACHE="${GOCACHE:-/tmp/tally-go-cache-dlv-plat-006}"

echo "== Idempotency focused tests =="
go test ./internal/platform/idempotency

echo "== Idempotency race tests =="
go test -race ./internal/platform/idempotency

echo "== Idempotency vet =="
go vet ./internal/platform/idempotency

echo "== Package ownership audit =="
if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|database|fixedassets|fiscalperiod|gl|httpapi|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' internal/platform/idempotency --glob '*.go'; then
  echo "idempotency must not import finance bounded-context or infrastructure packages" >&2
  exit 1
fi

echo "== OpenAPI preservation audit =="
if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "idempotency verification changed OpenAPI source or generated artifacts" >&2
  exit 1
fi
if ! git diff --cached --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "idempotency verification has staged OpenAPI source or generated artifacts" >&2
  exit 1
fi

echo "== Diff check =="
git diff --check

echo "idempotency verification passed"
