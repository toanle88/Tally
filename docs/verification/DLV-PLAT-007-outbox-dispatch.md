# DLV-PLAT-007 User Story 4 — Outbox dispatch verification

## Verification status

Implemented on 2026-08-29 from branch
`feat/dlv-plat-007-us4-outbox-dispatch`.

## Evidence scope

- Due outbox work is claimed in deterministic order with `FOR UPDATE SKIP LOCKED`.
- Lease renewal, establishment, rescheduling, and managed-exception transitions
  are fenced by the current owner and an unexpired lease.
- Typed transient dependency failures use bounded default backoff and a fixed
  ten-attempt limit.
- Non-transient failures become managed exceptions without automatic retry.
- Managed exception evidence remains durable and is excluded from future claims.

## Verification command

```bash
make outbox-dispatch-check
```

The focused gate runs dispatcher unit, race, vet, package-boundary, migration,
SQLC, and PostgreSQL integration checks.

## Non-claims

This verification does not claim worker-host lifecycle, shutdown orchestration,
replay, delivery ordering, semantic payload minimization, external broker
support, or finance-level exactly-once business effects. Those remain User Story
5 or owning bounded-context scope.
