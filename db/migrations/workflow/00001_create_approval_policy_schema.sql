-- +goose Up
CREATE TABLE workflow.approval_policy (
    id uuid NOT NULL,
    code text NOT NULL,
    current_policy_version text NOT NULL,
    status text NOT NULL,
    aggregate_version bigint NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_audit_reference uuid,

    CONSTRAINT approval_policy_pk PRIMARY KEY (id),
    CONSTRAINT approval_policy_code_unique UNIQUE (code),
    CONSTRAINT approval_policy_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT approval_policy_current_version_not_blank CHECK (btrim(current_policy_version) <> ''),
    CONSTRAINT approval_policy_status_check CHECK (status IN ('Draft', 'PendingApproval', 'Active', 'Superseded')),
    CONSTRAINT approval_policy_version_check CHECK (aggregate_version >= 1)
);

CREATE TABLE workflow.approval_policy_version_identity (
    policy_id uuid NOT NULL,
    policy_version text NOT NULL,
    code text NOT NULL,
    candidate_fingerprint text NOT NULL,
    selector_fingerprint text NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    created_at timestamptz NOT NULL,

    CONSTRAINT approval_policy_version_identity_pk PRIMARY KEY (policy_id, policy_version),
    CONSTRAINT approval_policy_version_identity_fk FOREIGN KEY (policy_id)
        REFERENCES workflow.approval_policy (id) ON DELETE RESTRICT,
    CONSTRAINT approval_policy_version_identity_code_version_unique UNIQUE (code, policy_version),
    CONSTRAINT approval_policy_version_identity_version_not_blank CHECK (btrim(policy_version) <> ''),
    CONSTRAINT approval_policy_version_identity_fingerprint_not_blank CHECK (btrim(candidate_fingerprint) <> ''),
    CONSTRAINT approval_policy_version_identity_selector_not_blank CHECK (btrim(selector_fingerprint) <> ''),
    CONSTRAINT approval_policy_version_identity_date_check CHECK (effective_to IS NULL OR effective_to > effective_from)
);

CREATE TABLE workflow.approval_policy_revision (
    policy_id uuid NOT NULL,
    revision_version bigint NOT NULL,
    policy_version text NOT NULL,
    code text NOT NULL,
    name text NOT NULL DEFAULT '',
    status text NOT NULL,
    applicability jsonb NOT NULL,
    scope jsonb NOT NULL,
    rules jsonb NOT NULL,
    steps jsonb NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    approval jsonb,
    approval_request_id uuid,
    activation_outcome text NOT NULL,
    dependent_workflow_impact text NOT NULL DEFAULT '',
    audit_reference uuid NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT approval_policy_revision_pk PRIMARY KEY (policy_id, revision_version),
    CONSTRAINT approval_policy_revision_policy_fk FOREIGN KEY (policy_id)
        REFERENCES workflow.approval_policy (id) ON DELETE RESTRICT,
    CONSTRAINT approval_policy_revision_version_identity_fk FOREIGN KEY (policy_id, policy_version)
        REFERENCES workflow.approval_policy_version_identity (policy_id, policy_version) ON DELETE RESTRICT,
    CONSTRAINT approval_policy_revision_version_check CHECK (revision_version >= 1),
    CONSTRAINT approval_policy_revision_status_check CHECK (status IN ('Draft', 'PendingApproval', 'Active', 'Superseded')),
    CONSTRAINT approval_policy_revision_applicability_object CHECK (jsonb_typeof(applicability) = 'object'),
    CONSTRAINT approval_policy_revision_scope_object CHECK (jsonb_typeof(scope) = 'object'),
    CONSTRAINT approval_policy_revision_rules_array CHECK (jsonb_typeof(rules) = 'array'),
    CONSTRAINT approval_policy_revision_steps_array CHECK (jsonb_typeof(steps) = 'array'),
    CONSTRAINT approval_policy_revision_date_check CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT approval_policy_revision_audit_not_nil CHECK (audit_reference <> '00000000-0000-0000-0000-000000000000')
);

-- Keep the mutable aggregate pointer attached to an immutable current
-- revision. The constraint is deferred because the repository writes the
-- root pointer and its new revision in one transaction.
ALTER TABLE workflow.approval_policy_revision
    ADD CONSTRAINT approval_policy_revision_current_key
    UNIQUE (policy_id, revision_version, policy_version);

