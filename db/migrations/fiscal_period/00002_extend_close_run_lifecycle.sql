-- +goose Up
ALTER TABLE fiscal_period.fiscal_period
    DROP CONSTRAINT fiscal_period_state_check,
    ADD CONSTRAINT fiscal_period_state_check CHECK (state IN ('Open', 'SoftClosed', 'Closing', 'HardClosed'));

ALTER TABLE fiscal_period.soft_close_run
    DROP CONSTRAINT soft_close_run_status_check,
    ADD CONSTRAINT soft_close_run_status_check CHECK (status IN ('Active', 'HandoffPending', 'Completed', 'Superseded'));

ALTER TABLE fiscal_period.close_run
    ADD COLUMN close_run_details jsonb;

ALTER TABLE fiscal_period.close_run
    DROP CONSTRAINT close_run_status_check,
    ADD CONSTRAINT close_run_status_check CHECK (status IN ('Initiating', 'BarrierAcquired', 'InProgress', 'AwaitingApproval', 'Finalizing', 'Completed', 'Failed', 'Aborting', 'Aborted')),
    ADD CONSTRAINT close_run_details_check CHECK (close_run_details IS NULL OR jsonb_typeof(close_run_details) = 'object');

DROP INDEX fiscal_period.close_run_active_period_unique;
CREATE UNIQUE INDEX close_run_active_period_unique
    ON fiscal_period.close_run (accounting_scope_id, fiscal_period_id)
    WHERE status IN ('Initiating', 'BarrierAcquired', 'InProgress', 'AwaitingApproval', 'Finalizing', 'Failed', 'Aborting');

-- +goose Down
DROP INDEX IF EXISTS fiscal_period.close_run_active_period_unique;
CREATE UNIQUE INDEX close_run_active_period_unique
    ON fiscal_period.close_run (accounting_scope_id, fiscal_period_id)
    WHERE status IN ('Initiating', 'BarrierAcquired', 'Failed');

ALTER TABLE fiscal_period.close_run
    DROP CONSTRAINT close_run_status_check,
    DROP CONSTRAINT close_run_details_check,
    ADD CONSTRAINT close_run_status_check CHECK (status IN ('Initiating', 'BarrierAcquired', 'Failed'));

ALTER TABLE fiscal_period.close_run
    DROP COLUMN close_run_details;

ALTER TABLE fiscal_period.soft_close_run
    DROP CONSTRAINT soft_close_run_status_check,
    ADD CONSTRAINT soft_close_run_status_check CHECK (status IN ('Active', 'HandoffPending'));

ALTER TABLE fiscal_period.fiscal_period
    DROP CONSTRAINT fiscal_period_state_check,
    ADD CONSTRAINT fiscal_period_state_check CHECK (state IN ('Open', 'SoftClosed', 'Closing'));
