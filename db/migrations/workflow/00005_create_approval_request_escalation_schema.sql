-- +goose Up
CREATE TABLE workflow.approval_request_escalation (
    id uuid NOT NULL,
    approval_request_id uuid NOT NULL,
    step_id uuid NOT NULL,
    reason text NOT NULL,
    blocker text NOT NULL DEFAULT '',
    responsible_roles jsonb NOT NULL,
    target_roles jsonb NOT NULL,
    actor_user_id uuid,
    actor_subject_reference text NOT NULL DEFAULT '',
    process_reference text NOT NULL DEFAULT '',
    policy_id uuid NOT NULL,
    policy_version text NOT NULL,
    subject_version bigint NOT NULL,
    candidate_fingerprint text NOT NULL,
    due_at timestamptz,
    occurred_at timestamptz NOT NULL,
    audit_reference uuid NOT NULL,
    correlation_id uuid,
    causation_id uuid,

    CONSTRAINT approval_request_escalation_pk PRIMARY KEY (id),
    CONSTRAINT approval_request_escalation_request_fk FOREIGN KEY (approval_request_id)
        REFERENCES workflow.approval_request (id) ON DELETE RESTRICT,
    CONSTRAINT approval_request_escalation_policy_version_fk FOREIGN KEY (policy_id, policy_version)
        REFERENCES workflow.approval_policy_version_identity (policy_id, policy_version) ON DELETE RESTRICT,
    CONSTRAINT approval_request_escalation_step_unique UNIQUE (approval_request_id, step_id),
    CONSTRAINT approval_request_escalation_reason_check CHECK (reason IN ('overdue', 'blocked')),
    CONSTRAINT approval_request_escalation_blocker_check CHECK (reason = 'overdue' OR btrim(blocker) <> ''),
    CONSTRAINT approval_request_escalation_blocker_length CHECK (length(blocker) <= 500),
    CONSTRAINT approval_request_escalation_subject_version_check CHECK (subject_version >= 1),
    CONSTRAINT approval_request_escalation_fingerprint_check CHECK (btrim(candidate_fingerprint) <> '' AND length(candidate_fingerprint) <= 240),
    CONSTRAINT approval_request_escalation_roles_check CHECK (
        jsonb_typeof(responsible_roles) = 'array'
        AND jsonb_typeof(target_roles) = 'array'
        AND jsonb_array_length(target_roles) > 0
    ),
    CONSTRAINT approval_request_escalation_actor_check CHECK (
        (actor_user_id IS NOT NULL AND btrim(actor_subject_reference) <> '' AND btrim(process_reference) = '')
        OR (actor_user_id IS NULL AND btrim(actor_subject_reference) = '' AND btrim(process_reference) <> '')
    ),
    CONSTRAINT approval_request_escalation_audit_not_nil CHECK (audit_reference <> '00000000-0000-0000-0000-000000000000')
);

CREATE TRIGGER approval_request_escalation_immutable
BEFORE UPDATE OR DELETE ON workflow.approval_request_escalation
FOR EACH ROW EXECUTE FUNCTION workflow.reject_immutable_workflow_fact_mutation();

CREATE INDEX approval_request_escalation_request_idx
    ON workflow.approval_request_escalation (approval_request_id, occurred_at, id);
CREATE INDEX approval_request_escalation_step_idx
    ON workflow.approval_request_escalation (step_id, occurred_at DESC);

-- +goose Down
DROP INDEX IF EXISTS workflow.approval_request_escalation_step_idx;
DROP INDEX IF EXISTS workflow.approval_request_escalation_request_idx;
DROP TABLE IF EXISTS workflow.approval_request_escalation;
