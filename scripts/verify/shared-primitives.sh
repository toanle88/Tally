#!/usr/bin/env bash

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"

packages=(
	./internal/platform/money
	./internal/platform/accountingscope
	./internal/platform/identity
	./internal/platform/aggregateversion
)

echo "== Shared primitive unit tests =="
go test "${packages[@]}"

echo "== Shared primitive vet =="
go vet "${packages[@]}"

echo "== Money representation audit =="
if rg -n '\bfloat(32|64)?\b' internal/platform/money --glob '*.go'; then
	echo "money primitives must not use binary floating-point types" >&2
	exit 1
fi

echo "== Shared package ownership audit =="
if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|database|fixedassets|fiscalperiod|gl|httpapi|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' \
	internal/platform/money \
	internal/platform/accountingscope \
	internal/platform/identity \
	internal/platform/aggregateversion \
	--glob '*.go'; then
	echo "shared primitives must not import finance bounded-context adapters, schemas, persistence, or generated API packages" >&2
	exit 1
fi

echo "== API boundary representation audit =="
schema_block() {
	local schema="$1"
	awk -v schema="${schema}" '
		$0 == "  " schema ":" { in_schema = 1; print; next }
		in_schema && $0 ~ /^  [A-Za-z][A-Za-z0-9]*:$/ { exit }
		in_schema { print }
	' contracts/openapi/components/common.yaml
}

money_schema="$(schema_block Money)"
currency_schema="$(schema_block CurrencyCode)"
command_schema="$(schema_block CommandRequest)"
result_schema="$(schema_block EstablishedResult)"

if [[ -z "${money_schema}" || -z "${currency_schema}" || -z "${command_schema}" || -z "${result_schema}" ]]; then
	echo "required shared primitive API schemas are missing" >&2
	exit 1
fi

grep -Fq "    type: string" <<<"${money_schema}"
grep -Fq "    pattern: '^-?[0-9]+(\\.[0-9]+)?$'" <<<"${money_schema}"
grep -Fq "    type: string" <<<"${currency_schema}"
grep -Fq "    pattern: '^[A-Z]{3}$'" <<<"${currency_schema}"
grep -Fq '      expectedVersion: { type: integer,' <<<"${command_schema}"
grep -Fq '      aggregateVersion: { type: integer,' <<<"${result_schema}"

if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated; then
	echo "shared-primitives verification changed OpenAPI source or generated artifacts" >&2
	exit 1
fi

if ! git diff --cached --quiet -- contracts/openapi internal/platform/httpapi/generated; then
	echo "shared-primitives verification has staged OpenAPI source or generated artifacts" >&2
	exit 1
fi

echo "shared primitives verification passed"
