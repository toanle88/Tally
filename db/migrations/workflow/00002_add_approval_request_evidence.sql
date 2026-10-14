-- +goose Up
ALTER TABLE workflow.approval_request
    ADD COLUMN subject_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN snapshot_fingerprint text NOT NULL DEFAULT '',
    ADD COLUMN source_context text NOT NULL DEFAULT '',
    ADD COLUMN operation text NOT NULL DEFAULT '',
    ADD COLUMN transaction_type text NOT NULL DEFAULT '',
    ADD COLUMN data_classification text NOT NULL DEFAULT '',
    ADD COLUMN dimensions jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN amount numeric,
    ADD COLUMN currency text NOT NULL DEFAULT '',
    ADD COLUMN due_at timestamptz,
    ADD COLUMN responsible_roles jsonb NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE workflow.approval_request
    ADD CONSTRAINT approval_request_subject_snapshot_object CHECK (jsonb_typeof(subject_snapshot) = 'object'),
    ADD CONSTRAINT approval_request_dimensions_object CHECK (jsonb_typeof(dimensions) = 'object'),
    ADD CONSTRAINT approval_request_responsible_roles_array CHECK (jsonb_typeof(responsible_roles) = 'array'),
    ADD CONSTRAINT approval_request_amount_check CHECK (amount IS NULL OR amount >= 0),
    ADD CONSTRAINT approval_request_currency_check CHECK (currency = '' OR currency ~ '^[A-Z]{3}$');

CREATE INDEX approval_request_status_idx ON workflow.approval_request (status, created_at DESC);
CREATE INDEX approval_request_scope_idx ON workflow.approval_request USING gin ((scope -> 'scopeIds'));
CREATE INDEX approval_request_due_idx ON workflow.approval_request (due_at) WHERE due_at IS NOT NULL;
CREATE INDEX approval_request_subject_idx ON workflow.approval_request (subject_type, subject_id, subject_version);

-- +goose Down
DROP INDEX IF EXISTS workflow.approval_request_subject_idx;
DROP INDEX IF EXISTS workflow.approval_request_due_idx;
DROP INDEX IF EXISTS workflow.approval_request_scope_idx;
DROP INDEX IF EXISTS workflow.approval_request_status_idx;
ALTER TABLE workflow.approval_request
    DROP CONSTRAINT IF EXISTS approval_request_currency_check,
    DROP CONSTRAINT IF EXISTS approval_request_amount_check,
    DROP CONSTRAINT IF EXISTS approval_request_responsible_roles_array,
    DROP CONSTRAINT IF EXISTS approval_request_dimensions_object,
    DROP CONSTRAINT IF EXISTS approval_request_subject_snapshot_object,
    DROP COLUMN IF EXISTS responsible_roles,
    DROP COLUMN IF EXISTS due_at,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS amount,
    DROP COLUMN IF EXISTS dimensions,
    DROP COLUMN IF EXISTS data_classification,
    DROP COLUMN IF EXISTS transaction_type,
    DROP COLUMN IF EXISTS operation,
    DROP COLUMN IF EXISTS source_context,
    DROP COLUMN IF EXISTS snapshot_fingerprint,
    DROP COLUMN IF EXISTS subject_snapshot;
