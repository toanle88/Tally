#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

export GOCACHE="${GOCACHE:-/tmp/tally-go-cache-dlv-plat-006}"

echo "== Idempotency migration checks =="
bash scripts/db/migrate.sh check
bash scripts/db/migrate.sh validate

echo "== Idempotency SQLC checks =="
make sqlc-compile
make sqlc-check

echo "== Idempotency PostgreSQL integration tests =="
go test -tags=integration ./internal/platform/database \
	-run '^TestDurableIdempotency(ReservationAndFinalization|ConcurrentReservationsHaveOneOwner)$' \
	-count=1 -timeout=10m

echo "idempotency persistence verification passed"
