package gl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	platformevents "github.com/toanle88/Tally/internal/platform/events"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

// PostgresSoftCloseGateAuditWriter is supplied by the audit owner. GL passes
// the open transaction and never writes another bounded context's audit table.
type PostgresSoftCloseGateAuditWriter func(context.Context, pgx.Tx, SoftCloseGateAuditRecord) (uuid.UUID, error)

type PostgresSoftCloseGateRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresSoftCloseGateAuditWriter
}

func NewPostgresSoftCloseGateRepository(pool *pgxpool.Pool, auditWriter PostgresSoftCloseGateAuditWriter) (*PostgresSoftCloseGateRepository, error) {
	if pool == nil || auditWriter == nil {
		return nil, ErrInvalidSoftCloseGateService
	}
	return &PostgresSoftCloseGateRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresSoftCloseGateRepository) GetSoftCloseGate(ctx context.Context, scopeID, periodID uuid.UUID) (PeriodPostingGate, error) {
	if repository == nil || repository.pool == nil {
		return PeriodPostingGate{}, ErrSoftCloseGateReferenceUnavailable
	}
	return repository.getSoftCloseGate(ctx, repository.pool, scopeID, periodID, false)
}

func (repository *PostgresSoftCloseGateRepository) GetPostingGateStatus(ctx context.Context, scopeID, periodID uuid.UUID) (PeriodPostingGate, error) {
	return repository.GetSoftCloseGate(ctx, scopeID, periodID)
}

func (repository *PostgresSoftCloseGateRepository) CommitSoftCloseGateMutation(ctx context.Context, mutation SoftCloseGateMutation) (SoftCloseGateResult, error) {
	return repository.commitSoftCloseGateMutation(ctx, mutation, nil)
}

func (repository *PostgresSoftCloseGateRepository) CommitSoftCloseGateMutationWithIdempotency(ctx context.Context, mutation SoftCloseGateMutation, commit DurableSoftCloseGateCommit) (SoftCloseGateResult, error) {
	return repository.commitSoftCloseGateMutation(ctx, mutation, &commit)
}

