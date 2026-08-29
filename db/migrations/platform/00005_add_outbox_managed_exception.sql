-- +goose Up
ALTER TABLE integration.outbox
    ADD COLUMN managed_exception_at timestamptz;

ALTER TABLE integration.outbox
    ADD CONSTRAINT integration_outbox_established_managed_check
        CHECK (NOT (established_at IS NOT NULL AND managed_exception_at IS NOT NULL));

DROP INDEX integration.integration_outbox_due_idx;

CREATE INDEX integration_outbox_due_idx
    ON integration.outbox (available_at, outbox_id)
    WHERE established_at IS NULL AND managed_exception_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS integration.integration_outbox_due_idx;

CREATE INDEX integration_outbox_due_idx
    ON integration.outbox (available_at, outbox_id)
    WHERE established_at IS NULL;

ALTER TABLE integration.outbox
    DROP CONSTRAINT IF EXISTS integration_outbox_established_managed_check;

ALTER TABLE integration.outbox
    DROP COLUMN IF EXISTS managed_exception_at;
