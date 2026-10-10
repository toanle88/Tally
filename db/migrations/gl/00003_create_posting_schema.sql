-- +goose Up
-- The existing accounting-book table is keyed by book ID alone. Posting rows
-- carry the accounting scope as part of every ownership reference, so add the
-- same-scope uniqueness needed for a composite foreign key without changing
-- the earlier immutable migration.
CREATE UNIQUE INDEX accounting_book_scope_unique
    ON gl.accounting_book (accounting_book_id, accounting_scope_id);
CREATE UNIQUE INDEX account_scope_unique
    ON gl.account (account_id, accounting_scope_id);

CREATE TABLE gl.period_posting_gate (
    accounting_scope_id uuid NOT NULL,
    fiscal_period_id uuid NOT NULL,
    gate_mode text NOT NULL DEFAULT 'Open',
    period_state_version bigint NOT NULL,
    gate_version bigint NOT NULL,
    next_ledger_position bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT period_posting_gate_pk PRIMARY KEY (accounting_scope_id, fiscal_period_id),
    CONSTRAINT period_posting_gate_mode_check CHECK (gate_mode IN ('Open', 'SoftClosePolicy', 'CloseOnly', 'HardClosed', 'ScopedReopen', 'OperationalReopen')),
    CONSTRAINT period_posting_gate_period_version_check CHECK (period_state_version >= 1),
    CONSTRAINT period_posting_gate_version_check CHECK (gate_version >= 1),
    CONSTRAINT period_posting_gate_position_check CHECK (next_ledger_position >= 1)
);

CREATE TABLE gl.journal_entry (
    journal_entry_id uuid NOT NULL,
    journal_number text NOT NULL,
    accounting_scope_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    ledger_id uuid NOT NULL,
    accounting_book_id uuid NOT NULL,
    functional_currency text NOT NULL,
    source_context text NOT NULL,
    source_aggregate_type text NOT NULL,
    source_aggregate_id uuid NOT NULL,
    source_version bigint NOT NULL,
    request_id uuid NOT NULL,
    idempotency_key text NOT NULL,
    request_fingerprint text NOT NULL,
    posting_date date NOT NULL,
    fiscal_period_id uuid NOT NULL,
    period_state_version bigint NOT NULL,
    posting_gate_version bigint NOT NULL,
    posting_purpose text NOT NULL,
    adjustment_period_indicator boolean NOT NULL DEFAULT false,
    posting_authorization_id uuid,
    close_run_id uuid,
    reopen_request_id uuid,
    operational_reopen_request_id uuid,
    control_authority_epoch bigint,
    transaction_currency text NOT NULL,
    conversion_evidence jsonb,
    description text NOT NULL DEFAULT '',
    lifecycle_status text NOT NULL,
    approval_reference jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    ledger_position bigint,
    audit_reference uuid NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT journal_entry_pk PRIMARY KEY (journal_entry_id),
    CONSTRAINT journal_entry_scope_unique UNIQUE (journal_entry_id, accounting_scope_id),
    CONSTRAINT journal_entry_ledger_fk FOREIGN KEY (ledger_id, accounting_scope_id) REFERENCES gl.ledger (ledger_id, accounting_scope_id) ON DELETE RESTRICT,
    CONSTRAINT journal_entry_book_fk FOREIGN KEY (accounting_book_id, accounting_scope_id) REFERENCES gl.accounting_book (accounting_book_id, accounting_scope_id) ON DELETE RESTRICT,
    CONSTRAINT journal_entry_status_check CHECK (lifecycle_status IN ('PendingApproval', 'Posted')),
    CONSTRAINT journal_entry_purpose_check CHECK (posting_purpose IN ('Ordinary', 'Close', 'ReopenCorrection', 'OperationalReopen', 'PolicyAdjustment')),
    CONSTRAINT journal_entry_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT journal_entry_source_version_check CHECK (source_version >= 1),
    CONSTRAINT journal_entry_period_version_check CHECK (period_state_version >= 1),
    CONSTRAINT journal_entry_gate_version_check CHECK (posting_gate_version >= 1),
    CONSTRAINT journal_entry_currency_check CHECK (functional_currency ~ '^[A-Z]{3}$' AND transaction_currency ~ '^[A-Z]{3}$'),
    CONSTRAINT journal_entry_identity_check CHECK (btrim(journal_number) <> '' AND btrim(source_context) <> '' AND btrim(source_aggregate_type) <> '' AND btrim(idempotency_key) <> '' AND btrim(request_fingerprint) <> '')
);

