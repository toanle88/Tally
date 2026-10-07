-- name: GetSegmentChangeRequest :one
SELECT segment_change_request_id, scope_id, change_type, subject_id, subject_version,
       requested_effective_date, approval_request_id, approval_status, application_status,
       validation_outcome, conflict_code, rejection_reason, next_action, proposed_change,
       subject_fingerprint, proposed_fingerprint, aggregate_version, revision_number,
       created_by, created_at, updated_at, last_audit_reference
FROM coa.segment_change_request
WHERE segment_change_request_id = sqlc.arg(segment_change_request_id);

-- name: ListSegmentChangeRequests :many
SELECT segment_change_request_id, scope_id, change_type, subject_id, subject_version,
       requested_effective_date, approval_request_id, approval_status, application_status,
       validation_outcome, conflict_code, rejection_reason, next_action, proposed_change,
       subject_fingerprint, proposed_fingerprint, aggregate_version, revision_number,
       created_by, created_at, updated_at, last_audit_reference
FROM coa.segment_change_request
WHERE (sqlc.narg(scope_id)::uuid IS NULL OR scope_id = sqlc.narg(scope_id)::uuid)
ORDER BY requested_effective_date, segment_change_request_id;
