-- name: GetLegalEntityRoot :one
SELECT id, scope_id, legal_name, functional_currency, presentation_currency,
       tax_registration_id, status, effective_from, effective_to,
       aggregate_version, revision_number, approval_reference,
       created_at, updated_at, last_audit_reference
FROM organization.legal_entity
WHERE id = sqlc.arg(id);

-- name: ListLegalEntityRoots :many
SELECT id, scope_id, legal_name, functional_currency, presentation_currency,
       tax_registration_id, status, effective_from, effective_to,
       aggregate_version, revision_number, approval_reference,
       created_at, updated_at, last_audit_reference
FROM organization.legal_entity
WHERE (sqlc.narg(scope_id)::uuid IS NULL OR scope_id = sqlc.narg(scope_id)::uuid)
ORDER BY id;
