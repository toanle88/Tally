-- +goose Up
CREATE TABLE identity.emergency_access_grant (
    id uuid NOT NULL,
    target_actor_id uuid NOT NULL,
    current_version bigint NOT NULL DEFAULT 1,
    status text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,

    CONSTRAINT emergency_access_grant_pk PRIMARY KEY (id),
    CONSTRAINT emergency_access_grant_status_check CHECK (status IN ('active', 'revoked')),
    CONSTRAINT emergency_access_grant_version_check CHECK (current_version >= 1)
);

CREATE TABLE identity.emergency_access_grant_revision (
    grant_id uuid NOT NULL,
    revision_version bigint NOT NULL,
    target_actor_id uuid NOT NULL,
    status text NOT NULL,
    reason_code text NOT NULL,
    granting_actor_id uuid NOT NULL,
    approver_user_id uuid NOT NULL,
    approval_request_id uuid NOT NULL,
    approval_decision_id uuid NOT NULL,
    policy_version text NOT NULL,
    decision_version bigint NOT NULL,
    subject_version bigint NOT NULL,
    candidate_fingerprint text NOT NULL,
    start_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    review_status text NOT NULL,
    review_outcome_code text,
    review_reference text,
    review_due_at timestamptz NOT NULL,
    revoked_at timestamptz,
    revoked_by uuid,
    revocation_reason text,
    audit_reference uuid NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT emergency_access_grant_revision_pk PRIMARY KEY (grant_id, revision_version),
    CONSTRAINT emergency_access_grant_revision_grant_fk
        FOREIGN KEY (grant_id)
        REFERENCES identity.emergency_access_grant (id)
        ON DELETE RESTRICT,
    CONSTRAINT emergency_access_grant_revision_status_check CHECK (status IN ('active', 'revoked')),
    CONSTRAINT emergency_access_grant_revision_reason_check CHECK (btrim(reason_code) <> ''),
    CONSTRAINT emergency_access_grant_revision_policy_check CHECK (btrim(policy_version) <> ''),
    CONSTRAINT emergency_access_grant_revision_version_check CHECK (revision_version >= 1),
    CONSTRAINT emergency_access_grant_revision_interval_check CHECK (expires_at > start_at AND expires_at <= start_at + interval '4 hours'),
    CONSTRAINT emergency_access_grant_revision_review_status_check CHECK (review_status IN ('pending', 'completed')),
    CONSTRAINT emergency_access_grant_revision_review_outcome_check CHECK (review_status <> 'completed' OR btrim(COALESCE(review_outcome_code, '')) <> ''),
    CONSTRAINT emergency_access_grant_revision_review_due_check CHECK (review_due_at IS NOT NULL),
    CONSTRAINT emergency_access_grant_revision_revocation_check CHECK ((status = 'active' AND revoked_at IS NULL AND revoked_by IS NULL AND revocation_reason IS NULL) OR (status = 'revoked' AND revoked_at IS NOT NULL AND revoked_by IS NOT NULL AND btrim(COALESCE(revocation_reason, '')) <> '')),
    CONSTRAINT emergency_access_grant_revision_audit_reference_check CHECK (audit_reference <> '00000000-0000-0000-0000-000000000000')
);

CREATE TABLE identity.emergency_access_grant_permission (
    grant_id uuid NOT NULL,
    grant_version bigint NOT NULL,
    permission text NOT NULL,
    CONSTRAINT emergency_access_grant_permission_pk PRIMARY KEY (grant_id, grant_version, permission),
    CONSTRAINT emergency_access_grant_permission_revision_fk
        FOREIGN KEY (grant_id, grant_version)
        REFERENCES identity.emergency_access_grant_revision (grant_id, revision_version)
        ON DELETE RESTRICT,
    CONSTRAINT emergency_access_grant_permission_not_blank CHECK (btrim(permission) <> '')
);

CREATE TABLE identity.emergency_access_grant_scope (
    grant_id uuid NOT NULL,
    grant_version bigint NOT NULL,
    scope_id text NOT NULL,
    CONSTRAINT emergency_access_grant_scope_pk PRIMARY KEY (grant_id, grant_version, scope_id),
    CONSTRAINT emergency_access_grant_scope_revision_fk
        FOREIGN KEY (grant_id, grant_version)
        REFERENCES identity.emergency_access_grant_revision (grant_id, revision_version)
        ON DELETE RESTRICT,
    CONSTRAINT emergency_access_grant_scope_not_blank CHECK (btrim(scope_id) <> '')
);

ALTER TABLE identity.emergency_access_grant
    ADD CONSTRAINT emergency_access_grant_current_revision_fk
    FOREIGN KEY (id, current_version)
    REFERENCES identity.emergency_access_grant_revision (grant_id, revision_version)
    DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION identity.reject_emergency_access_grant_revision_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'identity emergency access grant revisions are immutable';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER emergency_access_grant_revision_immutable
BEFORE UPDATE OR DELETE ON identity.emergency_access_grant_revision
FOR EACH ROW EXECUTE FUNCTION identity.reject_emergency_access_grant_revision_mutation();

CREATE INDEX emergency_access_grant_status_idx
    ON identity.emergency_access_grant (status);

CREATE INDEX emergency_access_grant_revision_expiry_idx
    ON identity.emergency_access_grant_revision (status, expires_at);

CREATE INDEX emergency_access_grant_revision_target_idx
    ON identity.emergency_access_grant_revision (target_actor_id, status);

-- +goose Down
ALTER TABLE identity.emergency_access_grant
    DROP CONSTRAINT IF EXISTS emergency_access_grant_current_revision_fk;

DROP TRIGGER IF EXISTS emergency_access_grant_revision_immutable ON identity.emergency_access_grant_revision;
DROP FUNCTION IF EXISTS identity.reject_emergency_access_grant_revision_mutation();
DROP TABLE IF EXISTS identity.emergency_access_grant_scope;
DROP TABLE IF EXISTS identity.emergency_access_grant_permission;
DROP TABLE IF EXISTS identity.emergency_access_grant_revision;
DROP TABLE IF EXISTS identity.emergency_access_grant;
