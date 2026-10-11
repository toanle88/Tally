-- +goose Up
-- Story 6 adds the GL-owned state required to establish and release a
-- soft-close owner. Existing rows receive a deterministic UUID because this
-- repository does not require a database UUID extension.
ALTER TABLE gl.period_posting_gate
    ADD COLUMN period_posting_gate_id uuid,
    ADD COLUMN aggregate_version bigint NOT NULL DEFAULT 1,
    ADD COLUMN active_soft_close_run_id uuid,
    ADD COLUMN active_soft_close_control_epoch bigint,
    ADD COLUMN soft_close_policy_version text,
    ADD COLUMN soft_close_restrictions jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN active_gate_admission_counters jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN last_frozen_admission_summary jsonb,
    ADD COLUMN frozen_admission_summary_history jsonb NOT NULL DEFAULT '[]'::jsonb;

UPDATE gl.period_posting_gate
SET period_posting_gate_id = md5(accounting_scope_id::text || ':' || fiscal_period_id::text)::uuid,
    aggregate_version = gate_version
WHERE period_posting_gate_id IS NULL;

ALTER TABLE gl.period_posting_gate
    ALTER COLUMN period_posting_gate_id SET NOT NULL;

ALTER TABLE gl.period_posting_gate
    ADD CONSTRAINT period_posting_gate_id_unique UNIQUE (period_posting_gate_id),
    ADD CONSTRAINT period_posting_gate_aggregate_version_check CHECK (aggregate_version >= 1),
    ADD CONSTRAINT period_posting_gate_control_epoch_check CHECK (active_soft_close_control_epoch IS NULL OR active_soft_close_control_epoch >= 1),
    ADD CONSTRAINT period_posting_gate_restrictions_check CHECK (jsonb_typeof(soft_close_restrictions) IS NOT NULL),
    ADD CONSTRAINT period_posting_gate_counters_check CHECK (jsonb_typeof(active_gate_admission_counters) = 'object'),
    ADD CONSTRAINT period_posting_gate_last_summary_check CHECK (last_frozen_admission_summary IS NULL OR jsonb_typeof(last_frozen_admission_summary) = 'object'),
    ADD CONSTRAINT period_posting_gate_history_check CHECK (jsonb_typeof(frozen_admission_summary_history) = 'array');

CREATE INDEX period_posting_gate_active_owner_idx
    ON gl.period_posting_gate (accounting_scope_id, active_soft_close_run_id, active_soft_close_control_epoch)
    WHERE active_soft_close_run_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS gl.period_posting_gate_active_owner_idx;
ALTER TABLE gl.period_posting_gate
    DROP CONSTRAINT IF EXISTS period_posting_gate_history_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_last_summary_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_counters_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_restrictions_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_control_epoch_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_aggregate_version_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_id_unique,
    DROP COLUMN IF EXISTS frozen_admission_summary_history,
    DROP COLUMN IF EXISTS last_frozen_admission_summary,
    DROP COLUMN IF EXISTS active_gate_admission_counters,
    DROP COLUMN IF EXISTS soft_close_restrictions,
    DROP COLUMN IF EXISTS soft_close_policy_version,
    DROP COLUMN IF EXISTS active_soft_close_control_epoch,
    DROP COLUMN IF EXISTS active_soft_close_run_id,
    DROP COLUMN IF EXISTS aggregate_version,
    DROP COLUMN IF EXISTS period_posting_gate_id;
