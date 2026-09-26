-- +goose Up
CREATE TABLE identity.segregation_rule (
    id uuid NOT NULL,
    code text NOT NULL,
    current_version bigint NOT NULL DEFAULT 1,
    status text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,

    CONSTRAINT segregation_rule_pk PRIMARY KEY (id),
    CONSTRAINT segregation_rule_code_unique UNIQUE (code),
    CONSTRAINT segregation_rule_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT segregation_rule_status_check CHECK (status IN ('active', 'retired')),
    CONSTRAINT segregation_rule_version_check CHECK (current_version >= 1)
);

CREATE TABLE identity.segregation_rule_revision (
    rule_id uuid NOT NULL,
    revision_version bigint NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    conflicting_permissions text[] NOT NULL,
    enforcement_mode text NOT NULL,
    scope_ids text[] NOT NULL DEFAULT '{}'::text[],
    amount_threshold numeric,
    cooling_off_seconds bigint NOT NULL DEFAULT 0,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    approval_request_id uuid NOT NULL,
    approval_decision_id uuid NOT NULL,
    approver_user_id uuid NOT NULL,
    policy_version text NOT NULL,
    decision_version bigint NOT NULL,
    subject_version bigint NOT NULL,
    candidate_fingerprint text NOT NULL,
    audit_reference uuid NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT segregation_rule_revision_pk PRIMARY KEY (rule_id, revision_version),
    CONSTRAINT segregation_rule_revision_rule_fk
        FOREIGN KEY (rule_id)
        REFERENCES identity.segregation_rule (id)
        ON DELETE RESTRICT,
    CONSTRAINT segregation_rule_revision_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT segregation_rule_revision_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT segregation_rule_revision_status_check CHECK (status IN ('active', 'retired')),
    CONSTRAINT segregation_rule_revision_version_check CHECK (revision_version >= 1),
    CONSTRAINT segregation_rule_revision_permissions_check CHECK (cardinality(conflicting_permissions) >= 2),
    CONSTRAINT segregation_rule_revision_mode_check CHECK (enforcement_mode IN ('block', 'exception-required')),
    CONSTRAINT segregation_rule_revision_scope_values_check CHECK (array_position(scope_ids, '') IS NULL),
    CONSTRAINT segregation_rule_revision_threshold_check CHECK (amount_threshold IS NULL OR amount_threshold >= 0),
    CONSTRAINT segregation_rule_revision_cooling_off_check CHECK (cooling_off_seconds >= 0),
    CONSTRAINT segregation_rule_revision_effective_date_check CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT segregation_rule_revision_policy_version_check CHECK (btrim(policy_version) <> ''),
    CONSTRAINT segregation_rule_revision_decision_version_check CHECK (decision_version >= 1),
    CONSTRAINT segregation_rule_revision_subject_version_check CHECK (subject_version >= 1),
    CONSTRAINT segregation_rule_revision_fingerprint_check CHECK (btrim(candidate_fingerprint) <> ''),
    CONSTRAINT segregation_rule_revision_audit_reference_check CHECK (audit_reference <> '00000000-0000-0000-0000-000000000000')
);

ALTER TABLE identity.segregation_rule
    ADD CONSTRAINT segregation_rule_current_revision_fk
    FOREIGN KEY (id, current_version)
    REFERENCES identity.segregation_rule_revision (rule_id, revision_version)
    DEFERRABLE INITIALLY DEFERRED;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION identity.reject_segregation_rule_revision_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'identity segregation rule revisions are immutable';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER segregation_rule_revision_immutable
BEFORE UPDATE OR DELETE ON identity.segregation_rule_revision
FOR EACH ROW EXECUTE FUNCTION identity.reject_segregation_rule_revision_mutation();

CREATE INDEX segregation_rule_status_idx
    ON identity.segregation_rule (status);

CREATE INDEX segregation_rule_revision_effective_idx
    ON identity.segregation_rule_revision (status, effective_from, effective_to);

CREATE INDEX segregation_rule_revision_permissions_idx
    ON identity.segregation_rule_revision USING gin (conflicting_permissions);

INSERT INTO identity.segregation_rule (id, code, current_version, status, created_at, updated_at)
VALUES
    ('00000000-0000-0000-0000-000000000501', 'payment-batch-preparation-approval', 1, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000502', 'fiscal-period-reopen-request-approval', 1, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000503', 'vendor-bank-detail-payment-release-cooling-off', 1, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000504', 'manual-journal-self-approval-threshold', 1, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000505', 'payroll-detail-summary-ledger', 1, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000506', 'independent-policy-approval', 1, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');

