#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"
export GOCACHE="${GOCACHE:-/tmp/tally-go-cache-dlv-ops-001-us1}"

packages=(
  ./internal/platform/telemetry
  ./internal/platform/httpx
  ./internal/platform/integration
  ./internal/platform/worker
)

go test "${packages[@]}"
go test -race "${packages[@]}"
go vet "${packages[@]}"
git diff --check

if rg -n 'Authorization:|Bearer |password|secret|token|bank|payroll|tax' internal/platform/telemetry --glob '*.go'; then
  echo "telemetry context package contains prohibited sensitive fields" >&2
  exit 1
fi

echo "telemetry context verification passed"
