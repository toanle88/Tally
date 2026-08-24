-- name: InsertOutbox :one
INSERT INTO integration.outbox (
    outbox_id,
    event_type,
    event_version,
    occurred_at,
    source_context,
    aggregate_id,
    aggregate_version,
    accounting_scope_id,
    correlation_id,
    causation_id,
    payload,
    payload_fingerprint,
    data_classification,
    available_at
)
VALUES (
    sqlc.arg(outbox_id),
    sqlc.arg(event_type),
    sqlc.arg(event_version),
    sqlc.arg(occurred_at),
    sqlc.arg(source_context),
    sqlc.arg(aggregate_id),
    sqlc.arg(aggregate_version),
    sqlc.narg(accounting_scope_id),
    sqlc.arg(correlation_id),
    sqlc.arg(causation_id),
    sqlc.arg(payload)::jsonb,
    sqlc.arg(payload_fingerprint),
    sqlc.arg(data_classification),
    sqlc.arg(available_at)
)
RETURNING *;

-- name: GetOutboxBySourceIdentity :one
SELECT *
FROM integration.outbox
WHERE source_context = sqlc.arg(source_context)
  AND aggregate_id = sqlc.arg(aggregate_id)
  AND aggregate_version = sqlc.arg(aggregate_version)
  AND event_type = sqlc.arg(event_type);

-- name: ClaimDueOutbox :many
WITH claim AS (
    SELECT outbox_id
    FROM integration.outbox
    WHERE established_at IS NULL
      AND available_at <= clock_timestamp()
      AND (claimed_until IS NULL OR claimed_until < clock_timestamp())
    ORDER BY available_at, outbox_id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
UPDATE integration.outbox AS outbox
SET claimed_until = clock_timestamp() + sqlc.arg(lease_duration)::interval,
    claim_owner = sqlc.arg(claim_owner),
    attempt_count = outbox.attempt_count + 1
FROM claim
WHERE outbox.outbox_id = claim.outbox_id
RETURNING outbox.*;

-- name: ListExpiredOutbox :many
SELECT *
FROM integration.outbox
WHERE established_at IS NULL
  AND claimed_until IS NOT NULL
  AND claimed_until < clock_timestamp()
ORDER BY claimed_until, outbox_id
LIMIT sqlc.arg(max_rows);
