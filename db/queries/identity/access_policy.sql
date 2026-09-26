-- name: ListCurrentIdentityAccessPolicies :many
SELECT
    r.policy_id,
    r.policy_version,
    p.status,
    r.effective_from,
    r.effective_to,
    to_jsonb(r.subject_actor_ids) AS subject_actor_ids,
    to_jsonb(r.subject_role_ids) AS subject_role_ids,
    to_jsonb(r.permissions) AS permissions,
    r.rules
FROM identity.access_policy_revision AS r
JOIN identity.access_policy AS p
  ON p.id = r.policy_id
 AND p.current_version = r.policy_version
ORDER BY r.policy_id;

-- name: GetCurrentIdentityAccessPolicyVersion :one
SELECT current_version
FROM identity.access_policy
WHERE id = sqlc.arg(policy_id);
