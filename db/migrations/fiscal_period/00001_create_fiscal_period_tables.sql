-- +goose Up
CREATE TABLE fiscal_period.fiscal_period (
    fiscal_period_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    state text NOT NULL,
    state_version bigint NOT NULL DEFAULT 1,
    aggregate_version bigint NOT NULL DEFAULT 1,
    active_soft_close_run_id uuid,
    active_close_run_id uuid,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,

    CONSTRAINT fiscal_period_pk PRIMARY KEY (fiscal_period_id),
    CONSTRAINT fiscal_period_scope_period_unique UNIQUE (accounting_scope_id, fiscal_period_id),
    CONSTRAINT fiscal_period_state_check CHECK (state IN ('Open', 'SoftClosed', 'Closing')),
    CONSTRAINT fiscal_period_state_version_check CHECK (state_version >= 1),
    CONSTRAINT fiscal_period_aggregate_version_check CHECK (aggregate_version >= 1)
);

CREATE TABLE fiscal_period.soft_close_run (
    soft_close_run_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    fiscal_period_id uuid NOT NULL,
    status text NOT NULL,
    control_epoch bigint NOT NULL,
    opened_gate_version bigint NOT NULL,
    policy_version text NOT NULL,
    aggregate_version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT soft_close_run_pk PRIMARY KEY (soft_close_run_id),
    CONSTRAINT soft_close_run_period_fk FOREIGN KEY (fiscal_period_id) REFERENCES fiscal_period.fiscal_period (fiscal_period_id),
    CONSTRAINT soft_close_run_status_check CHECK (status IN ('Active', 'HandoffPending')),
    CONSTRAINT soft_close_run_epoch_check CHECK (control_epoch >= 1),
    CONSTRAINT soft_close_run_gate_version_check CHECK (opened_gate_version >= 1),
    CONSTRAINT soft_close_run_policy_check CHECK (btrim(policy_version) <> ''),
    CONSTRAINT soft_close_run_aggregate_version_check CHECK (aggregate_version >= 1)
);

CREATE INDEX soft_close_run_scope_period_idx
    ON fiscal_period.soft_close_run (accounting_scope_id, fiscal_period_id, status);

CREATE TABLE fiscal_period.soft_close_control_epoch (
    soft_close_run_id uuid NOT NULL,
    fiscal_period_id uuid NOT NULL,
    epoch_number bigint NOT NULL,
    opened_gate_version bigint NOT NULL,
    policy_version text NOT NULL,
    status text NOT NULL,
    handoff_close_run_id uuid,
    frozen_summary jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT soft_close_control_epoch_pk PRIMARY KEY (soft_close_run_id, epoch_number),
    CONSTRAINT soft_close_control_epoch_run_fk FOREIGN KEY (soft_close_run_id) REFERENCES fiscal_period.soft_close_run (soft_close_run_id),
    CONSTRAINT soft_close_control_epoch_period_fk FOREIGN KEY (fiscal_period_id) REFERENCES fiscal_period.fiscal_period (fiscal_period_id),
    CONSTRAINT soft_close_control_epoch_status_check CHECK (status IN ('Active', 'HandoffPending')),
    CONSTRAINT soft_close_control_epoch_number_check CHECK (epoch_number >= 1),
    CONSTRAINT soft_close_control_epoch_gate_version_check CHECK (opened_gate_version >= 1),
    CONSTRAINT soft_close_control_epoch_policy_check CHECK (btrim(policy_version) <> ''),
    CONSTRAINT soft_close_control_epoch_aggregate_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT soft_close_control_epoch_summary_check CHECK (frozen_summary IS NULL OR jsonb_typeof(frozen_summary) = 'object')
);

CREATE TABLE fiscal_period.close_run (
    close_run_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    fiscal_period_id uuid NOT NULL,
    close_type text NOT NULL,
    status text NOT NULL,
    aggregate_version bigint NOT NULL DEFAULT 1,
    command_fingerprint text NOT NULL,
    barrier_command_id uuid NOT NULL,
    barrier_idempotency_key text NOT NULL,
    barrier_evidence jsonb,
    last_failure_code text,
    last_failure_detail text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT close_run_pk PRIMARY KEY (close_run_id),
    CONSTRAINT close_run_period_fk FOREIGN KEY (fiscal_period_id) REFERENCES fiscal_period.fiscal_period (fiscal_period_id),
    CONSTRAINT close_run_close_type_check CHECK (close_type IN ('InitialHardClose')),
    CONSTRAINT close_run_status_check CHECK (status IN ('Initiating', 'BarrierAcquired', 'Failed')),
    CONSTRAINT close_run_aggregate_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT close_run_fingerprint_check CHECK (btrim(command_fingerprint) <> ''),
    CONSTRAINT close_run_barrier_command_check CHECK (barrier_command_id <> '00000000-0000-0000-0000-000000000000'::uuid),
    CONSTRAINT close_run_barrier_key_check CHECK (btrim(barrier_idempotency_key) <> ''),
    CONSTRAINT close_run_evidence_check CHECK (barrier_evidence IS NULL OR jsonb_typeof(barrier_evidence) = 'object')
);

CREATE UNIQUE INDEX close_run_active_period_unique
    ON fiscal_period.close_run (accounting_scope_id, fiscal_period_id)
    WHERE status IN ('Initiating', 'BarrierAcquired', 'Failed');

CREATE INDEX close_run_scope_status_idx
    ON fiscal_period.close_run (accounting_scope_id, status, updated_at);

-- +goose Down
DROP TABLE IF EXISTS fiscal_period.close_run;
DROP TABLE IF EXISTS fiscal_period.soft_close_control_epoch;
DROP TABLE IF EXISTS fiscal_period.soft_close_run;
DROP TABLE IF EXISTS fiscal_period.fiscal_period;
