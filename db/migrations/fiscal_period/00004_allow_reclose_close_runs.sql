-- +goose Up
ALTER TABLE fiscal_period.close_run
    DROP CONSTRAINT close_run_close_type_check,
    ADD CONSTRAINT close_run_close_type_check CHECK (close_type IN ('InitialHardClose', 'Reclose'));

-- +goose Down
ALTER TABLE fiscal_period.close_run
    DROP CONSTRAINT close_run_close_type_check,
    ADD CONSTRAINT close_run_close_type_check CHECK (close_type IN ('InitialHardClose'));