INSERT INTO identity.segregation_rule_revision (
    rule_id, revision_version, code, name, status, conflicting_permissions, enforcement_mode,
    scope_ids, amount_threshold, cooling_off_seconds, effective_from,
    approval_request_id, approval_decision_id, approver_user_id, policy_version,
    decision_version, subject_version, candidate_fingerprint, audit_reference, created_at
)
VALUES
    ('00000000-0000-0000-0000-000000000501', 1, 'payment-batch-preparation-approval', 'Payment batch preparer and approver', 'active', ARRAY['finance.pcm.prepare.payment.batch', 'finance.pcm.apply.payment.batch.approval.decision'], 'exception-required', '{}', NULL, 0, '2026-01-01T00:00:00Z', '10000000-0000-0000-0000-000000000501', '20000000-0000-0000-0000-000000000501', '30000000-0000-0000-0000-000000000501', 'iam-segregation-baseline-v1', 1, 1, 'sha256:baseline-payment-batch-preparation-approval', '40000000-0000-0000-0000-000000000501', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000502', 1, 'fiscal-period-reopen-request-approval', 'Fiscal-period reopen requester and approver', 'active', ARRAY['finance.fpm.request.reopen', 'finance.fpm.apply.reopen.approval.decision'], 'block', '{}', NULL, 0, '2026-01-01T00:00:00Z', '10000000-0000-0000-0000-000000000502', '20000000-0000-0000-0000-000000000502', '30000000-0000-0000-0000-000000000502', 'iam-segregation-baseline-v1', 1, 1, 'sha256:baseline-fiscal-period-reopen-request-approval', '40000000-0000-0000-0000-000000000502', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000503', 1, 'vendor-bank-detail-payment-release-cooling-off', 'Vendor bank-detail cooling-off', 'active', ARRAY['finance.omd.maintain.vendor.profiles', 'finance.pcm.submit.payment.instruction'], 'block', '{}', NULL, 86400, '2026-01-01T00:00:00Z', '10000000-0000-0000-0000-000000000503', '20000000-0000-0000-0000-000000000503', '30000000-0000-0000-0000-000000000503', 'iam-segregation-baseline-v1', 1, 1, 'sha256:baseline-vendor-bank-detail-payment-release-cooling-off', '40000000-0000-0000-0000-000000000503', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000504', 1, 'manual-journal-self-approval-threshold', 'Manual-journal self approval threshold', 'active', ARRAY['finance.gl.submit.posting.request', 'finance.gl.apply.journal.approval.decision'], 'block', '{}', 10000.00, 0, '2026-01-01T00:00:00Z', '10000000-0000-0000-0000-000000000504', '20000000-0000-0000-0000-000000000504', '30000000-0000-0000-0000-000000000504', 'iam-segregation-baseline-v1', 1, 1, 'sha256:baseline-manual-journal-self-approval-threshold', '40000000-0000-0000-0000-000000000504', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000505', 1, 'payroll-detail-summary-ledger', 'Payroll detail and summary ledger', 'active', ARRAY['finance.payr.maintain.employee.payroll.profiles', 'finance.rpt.generate.and.publish.ledger.financial.statements'], 'block', '{}', NULL, 0, '2026-01-01T00:00:00Z', '10000000-0000-0000-0000-000000000505', '20000000-0000-0000-0000-000000000505', '30000000-0000-0000-0000-000000000505', 'iam-segregation-baseline-v1', 1, 1, 'sha256:baseline-payroll-detail-summary-ledger', '40000000-0000-0000-0000-000000000505', '2026-01-01T00:00:00Z'),
    ('00000000-0000-0000-0000-000000000506', 1, 'independent-policy-approval', 'Independent policy approval', 'active', ARRAY['finance.iam.manage.access.policies', 'finance.wfa.decide.approval.request'], 'block', '{}', NULL, 0, '2026-01-01T00:00:00Z', '10000000-0000-0000-0000-000000000506', '20000000-0000-0000-0000-000000000506', '30000000-0000-0000-0000-000000000506', 'iam-segregation-baseline-v1', 1, 1, 'sha256:baseline-independent-policy-approval', '40000000-0000-0000-0000-000000000506', '2026-01-01T00:00:00Z');

-- +goose Down
ALTER TABLE identity.segregation_rule
    DROP CONSTRAINT IF EXISTS segregation_rule_current_revision_fk;

DROP TRIGGER IF EXISTS segregation_rule_revision_immutable ON identity.segregation_rule_revision;
DROP FUNCTION IF EXISTS identity.reject_segregation_rule_revision_mutation();
DROP TABLE IF EXISTS identity.segregation_rule_revision;
DROP TABLE IF EXISTS identity.segregation_rule;
