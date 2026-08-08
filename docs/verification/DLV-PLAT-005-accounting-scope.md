# DLV-PLAT-005 User Story 2 Verification

This record verifies User Story 2 only. It does not claim completion of the
remaining shared primitive stories or any finance capability.

## Implementation

- Package: `internal/platform/accountingscope`
- Components: tenant ID, legal-entity ID, ledger ID, accounting-book ID, and functional-currency code
- Component IDs: `google/uuid.UUID`
- Currency validation: canonical three uppercase ASCII letters, returning `money.ErrMalformedCurrency`
- Relationship validation: deferred to Organization & Master Data and GL owning contexts
- Serialization: canonical JSON object with all five components
- Verification command: `make accounting-scope-check`

## JSON representation

```json
{
  "tenantId": "0195a91b-20ab-7c15-8aa8-4e111a8bd618",
  "legalEntityId": "0195a91b-20ab-7c15-8aa8-4e111a8bd619",
  "ledgerId": "0195a91b-20ab-7c15-8aa8-4e111a8bd620",
  "accountingBookId": "0195a91b-20ab-7c15-8aa8-4e111a8bd621",
  "functionalCurrency": "USD"
}
```

## Reproducible commands

```bash
GOCACHE=/tmp/tally-go-cache make accounting-scope-check
GOCACHE=/tmp/tally-go-cache go test ./...
GOCACHE=/tmp/tally-go-cache go vet ./...
git diff --check
```

## Evidence

- Zero UUID components are rejected deterministically.
- Malformed UUID strings are rejected during JSON deserialization.
- Currency-code validation is syntax-only and reuses `money.ErrMalformedCurrency`.
- Equality includes all five scope components.
- Different ledgers and accounting books remain distinct under the same legal entity.
- JSON serialization includes all five components and round-trips without loss.
- The package does not read ambient request or authentication context.
- Shared primitive packages do not import finance bounded-context packages.
- OpenAPI source and generated artifacts remain unchanged.