func (repository *PostgresSoftCloseGateRepository) commitSoftCloseGateMutation(ctx context.Context, mutation SoftCloseGateMutation, durable *DurableSoftCloseGateCommit) (SoftCloseGateResult, error) {
	if repository == nil || repository.pool == nil || repository.auditWriter == nil {
		return SoftCloseGateResult{}, ErrInvalidSoftCloseGateService
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	current, err := repository.getSoftCloseGate(ctx, tx, mutation.Before.AccountingScopeID, mutation.Before.FiscalPeriodID, true)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	if current.ID != mutation.Before.ID || current.GateVersion != mutation.Before.GateVersion {
		return SoftCloseGateResult{}, ErrSoftCloseGateVersionConflict
	}
	if mutation.After.ID != current.ID || mutation.After.GateVersion != current.GateVersion+1 {
		return SoftCloseGateResult{}, ErrSoftCloseGateVersionConflict
	}

	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	if auditReference == uuid.Nil {
		return SoftCloseGateResult{}, ErrSoftCloseGateAuditUnavailable
	}
	if err := repository.updateSoftCloseGate(ctx, tx, mutation); err != nil {
		return SoftCloseGateResult{}, err
	}

	result := SoftCloseGateResult{
		Operation:                  mutation.Operation,
		Gate:                       clonePeriodPostingGate(mutation.After),
		DecisionReference:          mutation.Authorization.DecisionReference,
		PolicyReference:            mutation.Authorization.PolicyReference,
		AuthorizationPolicyVersion: mutation.Authorization.PolicyVersion,
		ValidationOutcome:          "valid",
		AuditReference:             auditReference,
		NextAction:                 softCloseGateNextAction(mutation.Operation),
	}
	if err := repository.writeSoftCloseGateEvent(ctx, tx, mutation, result); err != nil {
		return SoftCloseGateResult{}, fmt.Errorf("%w: %v", ErrSoftCloseGateOutboxUnavailable, err)
	}
	if durable != nil {
		if durable.Coordinator == nil || durable.Acquisition.Decision() != platformidempotency.DecisionExecute {
			return SoftCloseGateResult{}, platformidempotency.ErrInvalidAcquisition
		}
		body, err := json.Marshal(result)
		if err != nil {
			return SoftCloseGateResult{}, err
		}
		status := 200
		aggregateID := result.AggregateID()
		processID := postingGateProcessID(mutation.After)
		metadata, err := platformidempotency.NewCommandResultMetadata(
			durable.Acquisition.Result().Identity(),
			durable.Acquisition.Result().Fingerprint(),
			durable.Acquisition.Result().OperationID(),
			platformidempotency.StateEstablished, &status, body, &aggregateID, processID,
		)
		if err != nil {
			return SoftCloseGateResult{}, err
		}
		if err := durable.Coordinator.Finalize(ctx, tx, durable.Acquisition, metadata); err != nil {
			return SoftCloseGateResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SoftCloseGateResult{}, err
	}
	committed = true
	return result, nil
}

type softCloseGateDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (repository *PostgresSoftCloseGateRepository) getSoftCloseGate(ctx context.Context, db softCloseGateDB, scopeID, periodID uuid.UUID, forUpdate bool) (PeriodPostingGate, error) {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE"
	}
	var (
		gateID, accountingScopeID, fiscalPeriodUUID                                                                  uuid.UUID
		mode, policyVersion, activeCloseType, priorControlProcessType, priorGateMode, finalizationCommandFingerprint string
		periodStateVersion, gateVersion, aggregateVersion, nextLedgerPosition, barrierLedgerPosition                 int64
		activeRunID, activeCloseRunID, priorControlProcessID, finalizedCloseRunID                                    *uuid.UUID
		activeEpoch, priorControlEpoch, finalLedgerWatermark                                                         *int64
		restrictions, countersJSON                                                                                   []byte
		lastSummaryJSON, historyJSON                                                                                 []byte
		createdAt, updatedAt                                                                                         time.Time
	)
	query := `SELECT period_posting_gate_id, accounting_scope_id, fiscal_period_id, gate_mode, period_state_version, gate_version, aggregate_version, next_ledger_position, active_soft_close_run_id, active_soft_close_control_epoch, COALESCE(soft_close_policy_version, ''), soft_close_restrictions, active_close_run_id, COALESCE(active_close_type, ''), prior_control_process_id, COALESCE(prior_control_process_type, ''), prior_control_epoch, COALESCE(prior_gate_mode, ''), barrier_ledger_position, final_ledger_watermark, finalized_close_run_id, COALESCE(finalization_command_fingerprint, ''), active_gate_admission_counters, last_frozen_admission_summary, frozen_admission_summary_history, created_at, updated_at FROM gl.period_posting_gate WHERE accounting_scope_id = $1 AND fiscal_period_id = $2` + lock
	err := db.QueryRow(ctx, query, scopeID, periodID).Scan(&gateID, &accountingScopeID, &fiscalPeriodUUID, &mode, &periodStateVersion, &gateVersion, &aggregateVersion, &nextLedgerPosition, &activeRunID, &activeEpoch, &policyVersion, &restrictions, &activeCloseRunID, &activeCloseType, &priorControlProcessID, &priorControlProcessType, &priorControlEpoch, &priorGateMode, &barrierLedgerPosition, &finalLedgerWatermark, &finalizedCloseRunID, &finalizationCommandFingerprint, &countersJSON, &lastSummaryJSON, &historyJSON, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return PeriodPostingGate{}, ErrSoftCloseGateNotFound
	}
	if err != nil {
		return PeriodPostingGate{}, mapPostingPostgresError(err)
	}
	gate := PeriodPostingGate{ID: gateID, AccountingScopeID: accountingScopeID, FiscalPeriodID: fiscalPeriodUUID, GateMode: mode, PeriodStateVersion: periodStateVersion, GateVersion: gateVersion, AggregateVersion: aggregateVersion, NextLedgerPosition: nextLedgerPosition, ActiveSoftCloseRunID: activeRunID, SoftClosePolicyVersion: policyVersion, SoftCloseRestrictions: append(json.RawMessage(nil), restrictions...), ActiveCloseRunID: activeCloseRunID, ActiveCloseType: activeCloseType, PriorControlProcessID: priorControlProcessID, PriorControlProcessType: priorControlProcessType, PriorControlEpoch: priorControlEpoch, PriorGateMode: priorGateMode, BarrierLedgerPosition: barrierLedgerPosition, FinalLedgerWatermark: finalLedgerWatermark, FinalizedCloseRunID: finalizedCloseRunID, FinalizationCommandFingerprint: finalizationCommandFingerprint, CreatedAt: createdAt.UTC(), UpdatedAt: updatedAt.UTC()}
	if activeEpoch != nil {
		gate.ActiveSoftCloseControlEpoch = *activeEpoch
	}
	if len(countersJSON) > 0 {
		if err := json.Unmarshal(countersJSON, &gate.ActiveAdmissionCounters); err != nil {
			return PeriodPostingGate{}, err
		}
	}
	if len(lastSummaryJSON) > 0 && string(lastSummaryJSON) != "null" {
		var summary FrozenGateAdmissionSummary
		if err := json.Unmarshal(lastSummaryJSON, &summary); err != nil {
			return PeriodPostingGate{}, err
		}
		gate.LastFrozenAdmissionSummary = &summary
	}
	if len(historyJSON) > 0 {
		if err := json.Unmarshal(historyJSON, &gate.FrozenAdmissionSummaryHistory); err != nil {
			return PeriodPostingGate{}, err
		}
	}
	if gate.SoftCloseRestrictions == nil {
		gate.SoftCloseRestrictions = json.RawMessage(`{}`)
	}
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	return gate, nil
}

func (repository *PostgresSoftCloseGateRepository) updateSoftCloseGate(ctx context.Context, tx pgx.Tx, mutation SoftCloseGateMutation) error {
	countersJSON, err := json.Marshal(mutation.After.ActiveAdmissionCounters)
	if err != nil {
		return err
	}
	var lastSummaryJSON any
	if mutation.After.LastFrozenAdmissionSummary != nil {
		lastSummaryJSON, err = json.Marshal(mutation.After.LastFrozenAdmissionSummary)
		if err != nil {
			return err
		}
	}
	history := mutation.After.FrozenAdmissionSummaryHistory
	if history == nil {
		history = []FrozenGateAdmissionSummary{}
	}
	historyJSON, err := json.Marshal(history)
	if err != nil {
		return err
	}
	var policyVersion any
	if mutation.After.SoftClosePolicyVersion != "" {
		policyVersion = mutation.After.SoftClosePolicyVersion
	}
	var activeRunID any
	if mutation.After.ActiveSoftCloseRunID != nil {
		activeRunID = *mutation.After.ActiveSoftCloseRunID
	}
	var activeEpoch any
	if mutation.After.ActiveSoftCloseControlEpoch > 0 {
		activeEpoch = mutation.After.ActiveSoftCloseControlEpoch
	}
	var activeCloseRunID any
	if mutation.After.ActiveCloseRunID != nil {
		activeCloseRunID = *mutation.After.ActiveCloseRunID
	}
	var activeCloseType any
	if mutation.After.ActiveCloseType != "" {
		activeCloseType = mutation.After.ActiveCloseType
	}
	var priorControlProcessID any
	if mutation.After.PriorControlProcessID != nil {
		priorControlProcessID = *mutation.After.PriorControlProcessID
	}
	var priorControlProcessType any
	if mutation.After.PriorControlProcessType != "" {
		priorControlProcessType = mutation.After.PriorControlProcessType
	}
	var priorControlEpoch any
	if mutation.After.PriorControlEpoch != nil {
		priorControlEpoch = *mutation.After.PriorControlEpoch
	}
	var priorGateMode any
	if mutation.After.PriorGateMode != "" {
		priorGateMode = mutation.After.PriorGateMode
	}
	var finalLedgerWatermark any
	if mutation.After.FinalLedgerWatermark != nil {
		finalLedgerWatermark = *mutation.After.FinalLedgerWatermark
	}
	var finalizedCloseRunID any
	if mutation.After.FinalizedCloseRunID != nil {
		finalizedCloseRunID = *mutation.After.FinalizedCloseRunID
	}
	var finalizationCommandFingerprint any
	if mutation.After.FinalizationCommandFingerprint != "" {
		finalizationCommandFingerprint = mutation.After.FinalizationCommandFingerprint
	}
	commandTag, err := tx.Exec(ctx, `UPDATE gl.period_posting_gate SET gate_mode = $3, period_state_version = $4, gate_version = $5, aggregate_version = $6, active_soft_close_run_id = $7, active_soft_close_control_epoch = $8, soft_close_policy_version = $9, soft_close_restrictions = $10, active_close_run_id = $11, active_close_type = $12, prior_control_process_id = $13, prior_control_process_type = $14, prior_control_epoch = $15, prior_gate_mode = $16, barrier_ledger_position = $17, final_ledger_watermark = $18, finalized_close_run_id = $19, finalization_command_fingerprint = $20, active_gate_admission_counters = $21, last_frozen_admission_summary = $22, frozen_admission_summary_history = $23, updated_at = $24 WHERE accounting_scope_id = $1 AND fiscal_period_id = $2 AND gate_version = $25`, mutation.After.AccountingScopeID, mutation.After.FiscalPeriodID, mutation.After.GateMode, mutation.After.PeriodStateVersion, mutation.After.GateVersion, mutation.After.AggregateVersion, activeRunID, activeEpoch, policyVersion, mutation.After.SoftCloseRestrictions, activeCloseRunID, activeCloseType, priorControlProcessID, priorControlProcessType, priorControlEpoch, priorGateMode, mutation.After.BarrierLedgerPosition, finalLedgerWatermark, finalizedCloseRunID, finalizationCommandFingerprint, countersJSON, lastSummaryJSON, historyJSON, mutation.After.UpdatedAt, mutation.Before.GateVersion)
	if err := mapPostingPostgresError(err); err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return ErrSoftCloseGateVersionConflict
	}
	return nil
}