ALTER TABLE workflow.approval_policy
    ADD CONSTRAINT approval_policy_current_revision_fk
    FOREIGN KEY (id, aggregate_version, current_policy_version)
    REFERENCES workflow.approval_policy_revision (policy_id, revision_version, policy_version)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE workflow.approval_policy_rule (
    policy_id uuid NOT NULL,
    revision_version bigint NOT NULL,
    rule_id uuid NOT NULL,
    name text NOT NULL,
    minimum_amount numeric,
    maximum_amount numeric,
    currency char(3) NOT NULL,
    required_roles jsonb NOT NULL,
    delegation jsonb NOT NULL,
    escalation jsonb NOT NULL,

    CONSTRAINT approval_policy_rule_pk PRIMARY KEY (policy_id, revision_version, rule_id),
    CONSTRAINT approval_policy_rule_revision_fk FOREIGN KEY (policy_id, revision_version)
        REFERENCES workflow.approval_policy_revision (policy_id, revision_version) ON DELETE RESTRICT,
    CONSTRAINT approval_policy_rule_currency_check CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT approval_policy_rule_amount_check CHECK ((minimum_amount IS NULL OR minimum_amount >= 0) AND (maximum_amount IS NULL OR maximum_amount >= 0) AND (minimum_amount IS NULL OR maximum_amount IS NULL OR minimum_amount <= maximum_amount)),
    CONSTRAINT approval_policy_rule_roles_array CHECK (jsonb_typeof(required_roles) = 'array')
);

CREATE TABLE workflow.approval_policy_step_definition (
    policy_id uuid NOT NULL,
    revision_version bigint NOT NULL,
    step_id uuid NOT NULL,
    sequence integer NOT NULL,
    name text NOT NULL,
    required_roles jsonb NOT NULL,
    minimum_approvals integer NOT NULL,
    delegation jsonb NOT NULL,
    escalation jsonb NOT NULL,

    CONSTRAINT approval_policy_step_pk PRIMARY KEY (policy_id, revision_version, step_id),
    CONSTRAINT approval_policy_step_revision_fk FOREIGN KEY (policy_id, revision_version)
        REFERENCES workflow.approval_policy_revision (policy_id, revision_version) ON DELETE RESTRICT,
    CONSTRAINT approval_policy_step_sequence_unique UNIQUE (policy_id, revision_version, sequence),
    CONSTRAINT approval_policy_step_sequence_check CHECK (sequence >= 1),
    CONSTRAINT approval_policy_step_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT approval_policy_step_minimum_check CHECK (minimum_approvals >= 1),
    CONSTRAINT approval_policy_step_roles_array CHECK (jsonb_typeof(required_roles) = 'array')
);

CREATE TABLE workflow.approval_request (
    id uuid NOT NULL,
    subject_type text NOT NULL,
    subject_id text NOT NULL,
    subject_version bigint NOT NULL,
    subject_fingerprint text NOT NULL,
    policy_id uuid NOT NULL,
    policy_version text NOT NULL,
    scope jsonb NOT NULL,
    steps jsonb NOT NULL,
    status text NOT NULL,
    aggregate_version bigint NOT NULL,
    requester_user_id uuid NOT NULL,
    requester_subject_reference text NOT NULL,
    correlation_id uuid,
    causation_id uuid,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT approval_request_pk PRIMARY KEY (id),
    CONSTRAINT approval_request_subject_version_check CHECK (subject_version >= 1),
    CONSTRAINT approval_request_fingerprint_not_blank CHECK (btrim(subject_fingerprint) <> ''),
    CONSTRAINT approval_request_status_check CHECK (status IN ('Pending', 'Approved', 'Rejected')),
    CONSTRAINT approval_request_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT approval_request_steps_array CHECK (jsonb_typeof(steps) = 'array')
);

ALTER TABLE workflow.approval_request
    ADD CONSTRAINT approval_request_policy_version_fk
    FOREIGN KEY (policy_id, policy_version)
    REFERENCES workflow.approval_policy_version_identity (policy_id, policy_version)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE workflow.approval_decision (
    id uuid NOT NULL,
    approval_request_id uuid NOT NULL,
    decision_version bigint NOT NULL,
    outcome text NOT NULL,
    approver_user_id uuid NOT NULL,
    approver_subject_reference text NOT NULL,
    reason text NOT NULL DEFAULT '',
    policy_version text NOT NULL,
    subject_version bigint NOT NULL,
    candidate_fingerprint text NOT NULL,
    recorded_at timestamptz NOT NULL,

    CONSTRAINT approval_decision_pk PRIMARY KEY (id),
    CONSTRAINT approval_decision_request_fk FOREIGN KEY (approval_request_id)
        REFERENCES workflow.approval_request (id) ON DELETE RESTRICT,
    CONSTRAINT approval_decision_identity_unique UNIQUE (approval_request_id, decision_version),
    CONSTRAINT approval_decision_version_check CHECK (decision_version >= 1),
    CONSTRAINT approval_decision_outcome_check CHECK (outcome IN ('approved', 'rejected')),
    CONSTRAINT approval_decision_subject_version_check CHECK (subject_version >= 1),
    CONSTRAINT approval_decision_fingerprint_not_blank CHECK (btrim(candidate_fingerprint) <> '')
);

