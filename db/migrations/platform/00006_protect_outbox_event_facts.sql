-- +goose Up
CREATE INDEX integration_outbox_replay_idx
    ON integration.outbox (source_context, created_at, outbox_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION integration.prevent_outbox_event_fact_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.outbox_id IS DISTINCT FROM OLD.outbox_id
       OR NEW.event_type IS DISTINCT FROM OLD.event_type
       OR NEW.event_version IS DISTINCT FROM OLD.event_version
       OR NEW.occurred_at IS DISTINCT FROM OLD.occurred_at
       OR NEW.source_context IS DISTINCT FROM OLD.source_context
       OR NEW.aggregate_id IS DISTINCT FROM OLD.aggregate_id
       OR NEW.aggregate_version IS DISTINCT FROM OLD.aggregate_version
       OR NEW.accounting_scope_id IS DISTINCT FROM OLD.accounting_scope_id
       OR NEW.correlation_id IS DISTINCT FROM OLD.correlation_id
       OR NEW.causation_id IS DISTINCT FROM OLD.causation_id
       OR NEW.payload IS DISTINCT FROM OLD.payload
       OR NEW.payload_fingerprint IS DISTINCT FROM OLD.payload_fingerprint
       OR NEW.data_classification IS DISTINCT FROM OLD.data_classification
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'integration outbox event facts are immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER integration_outbox_event_facts_immutable
    BEFORE UPDATE ON integration.outbox
    FOR EACH ROW
    EXECUTE FUNCTION integration.prevent_outbox_event_fact_mutation();

-- Replay projections are intentionally local. The transaction-local marker is
-- set by the replay coordinator and prevents trusted replay effects from
-- accidentally creating new integration work through a raw pgx transaction.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION integration.prevent_replay_outbox_publication()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF current_setting('tally.replay', true) = 'on' THEN
        RAISE EXCEPTION 'replay transactions cannot publish integration events';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER integration_outbox_replay_publication_guard
    BEFORE INSERT ON integration.outbox
    FOR EACH ROW
    EXECUTE FUNCTION integration.prevent_replay_outbox_publication();

-- +goose Down
DROP TRIGGER IF EXISTS integration_outbox_replay_publication_guard ON integration.outbox;
DROP FUNCTION IF EXISTS integration.prevent_replay_outbox_publication();
DROP TRIGGER IF EXISTS integration_outbox_event_facts_immutable ON integration.outbox;
DROP FUNCTION IF EXISTS integration.prevent_outbox_event_fact_mutation();
DROP INDEX IF EXISTS integration.integration_outbox_replay_idx;
