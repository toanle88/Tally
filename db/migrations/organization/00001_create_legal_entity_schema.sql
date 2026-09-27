-- +goose Up
CREATE SCHEMA IF NOT EXISTS organization;

CREATE TABLE organization.legal_entity (
    id uuid NOT NULL,
    scope_id uuid NOT NULL,
    legal_name text NOT NULL,
    functional_currency char(3) NOT NULL,
    presentation_currency char(3) NOT NULL,
    tax_registration_id text,
    status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    approval_reference jsonb,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT legal_entity_pk PRIMARY KEY (id),
    CONSTRAINT legal_entity_status_check CHECK (status IN ('draft', 'active', 'end_dated')),
    CONSTRAINT legal_entity_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT legal_entity_revision_check CHECK (revision_number >= 1),
    CONSTRAINT legal_entity_name_check CHECK (btrim(legal_name) <> ''),
    CONSTRAINT legal_entity_currency_check CHECK (functional_currency ~ '^[A-Z]{3}$' AND presentation_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT legal_entity_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT legal_entity_ended_date_check CHECK ((status = 'end_dated') = (effective_to IS NOT NULL))
);

CREATE UNIQUE INDEX legal_entity_scope_name_active_unique
    ON organization.legal_entity (scope_id, lower(legal_name))
    WHERE status <> 'end_dated';

CREATE TABLE organization.legal_entity_registration (
    id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    registration_type text NOT NULL,
    identifier text NOT NULL,
    jurisdiction text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    CONSTRAINT legal_entity_registration_pk PRIMARY KEY (id),
    CONSTRAINT legal_entity_registration_entity_fk FOREIGN KEY (legal_entity_id) REFERENCES organization.legal_entity (id) ON DELETE CASCADE,
    CONSTRAINT legal_entity_registration_value_check CHECK (btrim(registration_type) <> '' AND btrim(identifier) <> '' AND jurisdiction ~ '^[A-Z]{2,3}$'),
    CONSTRAINT legal_entity_registration_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX legal_entity_registration_entity_idx ON organization.legal_entity_registration (legal_entity_id);

CREATE TABLE organization.legal_entity_address (
    id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    address_type text NOT NULL,
    line1 text NOT NULL,
    line2 text,
    locality text NOT NULL,
    region text,
    postal_code text NOT NULL,
    country_code text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    CONSTRAINT legal_entity_address_pk PRIMARY KEY (id),
    CONSTRAINT legal_entity_address_entity_fk FOREIGN KEY (legal_entity_id) REFERENCES organization.legal_entity (id) ON DELETE CASCADE,
    CONSTRAINT legal_entity_address_value_check CHECK (btrim(address_type) <> '' AND btrim(line1) <> '' AND btrim(locality) <> '' AND btrim(postal_code) <> '' AND country_code ~ '^[A-Z]{2,3}$'),
    CONSTRAINT legal_entity_address_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX legal_entity_address_entity_idx ON organization.legal_entity_address (legal_entity_id);

CREATE TABLE organization.legal_entity_ownership_interest (
    id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    owner_reference text NOT NULL,
    percentage numeric(9,6) NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    CONSTRAINT legal_entity_ownership_pk PRIMARY KEY (id),
    CONSTRAINT legal_entity_ownership_entity_fk FOREIGN KEY (legal_entity_id) REFERENCES organization.legal_entity (id) ON DELETE CASCADE,
    CONSTRAINT legal_entity_ownership_value_check CHECK (btrim(owner_reference) <> '' AND percentage > 0 AND percentage <= 100),
    CONSTRAINT legal_entity_ownership_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX legal_entity_ownership_entity_idx ON organization.legal_entity_ownership_interest (legal_entity_id);

CREATE TABLE organization.legal_entity_revision (
    legal_entity_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT legal_entity_revision_pk PRIMARY KEY (legal_entity_id, revision_number),
    CONSTRAINT legal_entity_revision_entity_fk FOREIGN KEY (legal_entity_id) REFERENCES organization.legal_entity (id) ON DELETE CASCADE,
    CONSTRAINT legal_entity_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT legal_entity_revision_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX legal_entity_scope_status_idx ON organization.legal_entity (scope_id, status);
CREATE INDEX legal_entity_revision_effective_idx ON organization.legal_entity_revision (legal_entity_id, effective_from);

-- +goose Down
DROP SCHEMA IF EXISTS organization CASCADE;
