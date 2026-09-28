-- +goose Up
CREATE TABLE organization.fiscal_calendar (
    id uuid NOT NULL,
    scope_id uuid NOT NULL,
    calendar_type text NOT NULL,
    period_pattern text NOT NULL,
    status text NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    approval_reference jsonb,
    aggregate_version bigint NOT NULL DEFAULT 1,
    revision_number bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,
    CONSTRAINT fiscal_calendar_pk PRIMARY KEY (id),
    CONSTRAINT fiscal_calendar_status_check CHECK (status IN ('draft', 'active', 'end_dated')),
    CONSTRAINT fiscal_calendar_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT fiscal_calendar_revision_check CHECK (revision_number >= 1),
    CONSTRAINT fiscal_calendar_type_check CHECK (calendar_type ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT fiscal_calendar_pattern_check CHECK (period_pattern ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT fiscal_calendar_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT fiscal_calendar_ended_date_check CHECK ((status = 'end_dated') = (effective_to IS NOT NULL))
);

CREATE UNIQUE INDEX fiscal_calendar_scope_type_active_unique
    ON organization.fiscal_calendar (scope_id, calendar_type)
    WHERE status <> 'end_dated';
CREATE INDEX fiscal_calendar_scope_status_idx
    ON organization.fiscal_calendar (scope_id, status);

CREATE TABLE organization.fiscal_calendar_period (
    fiscal_calendar_id uuid NOT NULL,
    period_id uuid NOT NULL,
    period_reference text NOT NULL,
    period_ordinal integer NOT NULL,
    start_date date NOT NULL,
    end_date date NOT NULL,
    CONSTRAINT fiscal_calendar_period_pk PRIMARY KEY (fiscal_calendar_id, period_id),
    CONSTRAINT fiscal_calendar_period_calendar_fk FOREIGN KEY (fiscal_calendar_id) REFERENCES organization.fiscal_calendar (id) ON DELETE CASCADE,
    CONSTRAINT fiscal_calendar_period_reference_check CHECK (period_reference ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    CONSTRAINT fiscal_calendar_period_ordinal_check CHECK (period_ordinal >= 1),
    CONSTRAINT fiscal_calendar_period_date_check CHECK (end_date >= start_date)
);
CREATE UNIQUE INDEX fiscal_calendar_period_reference_unique
    ON organization.fiscal_calendar_period (fiscal_calendar_id, period_reference);
CREATE UNIQUE INDEX fiscal_calendar_period_ordinal_unique
    ON organization.fiscal_calendar_period (fiscal_calendar_id, period_ordinal);
CREATE INDEX fiscal_calendar_period_date_idx
    ON organization.fiscal_calendar_period (fiscal_calendar_id, start_date, end_date);

CREATE TABLE organization.fiscal_calendar_revision (
    fiscal_calendar_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    aggregate_version bigint NOT NULL,
    snapshot jsonb NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    created_at timestamptz NOT NULL,
    CONSTRAINT fiscal_calendar_revision_pk PRIMARY KEY (fiscal_calendar_id, revision_number),
    CONSTRAINT fiscal_calendar_revision_calendar_fk FOREIGN KEY (fiscal_calendar_id) REFERENCES organization.fiscal_calendar (id) ON DELETE CASCADE,
    CONSTRAINT fiscal_calendar_revision_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT fiscal_calendar_revision_effective_check CHECK (effective_to IS NULL OR effective_to > effective_from)
);
CREATE INDEX fiscal_calendar_revision_effective_idx
    ON organization.fiscal_calendar_revision (fiscal_calendar_id, effective_from);

-- +goose Down
DROP TABLE IF EXISTS organization.fiscal_calendar_revision;
DROP TABLE IF EXISTS organization.fiscal_calendar_period;
DROP TABLE IF EXISTS organization.fiscal_calendar;
