-- +goose Up
CREATE TABLE gl.chart_of_accounts (
    chart_of_accounts_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    ledger_id uuid NOT NULL,
    account_code_policy text NOT NULL,
    lifecycle_status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    approval_reference jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT chart_of_accounts_pk PRIMARY KEY (chart_of_accounts_id),
    CONSTRAINT chart_of_accounts_scope_unique UNIQUE (chart_of_accounts_id, accounting_scope_id),
    CONSTRAINT chart_of_accounts_ledger_fk FOREIGN KEY (ledger_id, accounting_scope_id) REFERENCES gl.ledger (ledger_id, accounting_scope_id) ON DELETE RESTRICT,
    CONSTRAINT chart_of_accounts_status_check CHECK (lifecycle_status IN ('draft', 'active', 'suspended', 'retired')),
    CONSTRAINT chart_of_accounts_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT chart_of_accounts_revision_check CHECK (revision_number >= 1),
    CONSTRAINT chart_of_accounts_policy_check CHECK (btrim(account_code_policy) <> ''),
    CONSTRAINT chart_of_accounts_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE UNIQUE INDEX chart_of_accounts_effective_identity_unique
    ON gl.chart_of_accounts (accounting_scope_id, ledger_id, account_code_policy, effective_from, COALESCE(effective_to, DATE '9999-12-31'));

CREATE INDEX chart_of_accounts_scope_status_idx
    ON gl.chart_of_accounts (accounting_scope_id, lifecycle_status);

CREATE INDEX chart_of_accounts_ledger_idx
    ON gl.chart_of_accounts (ledger_id, effective_from);

CREATE TABLE gl.chart_of_accounts_revision (
    chart_of_accounts_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT chart_of_accounts_revision_pk PRIMARY KEY (chart_of_accounts_id, revision_number),
    CONSTRAINT chart_of_accounts_revision_fk FOREIGN KEY (chart_of_accounts_id) REFERENCES gl.chart_of_accounts (chart_of_accounts_id) ON DELETE CASCADE,
    CONSTRAINT chart_of_accounts_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT chart_of_accounts_revision_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX chart_of_accounts_revision_effective_idx
    ON gl.chart_of_accounts_revision (chart_of_accounts_id, effective_from);

CREATE TABLE gl.account (
    account_id uuid NOT NULL,
    accounting_scope_id uuid NOT NULL,
    chart_of_accounts_id uuid NOT NULL,
    account_code text NOT NULL,
    account_name text NOT NULL,
    account_type text NOT NULL,
    normal_balance text NOT NULL,
    lifecycle_status text NOT NULL,
    restrictions jsonb NOT NULL DEFAULT '[]'::jsonb,
    currency_policy text NOT NULL,
    reporting_mappings jsonb NOT NULL DEFAULT '[]'::jsonb,
    effective_from date NOT NULL,
    effective_to date,
    approval_reference jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT account_pk PRIMARY KEY (account_id),
    CONSTRAINT account_chart_fk FOREIGN KEY (chart_of_accounts_id, accounting_scope_id) REFERENCES gl.chart_of_accounts (chart_of_accounts_id, accounting_scope_id) ON DELETE RESTRICT,
    CONSTRAINT account_status_check CHECK (lifecycle_status IN ('draft', 'active', 'suspended', 'retired')),
    CONSTRAINT account_normal_balance_check CHECK (normal_balance IN ('debit', 'credit')),
    CONSTRAINT account_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT account_revision_check CHECK (revision_number >= 1),
    CONSTRAINT account_code_check CHECK (btrim(account_code) <> ''),
    CONSTRAINT account_name_check CHECK (btrim(account_name) <> ''),
    CONSTRAINT account_type_check CHECK (btrim(account_type) <> ''),
    CONSTRAINT account_currency_policy_check CHECK (btrim(currency_policy) <> ''),
    CONSTRAINT account_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT account_restrictions_array_check CHECK (jsonb_typeof(restrictions) = 'array'),
    CONSTRAINT account_reporting_mappings_array_check CHECK (jsonb_typeof(reporting_mappings) = 'array')
);

CREATE UNIQUE INDEX account_effective_identity_unique
    ON gl.account (accounting_scope_id, chart_of_accounts_id, account_code, effective_from, COALESCE(effective_to, DATE '9999-12-31'));

CREATE INDEX account_scope_status_idx
    ON gl.account (accounting_scope_id, lifecycle_status);

CREATE INDEX account_chart_code_idx
    ON gl.account (chart_of_accounts_id, account_code, effective_from);

CREATE TABLE gl.account_revision (
    account_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT account_revision_pk PRIMARY KEY (account_id, revision_number),
    CONSTRAINT account_revision_fk FOREIGN KEY (account_id) REFERENCES gl.account (account_id) ON DELETE CASCADE,
    CONSTRAINT account_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT account_revision_effective_check CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX account_revision_effective_idx
    ON gl.account_revision (account_id, effective_from);

-- +goose Down
DROP TABLE IF EXISTS gl.account_revision;
DROP TABLE IF EXISTS gl.account;
DROP TABLE IF EXISTS gl.chart_of_accounts_revision;
DROP TABLE IF EXISTS gl.chart_of_accounts;
