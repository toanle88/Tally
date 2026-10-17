-- +goose Up
CREATE SCHEMA IF NOT EXISTS invoicing;

CREATE TABLE invoicing.invoice_template (
    invoice_template_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    billing_frequency text NOT NULL,
    payment_method jsonb NOT NULL,
    applicability jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT invoice_template_pk PRIMARY KEY (invoice_template_id),
    CONSTRAINT invoice_template_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT invoice_template_revision_check CHECK (revision_number >= 1),
    CONSTRAINT invoice_template_code_check CHECK (btrim(code) <> ''),
    CONSTRAINT invoice_template_name_check CHECK (btrim(name) <> ''),
    CONSTRAINT invoice_template_frequency_check CHECK (billing_frequency IN ('one_time', 'daily', 'weekly', 'monthly', 'quarterly', 'annual')),
    CONSTRAINT invoice_template_payment_method_object_check CHECK (jsonb_typeof(payment_method) = 'object'),
    CONSTRAINT invoice_template_applicability_object_check CHECK (jsonb_typeof(applicability) = 'object'),
    CONSTRAINT invoice_template_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX invoice_template_scope_code_idx
    ON invoicing.invoice_template (accounting_scope_id, code, effective_from);

CREATE TABLE invoicing.invoice_template_line_item (
    invoice_template_id uuid NOT NULL,
    line_item_id uuid NOT NULL,
    line_order integer NOT NULL,
    code text NOT NULL,
    description text NOT NULL,
    charge_rule jsonb NOT NULL,
    tax_category jsonb NOT NULL,
    CONSTRAINT invoice_template_line_item_pk PRIMARY KEY (invoice_template_id, line_item_id),
    CONSTRAINT invoice_template_line_item_fk FOREIGN KEY (invoice_template_id) REFERENCES invoicing.invoice_template (invoice_template_id) ON DELETE CASCADE,
    CONSTRAINT invoice_template_line_item_order_check CHECK (line_order >= 0),
    CONSTRAINT invoice_template_line_item_code_check CHECK (btrim(code) <> ''),
    CONSTRAINT invoice_template_line_item_description_check CHECK (btrim(description) <> ''),
    CONSTRAINT invoice_template_line_item_charge_rule_object_check CHECK (jsonb_typeof(charge_rule) = 'object'),
    CONSTRAINT invoice_template_line_item_tax_category_object_check CHECK (jsonb_typeof(tax_category) = 'object')
);

CREATE INDEX invoice_template_line_item_order_idx
    ON invoicing.invoice_template_line_item (invoice_template_id, line_order);

CREATE TABLE invoicing.invoice_template_revision (
    invoice_template_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    snapshot_fingerprint text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT invoice_template_revision_pk PRIMARY KEY (invoice_template_id, revision_number),
    CONSTRAINT invoice_template_revision_fk FOREIGN KEY (invoice_template_id) REFERENCES invoicing.invoice_template (invoice_template_id) ON DELETE CASCADE,
    CONSTRAINT invoice_template_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT invoice_template_revision_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT invoice_template_revision_fingerprint_check CHECK (btrim(snapshot_fingerprint) <> '')
);

CREATE INDEX invoice_template_revision_version_idx
    ON invoicing.invoice_template_revision (invoice_template_id, aggregate_version);

-- +goose Down
DROP TABLE IF EXISTS invoicing.invoice_template_revision;
DROP TABLE IF EXISTS invoicing.invoice_template_line_item;
DROP TABLE IF EXISTS invoicing.invoice_template;
