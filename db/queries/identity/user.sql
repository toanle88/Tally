-- name: GetIdentityUser :one
SELECT
    id,
    authentication_subject_oid,
    authentication_subject_tid,
    authentication_subject_sub,
    status,
    aggregate_version,
    created_at,
    updated_at,
    last_audit_reference
FROM identity.user_account
WHERE id = sqlc.arg(id);

-- name: FindIdentityUserBySubject :one
SELECT
    id,
    authentication_subject_oid,
    authentication_subject_tid,
    authentication_subject_sub,
    status,
    aggregate_version,
    created_at,
    updated_at,
    last_audit_reference
FROM identity.user_account
WHERE authentication_subject_oid = sqlc.arg(authentication_subject_oid)
  AND authentication_subject_tid = sqlc.arg(authentication_subject_tid)
  AND authentication_subject_sub = sqlc.arg(authentication_subject_sub);

-- name: ListIdentityUsers :many
SELECT
    id,
    authentication_subject_oid,
    authentication_subject_tid,
    authentication_subject_sub,
    status,
    aggregate_version,
    created_at,
    updated_at,
    last_audit_reference
FROM identity.user_account
ORDER BY id;

-- name: CreateIdentityUser :exec
INSERT INTO identity.user_account (
    id,
    authentication_subject_oid,
    authentication_subject_tid,
    authentication_subject_sub,
    status,
    aggregate_version,
    created_at,
    updated_at,
    last_audit_reference
) VALUES (
    sqlc.arg(id),
    sqlc.arg(authentication_subject_oid),
    sqlc.arg(authentication_subject_tid),
    sqlc.arg(authentication_subject_sub),
    sqlc.arg(status),
    sqlc.arg(aggregate_version),
    sqlc.arg(created_at),
    sqlc.arg(updated_at),
    sqlc.arg(last_audit_reference)
);

-- name: UpdateIdentityUser :execrows
UPDATE identity.user_account
SET status = sqlc.arg(status),
    aggregate_version = sqlc.arg(next_aggregate_version),
    updated_at = sqlc.arg(updated_at),
    last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id)
  AND aggregate_version = sqlc.arg(expected_aggregate_version);

-- name: SetIdentityUserAuditReference :exec
UPDATE identity.user_account
SET last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id);

-- name: DeleteIdentityUserAssignments :exec
DELETE FROM identity.user_role_assignment
WHERE user_id = sqlc.arg(user_id);

-- name: CreateIdentityUserRoleAssignment :exec
INSERT INTO identity.user_role_assignment (user_id, role_id)
VALUES (sqlc.arg(user_id), sqlc.arg(role_id));

-- name: CreateIdentityUserRoleAssignmentScope :exec
INSERT INTO identity.user_role_assignment_scope (user_id, role_id, scope_id)
VALUES (sqlc.arg(user_id), sqlc.arg(role_id), sqlc.arg(scope_id));

-- name: ListIdentityUserRoleAssignments :many
SELECT
    user_id,
    role_id,
    scope_id
FROM identity.user_role_assignment_scope
WHERE user_id = sqlc.arg(user_id)
ORDER BY role_id, scope_id;
