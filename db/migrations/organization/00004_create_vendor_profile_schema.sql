-- +goose Up
CREATE TABLE organization.vendor_profile (
    id uuid NOT NULL,
    scope_id uuid NOT NULL,
    party_id uuid NOT NULL,
    party_version bigint NOT NULL,
    payment_terms text NOT NULL,
    withholding_treatment text NOT NULL,
    remittance_preference text NOT NULL,
    status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    approval_reference jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT vendor_profile_pk PRIMARY KEY (id),
    CONSTRAINT vendor_profile_party_fk FOREIGN KEY (party_id) REFERENCES organization.party (id) ON DELETE RESTRICT,
    CONSTRAINT vendor_profile_status_check CHECK (status IN ('draft', 'active', 'end_dated')),
    CONSTRAINT vendor_profile_party_version_check CHECK (party_version >= 1),
    CONSTRAINT vendor_profile_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT vendor_profile_revision_check CHECK (revision_number >= 1),
    CONSTRAINT vendor_profile_payment_terms_check CHECK (payment_terms ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT vendor_profile_withholding_check CHECK (withholding_treatment ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT vendor_profile_remittance_check CHECK (remittance_preference ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT vendor_profile_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT vendor_profile_ended_date_check CHECK ((status = 'end_dated') = (effective_to IS NOT NULL))
);

CREATE UNIQUE INDEX vendor_profile_scope_party_unique
    ON organization.vendor_profile (scope_id, party_id);
CREATE INDEX vendor_profile_scope_status_idx
    ON organization.vendor_profile (scope_id, status);
CREATE INDEX vendor_profile_party_idx
    ON organization.vendor_profile (party_id);

CREATE TABLE organization.vendor_profile_revision (
    vendor_profile_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT vendor_profile_revision_pk PRIMARY KEY (vendor_profile_id, revision_number),
    CONSTRAINT vendor_profile_revision_profile_fk FOREIGN KEY (vendor_profile_id) REFERENCES organization.vendor_profile (id) ON DELETE CASCADE,
    CONSTRAINT vendor_profile_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT vendor_profile_revision_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX vendor_profile_revision_effective_idx
    ON organization.vendor_profile_revision (vendor_profile_id, effective_from);

-- +goose Down
DROP TABLE IF EXISTS organization.vendor_profile_revision;
DROP TABLE IF EXISTS organization.vendor_profile;
