-- name: InsertInbox :one
INSERT INTO integration.inbox (
    consumer_name,
    message_id,
    message_fingerprint,
    state,
    result_reference,
    established_at
)
VALUES (
    sqlc.arg(consumer_name),
    sqlc.arg(message_id),
    sqlc.arg(message_fingerprint),
    sqlc.arg(state),
    sqlc.narg(result_reference)::jsonb,
    sqlc.narg(established_at)
)
RETURNING *;

-- name: InsertInboxIfAbsent :one
INSERT INTO integration.inbox (
    consumer_name,
    message_id,
    message_fingerprint,
    state
)
VALUES (
    sqlc.arg(consumer_name),
    sqlc.arg(message_id),
    sqlc.arg(message_fingerprint),
    'processing'
)
ON CONFLICT (consumer_name, message_id) DO NOTHING
RETURNING *;

-- name: GetInboxByIdentity :one
SELECT *
FROM integration.inbox
WHERE consumer_name = sqlc.arg(consumer_name)
  AND message_id = sqlc.arg(message_id);

-- name: GetInboxForUpdate :one
SELECT *
FROM integration.inbox
WHERE consumer_name = sqlc.arg(consumer_name)
  AND message_id = sqlc.arg(message_id)
FOR UPDATE;

-- name: EstablishInbox :execrows
UPDATE integration.inbox
SET state = 'established',
    result_reference = sqlc.narg(result_reference)::jsonb,
    established_at = clock_timestamp()
WHERE consumer_name = sqlc.arg(consumer_name)
  AND message_id = sqlc.arg(message_id)
  AND state IN ('processing', 'failed');

-- name: RecordInboxFailure :execrows
INSERT INTO integration.inbox (
    consumer_name,
    message_id,
    message_fingerprint,
    state
)
VALUES (
    sqlc.arg(consumer_name),
    sqlc.arg(message_id),
    sqlc.arg(message_fingerprint),
    'failed'
)
ON CONFLICT (consumer_name, message_id) DO UPDATE
SET state = 'failed',
    result_reference = NULL,
    established_at = NULL
WHERE integration.inbox.state <> 'established';

-- name: ListInboxForReconciliation :many
SELECT *
FROM integration.inbox
WHERE state IN ('processing', 'failed')
ORDER BY first_received_at, consumer_name, message_id
LIMIT sqlc.arg(max_rows);
