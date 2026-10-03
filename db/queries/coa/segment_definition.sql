-- name: GetSegmentDefinitionRoot :one
SELECT segment_definition_id, scope_id, segment_type, code, name, status,
       effective_from, effective_to, aggregate_version, revision_number,
       created_at, updated_at, last_audit_reference
FROM coa.segment_definition
WHERE segment_definition_id = sqlc.arg(segment_definition_id);

-- name: ListSegmentDefinitionRoots :many
SELECT segment_definition_id, scope_id, segment_type, code, name, status,
       effective_from, effective_to, aggregate_version, revision_number,
       created_at, updated_at, last_audit_reference
FROM coa.segment_definition
WHERE (sqlc.narg(scope_id)::uuid IS NULL OR scope_id = sqlc.narg(scope_id)::uuid)
ORDER BY segment_definition_id;
