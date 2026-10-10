-- +goose Up
ALTER TABLE coa.segment_change_request
    ADD COLUMN approval_decision_id uuid,
    ADD COLUMN approval_policy_version text,
    ADD COLUMN approval_decision_version bigint,
    ADD COLUMN approval_subject_version bigint,
    ADD COLUMN approval_candidate_fingerprint text,
    ADD COLUMN approval_approver_user_id uuid,
    ADD COLUMN approval_decided_at timestamptz,
    ADD COLUMN approval_applied_at timestamptz,
    ADD COLUMN applied_subject_version bigint,
    ADD COLUMN resulting_subject_version bigint,
    ADD COLUMN decision_fingerprint text;

ALTER TABLE coa.segment_change_request
    DROP CONSTRAINT segment_change_request_application_status_check,
    ADD CONSTRAINT segment_change_request_application_status_check
        CHECK (application_status IN ('not-applied', 'applied', 'unchanged', 'conflict')),
    ADD CONSTRAINT segment_change_request_approval_terminal_check
        CHECK (
            (approval_status = 'pending' AND application_status = 'not-applied'
             AND next_action = 'await-approval' AND decision_fingerprint IS NULL)
            OR
            (approval_status IN ('approved', 'rejected')
             AND (
                 (approval_status = 'approved' AND application_status IN ('applied', 'unchanged', 'conflict') AND btrim(COALESCE(rejection_reason, '')) = '')
                 OR
                 (approval_status = 'rejected' AND application_status = 'unchanged' AND btrim(COALESCE(rejection_reason, '')) <> '')
             )
             AND (
                 (application_status = 'conflict' AND next_action = 'resolve-conflict' AND btrim(COALESCE(conflict_code, '')) <> '')
                 OR
                 (application_status IN ('applied', 'unchanged') AND next_action = 'completed')
             )
             AND approval_decision_id IS NOT NULL
             AND btrim(COALESCE(approval_policy_version, '')) <> ''
             AND approval_decision_version >= 1
             AND approval_subject_version >= 1
             AND btrim(COALESCE(approval_candidate_fingerprint, '')) <> ''
             AND approval_approver_user_id IS NOT NULL
             AND approval_decided_at IS NOT NULL
             AND approval_applied_at IS NOT NULL
             AND applied_subject_version >= 1
             AND resulting_subject_version >= 1
             AND btrim(COALESCE(decision_fingerprint, '')) <> '')
        ),
    ADD CONSTRAINT segment_change_request_approval_version_check
        CHECK (approval_decision_version IS NULL OR approval_decision_version >= 1),
    ADD CONSTRAINT segment_change_request_applied_version_check
        CHECK (applied_subject_version IS NULL OR applied_subject_version >= 1),
    ADD CONSTRAINT segment_change_request_resulting_version_check
        CHECK (resulting_subject_version IS NULL OR resulting_subject_version >= 1);

CREATE INDEX segment_change_request_decision_idx
    ON coa.segment_change_request (approval_decision_id)
    WHERE approval_decision_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS coa.segment_change_request_decision_idx;
ALTER TABLE coa.segment_change_request
    DROP CONSTRAINT IF EXISTS segment_change_request_approval_terminal_check,
    DROP CONSTRAINT IF EXISTS segment_change_request_application_status_check;
UPDATE coa.segment_change_request
SET application_status = 'conflict'
WHERE application_status = 'unchanged';
ALTER TABLE coa.segment_change_request
    DROP CONSTRAINT IF EXISTS segment_change_request_resulting_version_check,
    DROP CONSTRAINT IF EXISTS segment_change_request_applied_version_check,
    DROP CONSTRAINT IF EXISTS segment_change_request_approval_version_check,
    ADD CONSTRAINT segment_change_request_application_status_check
        CHECK (application_status IN ('not-applied', 'applied', 'conflict'));
ALTER TABLE coa.segment_change_request
    DROP COLUMN IF EXISTS approval_decision_id,
    DROP COLUMN IF EXISTS approval_policy_version,
    DROP COLUMN IF EXISTS approval_decision_version,
    DROP COLUMN IF EXISTS approval_subject_version,
    DROP COLUMN IF EXISTS approval_candidate_fingerprint,
    DROP COLUMN IF EXISTS approval_approver_user_id,
    DROP COLUMN IF EXISTS approval_decided_at,
    DROP COLUMN IF EXISTS approval_applied_at,
    DROP COLUMN IF EXISTS applied_subject_version,
    DROP COLUMN IF EXISTS resulting_subject_version,
    DROP COLUMN IF EXISTS decision_fingerprint;
