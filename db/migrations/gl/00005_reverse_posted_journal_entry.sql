-- +goose Up
ALTER TABLE gl.journal_entry
    ADD COLUMN posting_gate_mode text NOT NULL DEFAULT 'Open',
    ADD COLUMN reversal_of_journal_entry_id uuid,
    ADD COLUMN correction_reason text;

ALTER TABLE gl.journal_entry
    ADD CONSTRAINT journal_entry_gate_mode_check
        CHECK (posting_gate_mode IN ('Open', 'SoftClosePolicy', 'CloseOnly', 'HardClosed', 'ScopedReopen', 'OperationalReopen')),
    ADD CONSTRAINT journal_entry_reversal_fk
        FOREIGN KEY (reversal_of_journal_entry_id, accounting_scope_id)
        REFERENCES gl.journal_entry (journal_entry_id, accounting_scope_id)
        ON DELETE RESTRICT,
    ADD CONSTRAINT journal_entry_reversal_reason_check
        CHECK (reversal_of_journal_entry_id IS NULL OR btrim(COALESCE(correction_reason, '')) <> '');

CREATE UNIQUE INDEX journal_entry_reversal_unique
    ON gl.journal_entry (accounting_scope_id, reversal_of_journal_entry_id)
    WHERE reversal_of_journal_entry_id IS NOT NULL;

ALTER TABLE gl.posting_attempt
    DROP CONSTRAINT posting_attempt_outcome_check,
    ADD CONSTRAINT posting_attempt_outcome_check
        CHECK (outcome IN ('JournalEntryPosted', 'JournalEntryReversed', 'PostingRejected', 'PostingPendingApproval', 'IdempotencyConflict')),
    DROP CONSTRAINT posting_attempt_operation_check,
    ADD CONSTRAINT posting_attempt_operation_check
        CHECK (operation IN ('submit-posting-request', 'apply-journal-approval-decision', 'reverse-journal-entry'));

-- +goose Down
DROP INDEX IF EXISTS gl.journal_entry_reversal_unique;

ALTER TABLE gl.posting_attempt
    DROP CONSTRAINT IF EXISTS posting_attempt_operation_check,
    ADD CONSTRAINT posting_attempt_operation_check
        CHECK (operation IN ('submit-posting-request', 'apply-journal-approval-decision')),
    DROP CONSTRAINT IF EXISTS posting_attempt_outcome_check,
    ADD CONSTRAINT posting_attempt_outcome_check
        CHECK (outcome IN ('JournalEntryPosted', 'PostingRejected', 'PostingPendingApproval', 'IdempotencyConflict'));

ALTER TABLE gl.journal_entry
    DROP CONSTRAINT IF EXISTS journal_entry_reversal_reason_check,
    DROP CONSTRAINT IF EXISTS journal_entry_reversal_fk,
    DROP CONSTRAINT IF EXISTS journal_entry_gate_mode_check,
    DROP COLUMN IF EXISTS correction_reason,
    DROP COLUMN IF EXISTS reversal_of_journal_entry_id,
    DROP COLUMN IF EXISTS posting_gate_mode;