-- Bootstrap is deliberately not an IAM row. It is a workflow-owned,
-- immutable authority used only to establish the first generic approval
-- authority; subsequent changes must use ApprovalRequested and
-- ApprovalDecisionRecorded.
CREATE TABLE workflow.bootstrap_approval_authority (
    authority_id uuid NOT NULL,
    authority_kind text NOT NULL,
    permission text NOT NULL,
    scope_id text NOT NULL,
    immutable boolean NOT NULL DEFAULT true,
    audit_evidence text NOT NULL,
    created_at timestamptz NOT NULL,

    CONSTRAINT bootstrap_approval_authority_pk PRIMARY KEY (authority_id),
    CONSTRAINT bootstrap_approval_authority_immutable_check CHECK (immutable),
    CONSTRAINT bootstrap_approval_authority_kind_check CHECK (authority_kind = 'bootstrap'),
    CONSTRAINT bootstrap_approval_authority_permission_check CHECK (permission = 'finance.wfa.maintain.approval.policies'),
    CONSTRAINT bootstrap_approval_authority_scope_not_blank CHECK (btrim(scope_id) <> ''),
    CONSTRAINT bootstrap_approval_authority_audit_not_blank CHECK (btrim(audit_evidence) <> '')
);

INSERT INTO workflow.bootstrap_approval_authority (
    authority_id, authority_kind, permission, scope_id, immutable, audit_evidence, created_at
) VALUES (
    '00000000-0000-0000-0000-000000000701',
    'bootstrap',
    'finance.wfa.maintain.approval.policies',
    'bootstrap-workflow-authority',
    true,
    'Controlled migration 00001 established the sole bootstrap approval authority; all later policy changes require generic WFA approval evidence.',
    clock_timestamp()
);

-- Policy and authority facts are append-only. New state is represented by a
-- new aggregate/revision row, never by rewriting an established snapshot.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION workflow.reject_immutable_workflow_fact_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'workflow approval facts are immutable';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER approval_policy_version_identity_immutable
BEFORE UPDATE OR DELETE ON workflow.approval_policy_version_identity
FOR EACH ROW EXECUTE FUNCTION workflow.reject_immutable_workflow_fact_mutation();

CREATE TRIGGER approval_policy_revision_immutable
BEFORE UPDATE OR DELETE ON workflow.approval_policy_revision
FOR EACH ROW EXECUTE FUNCTION workflow.reject_immutable_workflow_fact_mutation();

CREATE TRIGGER approval_policy_rule_immutable
BEFORE UPDATE OR DELETE ON workflow.approval_policy_rule
FOR EACH ROW EXECUTE FUNCTION workflow.reject_immutable_workflow_fact_mutation();

CREATE TRIGGER approval_policy_step_immutable
BEFORE UPDATE OR DELETE ON workflow.approval_policy_step_definition
FOR EACH ROW EXECUTE FUNCTION workflow.reject_immutable_workflow_fact_mutation();

CREATE TRIGGER approval_decision_immutable
BEFORE UPDATE OR DELETE ON workflow.approval_decision
FOR EACH ROW EXECUTE FUNCTION workflow.reject_immutable_workflow_fact_mutation();

CREATE TRIGGER bootstrap_approval_authority_immutable
BEFORE UPDATE OR DELETE ON workflow.bootstrap_approval_authority
FOR EACH ROW EXECUTE FUNCTION workflow.reject_immutable_workflow_fact_mutation();

CREATE INDEX approval_policy_status_idx ON workflow.approval_policy (status);
CREATE INDEX approval_policy_revision_effective_idx ON workflow.approval_policy_revision (status, effective_from, effective_to);
CREATE INDEX approval_policy_revision_selector_idx ON workflow.approval_policy_version_identity (selector_fingerprint, effective_from, effective_to);
CREATE INDEX approval_request_policy_idx ON workflow.approval_request (policy_id, policy_version);

-- +goose Down
ALTER TABLE workflow.approval_policy
    DROP CONSTRAINT IF EXISTS approval_policy_current_revision_fk;
ALTER TABLE workflow.approval_request
    DROP CONSTRAINT IF EXISTS approval_request_policy_version_fk;
DROP TRIGGER IF EXISTS bootstrap_approval_authority_immutable ON workflow.bootstrap_approval_authority;
DROP TRIGGER IF EXISTS approval_decision_immutable ON workflow.approval_decision;
DROP TRIGGER IF EXISTS approval_policy_step_immutable ON workflow.approval_policy_step_definition;
DROP TRIGGER IF EXISTS approval_policy_rule_immutable ON workflow.approval_policy_rule;
DROP TRIGGER IF EXISTS approval_policy_revision_immutable ON workflow.approval_policy_revision;
DROP TRIGGER IF EXISTS approval_policy_version_identity_immutable ON workflow.approval_policy_version_identity;
DROP FUNCTION IF EXISTS workflow.reject_immutable_workflow_fact_mutation();
DROP TABLE IF EXISTS workflow.bootstrap_approval_authority;
DROP TABLE IF EXISTS workflow.approval_decision;
DROP TABLE IF EXISTS workflow.approval_request;
DROP TABLE IF EXISTS workflow.approval_policy_step_definition;
DROP TABLE IF EXISTS workflow.approval_policy_rule;
DROP TABLE IF EXISTS workflow.approval_policy_revision;
DROP TABLE IF EXISTS workflow.approval_policy_version_identity;
DROP TABLE IF EXISTS workflow.approval_policy;
