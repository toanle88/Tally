-- +goose Up
CREATE TABLE invoicing.billing_schedule (
    billing_schedule_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    contract_reference text NOT NULL,
    charges jsonb NOT NULL,
    next_billing_date date NOT NULL,
    time_zone text NOT NULL,
    status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT billing_schedule_pk PRIMARY KEY (billing_schedule_id),
    CONSTRAINT billing_schedule_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT billing_schedule_revision_check CHECK (revision_number >= 1),
    CONSTRAINT billing_schedule_contract_reference_check CHECK (btrim(contract_reference) <> ''),
    CONSTRAINT billing_schedule_charges_array_check CHECK (jsonb_typeof(charges) = 'array' AND jsonb_array_length(charges) > 0),
    CONSTRAINT billing_schedule_time_zone_check CHECK (btrim(time_zone) <> ''),
    CONSTRAINT billing_schedule_status_check CHECK (status = 'configured'),
    CONSTRAINT billing_schedule_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT billing_schedule_next_date_check CHECK (next_billing_date >= effective_from AND (effective_to IS NULL OR next_billing_date <= effective_to))
);

CREATE INDEX billing_schedule_scope_next_date_idx
    ON invoicing.billing_schedule (accounting_scope_id, next_billing_date, billing_schedule_id);

CREATE INDEX billing_schedule_customer_contract_idx
    ON invoicing.billing_schedule (accounting_scope_id, customer_id, contract_reference, effective_from);

CREATE TABLE invoicing.billing_schedule_revision (
    billing_schedule_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    snapshot_fingerprint text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT billing_schedule_revision_pk PRIMARY KEY (billing_schedule_id, revision_number),
    CONSTRAINT billing_schedule_revision_fk FOREIGN KEY (billing_schedule_id) REFERENCES invoicing.billing_schedule (billing_schedule_id) ON DELETE CASCADE,
    CONSTRAINT billing_schedule_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT billing_schedule_revision_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT billing_schedule_revision_fingerprint_check CHECK (btrim(snapshot_fingerprint) <> '')
);

CREATE INDEX billing_schedule_revision_version_idx
    ON invoicing.billing_schedule_revision (billing_schedule_id, aggregate_version);

-- +goose Down
DROP TABLE IF EXISTS invoicing.billing_schedule_revision;
DROP TABLE IF EXISTS invoicing.billing_schedule;
