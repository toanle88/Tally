-- +goose Up
CREATE TABLE organization.master_data_publication (
    publication_id uuid NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    scope_id uuid NOT NULL,
    aggregate_version bigint NOT NULL,
    revision_number bigint NOT NULL,
    status text NOT NULL,
    dependent_availability text NOT NULL,
    event_type text NOT NULL,
    event_version integer NOT NULL,
    message_id uuid NOT NULL,
    effective_from date,
    effective_to date,
    approval_reference jsonb,
    source_fingerprint text NOT NULL,
    audit_reference uuid NOT NULL,
    published_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT master_data_publication_pk PRIMARY KEY (publication_id),
    CONSTRAINT master_data_publication_source_unique UNIQUE (aggregate_type, aggregate_id, aggregate_version),
    CONSTRAINT master_data_publication_type_check CHECK (aggregate_type IN ('legal_entity', 'party', 'customer_profile', 'vendor_profile', 'fiscal_calendar')),
    CONSTRAINT master_data_publication_status_check CHECK (status IN ('pending_approval', 'approved', 'published', 'rejected', 'stale', 'unavailable')),
    CONSTRAINT master_data_publication_availability_check CHECK (dependent_availability IN ('pending', 'available', 'unavailable')),
    CONSTRAINT master_data_publication_version_check CHECK (aggregate_version >= 1 AND revision_number >= 1),
    CONSTRAINT master_data_publication_event_version_check CHECK (event_version = 1),
    CONSTRAINT master_data_publication_fingerprint_check CHECK (btrim(source_fingerprint) <> ''),
    CONSTRAINT master_data_publication_audit_check CHECK (audit_reference <> '00000000-0000-0000-0000-000000000000'),
    CONSTRAINT master_data_publication_effective_check CHECK (effective_to IS NULL OR (effective_from IS NOT NULL AND effective_to > effective_from))
);

CREATE UNIQUE INDEX master_data_publication_message_unique
    ON organization.master_data_publication (message_id);
CREATE INDEX master_data_publication_scope_status_idx
    ON organization.master_data_publication (scope_id, status, updated_at);
CREATE INDEX master_data_publication_aggregate_idx
    ON organization.master_data_publication (aggregate_type, aggregate_id, updated_at);

-- +goose Down
DROP TABLE IF EXISTS organization.master_data_publication;
