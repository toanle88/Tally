#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

go_command="${GO_BIN:-go}"
packages=(
  ./internal/identity
  ./internal/platform/httpapi
  ./internal/platform/telemetry
)

"${go_command}" test "${packages[@]}"
"${go_command}" test -race "${packages[@]}"
"${go_command}" vet "${packages[@]}"

pnpm --dir web exec vitest run \
  src/components/record-context/record-context.test.tsx \
  src/components/operational-components.test.tsx \
  src/app/identity-access-workspace.test.tsx
pnpm --dir web exec tsc -b
pnpm --dir web exec tsc -p tsconfig.playwright.json --noEmit
pnpm --dir web exec playwright test --config=playwright.config.ts tests/a11y/semantic.spec.ts

git diff --check

echo "IAM sensitive-access evidence verification passed"