func (repository *PostgresSoftCloseGateRepository) writeSoftCloseGateEvent(ctx context.Context, tx pgx.Tx, mutation SoftCloseGateMutation, result SoftCloseGateResult) error {
	payload, err := json.Marshal(struct {
		Operation          string            `json:"operation"`
		Gate               PeriodPostingGate `json:"gate"`
		AuditReference     uuid.UUID         `json:"auditReference"`
		CommandFingerprint string            `json:"commandFingerprint"`
	}{mutation.Operation, result.Gate, result.AuditReference, mutation.CommandFingerprint})
	if err != nil {
		return err
	}
	payloadFingerprint, err := platformevents.ComputePayloadFingerprint(payload)
	if err != nil {
		return err
	}
	eventType := softCloseGateEventType(mutation.Operation)
	event, err := platformevents.NewEnvelope(platformevents.EnvelopeInput{MessageID: uuid.NewString(), EventType: eventType, EventVersion: 1, OccurredAt: mutation.Now, SourceContext: "gl", AggregateID: result.AggregateID().String(), AggregateVersion: result.AggregateVersion(), AccountingScopeID: mutation.After.AccountingScopeID.String(), CorrelationID: mutation.Audit.CorrelationID, CausationID: mutation.Audit.CausationID, DataClassification: platformevents.Internal, PayloadFingerprint: payloadFingerprint, Data: payload})
	if err != nil {
		return err
	}
	return platformintegration.WritePublication(ctx, tx, platformintegration.Publication{Event: event, AvailableAt: mutation.Now})
}
