# DLV-PLAT-005 User Story 4 Verification

This record verifies User Story 4 only. It does not claim completion of User
Story 5 or any finance capability.

## Implementation

- Package: `internal/platform/aggregateversion`
- Type: `AggregateVersion`, valid range `1..math.MaxInt64`
- Initial version: `1`
- Domain representation: typed Go `int64`
- Persistence boundary: PostgreSQL `bigint`, converted with `FromInt64` and `Value`
- API boundary: JSON integer, with no OpenAPI schema change
- Verification command: `make aggregate-version-check`

## Reproducible commands

```bash
GOCACHE=/tmp/tally-go-cache make aggregate-version-check
GOCACHE=/tmp/tally-go-cache go test ./...
GOCACHE=/tmp/tally-go-cache go vet ./...
git diff --check
```

## Evidence

- New aggregates begin at version `1`.
- Zero and negative versions are rejected.
- Advancement is monotonic and maximum-version overflow is rejected.
- Matching and stale expected versions are distinguished.
- JSON round-trips preserve versions without string conversion.
- `FromInt64(v.Value())` round-trips initial, interior, and maximum values.
- The package does not import finance bounded-context packages.
- OpenAPI source and generated artifacts remain unchanged.
