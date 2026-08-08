#!/usr/bin/env bash

set -euo pipefail

go test ./internal/platform/money

if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|fixedassets|fiscalperiod|gl|identity|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' internal/platform/money --glob '*.go'; then
  echo "money package must not import finance bounded-context packages" >&2
  exit 1
fi

if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "money verification changed OpenAPI source or generated artifacts" >&2
  exit 1
fi

echo "money primitive verification passed"
