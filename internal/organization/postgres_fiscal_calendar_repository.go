package organization

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

type PostgresFiscalCalendarAuditWriter func(context.Context, pgx.Tx, FiscalCalendarAuditRecord) (uuid.UUID, error)

type PostgresFiscalCalendarRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresFiscalCalendarAuditWriter
}

func NewPostgresFiscalCalendarRepository(pool *pgxpool.Pool, auditWriter PostgresFiscalCalendarAuditWriter) (*PostgresFiscalCalendarRepository, error) {
	if pool == nil {
		return nil, ErrInvalidFiscalCalendarService
	}
	return &PostgresFiscalCalendarRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresFiscalCalendarRepository) Get(ctx context.Context, id uuid.UUID) (FiscalCalendar, error) {
	if repository == nil || repository.pool == nil {
		return FiscalCalendar{}, ErrInvalidFiscalCalendarService
	}
	return readFiscalCalendar(ctx, repository.pool, id)
}

func (repository *PostgresFiscalCalendarRepository) List(ctx context.Context, scopeID *uuid.UUID) ([]FiscalCalendar, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidFiscalCalendarService
	}
	rows, err := repository.pool.Query(ctx, `SELECT id FROM organization.fiscal_calendar WHERE ($1::uuid IS NULL OR scope_id = $1) ORDER BY id`, nullableUUID(scopeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]FiscalCalendar, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		calendar, err := readFiscalCalendar(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, calendar)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (repository *PostgresFiscalCalendarRepository) CommitFiscalCalendarMutation(ctx context.Context, mutation FiscalCalendarMutation) error {
	return repository.commit(ctx, mutation, nil)
}

func (repository *PostgresFiscalCalendarRepository) CommitFiscalCalendarMutationWithIdempotency(ctx context.Context, mutation FiscalCalendarMutation, durable DurableFiscalCalendarMutationCommit) error {
	return repository.commit(ctx, mutation, &durable)
}

func (repository *PostgresFiscalCalendarRepository) commit(ctx context.Context, mutation FiscalCalendarMutation, durable *DurableFiscalCalendarMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidFiscalCalendarService
	}
	if repository.auditWriter == nil {
		return ErrFiscalCalendarAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrFiscalCalendarVersionConflict
		}
	} else if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrFiscalCalendarVersionConflict
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

	approval, err := fiscalCalendarApprovalJSON(mutation.After.Approval)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO organization.fiscal_calendar (id, scope_id, calendar_type, period_pattern, status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, mutation.After.ID, mutation.After.ScopeID, mutation.After.CalendarType, mutation.After.PeriodPattern, mutation.After.Status, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE organization.fiscal_calendar SET calendar_type=$1, period_pattern=$2, status=$3, effective_from=$4, effective_to=$5, approval_reference=$6, aggregate_version=$7, revision_number=$8, updated_at=$9 WHERE id=$10 AND scope_id=$11 AND aggregate_version=$12`, mutation.After.CalendarType, mutation.After.PeriodPattern, mutation.After.Status, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.After.ScopeID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrFiscalCalendarVersionConflict
		}
		if err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM organization.fiscal_calendar_period WHERE fiscal_calendar_id=$1`, mutation.After.ID)
		}
	}
	if err != nil {
		return mapFiscalCalendarPostgresError(err)
	}
	for _, period := range mutation.After.Periods {
		_, err = tx.Exec(ctx, `INSERT INTO organization.fiscal_calendar_period (fiscal_calendar_id, period_id, period_reference, period_ordinal, start_date, end_date) VALUES ($1,$2,$3,$4,$5,$6)`, mutation.After.ID, period.ID, period.Reference, period.Ordinal, period.StartDate, period.EndDate)
		if err != nil {
			return mapFiscalCalendarPostgresError(err)
		}
	}
	snapshot, err := json.Marshal(mutation.After.Snapshot())
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO organization.fiscal_calendar_revision (fiscal_calendar_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, mutation.After.ID, mutation.After.RevisionNumber, mutation.After.Version.Value(), snapshot, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, mutation.After.UpdatedAt); err != nil {
		return mapFiscalCalendarPostgresError(err)
	}
	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrFiscalCalendarAuditUnavailable
	}
	if _, err = tx.Exec(ctx, `UPDATE organization.fiscal_calendar SET last_audit_reference=$1 WHERE id=$2`, auditReference, mutation.After.ID); err != nil {
		return err
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidFiscalCalendarService
		}
		if err := durable.Coordinator.Finalize(ctx, tx, durable.Acquisition, durable.Result); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func fiscalCalendarApprovalJSON(value *ApprovalDecisionReference) (any, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readFiscalCalendar(ctx context.Context, queryer pgxQueryer, id uuid.UUID) (FiscalCalendar, error) {
	var calendar FiscalCalendar
	var effectiveTo *time.Time
	var approvalJSON []byte
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT id, scope_id, calendar_type, period_pattern, status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at FROM organization.fiscal_calendar WHERE id=$1`, id).Scan(&calendar.ID, &calendar.ScopeID, &calendar.CalendarType, &calendar.PeriodPattern, &calendar.Status, &calendar.EffectiveFrom, &effectiveTo, &approvalJSON, &version, &revision, &calendar.CreatedAt, &calendar.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return FiscalCalendar{}, ErrFiscalCalendarNotFound
	}
	if err != nil {
		return FiscalCalendar{}, err
	}
	calendar.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return FiscalCalendar{}, err
	}
	calendar.RevisionNumber = revision
	calendar.EffectiveFrom = dateOnly(calendar.EffectiveFrom)
	if effectiveTo != nil {
		calendar.EffectiveTo = ptrDate(*effectiveTo)
	}
	if len(approvalJSON) > 0 && string(approvalJSON) != "null" {
		var approval ApprovalDecisionReference
		if err := json.Unmarshal(approvalJSON, &approval); err != nil {
			return FiscalCalendar{}, err
		}
		calendar.Approval = &approval
	}
	calendar.Periods, err = queryFiscalCalendarPeriods(ctx, queryer, id)
	if err != nil {
		return FiscalCalendar{}, err
	}
	calendar.Revisions, err = queryFiscalCalendarRevisions(ctx, queryer, id)
	if err != nil {
		return FiscalCalendar{}, err
	}
	if err := calendar.Validate(); err != nil {
		return FiscalCalendar{}, err
	}
	return calendar, nil
}

func queryFiscalCalendarPeriods(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]CalendarPeriod, error) {
	rows, err := queryer.Query(ctx, `SELECT period_id, period_reference, period_ordinal, start_date, end_date FROM organization.fiscal_calendar_period WHERE fiscal_calendar_id=$1 ORDER BY period_ordinal`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]CalendarPeriod, 0)
	for rows.Next() {
		var period CalendarPeriod
		if err := rows.Scan(&period.ID, &period.Reference, &period.Ordinal, &period.StartDate, &period.EndDate); err != nil {
			return nil, err
		}
		period.Reference = canonicalFiscalCalendarCode(period.Reference)
		period.StartDate, period.EndDate = dateOnly(period.StartDate), dateOnly(period.EndDate)
		result = append(result, period)
	}
	return result, rows.Err()
}

func queryFiscalCalendarRevisions(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]FiscalCalendarRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM organization.fiscal_calendar_revision WHERE fiscal_calendar_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]FiscalCalendarRevision, 0)
	for rows.Next() {
		var revision FiscalCalendarRevision
		var version int64
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &version, &snapshot, &revision.CreatedAt); err != nil {
			return nil, err
		}
		revision.Version, err = aggregateversion.FromInt64(version)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snapshot, &revision.Snapshot); err != nil {
			return nil, err
		}
		result = append(result, revision)
	}
	return result, rows.Err()
}

func mapFiscalCalendarPostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrFiscalCalendarDuplicate
		case "23514":
			return ErrInvalidFiscalCalendar
		}
	}
	return err
}

var _ FiscalCalendarRepository = (*PostgresFiscalCalendarRepository)(nil)
var _ DurableFiscalCalendarRepository = (*PostgresFiscalCalendarRepository)(nil)
