-- +goose Up
-- Story 8 records the GL-owned scoped-reopen handoff and admission authority.
-- Existing gates remain unaffected and receive empty scope/evidence defaults.
ALTER TABLE gl.period_posting_gate
    ADD COLUMN active_reopen_request_id uuid,
    ADD COLUMN reopen_authorization_scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN reopen_authorization_expiry timestamptz,
    ADD COLUMN reopen_control_authority_epoch bigint,
    ADD COLUMN reopen_approval jsonb,
    ADD COLUMN reopen_no_change_close_permitted boolean NOT NULL DEFAULT false;

ALTER TABLE gl.period_posting_gate
    ADD CONSTRAINT period_posting_gate_reopen_scope_check
        CHECK (jsonb_typeof(reopen_authorization_scope) = 'object'),
    ADD CONSTRAINT period_posting_gate_reopen_epoch_check
        CHECK (reopen_control_authority_epoch IS NULL OR reopen_control_authority_epoch >= 1),
    ADD CONSTRAINT period_posting_gate_reopen_expiry_check
        CHECK (reopen_authorization_expiry IS NULL OR active_reopen_request_id IS NOT NULL);

CREATE INDEX period_posting_gate_active_reopen_owner_idx
    ON gl.period_posting_gate (accounting_scope_id, active_reopen_request_id)
    WHERE active_reopen_request_id IS NOT NULL;

ALTER TABLE gl.journal_entry
    ADD COLUMN reopen_authorization_scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN reopen_authorization_expiry timestamptz;

ALTER TABLE gl.journal_entry
    ADD CONSTRAINT journal_entry_reopen_scope_check
        CHECK (jsonb_typeof(reopen_authorization_scope) = 'object'),
    ADD CONSTRAINT journal_entry_reopen_expiry_check
        CHECK (reopen_authorization_expiry IS NULL OR reopen_request_id IS NOT NULL);

-- The existing journal immutability trigger predates these evidence columns.
-- Keep the new correction authorization evidence immutable after posting too.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION gl.prevent_posted_reopen_evidence_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.lifecycle_status = 'Posted' AND (
        NEW.reopen_authorization_scope IS DISTINCT FROM OLD.reopen_authorization_scope OR
        NEW.reopen_authorization_expiry IS DISTINCT FROM OLD.reopen_authorization_expiry
    ) THEN
        RAISE EXCEPTION 'posted journal reopen evidence is immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER journal_entry_posted_reopen_evidence_immutable
    BEFORE UPDATE ON gl.journal_entry
    FOR EACH ROW
    EXECUTE FUNCTION gl.prevent_posted_reopen_evidence_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS journal_entry_posted_reopen_evidence_immutable ON gl.journal_entry;
DROP FUNCTION IF EXISTS gl.prevent_posted_reopen_evidence_mutation();
ALTER TABLE gl.journal_entry
    DROP CONSTRAINT IF EXISTS journal_entry_reopen_expiry_check,
    DROP CONSTRAINT IF EXISTS journal_entry_reopen_scope_check,
    DROP COLUMN IF EXISTS reopen_authorization_expiry,
    DROP COLUMN IF EXISTS reopen_authorization_scope;
DROP INDEX IF EXISTS gl.period_posting_gate_active_reopen_owner_idx;
ALTER TABLE gl.period_posting_gate
    DROP CONSTRAINT IF EXISTS period_posting_gate_reopen_expiry_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_reopen_epoch_check,
    DROP CONSTRAINT IF EXISTS period_posting_gate_reopen_scope_check,
    DROP COLUMN IF EXISTS reopen_no_change_close_permitted,
    DROP COLUMN IF EXISTS reopen_approval,
    DROP COLUMN IF EXISTS reopen_control_authority_epoch,
    DROP COLUMN IF EXISTS reopen_authorization_expiry,
    DROP COLUMN IF EXISTS reopen_authorization_scope,
    DROP COLUMN IF EXISTS active_reopen_request_id;
