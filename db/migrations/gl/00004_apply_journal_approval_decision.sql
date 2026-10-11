-- +goose Up
ALTER TABLE gl.journal_entry
    ADD COLUMN submitted_by_user_id uuid,
    ADD COLUMN approval_application_key text,
    ADD COLUMN approval_application_fingerprint text,
    ADD COLUMN approval_decision_id uuid,
    ADD COLUMN approval_applied_at timestamptz;

ALTER TABLE gl.journal_entry
    DROP CONSTRAINT journal_entry_status_check,
    ADD CONSTRAINT journal_entry_status_check CHECK (lifecycle_status IN ('PendingApproval', 'Posted', 'Cancelled'));

CREATE UNIQUE INDEX journal_entry_approval_application_unique
    ON gl.journal_entry (accounting_scope_id, journal_entry_id, approval_application_key)
    WHERE approval_application_key IS NOT NULL;

CREATE UNIQUE INDEX journal_entry_approval_decision_unique
    ON gl.journal_entry (accounting_scope_id, journal_entry_id, approval_decision_id)
    WHERE approval_decision_id IS NOT NULL;

ALTER TABLE gl.posting_attempt
    ADD COLUMN operation text NOT NULL DEFAULT 'submit-posting-request';

ALTER TABLE gl.posting_attempt
    ADD CONSTRAINT posting_attempt_operation_check CHECK (operation IN ('submit-posting-request', 'apply-journal-approval-decision'));

-- Applied approval decisions are terminal financial lifecycle facts. The
-- pending-to-terminal transition is allowed; later changes to a Posted or
-- Cancelled journal remain prohibited.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION gl.prevent_posted_journal_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.lifecycle_status IN ('Posted', 'Cancelled') THEN
            RAISE EXCEPTION 'established journal facts are immutable';
        END IF;
        RETURN OLD;
    END IF;
    IF OLD.lifecycle_status IN ('Posted', 'Cancelled') AND NEW IS DISTINCT FROM OLD THEN
        RAISE EXCEPTION 'established journal facts are immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION gl.prevent_posted_journal_line_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND EXISTS (
        SELECT 1 FROM gl.journal_entry
        WHERE journal_entry_id = OLD.journal_entry_id
          AND lifecycle_status IN ('Posted', 'Cancelled')
    ) THEN
        RAISE EXCEPTION 'established journal lines are immutable';
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') AND EXISTS (
        SELECT 1 FROM gl.journal_entry
        WHERE journal_entry_id = NEW.journal_entry_id
          AND lifecycle_status IN ('Posted', 'Cancelled')
    ) THEN
        RAISE EXCEPTION 'established journal lines are immutable';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS gl.journal_entry_approval_decision_unique;
DROP INDEX IF EXISTS gl.journal_entry_approval_application_unique;
ALTER TABLE gl.posting_attempt DROP CONSTRAINT IF EXISTS posting_attempt_operation_check;
ALTER TABLE gl.posting_attempt DROP COLUMN IF EXISTS operation;
ALTER TABLE gl.journal_entry DROP CONSTRAINT IF EXISTS journal_entry_status_check;
ALTER TABLE gl.journal_entry
    ADD CONSTRAINT journal_entry_status_check CHECK (lifecycle_status IN ('PendingApproval', 'Posted'));
ALTER TABLE gl.journal_entry
    DROP COLUMN IF EXISTS approval_applied_at,
    DROP COLUMN IF EXISTS approval_decision_id,
    DROP COLUMN IF EXISTS approval_application_fingerprint,
    DROP COLUMN IF EXISTS approval_application_key,
    DROP COLUMN IF EXISTS submitted_by_user_id;

-- Restore the original trigger semantics for a clean rollback.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION gl.prevent_posted_journal_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.lifecycle_status = 'Posted' THEN
            RAISE EXCEPTION 'posted journal facts are immutable';
        END IF;
        RETURN OLD;
    END IF;
    IF OLD.lifecycle_status = 'Posted' AND NEW IS DISTINCT FROM OLD THEN
        RAISE EXCEPTION 'posted journal facts are immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION gl.prevent_posted_journal_line_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND EXISTS (
        SELECT 1 FROM gl.journal_entry
        WHERE journal_entry_id = OLD.journal_entry_id
          AND lifecycle_status = 'Posted'
    ) THEN
        RAISE EXCEPTION 'posted journal lines are immutable';
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') AND EXISTS (
        SELECT 1 FROM gl.journal_entry
        WHERE journal_entry_id = NEW.journal_entry_id
          AND lifecycle_status = 'Posted'
    ) THEN
        RAISE EXCEPTION 'posted journal lines are immutable';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
