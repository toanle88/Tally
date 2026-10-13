-- name: GetWorkflowApprovalPolicy :one
SELECT
    id,
    code,
    current_policy_version,
    status,
    aggregate_version,
    created_at,
    updated_at,
    last_audit_reference
FROM workflow.approval_policy
WHERE id = sqlc.arg(id);

-- name: ListWorkflowApprovalPolicies :many
SELECT
    id,
    code,
    current_policy_version,
    status,
    aggregate_version,
    created_at,
    updated_at,
    last_audit_reference
FROM workflow.approval_policy
ORDER BY id;

-- name: ListSelectableWorkflowApprovalPolicies :many
SELECT
    p.id,
    p.code,
    p.current_policy_version,
    p.status,
    p.aggregate_version,
    p.created_at,
    p.updated_at,
    p.last_audit_reference,
    r.policy_version,
    r.name,
    r.applicability,
    r.scope,
    r.rules,
    r.steps,
    r.effective_from,
    r.effective_to,
    r.approval,
    r.approval_request_id,
    r.activation_outcome,
    r.dependent_workflow_impact,
    r.audit_reference,
    r.created_at AS revision_created_at
FROM workflow.approval_policy AS p
JOIN workflow.approval_policy_revision AS r
  ON r.policy_id = p.id
 AND r.revision_version = p.aggregate_version
WHERE r.status = 'Active'
ORDER BY p.id, r.revision_version;

-- name: GetWorkflowApprovalPolicyRevision :one
SELECT
    policy_id,
    revision_version,
    policy_version,
    code,
    name,
    status,
    applicability,
    scope,
    rules,
    steps,
    effective_from,
    effective_to,
    approval,
    approval_request_id,
    activation_outcome,
    dependent_workflow_impact,
    audit_reference,
    created_at
FROM workflow.approval_policy_revision
WHERE policy_id = sqlc.arg(policy_id)
  AND revision_version = sqlc.arg(revision_version);

-- name: CreateWorkflowApprovalPolicy :exec
INSERT INTO workflow.approval_policy (
    id, code, current_policy_version, status, aggregate_version,
    created_at, updated_at, last_audit_reference
) VALUES (
    sqlc.arg(id), sqlc.arg(code), sqlc.arg(current_policy_version), sqlc.arg(status),
    sqlc.arg(aggregate_version), sqlc.arg(created_at), sqlc.arg(updated_at), sqlc.arg(last_audit_reference)
);

-- name: UpdateWorkflowApprovalPolicy :execrows
UPDATE workflow.approval_policy
SET code = sqlc.arg(code),
    current_policy_version = sqlc.arg(current_policy_version),
    status = sqlc.arg(status),
    aggregate_version = sqlc.arg(next_aggregate_version),
    updated_at = sqlc.arg(updated_at),
    last_audit_reference = sqlc.arg(last_audit_reference)
WHERE id = sqlc.arg(id)
  AND aggregate_version = sqlc.arg(expected_aggregate_version);

-- name: CreateWorkflowApprovalPolicyVersionIdentity :exec
INSERT INTO workflow.approval_policy_version_identity (
    policy_id, policy_version, code, candidate_fingerprint, selector_fingerprint,
    effective_from, effective_to, created_at
) VALUES (
    sqlc.arg(policy_id), sqlc.arg(policy_version), sqlc.arg(code), sqlc.arg(candidate_fingerprint),
    sqlc.arg(selector_fingerprint), sqlc.arg(effective_from), sqlc.arg(effective_to), sqlc.arg(created_at)
);

