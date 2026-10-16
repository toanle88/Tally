package fiscalperiod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/gl"
	platformevents "github.com/toanle88/Tally/internal/platform/events"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

// PostgresAuditWriter is supplied by the audit owner. FPM passes the open
// transaction and does not write another bounded context's audit schema.
type PostgresAuditWriter func(context.Context, pgx.Tx, AuditRecord) (uuid.UUID, error)

type PostgresRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresAuditWriter
}

func NewPostgresRepository(pool *pgxpool.Pool, auditWriter PostgresAuditWriter) (*PostgresRepository, error) {
	if pool == nil {
		return nil, ErrInvalidFiscalPeriodService
	}
	return &PostgresRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresRepository) Get(ctx context.Context, scopeID, periodID uuid.UUID) (ControlState, error) {
	if repository == nil || repository.pool == nil {
		return ControlState{}, ErrInvalidFiscalPeriodService
	}
	return readControlState(ctx, repository.pool, scopeID, periodID, false)
}

func (repository *PostgresRepository) Commit(ctx context.Context, mutation Mutation) error {
	return repository.commitMutation(ctx, mutation, nil)
}

func (repository *PostgresRepository) CommitWithIdempotency(ctx context.Context, mutation Mutation, durable DurableMutationCommit) error {
	if durable.Coordinator == nil || len(durable.Acquisitions) != len(durable.Results) || len(durable.Acquisitions) == 0 {
		return ErrInvalidFiscalPeriodService
	}
	return repository.commitMutation(ctx, mutation, &durable)
}

