-- +goose Up
-- Story 7 adds the GL-owned hard-close barrier transfer and finalization
-- evidence. Existing soft-close rows remain open until a command establishes
-- the new fields.
ALTER TABLE gl.period_posting_gate
    ADD COLUMN active_close_run_id uuid,
    ADD COLUMN active_close_type text,
    ADD COLUMN prior_control_process_id uuid,
    ADD COLUMN prior_control_process_type text,
    ADD COLUMN prior_control_epoch bigint,
    ADD COLUMN prior_gate_mode text,
    ADD COLUMN barrier_ledger_position bigint NOT NULL DEFAULT 0,
    ADD COLUMN final_ledger_watermark bigint,
    ADD COLUMN finalized_close_run_id uuid,
    ADD COLUMN finalization_command_fingerprint text;

ALTER TABLE gl.period_posting_gate
    ADD CONSTRAINT period_posting_gate_active_close_type_check
        CHECK (active_close_type IS NULL OR active_close_type IN ('InitialHardClose', 'Reclose')),
    ADD CONSTRAINT period_posting_gate_prior_control_epoch_check
        CHECK (prior_control_epoch IS NULL OR prior_control_epoch >= 1),
    ADD CONSTRAINT period_posting_gate_barrier_position_check
        CHECK (barrier_ledger_position >= 0),
    ADD CONSTRAINT period_posting_gate_final_watermark_check
        CHECK (final_ledger_watermark IS NULL OR final_ledger_watermark >= 0);

CREATE INDEX period_posting_gate_active_close_owner_idx
    ON gl.period_posting_gate (accounting_scope_id, active_close_run_id)
    WHERE active_close_run_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS gl.period_posting_gate_active_close_owner_idx;
ALTER TABLE gl.period_posting_gate
    DROP CONSTRAINT IF EXISTS period_posting_gate_final_watermark_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_barrier_position_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_prior_control_epoch_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_active_close_type_check,
    DROP COLUMN IF EXISTS finalization_command_fingerprint,
    DROP COLUMN IF EXISTS finalized_close_run_id,
    DROP COLUMN IF EXISTS final_ledger_watermark,
    DROP COLUMN IF EXISTS barrier_ledger_position,
    DROP COLUMN IF EXISTS prior_gate_mode,
    DROP COLUMN IF EXISTS prior_control_epoch,
    DROP COLUMN IF EXISTS prior_control_process_type,
    DROP COLUMN IF EXISTS prior_control_process_id,
    DROP COLUMN IF EXISTS active_close_type,
    DROP COLUMN IF EXISTS active_close_run_id;
