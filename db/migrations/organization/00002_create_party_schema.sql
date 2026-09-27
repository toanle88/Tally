-- +goose Up
CREATE TABLE organization.party (
    id uuid NOT NULL,
    scope_id uuid NOT NULL,
    name text NOT NULL,
    party_type text NOT NULL,
    status text NOT NULL,
    tax_identifier text,
    bank_control jsonb NOT NULL DEFAULT '{"status":"not-required"}'::jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT party_pk PRIMARY KEY (id),
    CONSTRAINT party_name_check CHECK (btrim(name) <> ''),
    CONSTRAINT party_type_check CHECK (party_type ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT party_status_check CHECK (status ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT party_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT party_revision_check CHECK (revision_number >= 1),
    CONSTRAINT party_bank_control_object_check CHECK (jsonb_typeof(bank_control) = 'object')
);

CREATE UNIQUE INDEX party_scope_name_unique ON organization.party (scope_id, lower(name));
CREATE INDEX party_scope_status_idx ON organization.party (scope_id, status);

CREATE TABLE organization.party_contact_method (
    id uuid NOT NULL,
    party_id uuid NOT NULL,
    contact_type text NOT NULL,
    contact_value text NOT NULL,
    label text,
    CONSTRAINT party_contact_method_pk PRIMARY KEY (id),
    CONSTRAINT party_contact_method_party_fk FOREIGN KEY (party_id) REFERENCES organization.party (id) ON DELETE CASCADE,
    CONSTRAINT party_contact_method_value_check CHECK (btrim(contact_type) <> '' AND btrim(contact_value) <> '')
);
CREATE INDEX party_contact_method_party_idx ON organization.party_contact_method (party_id);

CREATE TABLE organization.party_address (
    id uuid NOT NULL,
    party_id uuid NOT NULL,
    address_type text NOT NULL,
    line1 text NOT NULL,
    line2 text,
    locality text NOT NULL,
    region text,
    postal_code text NOT NULL,
    country_code text NOT NULL,
    CONSTRAINT party_address_pk PRIMARY KEY (id),
    CONSTRAINT party_address_party_fk FOREIGN KEY (party_id) REFERENCES organization.party (id) ON DELETE CASCADE,
    CONSTRAINT party_address_value_check CHECK (btrim(address_type) <> '' AND btrim(line1) <> '' AND btrim(locality) <> '' AND btrim(postal_code) <> '' AND country_code ~ '^[A-Z]{2,3}$')
);
CREATE INDEX party_address_party_idx ON organization.party_address (party_id);

CREATE TABLE organization.party_classification (
    id uuid NOT NULL,
    party_id uuid NOT NULL,
    classification_code text NOT NULL,
    classification_value text NOT NULL,
    CONSTRAINT party_classification_pk PRIMARY KEY (id),
    CONSTRAINT party_classification_party_fk FOREIGN KEY (party_id) REFERENCES organization.party (id) ON DELETE CASCADE,
    CONSTRAINT party_classification_value_check CHECK (classification_code ~ '^[a-z0-9][a-z0-9_-]{0,63}$' AND btrim(classification_value) <> '')
);
CREATE INDEX party_classification_party_idx ON organization.party_classification (party_id);

CREATE TABLE organization.party_bank_detail_reference (
    id uuid NOT NULL,
    party_id uuid NOT NULL,
    reference text NOT NULL,
    provider_code text,
    consent_reference text,
    CONSTRAINT party_bank_reference_pk PRIMARY KEY (id),
    CONSTRAINT party_bank_reference_party_fk FOREIGN KEY (party_id) REFERENCES organization.party (id) ON DELETE CASCADE,
    CONSTRAINT party_bank_reference_opaque_check CHECK (reference ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{2,159}$' AND reference !~ '^[0-9][0-9._:-]*$' AND reference !~* '(account|accountnumber|account_number|iban|routing|swift|credential|password|provider[-_]?token|secret|raw-account)'),
    CONSTRAINT party_bank_reference_provider_check CHECK (provider_code IS NULL OR (provider_code ~ '^[a-z0-9][a-z0-9_-]{0,63}$' AND provider_code !~* '(account|accountnumber|iban|routing|swift|credential|password|provider[-_]?token|secret)')),
    CONSTRAINT party_bank_reference_consent_check CHECK (consent_reference IS NULL OR (consent_reference ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{2,159}$' AND consent_reference !~ '^[0-9][0-9._:-]*$' AND consent_reference !~* '(account|accountnumber|account_number|iban|routing|swift|credential|password|provider[-_]?token|secret|raw-account)'))
);
CREATE UNIQUE INDEX party_bank_reference_value_unique ON organization.party_bank_detail_reference (party_id, lower(reference));
CREATE INDEX party_bank_reference_party_idx ON organization.party_bank_detail_reference (party_id);

CREATE TABLE organization.party_revision (
    party_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT party_revision_pk PRIMARY KEY (party_id, revision_number),
    CONSTRAINT party_revision_party_fk FOREIGN KEY (party_id) REFERENCES organization.party (id) ON DELETE CASCADE,
    CONSTRAINT party_revision_version_check CHECK (aggregate_version >= 1)
);
CREATE INDEX party_revision_created_idx ON organization.party_revision (party_id, revision_number);

-- +goose Down
DROP TABLE IF EXISTS organization.party_revision;
DROP TABLE IF EXISTS organization.party_bank_detail_reference;
DROP TABLE IF EXISTS organization.party_classification;
DROP TABLE IF EXISTS organization.party_address;
DROP TABLE IF EXISTS organization.party_contact_method;
DROP TABLE IF EXISTS organization.party;
