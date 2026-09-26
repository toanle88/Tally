-- name: GetCurrentIdentitySegregationRule :one
SELECT
    r.id,
    r.code,
    r.current_version,
    r.status,
    r.created_at,
    r.updated_at,
    r.last_audit_reference,
    rr.name,
    rr.conflicting_permissions,
    rr.enforcement_mode,
    rr.scope_ids,
    rr.amount_threshold,
    rr.cooling_off_seconds,
    rr.effective_from,
    rr.effective_to,
    rr.approval_request_id,
    rr.approval_decision_id,
    rr.approver_user_id,
    rr.policy_version,
    rr.decision_version,
    rr.subject_version,
    rr.candidate_fingerprint,
    rr.audit_reference,
    rr.created_at AS revision_created_at
FROM identity.segregation_rule AS r
JOIN identity.segregation_rule_revision AS rr
  ON rr.rule_id = r.id
 AND rr.revision_version = r.current_version
WHERE r.id = sqlc.arg(id);

-- name: ListCurrentIdentitySegregationRules :many
SELECT
    r.id,
    r.code,
    r.current_version,
    r.status,
    r.created_at,
    r.updated_at,
    r.last_audit_reference,
    rr.name,
    rr.conflicting_permissions,
    rr.enforcement_mode,
    rr.scope_ids,
    rr.amount_threshold,
    rr.cooling_off_seconds,
    rr.effective_from,
    rr.effective_to,
    rr.approval_request_id,
    rr.approval_decision_id,
    rr.approver_user_id,
    rr.policy_version,
    rr.decision_version,
    rr.subject_version,
    rr.candidate_fingerprint,
    rr.audit_reference,
    rr.created_at AS revision_created_at
FROM identity.segregation_rule AS r
JOIN identity.segregation_rule_revision AS rr
  ON rr.rule_id = r.id
 AND rr.revision_version = r.current_version
ORDER BY r.code;

-- name: CreateIdentitySegregationRule :exec
INSERT INTO identity.segregation_rule (
    id, code, current_version, status, created_at, updated_at, last_audit_reference
) VALUES (
    sqlc.arg(id), sqlc.arg(code), sqlc.arg(current_version), sqlc.arg(status),
    sqlc.arg(created_at), sqlc.arg(updated_at), sqlc.arg(last_audit_reference)
);

-- name: UpdateIdentitySegregationRule :execrows
UPDATE identity.segregation_rule
SET code = sqlc.arg(code),
    current_version = sqlc.arg(next_current_version),
    status = sqlc.arg(status),
    updated_at = sqlc.arg(updated_at),
    last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id)
  AND current_version = sqlc.arg(expected_current_version);

-- name: CreateIdentitySegregationRuleRevision :exec
INSERT INTO identity.segregation_rule_revision (
    rule_id, revision_version, code, name, status, conflicting_permissions,
    enforcement_mode, scope_ids, amount_threshold, cooling_off_seconds,
    effective_from, effective_to, approval_request_id, approval_decision_id,
    approver_user_id, policy_version, decision_version, subject_version,
    candidate_fingerprint, audit_reference, created_at
) VALUES (
    sqlc.arg(rule_id), sqlc.arg(revision_version), sqlc.arg(code), sqlc.arg(name),
    sqlc.arg(status), sqlc.arg(conflicting_permissions), sqlc.arg(enforcement_mode),
    sqlc.arg(scope_ids), sqlc.arg(amount_threshold), sqlc.arg(cooling_off_seconds),
    sqlc.arg(effective_from), sqlc.arg(effective_to), sqlc.arg(approval_request_id),
    sqlc.arg(approval_decision_id), sqlc.arg(approver_user_id), sqlc.arg(policy_version),
    sqlc.arg(decision_version), sqlc.arg(subject_version), sqlc.arg(candidate_fingerprint),
    sqlc.arg(audit_reference), sqlc.arg(created_at)
);

-- name: SetIdentitySegregationRuleAuditReference :exec
UPDATE identity.segregation_rule
SET last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id);
