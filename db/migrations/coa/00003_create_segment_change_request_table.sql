-- +goose Up
CREATE TABLE coa.segment_change_request (
    segment_change_request_id uuid NOT NULL,
    scope_id uuid NOT NULL,
    change_type text NOT NULL,
    subject_id uuid NOT NULL,
    subject_version bigint NOT NULL,
    requested_effective_date date NOT NULL,
    approval_request_id uuid NOT NULL,
    approval_status text NOT NULL,
    application_status text NOT NULL,
    validation_outcome text NOT NULL,
    conflict_code text,
    rejection_reason text,
    next_action text NOT NULL,
    proposed_change jsonb NOT NULL,
    subject_fingerprint text NOT NULL,
    proposed_fingerprint text NOT NULL,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT segment_change_request_pk PRIMARY KEY (segment_change_request_id),
    CONSTRAINT segment_change_request_type_check CHECK (change_type IN ('definition', 'value')),
    CONSTRAINT segment_change_request_subject_version_check CHECK (subject_version >= 1),
    CONSTRAINT segment_change_request_approval_status_check CHECK (approval_status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT segment_change_request_application_status_check CHECK (application_status IN ('not-applied', 'applied', 'conflict')),
    CONSTRAINT segment_change_request_validation_check CHECK (validation_outcome IN ('valid', 'invalid')),
    CONSTRAINT segment_change_request_aggregate_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT segment_change_request_revision_check CHECK (revision_number >= 1),
    CONSTRAINT segment_change_request_timestamp_check CHECK (updated_at >= created_at),
    CONSTRAINT segment_change_request_fingerprint_check CHECK (btrim(subject_fingerprint) <> '' AND btrim(proposed_fingerprint) <> '')
);

CREATE INDEX segment_change_request_scope_status_idx
    ON coa.segment_change_request (scope_id, approval_status, application_status, requested_effective_date);

CREATE INDEX segment_change_request_subject_idx
    ON coa.segment_change_request (scope_id, change_type, subject_id, subject_version);

CREATE UNIQUE INDEX segment_change_request_pending_subject_fingerprint_unique
    ON coa.segment_change_request (scope_id, change_type, subject_id, subject_version, proposed_fingerprint)
    WHERE approval_status = 'pending' AND application_status = 'not-applied';

-- +goose Down
DROP TABLE IF EXISTS coa.segment_change_request;
