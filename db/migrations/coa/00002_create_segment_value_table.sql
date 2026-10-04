-- +goose Up
CREATE TABLE coa.segment_value (
    segment_value_id uuid NOT NULL,
    segment_definition_id uuid NOT NULL,
    value text NOT NULL,
    description text NOT NULL DEFAULT '',
    status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT segment_value_pk PRIMARY KEY (segment_value_id),
    CONSTRAINT segment_value_definition_fk FOREIGN KEY (segment_definition_id) REFERENCES coa.segment_definition (segment_definition_id) ON DELETE CASCADE,
    CONSTRAINT segment_value_status_check CHECK (status IN ('draft', 'active', 'suspended', 'retired')),
    CONSTRAINT segment_value_value_check CHECK (btrim(value) <> ''),
    CONSTRAINT segment_value_value_canonical_check CHECK (value = btrim(value)),
    CONSTRAINT segment_value_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT segment_value_timestamp_check CHECK (updated_at >= created_at)
);

CREATE UNIQUE INDEX segment_value_exact_identity_unique
    ON coa.segment_value (segment_definition_id, value, effective_from, COALESCE(effective_to, DATE '9999-12-31'));

CREATE INDEX segment_value_definition_value_idx
    ON coa.segment_value (segment_definition_id, value, effective_from);

CREATE INDEX segment_value_definition_status_idx
    ON coa.segment_value (segment_definition_id, status);

-- +goose Down
DROP TABLE IF EXISTS coa.segment_value;
