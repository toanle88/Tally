-- +goose Up
CREATE TABLE identity.access_policy (
    id uuid NOT NULL,
    current_version text NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT access_policy_pk PRIMARY KEY (id),
    CONSTRAINT access_policy_current_version_not_blank CHECK (btrim(current_version) <> ''),
    CONSTRAINT access_policy_status_check CHECK (status IN ('active', 'retired'))
);

CREATE TABLE identity.access_policy_revision (
    policy_id uuid NOT NULL,
    policy_version text NOT NULL,
    status text NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    subject_actor_ids uuid[] NOT NULL DEFAULT '{}'::uuid[],
    subject_role_ids uuid[] NOT NULL DEFAULT '{}'::uuid[],
    permissions text[] NOT NULL,
    rules jsonb NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT access_policy_revision_pk PRIMARY KEY (policy_id, policy_version),
    CONSTRAINT access_policy_revision_policy_fk
        FOREIGN KEY (policy_id)
        REFERENCES identity.access_policy (id)
        ON DELETE RESTRICT,
    CONSTRAINT access_policy_revision_version_not_blank CHECK (btrim(policy_version) <> ''),
    CONSTRAINT access_policy_revision_status_check CHECK (status IN ('active', 'retired')),
    CONSTRAINT access_policy_revision_effective_date_check
        CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT access_policy_revision_permissions_not_empty CHECK (cardinality(permissions) > 0),
    CONSTRAINT access_policy_revision_permissions_not_blank CHECK (array_position(permissions, '') IS NULL),
    CONSTRAINT access_policy_revision_rules_array CHECK (jsonb_typeof(rules) = 'array')
);

ALTER TABLE identity.access_policy
    ADD CONSTRAINT access_policy_current_revision_fk
    FOREIGN KEY (id, current_version)
    REFERENCES identity.access_policy_revision (policy_id, policy_version)
    DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION identity.reject_access_policy_revision_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'identity access policy revisions are immutable';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER access_policy_revision_immutable
BEFORE UPDATE OR DELETE ON identity.access_policy_revision
FOR EACH ROW EXECUTE FUNCTION identity.reject_access_policy_revision_mutation();

CREATE INDEX access_policy_current_version_idx
    ON identity.access_policy (id, current_version);

CREATE INDEX access_policy_revision_effective_lookup_idx
    ON identity.access_policy_revision (status, effective_from, effective_to);

CREATE INDEX access_policy_revision_permissions_idx
    ON identity.access_policy_revision USING gin (permissions);

-- +goose Down
ALTER TABLE identity.access_policy
    DROP CONSTRAINT IF EXISTS access_policy_current_revision_fk;

DROP TRIGGER IF EXISTS access_policy_revision_immutable ON identity.access_policy_revision;
DROP FUNCTION IF EXISTS identity.reject_access_policy_revision_mutation();
DROP TABLE IF EXISTS identity.access_policy_revision;
DROP TABLE IF EXISTS identity.access_policy;
