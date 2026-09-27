-- +goose Up
CREATE TABLE organization.customer_profile (
    id uuid NOT NULL,
    scope_id uuid NOT NULL,
    party_id uuid NOT NULL,
    party_version bigint NOT NULL,
    credit_terms text NOT NULL,
    credit_limit_amount numeric(38,12) NOT NULL,
    credit_limit_currency char(3) NOT NULL,
    billing_preference text NOT NULL,
    tax_treatment text NOT NULL,
    status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    approval_reference jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT customer_profile_pk PRIMARY KEY (id),
    CONSTRAINT customer_profile_party_fk FOREIGN KEY (party_id) REFERENCES organization.party (id) ON DELETE RESTRICT,
    CONSTRAINT customer_profile_status_check CHECK (status IN ('draft', 'active', 'end_dated')),
    CONSTRAINT customer_profile_party_version_check CHECK (party_version >= 1),
    CONSTRAINT customer_profile_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT customer_profile_revision_check CHECK (revision_number >= 1),
    CONSTRAINT customer_profile_terms_check CHECK (credit_terms ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT customer_profile_limit_check CHECK (credit_limit_amount >= 0),
    CONSTRAINT customer_profile_currency_check CHECK (credit_limit_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT customer_profile_billing_check CHECK (billing_preference ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT customer_profile_tax_check CHECK (tax_treatment ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT customer_profile_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT customer_profile_ended_date_check CHECK ((status = 'end_dated') = (effective_to IS NOT NULL))
);

CREATE UNIQUE INDEX customer_profile_scope_party_unique
    ON organization.customer_profile (scope_id, party_id);
CREATE INDEX customer_profile_scope_status_idx
    ON organization.customer_profile (scope_id, status);
CREATE INDEX customer_profile_party_idx
    ON organization.customer_profile (party_id);

CREATE TABLE organization.customer_profile_revision (
    customer_profile_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT customer_profile_revision_pk PRIMARY KEY (customer_profile_id, revision_number),
    CONSTRAINT customer_profile_revision_profile_fk FOREIGN KEY (customer_profile_id) REFERENCES organization.customer_profile (id) ON DELETE CASCADE,
    CONSTRAINT customer_profile_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT customer_profile_revision_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX customer_profile_revision_effective_idx
    ON organization.customer_profile_revision (customer_profile_id, effective_from);

-- +goose Down
DROP TABLE IF EXISTS organization.customer_profile_revision;
DROP TABLE IF EXISTS organization.customer_profile;
