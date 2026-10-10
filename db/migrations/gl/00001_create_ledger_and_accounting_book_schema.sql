-- +goose Up
CREATE SCHEMA IF NOT EXISTS gl;

CREATE TABLE gl.ledger (
    ledger_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    ledger_type text NOT NULL,
    functional_currency text NOT NULL,
    fiscal_calendar_id uuid NOT NULL,
    lifecycle_status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    approval_reference jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT ledger_pk PRIMARY KEY (ledger_id),
    CONSTRAINT ledger_scope_unique UNIQUE (ledger_id, accounting_scope_id),
    CONSTRAINT ledger_status_check CHECK (lifecycle_status IN ('draft', 'active', 'suspended', 'retired')),
    CONSTRAINT ledger_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT ledger_revision_check CHECK (revision_number >= 1),
    CONSTRAINT ledger_type_check CHECK (btrim(ledger_type) <> ''),
    CONSTRAINT ledger_currency_check CHECK (functional_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT ledger_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE UNIQUE INDEX ledger_effective_identity_unique
    ON gl.ledger (accounting_scope_id, legal_entity_id, ledger_type, functional_currency, fiscal_calendar_id, effective_from, COALESCE(effective_to, DATE '9999-12-31'));

CREATE INDEX ledger_scope_status_idx
    ON gl.ledger (accounting_scope_id, lifecycle_status);

CREATE INDEX ledger_scope_identity_idx
    ON gl.ledger (accounting_scope_id, legal_entity_id, ledger_type, functional_currency, fiscal_calendar_id, effective_from);

CREATE TABLE gl.ledger_revision (
    ledger_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT ledger_revision_pk PRIMARY KEY (ledger_id, revision_number),
    CONSTRAINT ledger_revision_fk FOREIGN KEY (ledger_id) REFERENCES gl.ledger (ledger_id) ON DELETE CASCADE,
    CONSTRAINT ledger_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT ledger_revision_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX ledger_revision_effective_idx
    ON gl.ledger_revision (ledger_id, effective_from);

CREATE TABLE gl.accounting_book (
    accounting_book_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    ledger_id uuid NOT NULL,
    book_type text NOT NULL,
    accounting_basis text NOT NULL,
    posting_policy_version text NOT NULL,
    lifecycle_status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    approval_reference jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT accounting_book_pk PRIMARY KEY (accounting_book_id),
    CONSTRAINT accounting_book_ledger_fk FOREIGN KEY (ledger_id, accounting_scope_id) REFERENCES gl.ledger (ledger_id, accounting_scope_id) ON DELETE RESTRICT,
    CONSTRAINT accounting_book_status_check CHECK (lifecycle_status IN ('draft', 'active', 'suspended', 'retired')),
    CONSTRAINT accounting_book_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT accounting_book_revision_check CHECK (revision_number >= 1),
    CONSTRAINT accounting_book_type_check CHECK (btrim(book_type) <> '' AND btrim(accounting_basis) <> '' AND btrim(posting_policy_version) <> ''),
    CONSTRAINT accounting_book_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE UNIQUE INDEX accounting_book_effective_identity_unique
    ON gl.accounting_book (accounting_scope_id, ledger_id, book_type, accounting_basis, effective_from, COALESCE(effective_to, DATE '9999-12-31'));

CREATE INDEX accounting_book_scope_status_idx
    ON gl.accounting_book (accounting_scope_id, lifecycle_status);

CREATE INDEX accounting_book_ledger_idx
    ON gl.accounting_book (ledger_id, book_type, accounting_basis, effective_from);

CREATE TABLE gl.accounting_book_revision (
    accounting_book_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT accounting_book_revision_pk PRIMARY KEY (accounting_book_id, revision_number),
    CONSTRAINT accounting_book_revision_fk FOREIGN KEY (accounting_book_id) REFERENCES gl.accounting_book (accounting_book_id) ON DELETE CASCADE,
    CONSTRAINT accounting_book_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT accounting_book_revision_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX accounting_book_revision_effective_idx
    ON gl.accounting_book_revision (accounting_book_id, effective_from);

-- +goose Down
DROP SCHEMA IF EXISTS gl CASCADE;
