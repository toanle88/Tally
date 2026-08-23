#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

export GOCACHE="${GOCACHE:-/tmp/tally-go-cache-dlv-plat-007}"

echo "== Outbox/inbox migration checks =="
bash scripts/db/migrate.sh check
bash scripts/db/migrate.sh validate

echo "== Outbox/inbox SQLC checks =="
make sqlc-compile
make sqlc-check

echo "== Outbox/inbox PostgreSQL integration tests =="
go test -tags=integration ./internal/platform/database \
	-run '^TestOutboxInboxPersistence$' \
	-count=1 -timeout=10m

git diff --check
echo "outbox/inbox persistence verification passed"