func (repository *PostgresRepository) commitMutation(ctx context.Context, mutation Mutation, durable *DurableMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidFiscalPeriodService
	}
	if repository.auditWriter == nil {
		return ErrFiscalPeriodAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	current, err := readControlState(ctx, tx, mutation.After.Period.AccountingScopeID, mutation.After.Period.ID, true)
	if err != nil {
		return err
	}
	if !sameVersions(current, mutation.Before) {
		return ErrFiscalPeriodVersionConflict
	}
	if err := updatePeriod(ctx, tx, mutation.Before.Period, mutation.After.Period); err != nil {
		return err
	}
	if err := updateSoftCloseRun(ctx, tx, mutation.Before.SoftClose, mutation.After.SoftClose); err != nil {
		return err
	}
	if err := updateControlEpoch(ctx, tx, mutation.Before.ControlEpoch, mutation.After.ControlEpoch); err != nil {
		return err
	}
	if err := updateCloseRun(ctx, tx, mutation.Before.CloseRun, mutation.After.CloseRun); err != nil {
		return err
	}

	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrFiscalPeriodAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE fiscal_period.fiscal_period SET last_audit_reference=$1 WHERE fiscal_period_id=$2 AND accounting_scope_id=$3`, auditReference, mutation.After.Period.ID, mutation.After.Period.AccountingScopeID); err != nil {
		return mapFiscalPeriodPostgresError(err)
	}
	if err := writeEvents(ctx, tx, mutation.Events); err != nil {
		return fmt.Errorf("%w: %v", ErrFiscalPeriodOutboxUnavailable, err)
	}
	if durable != nil {
		for index, acquisition := range durable.Acquisitions {
			if err := durable.Coordinator.Finalize(ctx, tx, acquisition, durable.Results[index]); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

// Seed is intended for local fixtures and persistence tests. Production
// periods are created by the earlier period-management story.
func (repository *PostgresRepository) Seed(ctx context.Context, state ControlState) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidFiscalPeriodService
	}
	if err := state.Validate(); err != nil {
		return err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if _, err := tx.Exec(ctx, `
		INSERT INTO fiscal_period.fiscal_period (fiscal_period_id, accounting_scope_id, state, state_version, aggregate_version, active_soft_close_run_id, active_close_run_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (fiscal_period_id) DO UPDATE SET state=EXCLUDED.state, state_version=EXCLUDED.state_version, aggregate_version=EXCLUDED.aggregate_version, active_soft_close_run_id=EXCLUDED.active_soft_close_run_id, active_close_run_id=EXCLUDED.active_close_run_id, updated_at=EXCLUDED.updated_at`,
		state.Period.ID, state.Period.AccountingScopeID, state.Period.State, state.Period.StateVersion, state.Period.AggregateVersion, state.Period.ActiveSoftCloseRunID, state.Period.ActiveCloseRunID, state.Period.CreatedAt, state.Period.UpdatedAt); err != nil {
		return mapFiscalPeriodPostgresError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO fiscal_period.soft_close_run (soft_close_run_id, accounting_scope_id, fiscal_period_id, status, control_epoch, opened_gate_version, policy_version, aggregate_version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (soft_close_run_id) DO UPDATE SET status=EXCLUDED.status, control_epoch=EXCLUDED.control_epoch, opened_gate_version=EXCLUDED.opened_gate_version, policy_version=EXCLUDED.policy_version, aggregate_version=EXCLUDED.aggregate_version, updated_at=EXCLUDED.updated_at`,
		state.SoftClose.ID, state.SoftClose.AccountingScopeID, state.SoftClose.FiscalPeriodID, state.SoftClose.Status, state.SoftClose.ControlEpoch, state.SoftClose.OpenedGateVersion, state.SoftClose.PolicyVersion, state.SoftClose.AggregateVersion, state.SoftClose.CreatedAt, state.SoftClose.UpdatedAt); err != nil {
		return mapFiscalPeriodPostgresError(err)
	}
	if err := upsertControlEpoch(ctx, tx, state.ControlEpoch); err != nil {
		return err
	}
	if state.CloseRun != nil {
		if err := updateCloseRun(ctx, tx, nil, state.CloseRun); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

type fiscalPeriodQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readControlState(ctx context.Context, queryer fiscalPeriodQueryer, scopeID, periodID uuid.UUID, forUpdate bool) (ControlState, error) {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE"
	}
	var period FiscalPeriod
	var activeSoftCloseRunID, activeCloseRunID *uuid.UUID
	err := queryer.QueryRow(ctx, `SELECT fiscal_period_id, accounting_scope_id, state, state_version, aggregate_version, active_soft_close_run_id, active_close_run_id, created_at, updated_at FROM fiscal_period.fiscal_period WHERE accounting_scope_id=$1 AND fiscal_period_id=$2`+lock, scopeID, periodID).Scan(&period.ID, &period.AccountingScopeID, &period.State, &period.StateVersion, &period.AggregateVersion, &activeSoftCloseRunID, &activeCloseRunID, &period.CreatedAt, &period.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ControlState{}, ErrFiscalPeriodNotFound
	}
	if err != nil {
		return ControlState{}, mapFiscalPeriodPostgresError(err)
	}
	period.ActiveSoftCloseRunID = activeSoftCloseRunID
	period.ActiveCloseRunID = activeCloseRunID

	var run SoftCloseRun
	softCloseQuery := `SELECT soft_close_run_id, accounting_scope_id, fiscal_period_id, status, control_epoch, opened_gate_version, policy_version, aggregate_version, created_at, updated_at FROM fiscal_period.soft_close_run WHERE accounting_scope_id=$1 AND fiscal_period_id=$2`
	softCloseArgs := []any{scopeID, periodID}
	if activeSoftCloseRunID != nil {
		softCloseQuery += ` AND soft_close_run_id=$3`
		softCloseArgs = append(softCloseArgs, *activeSoftCloseRunID)
	} else {
		softCloseQuery += ` ORDER BY updated_at DESC, soft_close_run_id DESC LIMIT 1`
	}
	err = queryer.QueryRow(ctx, softCloseQuery+lock, softCloseArgs...).Scan(&run.ID, &run.AccountingScopeID, &run.FiscalPeriodID, &run.Status, &run.ControlEpoch, &run.OpenedGateVersion, &run.PolicyVersion, &run.AggregateVersion, &run.CreatedAt, &run.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ControlState{}, ErrFiscalPeriodNotFound
	}
	if err != nil {
		return ControlState{}, mapFiscalPeriodPostgresError(err)
	}

	var epoch SoftCloseControlEpoch
	var handoffCloseRunID *uuid.UUID
	var frozenSummaryJSON []byte
	err = queryer.QueryRow(ctx, `SELECT soft_close_run_id, fiscal_period_id, epoch_number, opened_gate_version, policy_version, status, handoff_close_run_id, frozen_summary, aggregate_version, created_at, updated_at FROM fiscal_period.soft_close_control_epoch WHERE soft_close_run_id=$1 AND epoch_number=$2`+lock, run.ID, run.ControlEpoch).Scan(&epoch.SoftCloseRunID, &epoch.FiscalPeriodID, &epoch.Number, &epoch.OpenedGateVersion, &epoch.PolicyVersion, &epoch.Status, &handoffCloseRunID, &frozenSummaryJSON, &epoch.AggregateVersion, &epoch.CreatedAt, &epoch.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ControlState{}, ErrFiscalPeriodNotFound
	}
	if err != nil {
		return ControlState{}, mapFiscalPeriodPostgresError(err)
	}
	epoch.HandoffCloseRunID = handoffCloseRunID
	if len(frozenSummaryJSON) > 0 && string(frozenSummaryJSON) != "null" {
		var summary gl.FrozenGateAdmissionSummary
		if err := json.Unmarshal(frozenSummaryJSON, &summary); err != nil {
			return ControlState{}, err
		}
		epoch.FrozenSummary = &summary
	}

	var closeRun CloseRun
	var barrierEvidenceJSON []byte
	err = queryer.QueryRow(ctx, `SELECT close_run_id, accounting_scope_id, fiscal_period_id, close_type, status, aggregate_version, command_fingerprint, barrier_command_id, barrier_idempotency_key, barrier_evidence, last_failure_code, last_failure_detail, created_at, updated_at FROM fiscal_period.close_run WHERE accounting_scope_id=$1 AND fiscal_period_id=$2 ORDER BY updated_at DESC, close_run_id DESC LIMIT 1`+lock, scopeID, periodID).Scan(&closeRun.ID, &closeRun.AccountingScopeID, &closeRun.FiscalPeriodID, &closeRun.CloseType, &closeRun.Status, &closeRun.AggregateVersion, &closeRun.CommandFingerprint, &closeRun.BarrierCommandID, &closeRun.BarrierIdempotencyKey, &barrierEvidenceJSON, &closeRun.LastFailureCode, &closeRun.LastFailureDetail, &closeRun.CreatedAt, &closeRun.UpdatedAt)
	var closeRunPtr *CloseRun
	if errors.Is(err, pgx.ErrNoRows) {
		closeRunPtr = nil
	} else if err != nil {
		return ControlState{}, mapFiscalPeriodPostgresError(err)
	} else {
		if len(barrierEvidenceJSON) > 0 && string(barrierEvidenceJSON) != "null" {
			var evidence BarrierEvidence
			if err := json.Unmarshal(barrierEvidenceJSON, &evidence); err != nil {
				return ControlState{}, err
			}
			closeRun.BarrierEvidence = &evidence
		}
		closeRunPtr = &closeRun
	}

	state := ControlState{Period: period, SoftClose: run, ControlEpoch: epoch, CloseRun: closeRunPtr}
	if err := state.Validate(); err != nil {
		return ControlState{}, err
	}
	return state, nil
}

func updatePeriod(ctx context.Context, tx pgx.Tx, before, after FiscalPeriod) error {
	if before == after {
		return nil
	}
	tag, err := tx.Exec(ctx, `UPDATE fiscal_period.fiscal_period SET state=$1, state_version=$2, aggregate_version=$3, active_soft_close_run_id=$4, active_close_run_id=$5, updated_at=$6 WHERE fiscal_period_id=$7 AND accounting_scope_id=$8 AND aggregate_version=$9`, after.State, after.StateVersion, after.AggregateVersion, after.ActiveSoftCloseRunID, after.ActiveCloseRunID, after.UpdatedAt, after.ID, after.AccountingScopeID, before.AggregateVersion)
	if err != nil {
		return mapFiscalPeriodPostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func updateSoftCloseRun(ctx context.Context, tx pgx.Tx, before, after SoftCloseRun) error {
	if before == after {
		return nil
	}
	tag, err := tx.Exec(ctx, `UPDATE fiscal_period.soft_close_run SET status=$1, control_epoch=$2, opened_gate_version=$3, policy_version=$4, aggregate_version=$5, updated_at=$6 WHERE soft_close_run_id=$7 AND aggregate_version=$8`, after.Status, after.ControlEpoch, after.OpenedGateVersion, after.PolicyVersion, after.AggregateVersion, after.UpdatedAt, after.ID, before.AggregateVersion)
	if err != nil {
		return mapFiscalPeriodPostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func updateControlEpoch(ctx context.Context, tx pgx.Tx, before, after SoftCloseControlEpoch) error {
	if controlEpochEqual(before, after) {
		return nil
	}
	return upsertControlEpochWithVersion(ctx, tx, before, after)
}

func controlEpochEqual(left, right SoftCloseControlEpoch) bool {
	leftData, leftErr := json.Marshal(left)
	rightData, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftData) == string(rightData)
}

func upsertControlEpoch(ctx context.Context, tx pgx.Tx, epoch SoftCloseControlEpoch) error {
	return upsertControlEpochWithVersion(ctx, tx, SoftCloseControlEpoch{}, epoch)
}

func upsertControlEpochWithVersion(ctx context.Context, tx pgx.Tx, before, after SoftCloseControlEpoch) error {
	var frozenSummary any
	if after.FrozenSummary != nil {
		value, err := json.Marshal(after.FrozenSummary)
		if err != nil {
			return err
		}
		frozenSummary = value
	}
	if before.SoftCloseRunID == uuid.Nil {
		_, err := tx.Exec(ctx, `INSERT INTO fiscal_period.soft_close_control_epoch (soft_close_run_id, fiscal_period_id, epoch_number, opened_gate_version, policy_version, status, handoff_close_run_id, frozen_summary, aggregate_version, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, after.SoftCloseRunID, after.FiscalPeriodID, after.Number, after.OpenedGateVersion, after.PolicyVersion, after.Status, after.HandoffCloseRunID, frozenSummary, after.AggregateVersion, after.CreatedAt, after.UpdatedAt)
		return mapFiscalPeriodPostgresError(err)
	}
	tag, err := tx.Exec(ctx, `UPDATE fiscal_period.soft_close_control_epoch SET status=$1, handoff_close_run_id=$2, frozen_summary=$3, aggregate_version=$4, updated_at=$5 WHERE soft_close_run_id=$6 AND epoch_number=$7 AND aggregate_version=$8`, after.Status, after.HandoffCloseRunID, frozenSummary, after.AggregateVersion, after.UpdatedAt, after.SoftCloseRunID, after.Number, before.AggregateVersion)
	if err != nil {
		return mapFiscalPeriodPostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func updateCloseRun(ctx context.Context, tx pgx.Tx, before, after *CloseRun) error {
	if before == nil && after == nil {
		return nil
	}
	if before == nil {
		evidence, err := nullableJSON(after.BarrierEvidence)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO fiscal_period.close_run (close_run_id, accounting_scope_id, fiscal_period_id, close_type, status, aggregate_version, command_fingerprint, barrier_command_id, barrier_idempotency_key, barrier_evidence, last_failure_code, last_failure_detail, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, after.ID, after.AccountingScopeID, after.FiscalPeriodID, after.CloseType, after.Status, after.AggregateVersion, after.CommandFingerprint, after.BarrierCommandID, after.BarrierIdempotencyKey, evidence, nullableText(after.LastFailureCode), nullableText(after.LastFailureDetail), after.CreatedAt, after.UpdatedAt)
		return mapFiscalPeriodPostgresError(err)
	}
	if after == nil {
		return nil
	}
	evidence, err := nullableJSON(after.BarrierEvidence)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE fiscal_period.close_run SET status=$1, aggregate_version=$2, barrier_evidence=$3, last_failure_code=$4, last_failure_detail=$5, updated_at=$6 WHERE close_run_id=$7 AND aggregate_version=$8`, after.Status, after.AggregateVersion, evidence, nullableText(after.LastFailureCode), nullableText(after.LastFailureDetail), after.UpdatedAt, after.ID, before.AggregateVersion)
	if err != nil {
		return mapFiscalPeriodPostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func writeEvents(ctx context.Context, tx pgx.Tx, eventList []Event) error {
	for _, item := range eventList {
		payload := append([]byte(nil), item.Payload...)
		fingerprint, err := platformevents.ComputePayloadFingerprint(payload)
		if err != nil {
			return err
		}
		event, err := platformevents.NewEnvelope(platformevents.EnvelopeInput{
			MessageID: uuid.NewString(), EventType: item.EventType, EventVersion: 1, OccurredAt: item.OccurredAt,
			SourceContext: "fiscal_period", AggregateID: item.AggregateID.String(), AggregateVersion: item.AggregateVersion,
			AccountingScopeID: item.AccountingScopeID.String(), CorrelationID: item.CorrelationID, CausationID: item.CausationID,
			DataClassification: platformevents.Internal, PayloadFingerprint: fingerprint, Data: payload,
		})
		if err != nil {
			return err
		}
		if err := platformintegration.WritePublication(ctx, tx, platformintegration.Publication{Event: event, AvailableAt: item.OccurredAt}); err != nil {
			return err
		}
	}
	return nil
}

func nullableJSON(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func mapFiscalPeriodPostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrFiscalPeriodOwnerConflict
		case "23514":
			return ErrFiscalPeriodStateConflict
		}
	}
	return err
}

var _ Repository = (*PostgresRepository)(nil)
var _ DurableRepository = (*PostgresRepository)(nil)
