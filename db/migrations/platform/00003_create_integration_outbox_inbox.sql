-- +goose Up
CREATE SCHEMA IF NOT EXISTS integration;

CREATE TABLE integration.outbox (
    outbox_id uuid NOT NULL,
    event_type text NOT NULL,
    event_version integer NOT NULL,
    source_context text NOT NULL,
    aggregate_id uuid NOT NULL,
    aggregate_version bigint NOT NULL,
    accounting_scope_id uuid,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    payload jsonb NOT NULL,
    payload_fingerprint text NOT NULL,
    available_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    claimed_until timestamptz,
    claim_owner text,
    attempt_count integer NOT NULL DEFAULT 0,
    established_at timestamptz,
    last_error_code text,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),

    CONSTRAINT integration_outbox_pk
        PRIMARY KEY (outbox_id),
    CONSTRAINT integration_outbox_source_event_unique
        UNIQUE (source_context, aggregate_id, aggregate_version, event_type)
);

CREATE INDEX integration_outbox_due_idx
    ON integration.outbox (available_at, outbox_id)
    WHERE established_at IS NULL;

CREATE TABLE integration.inbox (
    consumer_name text NOT NULL,
    message_id uuid NOT NULL,
    message_fingerprint text NOT NULL,
    state text NOT NULL,
    result_reference jsonb,
    first_received_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    established_at timestamptz,

    CONSTRAINT integration_inbox_pk
        PRIMARY KEY (consumer_name, message_id),
    CONSTRAINT integration_inbox_state_check
        CHECK (state IN ('processing', 'established', 'failed'))
);

-- +goose Down
DROP TABLE IF EXISTS integration.inbox;
DROP TABLE IF EXISTS integration.outbox;
DROP SCHEMA IF EXISTS integration;
