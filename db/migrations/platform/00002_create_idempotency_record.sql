-- +goose Up
CREATE TABLE platform.idempotency_record (
    scope_key text NOT NULL,
    idempotency_key text NOT NULL,
    canonical_fingerprint text NOT NULL,
    operation_id text NOT NULL,
    state text NOT NULL,
    result_status integer,
    result_body jsonb,
    aggregate_id uuid,
    process_id uuid,
    owner_token uuid NOT NULL,
    lease_until timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    expires_at timestamptz NOT NULL,

    CONSTRAINT idempotency_record_pk
        PRIMARY KEY (scope_key, idempotency_key),
    CONSTRAINT idempotency_record_state_check
        CHECK (state IN ('in_progress', 'established', 'failed')),
    CONSTRAINT idempotency_record_scope_key_not_blank
        CHECK (btrim(scope_key) <> ''),
    CONSTRAINT idempotency_record_key_not_blank
        CHECK (btrim(idempotency_key) <> ''),
    CONSTRAINT idempotency_record_fingerprint_not_blank
        CHECK (btrim(canonical_fingerprint) <> ''),
    CONSTRAINT idempotency_record_operation_id_not_blank
        CHECK (btrim(operation_id) <> '')
);

CREATE INDEX idempotency_record_expires_at_idx
    ON platform.idempotency_record (expires_at);

-- +goose Down
DROP TABLE IF EXISTS platform.idempotency_record;
