-- name: GetPartyRoot :one
SELECT id, scope_id, name, party_type, status, tax_identifier, bank_control,
       aggregate_version, revision_number, created_at, updated_at, last_audit_reference
FROM organization.party
WHERE id = sqlc.arg(id);

-- name: ListPartyRoots :many
SELECT id, scope_id, name, party_type, status, tax_identifier, bank_control,
       aggregate_version, revision_number, created_at, updated_at, last_audit_reference
FROM organization.party
WHERE (sqlc.narg(scope_id)::uuid IS NULL OR scope_id = sqlc.narg(scope_id)::uuid)
ORDER BY id;