CREATE UNIQUE INDEX journal_entry_number_unique ON gl.journal_entry (accounting_scope_id, journal_number);
CREATE UNIQUE INDEX journal_entry_source_unique ON gl.journal_entry (accounting_scope_id, source_context, source_aggregate_type, source_aggregate_id, source_version);
CREATE UNIQUE INDEX journal_entry_idempotency_unique ON gl.journal_entry (accounting_scope_id, idempotency_key);
CREATE INDEX journal_entry_scope_period_idx ON gl.journal_entry (accounting_scope_id, fiscal_period_id, posting_date);

CREATE TABLE gl.journal_entry_line (
    journal_entry_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    line_number integer NOT NULL,
    account_id uuid NOT NULL,
    debit_or_credit text NOT NULL,
    line_currency_mode text NOT NULL,
    transaction_amount numeric(38,18) NOT NULL,
    functional_amount numeric(38,18) NOT NULL,
    segment_combination_id uuid NOT NULL,
    line_reference text NOT NULL DEFAULT '',
    CONSTRAINT journal_entry_line_pk PRIMARY KEY (journal_entry_id, line_number),
    CONSTRAINT journal_entry_line_entry_fk FOREIGN KEY (journal_entry_id, accounting_scope_id) REFERENCES gl.journal_entry (journal_entry_id, accounting_scope_id) ON DELETE CASCADE,
    CONSTRAINT journal_entry_line_account_fk FOREIGN KEY (account_id, accounting_scope_id) REFERENCES gl.account (account_id, accounting_scope_id) ON DELETE RESTRICT,
    CONSTRAINT journal_entry_line_number_check CHECK (line_number >= 1),
    CONSTRAINT journal_entry_line_direction_check CHECK (debit_or_credit IN ('debit', 'credit')),
    CONSTRAINT journal_entry_line_mode_check CHECK (line_currency_mode IN ('TransactionAndFunctional', 'FunctionalOnlyAdjustment')),
    CONSTRAINT journal_entry_line_amount_check CHECK (transaction_amount >= 0 AND functional_amount >= 0),
    CONSTRAINT journal_entry_line_functional_only_check CHECK (line_currency_mode <> 'FunctionalOnlyAdjustment' OR (transaction_amount = 0 AND functional_amount <> 0))
);

CREATE TABLE gl.posting_attempt (
    posting_attempt_id uuid NOT NULL,
    request_id uuid NOT NULL,
    journal_entry_id uuid,
    accounting_scope_id uuid NOT NULL,
    source_context text NOT NULL,
    source_aggregate_type text NOT NULL,
    source_aggregate_id uuid NOT NULL,
    source_version bigint NOT NULL,
    idempotency_key text NOT NULL,
    request_fingerprint text,
    outcome text NOT NULL,
    issues jsonb NOT NULL DEFAULT '[]'::jsonb,
    actor_user_id uuid NOT NULL,
    actor_subject_reference text NOT NULL,
    correlation_id uuid NOT NULL,
    causation_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT posting_attempt_pk PRIMARY KEY (posting_attempt_id),
    CONSTRAINT posting_attempt_outcome_check CHECK (outcome IN ('JournalEntryPosted', 'PostingRejected', 'PostingPendingApproval', 'IdempotencyConflict')),
    CONSTRAINT posting_attempt_source_version_check CHECK (source_version >= 1),
    CONSTRAINT posting_attempt_issues_check CHECK (jsonb_typeof(issues) = 'array')
);

CREATE INDEX posting_attempt_source_idx ON gl.posting_attempt (accounting_scope_id, source_context, source_aggregate_id, source_version, created_at);
CREATE INDEX posting_attempt_idempotency_idx ON gl.posting_attempt (accounting_scope_id, idempotency_key, created_at);

