-- name: GetIdentityEmergencyAccessGrant :one
SELECT
    g.id,
    g.target_actor_id,
    g.current_version,
    g.status,
    g.created_at,
    g.updated_at,
    g.last_audit_reference,
    r.reason_code,
    r.granting_actor_id,
    r.approver_user_id,
    r.approval_request_id,
    r.approval_decision_id,
    r.policy_version,
    r.decision_version,
    r.subject_version,
    r.candidate_fingerprint,
    r.start_at,
    r.expires_at,
    r.review_status,
    r.review_outcome_code,
    r.review_reference,
    r.review_due_at,
    r.revoked_at,
    r.revoked_by,
    r.revocation_reason,
    r.audit_reference,
    r.created_at AS revision_created_at
FROM identity.emergency_access_grant AS g
JOIN identity.emergency_access_grant_revision AS r
  ON r.grant_id = g.id
 AND r.revision_version = g.current_version
WHERE g.id = sqlc.arg(id);

-- name: ListIdentityEmergencyAccessGrants :many
SELECT
    g.id,
    g.target_actor_id,
    g.current_version,
    g.status,
    g.created_at,
    g.updated_at,
    g.last_audit_reference,
    r.reason_code,
    r.granting_actor_id,
    r.approver_user_id,
    r.approval_request_id,
    r.approval_decision_id,
    r.policy_version,
    r.decision_version,
    r.subject_version,
    r.candidate_fingerprint,
    r.start_at,
    r.expires_at,
    r.review_status,
    r.review_outcome_code,
    r.review_reference,
    r.review_due_at,
    r.revoked_at,
    r.revoked_by,
    r.revocation_reason,
    r.audit_reference,
    r.created_at AS revision_created_at
FROM identity.emergency_access_grant AS g
JOIN identity.emergency_access_grant_revision AS r
  ON r.grant_id = g.id
 AND r.revision_version = g.current_version
ORDER BY g.id;

-- name: ListIdentityEmergencyAccessGrantPermissions :many
SELECT permission
FROM identity.emergency_access_grant_permission
WHERE grant_id = sqlc.arg(grant_id)
  AND grant_version = sqlc.arg(grant_version)
ORDER BY permission;

-- name: ListIdentityEmergencyAccessGrantScopes :many
SELECT scope_id
FROM identity.emergency_access_grant_scope
WHERE grant_id = sqlc.arg(grant_id)
  AND grant_version = sqlc.arg(grant_version)
ORDER BY scope_id;

-- name: CreateIdentityEmergencyAccessGrant :exec
INSERT INTO identity.emergency_access_grant (
    id, target_actor_id, current_version, status, created_at, updated_at, last_audit_reference
) VALUES (
    sqlc.arg(id), sqlc.arg(target_actor_id), sqlc.arg(current_version), sqlc.arg(status),
    sqlc.arg(created_at), sqlc.arg(updated_at), sqlc.arg(last_audit_reference)
);

-- name: UpdateIdentityEmergencyAccessGrant :execrows
UPDATE identity.emergency_access_grant
SET current_version = sqlc.arg(next_current_version),
    status = sqlc.arg(status),
    updated_at = sqlc.arg(updated_at),
    last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id)
  AND current_version = sqlc.arg(expected_current_version);

-- name: CreateIdentityEmergencyAccessGrantRevision :exec
INSERT INTO identity.emergency_access_grant_revision (
    grant_id, revision_version, target_actor_id, status, reason_code, granting_actor_id,
    approver_user_id, approval_request_id, approval_decision_id, policy_version,
    decision_version, subject_version, candidate_fingerprint, start_at, expires_at,
    review_status, review_outcome_code, review_reference, review_due_at, revoked_at,
    revoked_by, revocation_reason, audit_reference, created_at
) VALUES (
    sqlc.arg(grant_id), sqlc.arg(revision_version), sqlc.arg(target_actor_id), sqlc.arg(status),
    sqlc.arg(reason_code), sqlc.arg(granting_actor_id), sqlc.arg(approver_user_id),
    sqlc.arg(approval_request_id), sqlc.arg(approval_decision_id), sqlc.arg(policy_version),
    sqlc.arg(decision_version), sqlc.arg(subject_version), sqlc.arg(candidate_fingerprint),
    sqlc.arg(start_at), sqlc.arg(expires_at), sqlc.arg(review_status), sqlc.arg(review_outcome_code),
    sqlc.arg(review_reference), sqlc.arg(review_due_at), sqlc.arg(revoked_at), sqlc.arg(revoked_by),
    sqlc.arg(revocation_reason), sqlc.arg(audit_reference), sqlc.arg(created_at)
);

-- name: CreateIdentityEmergencyAccessGrantPermission :exec
INSERT INTO identity.emergency_access_grant_permission (grant_id, grant_version, permission)
VALUES (sqlc.arg(grant_id), sqlc.arg(grant_version), sqlc.arg(permission));

-- name: CreateIdentityEmergencyAccessGrantScope :exec
INSERT INTO identity.emergency_access_grant_scope (grant_id, grant_version, scope_id)
VALUES (sqlc.arg(grant_id), sqlc.arg(grant_version), sqlc.arg(scope_id));

-- name: SetIdentityEmergencyAccessGrantAuditReference :exec
UPDATE identity.emergency_access_grant
SET last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id);
