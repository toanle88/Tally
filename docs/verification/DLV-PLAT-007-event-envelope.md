# DLV-PLAT-007 event envelope verification

The platform-owned envelope foundation is implemented in `internal/platform/events`.
It validates the canonical wire shape, structural identities and scope, version 1,
UTC timestamps, classifications, object payloads, deterministic canonical JSON, and
the unprefixed `sha256:<hex>` payload fingerprint.

Evidence command:

```bash
GOCACHE=/tmp/tally-go-cache-dlv-plat-007-us1 make event-envelope-check
```

The focused gate runs package tests, race tests, `go vet`, package-boundary checks,
API-artifact preservation checks, and `git diff --check`.

Non-claims: this does not implement or verify semantic payload minimization,
event-specific schemas, sensitive-field allowlists, semantic transformations,
transactional outbox/inbox persistence, workers, brokers, delivery retries, or
full replay/recovery behavior. DLV-PLAT-007 and its User Story 1 remain open until
the broader delivery evidence is complete.
