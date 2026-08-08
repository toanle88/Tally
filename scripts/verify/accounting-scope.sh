#!/usr/bin/env bash

set -euo pipefail

go test ./internal/platform/money ./internal/platform/accountingscope

if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|fixedassets|fiscalperiod|gl|identity|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' \
  internal/platform/money internal/platform/accountingscope --glob '*.go'; then
  echo "shared primitives must not import finance bounded-context packages" >&2
  exit 1
fi

if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "accounting-scope verification changed OpenAPI source or generated artifacts" >&2
  exit 1
fi

if ! git diff --cached --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "accounting-scope verification staged OpenAPI source or generated artifacts" >&2
  exit 1
fi

echo "accounting-scope primitive verification passed"
