-- +goose Up
CREATE SCHEMA IF NOT EXISTS coa;

CREATE TABLE coa.segment_definition (
    segment_definition_id uuid NOT NULL,
    scope_id uuid NOT NULL,
    segment_type text NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT segment_definition_pk PRIMARY KEY (segment_definition_id),
    CONSTRAINT segment_definition_status_check CHECK (status IN ('draft', 'active', 'suspended', 'retired')),
    CONSTRAINT segment_definition_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT segment_definition_revision_check CHECK (revision_number >= 1),
    CONSTRAINT segment_definition_type_check CHECK (btrim(segment_type) <> ''),
    CONSTRAINT segment_definition_code_check CHECK (btrim(code) <> ''),
    CONSTRAINT segment_definition_name_check CHECK (btrim(name) <> ''),
    CONSTRAINT segment_definition_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE UNIQUE INDEX segment_definition_exact_identity_unique
    ON coa.segment_definition (scope_id, segment_type, code, effective_from, COALESCE(effective_to, DATE '9999-12-31'));

CREATE INDEX segment_definition_scope_status_idx
    ON coa.segment_definition (scope_id, status);

CREATE INDEX segment_definition_scope_code_idx
    ON coa.segment_definition (scope_id, segment_type, code, effective_from);

CREATE TABLE coa.segment_definition_revision (
    segment_definition_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT segment_definition_revision_pk PRIMARY KEY (segment_definition_id, revision_number),
    CONSTRAINT segment_definition_revision_fk FOREIGN KEY (segment_definition_id) REFERENCES coa.segment_definition (segment_definition_id) ON DELETE CASCADE,
    CONSTRAINT segment_definition_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT segment_definition_revision_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX segment_definition_revision_effective_idx
    ON coa.segment_definition_revision (segment_definition_id, effective_from);

-- +goose Down
DROP SCHEMA IF EXISTS coa CASCADE;
