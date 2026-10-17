package invoicing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

type PostgresBillingScheduleAuditWriter func(context.Context, pgx.Tx, BillingScheduleAuditRecord) (uuid.UUID, error)

type PostgresBillingScheduleRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresBillingScheduleAuditWriter
}

func NewPostgresBillingScheduleRepository(pool *pgxpool.Pool, auditWriter PostgresBillingScheduleAuditWriter) (*PostgresBillingScheduleRepository, error) {
	if pool == nil {
		return nil, ErrInvalidBillingScheduleService
	}
	return &PostgresBillingScheduleRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresBillingScheduleRepository) GetBillingSchedule(ctx context.Context, id uuid.UUID) (BillingSchedule, error) {
	if repository == nil || repository.pool == nil {
		return BillingSchedule{}, ErrInvalidBillingScheduleService
	}
	return readBillingSchedule(ctx, repository.pool, id)
}

func (repository *PostgresBillingScheduleRepository) ListBillingSchedules(ctx context.Context, scopeID *uuid.UUID) ([]BillingSchedule, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidBillingScheduleService
	}
	rows, err := repository.pool.Query(ctx, `SELECT billing_schedule_id FROM invoicing.billing_schedule WHERE ($1::uuid IS NULL OR accounting_scope_id = $1) ORDER BY next_billing_date, billing_schedule_id`, nullableUUID(scopeID))
	if err != nil {
		return nil, mapBillingSchedulePostgresError(err)
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapBillingSchedulePostgresError(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, mapBillingSchedulePostgresError(err)
	}
	rows.Close()
	result := make([]BillingSchedule, 0, len(ids))
	for _, id := range ids {
		schedule, err := readBillingSchedule(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, schedule)
	}
	return result, nil
}

func (repository *PostgresBillingScheduleRepository) CommitBillingScheduleMutation(ctx context.Context, mutation BillingScheduleMutation) error {
	return repository.commitBillingScheduleMutation(ctx, mutation, nil)
}

func (repository *PostgresBillingScheduleRepository) CommitBillingScheduleMutationWithIdempotency(ctx context.Context, mutation BillingScheduleMutation, durable DurableBillingScheduleMutationCommit) error {
	return repository.commitBillingScheduleMutation(ctx, mutation, &durable)
}

func (repository *PostgresBillingScheduleRepository) commitBillingScheduleMutation(ctx context.Context, mutation BillingScheduleMutation, durable *DurableBillingScheduleMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidBillingScheduleService
	}
	if repository.auditWriter == nil {
		return ErrBillingScheduleAuditUnavailable
	}
	if err := validateBillingScheduleMutation(mutation); err != nil {
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
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, mutation.After.AccountingScopeID.String()); err != nil {
		return mapBillingSchedulePostgresError(err)
	}
	if err := repository.ensureNoOverlap(ctx, tx, mutation.After); err != nil {
		return err
	}
	charges, err := json.Marshal(mutation.After.Charges)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO invoicing.billing_schedule (billing_schedule_id, accounting_scope_id, customer_id, contract_reference, charges, next_billing_date, time_zone, status, effective_from, effective_to, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, mutation.After.ID, mutation.After.AccountingScopeID, mutation.After.CustomerID, mutation.After.ContractReference, charges, mutation.After.NextBillingDate, mutation.After.TimeZone, mutation.After.Status, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE invoicing.billing_schedule SET customer_id=$1, contract_reference=$2, charges=$3, next_billing_date=$4, time_zone=$5, status=$6, effective_from=$7, effective_to=$8, aggregate_version=$9, revision_number=$10, updated_at=$11 WHERE billing_schedule_id=$12 AND accounting_scope_id=$13 AND aggregate_version=$14`, mutation.After.CustomerID, mutation.After.ContractReference, charges, mutation.After.NextBillingDate, mutation.After.TimeZone, mutation.After.Status, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.After.AccountingScopeID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrBillingScheduleVersionConflict
		}
	}
	if err != nil {
		return mapBillingSchedulePostgresError(err)
	}
	if err := insertBillingScheduleRevision(ctx, tx, mutation.After); err != nil {
		return err
	}
	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrBillingScheduleAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE invoicing.billing_schedule SET last_audit_reference=$1 WHERE billing_schedule_id=$2`, auditReference, mutation.After.ID); err != nil {
		return mapBillingSchedulePostgresError(err)
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidBillingScheduleService
		}
		if err := durable.Coordinator.Finalize(ctx, tx, durable.Acquisition, durable.Result); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrBillingScheduleCommitAmbiguous, err)
	}
	committed = true
	return nil
}

type billingScheduleQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readBillingSchedule(ctx context.Context, queryer billingScheduleQueryer, id uuid.UUID) (BillingSchedule, error) {
	var schedule BillingSchedule
	var effectiveTo *time.Time
	var chargesBytes []byte
	var lastAudit *uuid.UUID
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT billing_schedule_id, accounting_scope_id, customer_id, contract_reference, charges, next_billing_date, time_zone, status, effective_from, effective_to, aggregate_version, revision_number, created_at, updated_at, last_audit_reference FROM invoicing.billing_schedule WHERE billing_schedule_id=$1`, id).Scan(&schedule.ID, &schedule.AccountingScopeID, &schedule.CustomerID, &schedule.ContractReference, &chargesBytes, &schedule.NextBillingDate, &schedule.TimeZone, &schedule.Status, &schedule.EffectiveDateFrom, &effectiveTo, &version, &revision, &schedule.CreatedAt, &schedule.UpdatedAt, &lastAudit)
	if errors.Is(err, pgx.ErrNoRows) {
		return BillingSchedule{}, ErrBillingScheduleNotFound
	}
	if err != nil {
		return BillingSchedule{}, mapBillingSchedulePostgresError(err)
	}
	if err := json.Unmarshal(chargesBytes, &schedule.Charges); err != nil {
		return BillingSchedule{}, err
	}
	schedule.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return BillingSchedule{}, err
	}
	schedule.RevisionNumber = revision
	schedule.NextBillingDate = dateOnly(schedule.NextBillingDate)
	schedule.EffectiveDateFrom = dateOnly(schedule.EffectiveDateFrom)
	schedule.EffectiveDateTo = cloneDate(effectiveTo)
	if lastAudit != nil {
		schedule.LastAuditReference = *lastAudit
	}
	schedule.Revisions, err = queryBillingScheduleRevisions(ctx, queryer, id)
	if err != nil {
		return BillingSchedule{}, err
	}
	return schedule, nil
}

func queryBillingScheduleRevisions(ctx context.Context, queryer billingScheduleQueryer, id uuid.UUID) ([]BillingScheduleRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, snapshot, snapshot_fingerprint, created_at FROM invoicing.billing_schedule_revision WHERE billing_schedule_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, mapBillingSchedulePostgresError(err)
	}
	defer rows.Close()
	result := make([]BillingScheduleRevision, 0)
	for rows.Next() {
		var revision BillingScheduleRevision
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &snapshot, &revision.Fingerprint, &revision.CapturedAt); err != nil {
			return nil, mapBillingSchedulePostgresError(err)
		}
		if err := json.Unmarshal(snapshot, &revision.Snapshot); err != nil {
			return nil, err
		}
		revision.Snapshot.Charges = cloneScheduledCharges(revision.Snapshot.Charges)
		revision.Snapshot.EffectiveDateTo = cloneDate(revision.Snapshot.EffectiveDateTo)
		result = append(result, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, mapBillingSchedulePostgresError(err)
	}
	return result, nil
}

func insertBillingScheduleRevision(ctx context.Context, tx pgx.Tx, schedule BillingSchedule) error {
	snapshot, err := json.Marshal(schedule.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO invoicing.billing_schedule_revision (billing_schedule_id, revision_number, aggregate_version, snapshot, snapshot_fingerprint, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, schedule.ID, schedule.RevisionNumber, schedule.Version.Value(), snapshot, FingerprintBillingSchedule(schedule), schedule.EffectiveDateFrom, schedule.EffectiveDateTo, schedule.UpdatedAt)
	return mapBillingSchedulePostgresError(err)
}

func (repository *PostgresBillingScheduleRepository) ensureNoOverlap(ctx context.Context, tx pgx.Tx, candidate BillingSchedule) error {
	rows, err := tx.Query(ctx, `SELECT billing_schedule_id FROM invoicing.billing_schedule WHERE accounting_scope_id=$1 AND customer_id=$2 AND contract_reference=$3 AND billing_schedule_id<>$4 AND effective_from <= COALESCE($5::date, DATE '9999-12-31') AND COALESCE(effective_to, DATE '9999-12-31') >= $6::date FOR UPDATE`, candidate.AccountingScopeID, candidate.CustomerID, candidate.ContractReference, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom)
	if err != nil {
		return mapBillingSchedulePostgresError(err)
	}
	defer rows.Close()
	if rows.Next() {
		return ErrBillingScheduleDuplicate
	}
	return mapBillingSchedulePostgresError(rows.Err())
}

func mapBillingSchedulePostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "billing_schedule_contract_reference_check", "billing_schedule_charges_array_check", "billing_schedule_time_zone_check", "billing_schedule_status_check", "billing_schedule_effective_check", "billing_schedule_next_date_check", "billing_schedule_version_check", "billing_schedule_revision_check", "billing_schedule_revision_effective_check", "billing_schedule_revision_fingerprint_check":
			return ErrInvalidBillingSchedule
		}
	}
	return err
}

var _ BillingScheduleRepository = (*PostgresBillingScheduleRepository)(nil)
var _ DurableBillingScheduleRepository = (*PostgresBillingScheduleRepository)(nil)
