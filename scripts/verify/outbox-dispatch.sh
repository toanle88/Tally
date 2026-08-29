#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

export GOCACHE="${GOCACHE:-/tmp/tally-go-cache-dlv-plat-007-us4}"

echo "== Outbox dispatcher unit tests =="
go test ./internal/platform/integration

echo "== Outbox dispatcher race tests =="
go test -race ./internal/platform/integration

echo "== Outbox dispatcher vet =="
go vet ./internal/platform/integration

echo "== Platform package ownership audit =="
if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|fixedassets|fiscalperiod|gl|httpapi|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' internal/platform/integration --glob '*.go'; then
  echo "integration must not import finance bounded-context packages" >&2
  exit 1
fi

echo "== Migration and SQLC checks =="
make db-migrate-validate
make db-migrate-check
make sqlc-compile
make sqlc-check

echo "== Outbox dispatcher PostgreSQL integration tests =="
go test -tags=integration ./internal/platform/database \
  -run '^TestOutboxDispatch' \
  -count=1 -timeout=20m

echo "== OpenAPI preservation audit =="
if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "outbox dispatcher changed OpenAPI source or generated artifacts" >&2
  exit 1
fi
if ! git diff --cached --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "outbox dispatcher has staged OpenAPI source or generated artifacts" >&2
  exit 1
fi

git diff --check
echo "outbox dispatcher verification passed"
