# DLV-PLAT-005 User Story 5 Verification

This record verifies the shared primitive serialization and boundary behavior.
It completes `DLV-PLAT-005` without claiming any finance capability,
idempotency, outbox, worker, or integration workflow.

## Implementation and boundaries

- Packages: `internal/platform/money`, `internal/platform/accountingscope`,
  `internal/platform/identity`, and `internal/platform/aggregateversion`
- Money: exact decimal backed by `shopspring/decimal`; canonical fixed-scale
  amount text; currency remains a separate value
- Accounting scope: tenant, legal entity, ledger, accounting book, and
  functional currency, serialized as one exact JSON object
- Identity: typed UUID values; application-created aggregate IDs use UUID v7;
  JSON uses canonical lowercase UUID text
- Aggregate version: typed Go `int64`, valid range `1..math.MaxInt64`,
  PostgreSQL `bigint` conversion through `FromInt64` and `Value`, JSON integer
  representation
- API boundary: exact-decimal money is a JSON string, currency is an uppercase
  three-letter string, and aggregate versions are JSON integers
- Ownership: shared packages do not import finance bounded-context adapters,
  schemas, persistence packages, or generated API packages

## Reproducible commands

```bash
GOCACHE=/tmp/tally-go-cache-story5 make shared-primitives-check
GOCACHE=/tmp/tally-go-cache-story5 go test ./...
GOCACHE=/tmp/tally-go-cache-story5 go vet ./...
make api-check
git diff --check
```

## Evidence

- Primitive tests cover valid, invalid, negative, zero, precision, overflow,
  stale-version, maximum-boundary, JSON, and explicit-conversion cases.
- Money tests and the source audit provide evidence that binary floating-point
  types are not used in the implementation or serialized amount output.
- Accounting-scope JSON includes every required component and rejects malformed,
  unknown, and trailing input.
- Identity JSON round-trips canonically and compile-time tests reject cross-type
  assignments.
- Aggregate versions remain integer JSON values and round-trip through explicit
  `int64` conversion at the initial, interior, and maximum values.
- The focused gate rejects direct ownership violations and both staged and
  unstaged OpenAPI/generated-artifact changes.
- `make api-check` remains the broader OpenAPI contract and generated-artifact
  validation gate.
