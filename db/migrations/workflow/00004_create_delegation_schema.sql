-- +goose Up
CREATE TABLE workflow.delegation (
    delegation_id uuid NOT NULL,
    delegator_user_id uuid NOT NULL,
    delegate_user_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    policy_version text NOT NULL,
    scope jsonb NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_to timestamptz NOT NULL,
    aggregate_version bigint NOT NULL,
    audit_reference uuid NOT NULL,
    correlation_id uuid,
    causation_id uuid,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,

    CONSTRAINT delegation_pk PRIMARY KEY (delegation_id),
    CONSTRAINT delegation_identity_distinct CHECK (delegator_user_id <> delegate_user_id),
    CONSTRAINT delegation_policy_version_fk FOREIGN KEY (policy_id, policy_version)
        REFERENCES workflow.approval_policy_version_identity (policy_id, policy_version)
        ON DELETE RESTRICT,
    CONSTRAINT delegation_scope_object CHECK (jsonb_typeof(scope) = 'object'),
    CONSTRAINT delegation_interval_check CHECK (effective_to > effective_from),
    CONSTRAINT delegation_version_check CHECK (aggregate_version >= 1),
    CONSTRAINT delegation_audit_not_nil CHECK (audit_reference <> '00000000-0000-0000-0000-000000000000')
);

CREATE TRIGGER delegation_immutable
BEFORE UPDATE OR DELETE ON workflow.delegation
FOR EACH ROW EXECUTE FUNCTION workflow.reject_immutable_workflow_fact_mutation();

CREATE INDEX delegation_policy_effective_idx
    ON workflow.delegation (policy_id, policy_version, effective_from, effective_to);
CREATE INDEX delegation_delegate_effective_idx
    ON workflow.delegation (delegate_user_id, effective_from, effective_to);
CREATE INDEX delegation_delegator_idx
    ON workflow.delegation (delegator_user_id, created_at DESC);
CREATE INDEX delegation_scope_idx
    ON workflow.delegation USING gin ((scope -> 'scopeIds'));

-- +goose Down
DROP INDEX IF EXISTS workflow.delegation_scope_idx;
DROP INDEX IF EXISTS workflow.delegation_delegator_idx;
DROP INDEX IF EXISTS workflow.delegation_delegate_effective_idx;
DROP INDEX IF EXISTS workflow.delegation_policy_effective_idx;
DROP TRIGGER IF EXISTS delegation_immutable ON workflow.delegation;
DROP TABLE IF EXISTS workflow.delegation;
