#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

export GOCACHE="${GOCACHE:-/tmp/tally-go-cache-dlv-plat-007-us5}"

echo "== Worker and replay unit tests =="
go test ./internal/platform/worker ./internal/platform/integration ./cmd/worker

echo "== Worker and replay race tests =="
go test -race ./internal/platform/worker ./internal/platform/integration ./cmd/worker

echo "== Worker and replay vet =="
go vet ./internal/platform/worker ./internal/platform/integration ./cmd/worker

echo "== Worker build =="
go build -o /tmp/tally-worker-dlv-plat-007-us5 ./cmd/worker

echo "== Platform package ownership audit =="
if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|fixedassets|fiscalperiod|gl|httpapi|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' internal/platform/{integration,worker} cmd/worker --glob '*.go'; then
  echo "worker and integration packages must not import finance bounded-context packages" >&2
  exit 1
fi

echo "== Migration and SQLC checks =="
make db-migrate-validate
make db-migrate-check
make sqlc-compile
make sqlc-check

echo "== Worker and replay PostgreSQL integration tests =="
go test -tags=integration ./internal/platform/database \
  -run '^Test(OutboxDispatch|Replay|TransactionalCoordination|CrashRecovery)' \
  -count=1 -timeout=20m

echo "== OpenAPI preservation audit =="
if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "worker/replay implementation changed OpenAPI source or generated artifacts" >&2
  exit 1
fi
if ! git diff --cached --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "worker/replay implementation has staged OpenAPI source or generated artifacts" >&2
  exit 1
fi

git diff --check
echo "worker and replay verification passed"
