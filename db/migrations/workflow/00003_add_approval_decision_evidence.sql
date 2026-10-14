-- +goose Up
ALTER TABLE workflow.approval_decision
    ADD COLUMN step_id uuid,
    ADD COLUMN delegation_reference text NOT NULL DEFAULT '';

ALTER TABLE workflow.approval_decision
    ADD CONSTRAINT approval_decision_delegation_reference_length CHECK (length(delegation_reference) <= 240);

-- +goose Down
ALTER TABLE workflow.approval_decision
    DROP CONSTRAINT IF EXISTS approval_decision_delegation_reference_length,
    DROP COLUMN IF EXISTS delegation_reference,
    DROP COLUMN IF EXISTS step_id;
