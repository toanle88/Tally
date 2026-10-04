-- name: ListSegmentValueRows :many
SELECT segment_value_id, segment_definition_id, value, description, status,
       effective_from, effective_to, created_at, updated_at
FROM coa.segment_value
WHERE segment_definition_id = sqlc.arg(segment_definition_id)
ORDER BY value, effective_from, segment_value_id;
