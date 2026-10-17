-- +goose Up
ALTER TABLE fiscal_period.fiscal_period
    DROP CONSTRAINT fiscal_period_state_check,
    ADD CONSTRAINT fiscal_period_state_check CHECK (state IN ('Open', 'SoftClosed', 'Closing', 'HardClosed', 'Reopening'));

CREATE TABLE fiscal_period.reopen_request (
    reopen_request_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    fiscal_period_id uuid NOT NULL,
    original_close_run_id uuid NOT NULL,
    reopen_mode text NOT NULL,
    status text NOT NULL,
    reason text NOT NULL,
    authorization_scope jsonb NOT NULL,
    proposed_corrections jsonb NOT NULL,
    impact jsonb NOT NULL,
    authorization_expiry timestamptz NOT NULL,
    expected_period_version bigint NOT NULL,
    expected_gate_version bigint NOT NULL,
    control_authority_epoch bigint NOT NULL,
    requested_by_user_id uuid NOT NULL,
    requested_by_subject_reference text NOT NULL,
    request_fingerprint text NOT NULL,
    approval_request_id uuid,
    approval_decision jsonb,
    gate_result jsonb,
    gate_version bigint,
    aggregate_version bigint NOT NULL DEFAULT 1,
    correlation_id text NOT NULL,
    causation_id text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT reopen_request_pk PRIMARY KEY (reopen_request_id),
    CONSTRAINT reopen_request_period_fk FOREIGN KEY (fiscal_period_id) REFERENCES fiscal_period.fiscal_period (fiscal_period_id),
    CONSTRAINT reopen_request_original_close_run_fk FOREIGN KEY (original_close_run_id) REFERENCES fiscal_period.close_run (close_run_id),
    CONSTRAINT reopen_request_mode_check CHECK (reopen_mode IN ('ScopedCorrection', 'Operational')),
    CONSTRAINT reopen_request_status_check CHECK (status IN ('Draft', 'PendingApproval', 'Approved', 'GateOpening', 'Active', 'ExpiredPendingClosure', 'GateClosing', 'CompletedNoChange', 'RecloseHandoff', 'RecloseInProgress', 'Completed', 'Rejected', 'Cancelled', 'Failed')),
    CONSTRAINT reopen_request_reason_check CHECK (btrim(reason) <> ''),
    CONSTRAINT reopen_request_scope_check CHECK (jsonb_typeof(authorization_scope) = 'object'),
    CONSTRAINT reopen_request_corrections_check CHECK (jsonb_typeof(proposed_corrections) = 'object'),
    CONSTRAINT reopen_request_impact_check CHECK (jsonb_typeof(impact) = 'object'),
    CONSTRAINT reopen_request_subject_check CHECK (btrim(requested_by_subject_reference) <> ''),
    CONSTRAINT reopen_request_fingerprint_check CHECK (btrim(request_fingerprint) <> ''),
    CONSTRAINT reopen_request_approval_check CHECK (approval_decision IS NULL OR jsonb_typeof(approval_decision) = 'object'),
    CONSTRAINT reopen_request_gate_result_check CHECK (gate_result IS NULL OR jsonb_typeof(gate_result) = 'object'),
    CONSTRAINT reopen_request_version_check CHECK (expected_period_version >= 1 AND expected_gate_version >= 1 AND control_authority_epoch >= 1 AND aggregate_version >= 1),
    CONSTRAINT reopen_request_gate_version_check CHECK (gate_version IS NULL OR gate_version >= 1)
);

CREATE UNIQUE INDEX reopen_request_fingerprint_unique
    ON fiscal_period.reopen_request (accounting_scope_id, fiscal_period_id, request_fingerprint);

CREATE UNIQUE INDEX reopen_request_active_period_unique
    ON fiscal_period.reopen_request (accounting_scope_id, fiscal_period_id)
    WHERE status IN ('Draft', 'PendingApproval', 'Approved', 'GateOpening', 'Active', 'ExpiredPendingClosure', 'GateClosing', 'RecloseHandoff', 'RecloseInProgress', 'Failed');

CREATE INDEX reopen_request_scope_status_idx
    ON fiscal_period.reopen_request (accounting_scope_id, fiscal_period_id, status, updated_at);

-- +goose Down
DROP INDEX IF EXISTS fiscal_period.reopen_request_scope_status_idx;
DROP INDEX IF EXISTS fiscal_period.reopen_request_active_period_unique;
DROP INDEX IF EXISTS fiscal_period.reopen_request_fingerprint_unique;
DROP TABLE IF EXISTS fiscal_period.reopen_request;

ALTER TABLE fiscal_period.fiscal_period
    DROP CONSTRAINT fiscal_period_state_check,
    ADD CONSTRAINT fiscal_period_state_check CHECK (state IN ('Open', 'SoftClosed', 'Closing', 'HardClosed'));
