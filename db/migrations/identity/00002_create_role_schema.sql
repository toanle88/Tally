-- +goose Up
CREATE TABLE identity.role (
    id uuid NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    aggregate_version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,

    CONSTRAINT role_pk PRIMARY KEY (id),
    CONSTRAINT role_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT role_status_check CHECK (status IN ('active', 'retired')),
    CONSTRAINT role_version_check CHECK (aggregate_version >= 1)
);

CREATE TABLE identity.role_revision (
    role_id uuid NOT NULL,
    revision_version bigint NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    approval_request_id uuid NOT NULL,
    approval_decision_id uuid NOT NULL,
    approver_user_id uuid NOT NULL,
    policy_version text NOT NULL,
    decision_version bigint NOT NULL,
    subject_version bigint NOT NULL,
    candidate_fingerprint text NOT NULL,
    audit_reference uuid NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT role_revision_pk PRIMARY KEY (role_id, revision_version),
    CONSTRAINT role_revision_role_fk
        FOREIGN KEY (role_id)
        REFERENCES identity.role (id)
        ON DELETE RESTRICT,
    CONSTRAINT role_revision_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT role_revision_status_check CHECK (status IN ('active', 'retired')),
    CONSTRAINT role_revision_version_check CHECK (revision_version >= 1),
    CONSTRAINT role_revision_policy_version_not_blank CHECK (btrim(policy_version) <> ''),
    CONSTRAINT role_revision_decision_version_check CHECK (decision_version >= 1),
    CONSTRAINT role_revision_subject_version_check CHECK (subject_version >= 1),
    CONSTRAINT role_revision_fingerprint_not_blank CHECK (btrim(candidate_fingerprint) <> ''),
    CONSTRAINT role_revision_audit_reference_not_nil CHECK (audit_reference <> '00000000-0000-0000-0000-000000000000')
);

CREATE TABLE identity.role_permission_grant (
    role_id uuid NOT NULL,
    role_version bigint NOT NULL,
    permission text NOT NULL,
    scope_id text NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,

    CONSTRAINT role_permission_grant_pk
        PRIMARY KEY (role_id, role_version, permission, scope_id, effective_from),
    CONSTRAINT role_permission_grant_revision_fk
        FOREIGN KEY (role_id, role_version)
        REFERENCES identity.role_revision (role_id, revision_version)
        ON DELETE RESTRICT,
    CONSTRAINT role_permission_grant_permission_not_blank
        CHECK (btrim(permission) <> ''),
    CONSTRAINT role_permission_grant_scope_not_blank
        CHECK (btrim(scope_id) <> ''),
    CONSTRAINT role_permission_grant_date_check
        CHECK (effective_to IS NULL OR effective_to > effective_from)
);

ALTER TABLE identity.user_role_assignment
    ADD CONSTRAINT user_role_assignment_role_fk
    FOREIGN KEY (role_id)
    REFERENCES identity.role (id)
    ON DELETE RESTRICT;

CREATE INDEX role_status_idx
    ON identity.role (status);

CREATE INDEX role_revision_current_idx
    ON identity.role_revision (role_id, revision_version DESC);

CREATE INDEX role_permission_grant_permission_scope_idx
    ON identity.role_permission_grant (permission, scope_id);

-- +goose Down
ALTER TABLE identity.user_role_assignment
    DROP CONSTRAINT IF EXISTS user_role_assignment_role_fk;

DROP TABLE IF EXISTS identity.role_permission_grant;
DROP TABLE IF EXISTS identity.role_revision;
DROP TABLE IF EXISTS identity.role;