-- name: CreateWorkflowApprovalPolicyRevision :exec
INSERT INTO workflow.approval_policy_revision (
    policy_id, revision_version, policy_version, code, name, status,
    applicability, scope, rules, steps, effective_from, effective_to, approval,
    approval_request_id, activation_outcome, dependent_workflow_impact, audit_reference, created_at
) VALUES (
    sqlc.arg(policy_id), sqlc.arg(revision_version), sqlc.arg(policy_version), sqlc.arg(code), sqlc.arg(name), sqlc.arg(status),
    sqlc.arg(applicability)::jsonb, sqlc.arg(scope)::jsonb, sqlc.arg(rules)::jsonb, sqlc.arg(steps)::jsonb, sqlc.arg(effective_from), sqlc.arg(effective_to),
    sqlc.narg(approval)::jsonb, sqlc.narg(approval_request_id), sqlc.arg(activation_outcome), sqlc.arg(dependent_workflow_impact), sqlc.arg(audit_reference), sqlc.arg(created_at)
);

-- name: CreateWorkflowApprovalPolicyRule :exec
INSERT INTO workflow.approval_policy_rule (
    policy_id, revision_version, rule_id, name, minimum_amount, maximum_amount,
    currency, required_roles, delegation, escalation
) VALUES (
    sqlc.arg(policy_id), sqlc.arg(revision_version), sqlc.arg(rule_id), sqlc.arg(name),
    sqlc.narg(minimum_amount), sqlc.narg(maximum_amount), sqlc.arg(currency),
    sqlc.arg(required_roles)::jsonb, sqlc.arg(delegation)::jsonb, sqlc.arg(escalation)::jsonb
);

-- name: CreateWorkflowApprovalPolicyStep :exec
INSERT INTO workflow.approval_policy_step_definition (
    policy_id, revision_version, step_id, sequence, name, required_roles,
    minimum_approvals, delegation, escalation
) VALUES (
    sqlc.arg(policy_id), sqlc.arg(revision_version), sqlc.arg(step_id), sqlc.arg(sequence), sqlc.arg(name),
    sqlc.arg(required_roles)::jsonb, sqlc.arg(minimum_approvals), sqlc.arg(delegation)::jsonb, sqlc.arg(escalation)::jsonb
);

-- name: GetWorkflowApprovalRequest :one
SELECT
    id, subject_type, subject_id, subject_version, subject_fingerprint,
    policy_id, policy_version, scope, steps, status, aggregate_version,
    requester_user_id, requester_subject_reference, correlation_id,
    causation_id, created_at, updated_at
FROM workflow.approval_request
WHERE id = sqlc.arg(id);

-- name: CreateWorkflowApprovalRequest :exec
INSERT INTO workflow.approval_request (
    id, subject_type, subject_id, subject_version, subject_fingerprint,
    policy_id, policy_version, scope, steps, status, aggregate_version,
    requester_user_id, requester_subject_reference, correlation_id,
    causation_id, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(subject_type), sqlc.arg(subject_id), sqlc.arg(subject_version), sqlc.arg(subject_fingerprint),
    sqlc.arg(policy_id), sqlc.arg(policy_version), sqlc.arg(scope)::jsonb, sqlc.arg(steps)::jsonb, sqlc.arg(status), sqlc.arg(aggregate_version),
    sqlc.arg(requester_user_id), sqlc.arg(requester_subject_reference), sqlc.narg(correlation_id), sqlc.narg(causation_id), sqlc.arg(created_at), sqlc.arg(updated_at)
);

-- name: UpdateWorkflowApprovalRequest :execrows
UPDATE workflow.approval_request
SET status = sqlc.arg(status),
    aggregate_version = sqlc.arg(next_aggregate_version),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND aggregate_version = sqlc.arg(expected_aggregate_version);

-- name: CreateWorkflowApprovalDecision :exec
INSERT INTO workflow.approval_decision (
    id, approval_request_id, decision_version, outcome, approver_user_id,
    approver_subject_reference, reason, policy_version, subject_version,
    candidate_fingerprint, recorded_at
) VALUES (
    sqlc.arg(id), sqlc.arg(approval_request_id), sqlc.arg(decision_version), sqlc.arg(outcome), sqlc.arg(approver_user_id),
    sqlc.arg(approver_subject_reference), sqlc.arg(reason), sqlc.arg(policy_version), sqlc.arg(subject_version),
    sqlc.arg(candidate_fingerprint), sqlc.arg(recorded_at)
);
