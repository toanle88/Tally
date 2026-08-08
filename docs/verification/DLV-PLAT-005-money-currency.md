# DLV-PLAT-005 User Story 1 Verification

This record verifies User Story 1 only. It does not claim completion of the
remaining shared primitive stories or any finance capability.

## Implementation

- Package: `internal/platform/money`
- Decimal implementation: `github.com/shopspring/decimal`
- Currency metadata: caller-provided immutable registry entries
- Domain bound: 38 total digits, 12 fractional digits, 26 integer digits
- Serialization: canonical fixed-scale amount text; currency remains separate
- Verification command: `make money-check`

## Reproducible commands

```bash
GOCACHE=/tmp/tally-go-cache make money-check
GOCACHE=/tmp/tally-go-cache go test ./...
```

## Evidence

- Currency metadata validates canonical three-letter uppercase codes and scale.
- Money rejects malformed values, excess currency precision, and numeric-bound overflow.
- Addition, subtraction, negation, equality, currency mismatch, and defensive
  decimal access are covered by unit tests.
- The focused verification checks that the package has no bounded-context
  imports and that OpenAPI source and generated artifacts remain unchanged.
- The database currency-code documentation uses the same three-letter regex as
  the OpenAPI contract.
- Verification was run with an isolated Go build cache because the host-shared
  cache is read-only in this environment.
