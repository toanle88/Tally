-- name: GetIdentityRole :one
SELECT
    r.id,
    r.name,
    r.status,
    r.aggregate_version,
    r.created_at,
    r.updated_at,
    r.last_audit_reference,
    rr.approval_request_id,
    rr.approval_decision_id,
    rr.approver_user_id,
    rr.policy_version,
    rr.decision_version,
    rr.subject_version,
    rr.candidate_fingerprint,
    rr.audit_reference,
    rr.created_at AS revision_created_at
FROM identity.role AS r
JOIN identity.role_revision AS rr
  ON rr.role_id = r.id
 AND rr.revision_version = r.aggregate_version
WHERE r.id = sqlc.arg(id);

-- name: ListIdentityRoles :many
SELECT
    r.id,
    r.name,
    r.status,
    r.aggregate_version,
    r.created_at,
    r.updated_at,
    r.last_audit_reference,
    rr.approval_request_id,
    rr.approval_decision_id,
    rr.approver_user_id,
    rr.policy_version,
    rr.decision_version,
    rr.subject_version,
    rr.candidate_fingerprint,
    rr.audit_reference,
    rr.created_at AS revision_created_at
FROM identity.role AS r
JOIN identity.role_revision AS rr
  ON rr.role_id = r.id
 AND rr.revision_version = r.aggregate_version
ORDER BY r.id;

-- name: ListIdentityRoleGrants :many
SELECT
    role_id,
    role_version,
    permission,
    scope_id,
    effective_from,
    effective_to
FROM identity.role_permission_grant
WHERE role_id = sqlc.arg(role_id)
  AND role_version = sqlc.arg(role_version)
ORDER BY permission, scope_id, effective_from;

-- name: CreateIdentityRole :exec
INSERT INTO identity.role (
    id,
    name,
    status,
    aggregate_version,
    created_at,
    updated_at,
    last_audit_reference
) VALUES (
    sqlc.arg(id),
    sqlc.arg(name),
    sqlc.arg(status),
    sqlc.arg(aggregate_version),
    sqlc.arg(created_at),
    sqlc.arg(updated_at),
    sqlc.arg(last_audit_reference)
);

-- name: UpdateIdentityRole :execrows
UPDATE identity.role
SET name = sqlc.arg(name),
    status = sqlc.arg(status),
    aggregate_version = sqlc.arg(next_aggregate_version),
    updated_at = sqlc.arg(updated_at),
    last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id)
  AND aggregate_version = sqlc.arg(expected_aggregate_version);

-- name: CreateIdentityRoleRevision :exec
INSERT INTO identity.role_revision (
    role_id,
    revision_version,
    name,
    status,
    approval_request_id,
    approval_decision_id,
    approver_user_id,
    policy_version,
    decision_version,
    subject_version,
    candidate_fingerprint,
    audit_reference,
    created_at
) VALUES (
    sqlc.arg(role_id),
    sqlc.arg(revision_version),
    sqlc.arg(name),
    sqlc.arg(status),
    sqlc.arg(approval_request_id),
    sqlc.arg(approval_decision_id),
    sqlc.arg(approver_user_id),
    sqlc.arg(policy_version),
    sqlc.arg(decision_version),
    sqlc.arg(subject_version),
    sqlc.arg(candidate_fingerprint),
    sqlc.arg(audit_reference),
    sqlc.arg(created_at)
);

-- name: CreateIdentityRoleGrant :exec
INSERT INTO identity.role_permission_grant (
    role_id,
    role_version,
    permission,
    scope_id,
    effective_from,
    effective_to
) VALUES (
    sqlc.arg(role_id),
    sqlc.arg(role_version),
    sqlc.arg(permission),
    sqlc.arg(scope_id),
    sqlc.arg(effective_from),
    sqlc.arg(effective_to)
);

-- name: SetIdentityRoleAuditReference :exec
UPDATE identity.role
SET last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id);
