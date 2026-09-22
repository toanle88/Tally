#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

go_command="${GO_BIN:-go}"
packages=(
  ./internal/platform/telemetry
  ./cmd/api
  ./cmd/worker
  ./internal/platform/idempotency
  ./internal/platform/integration
  ./internal/platform/worker
)

"${go_command}" test "${packages[@]}"
"${go_command}" test -race "${packages[@]}"
"${go_command}" vet "${packages[@]}"
git diff --check

echo "telemetry failure and sensitive-data verification passed"
