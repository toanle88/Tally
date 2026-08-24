-- +goose Up
ALTER TABLE integration.outbox
    ADD COLUMN occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    ADD COLUMN data_classification text NOT NULL DEFAULT 'internal';

ALTER TABLE integration.outbox
    ADD CONSTRAINT integration_outbox_data_classification_check
        CHECK (data_classification IN ('public', 'internal', 'confidential', 'highly_restricted'));

-- +goose Down
ALTER TABLE integration.outbox
    DROP CONSTRAINT IF EXISTS integration_outbox_data_classification_check;

ALTER TABLE integration.outbox
    DROP COLUMN IF EXISTS data_classification,
    DROP COLUMN IF EXISTS occurred_at;
