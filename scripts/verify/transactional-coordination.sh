#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

export GOCACHE="${GOCACHE:-/tmp/tally-go-cache-dlv-plat-007-us3}"

echo "== Transactional coordination unit tests =="
go test ./internal/platform/integration

echo "== Transactional coordination race tests =="
go test -race ./internal/platform/integration

echo "== Transactional coordination vet =="
go vet ./internal/platform/integration

echo "== Platform package ownership audit =="
if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|database|fixedassets|fiscalperiod|gl|httpapi|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' internal/platform/integration --glob '*.go'; then
  echo "integration must not import finance bounded-context packages" >&2
  exit 1
fi

echo "== SQLC source and generated-output checks =="
make sqlc-compile
make sqlc-check

echo "== Transactional coordination PostgreSQL integration tests =="
go test -tags=integration ./internal/platform/database \
	-run '^TestTransactionalCoordination' \
	-count=1 -timeout=20m

echo "== OpenAPI preservation audit =="
if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "transactional coordination changed OpenAPI source or generated artifacts" >&2
  exit 1
fi
if ! git diff --cached --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "transactional coordination has staged OpenAPI source or generated artifacts" >&2
  exit 1
fi

git diff --check
echo "transactional coordination verification passed"