-- Posted journal facts are immutable. Pending approval rows remain mutable only
-- through the future approval application workflow.
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
    IF OLD.lifecycle_status = 'Posted' AND (
        NEW.journal_entry_id IS DISTINCT FROM OLD.journal_entry_id OR
        NEW.journal_number IS DISTINCT FROM OLD.journal_number OR
        NEW.accounting_scope_id IS DISTINCT FROM OLD.accounting_scope_id OR
        NEW.tenant_id IS DISTINCT FROM OLD.tenant_id OR
        NEW.legal_entity_id IS DISTINCT FROM OLD.legal_entity_id OR
        NEW.ledger_id IS DISTINCT FROM OLD.ledger_id OR
        NEW.accounting_book_id IS DISTINCT FROM OLD.accounting_book_id OR
        NEW.functional_currency IS DISTINCT FROM OLD.functional_currency OR
        NEW.source_context IS DISTINCT FROM OLD.source_context OR
        NEW.source_aggregate_type IS DISTINCT FROM OLD.source_aggregate_type OR
        NEW.source_aggregate_id IS DISTINCT FROM OLD.source_aggregate_id OR
        NEW.source_version IS DISTINCT FROM OLD.source_version OR
        NEW.request_id IS DISTINCT FROM OLD.request_id OR
        NEW.idempotency_key IS DISTINCT FROM OLD.idempotency_key OR
        NEW.request_fingerprint IS DISTINCT FROM OLD.request_fingerprint OR
        NEW.posting_date IS DISTINCT FROM OLD.posting_date OR
        NEW.fiscal_period_id IS DISTINCT FROM OLD.fiscal_period_id OR
        NEW.period_state_version IS DISTINCT FROM OLD.period_state_version OR
        NEW.posting_gate_version IS DISTINCT FROM OLD.posting_gate_version OR
        NEW.posting_purpose IS DISTINCT FROM OLD.posting_purpose OR
        NEW.adjustment_period_indicator IS DISTINCT FROM OLD.adjustment_period_indicator OR
        NEW.posting_authorization_id IS DISTINCT FROM OLD.posting_authorization_id OR
        NEW.close_run_id IS DISTINCT FROM OLD.close_run_id OR
        NEW.reopen_request_id IS DISTINCT FROM OLD.reopen_request_id OR
        NEW.operational_reopen_request_id IS DISTINCT FROM OLD.operational_reopen_request_id OR
        NEW.control_authority_epoch IS DISTINCT FROM OLD.control_authority_epoch OR
        NEW.transaction_currency IS DISTINCT FROM OLD.transaction_currency OR
        NEW.conversion_evidence IS DISTINCT FROM OLD.conversion_evidence OR
        NEW.description IS DISTINCT FROM OLD.description OR
        NEW.lifecycle_status IS DISTINCT FROM OLD.lifecycle_status OR
        NEW.approval_reference IS DISTINCT FROM OLD.approval_reference OR
        NEW.aggregate_version IS DISTINCT FROM OLD.aggregate_version OR
        NEW.ledger_position IS DISTINCT FROM OLD.ledger_position OR
        NEW.audit_reference IS DISTINCT FROM OLD.audit_reference OR
        NEW.correlation_id IS DISTINCT FROM OLD.correlation_id OR
        NEW.causation_id IS DISTINCT FROM OLD.causation_id OR
        NEW.created_at IS DISTINCT FROM OLD.created_at
    ) THEN
        RAISE EXCEPTION 'posted journal facts are immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER journal_entry_posted_immutable
    BEFORE UPDATE OR DELETE ON gl.journal_entry
    FOR EACH ROW
    EXECUTE FUNCTION gl.prevent_posted_journal_mutation();

-- Journal lines are part of the posted journal fact. Keep the line boundary
-- immutable as well; changing or deleting a parent journal is already
-- protected by the journal trigger above, while this trigger protects direct
-- line-table mutations.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION gl.prevent_posted_journal_line_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') AND EXISTS (
        SELECT 1
        FROM gl.journal_entry
        WHERE journal_entry_id = OLD.journal_entry_id
          AND lifecycle_status = 'Posted'
    ) THEN
        RAISE EXCEPTION 'posted journal lines are immutable';
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') AND EXISTS (
        SELECT 1
        FROM gl.journal_entry
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

CREATE TRIGGER journal_entry_line_posted_immutable
    BEFORE INSERT OR UPDATE OR DELETE ON gl.journal_entry_line
    FOR EACH ROW
    EXECUTE FUNCTION gl.prevent_posted_journal_line_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS journal_entry_line_posted_immutable ON gl.journal_entry_line;
DROP FUNCTION IF EXISTS gl.prevent_posted_journal_line_mutation();
DROP TRIGGER IF EXISTS journal_entry_posted_immutable ON gl.journal_entry;
DROP FUNCTION IF EXISTS gl.prevent_posted_journal_mutation();
DROP TABLE IF EXISTS gl.posting_attempt;
DROP TABLE IF EXISTS gl.journal_entry_line;
DROP TABLE IF EXISTS gl.journal_entry;
DROP TABLE IF EXISTS gl.period_posting_gate;
DROP INDEX IF EXISTS gl.account_scope_unique;
DROP INDEX IF EXISTS gl.accounting_book_scope_unique;
