-- +goose Up
CREATE SCHEMA IF NOT EXISTS identity;

CREATE TABLE identity.user_account (
    id uuid NOT NULL,
    authentication_subject_oid text NOT NULL,
    authentication_subject_tid text NOT NULL,
    authentication_subject_sub text NOT NULL,
    status text NOT NULL,
    aggregate_version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,

    CONSTRAINT user_account_pk PRIMARY KEY (id),
    CONSTRAINT user_account_status_check
        CHECK (status IN ('inactive', 'active', 'suspended', 'terminated')),
    CONSTRAINT user_account_version_check
        CHECK (aggregate_version >= 1),
    CONSTRAINT user_account_subject_unique
        UNIQUE (
            authentication_subject_oid,
            authentication_subject_tid,
            authentication_subject_sub
        ),
    CONSTRAINT user_account_subject_oid_not_blank
        CHECK (btrim(authentication_subject_oid) <> ''),
    CONSTRAINT user_account_subject_tid_not_blank
        CHECK (btrim(authentication_subject_tid) <> ''),
    CONSTRAINT user_account_subject_sub_not_blank
        CHECK (btrim(authentication_subject_sub) <> '')
);

CREATE TABLE identity.user_role_assignment (
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,

    CONSTRAINT user_role_assignment_pk PRIMARY KEY (user_id, role_id),
    CONSTRAINT user_role_assignment_user_fk
        FOREIGN KEY (user_id)
        REFERENCES identity.user_account (id)
        ON DELETE CASCADE
);

CREATE TABLE identity.user_role_assignment_scope (
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    scope_id text NOT NULL,

    CONSTRAINT user_role_assignment_scope_pk
        PRIMARY KEY (user_id, role_id, scope_id),
    CONSTRAINT user_role_assignment_scope_assignment_fk
        FOREIGN KEY (user_id, role_id)
        REFERENCES identity.user_role_assignment (user_id, role_id)
        ON DELETE CASCADE,
    CONSTRAINT user_role_assignment_scope_not_blank
        CHECK (btrim(scope_id) <> '')
);

CREATE INDEX user_account_status_idx
    ON identity.user_account (status);

CREATE INDEX user_role_assignment_scope_scope_id_idx
    ON identity.user_role_assignment_scope (scope_id);

-- +goose Down
DROP SCHEMA IF EXISTS identity CASCADE;
