-- +goose Up
ALTER TABLE fiscal_period.soft_close_run
    ADD COLUMN control_authority jsonb,
    ADD CONSTRAINT soft_close_run_control_authority_check
        CHECK (control_authority IS NULL OR jsonb_typeof(control_authority) = 'object');

-- +goose Down
ALTER TABLE fiscal_period.soft_close_run
    DROP CONSTRAINT IF EXISTS soft_close_run_control_authority_check,
    DROP COLUMN IF EXISTS control_authority;
