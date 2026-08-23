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

-- name: GetInboxByIdentity :one
SELECT *
FROM integration.inbox
WHERE consumer_name = sqlc.arg(consumer_name)
  AND message_id = sqlc.arg(message_id);

-- name: ListInboxForReconciliation :many
SELECT *
FROM integration.inbox
WHERE state IN ('processing', 'failed')
ORDER BY first_received_at, consumer_name, message_id
LIMIT sqlc.arg(max_rows);
