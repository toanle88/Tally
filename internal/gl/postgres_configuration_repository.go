package gl

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

type PostgresLedgerAuditWriter func(context.Context, pgx.Tx, LedgerAuditRecord) (uuid.UUID, error)
type PostgresAccountingBookAuditWriter func(context.Context, pgx.Tx, AccountingBookAuditRecord) (uuid.UUID, error)

type PostgresConfigurationRepository struct {
	pool                      *pgxpool.Pool
	ledgerAuditWriter         PostgresLedgerAuditWriter
	accountingBookAuditWriter PostgresAccountingBookAuditWriter
}

func NewPostgresConfigurationRepository(pool *pgxpool.Pool, ledgerAuditWriter PostgresLedgerAuditWriter, accountingBookAuditWriter PostgresAccountingBookAuditWriter) (*PostgresConfigurationRepository, error) {
	if pool == nil {
		return nil, ErrInvalidLedgerService
	}
	return &PostgresConfigurationRepository{pool: pool, ledgerAuditWriter: ledgerAuditWriter, accountingBookAuditWriter: accountingBookAuditWriter}, nil
}

func (repository *PostgresConfigurationRepository) GetLedger(ctx context.Context, id uuid.UUID) (Ledger, error) {
	if repository == nil || repository.pool == nil {
		return Ledger{}, ErrInvalidLedgerService
	}
	return readLedger(ctx, repository.pool, id)
}

func (repository *PostgresConfigurationRepository) ListLedgers(ctx context.Context, scopeID *uuid.UUID) ([]Ledger, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidLedgerService
	}
	rows, err := repository.pool.Query(ctx, `SELECT ledger_id FROM gl.ledger WHERE ($1::uuid IS NULL OR accounting_scope_id = $1) ORDER BY ledger_id`, nullableUUID(scopeID))
	if err != nil {
		return nil, mapGLPostgresError(err)
	}
	defer rows.Close()
	result := make([]Ledger, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapGLPostgresError(err)
		}
		ledger, err := readLedger(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, ledger)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGLPostgresError(err)
	}
	sortLedgers(result)
	return result, nil
}

func (repository *PostgresConfigurationRepository) GetAccountingBook(ctx context.Context, id uuid.UUID) (AccountingBook, error) {
	if repository == nil || repository.pool == nil {
		return AccountingBook{}, ErrInvalidAccountingBookService
	}
	return readAccountingBook(ctx, repository.pool, id)
}

func (repository *PostgresConfigurationRepository) ListAccountingBooks(ctx context.Context, scopeID *uuid.UUID) ([]AccountingBook, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidAccountingBookService
	}
	rows, err := repository.pool.Query(ctx, `SELECT accounting_book_id FROM gl.accounting_book WHERE ($1::uuid IS NULL OR accounting_scope_id = $1) ORDER BY accounting_book_id`, nullableUUID(scopeID))
	if err != nil {
		return nil, mapGLPostgresError(err)
	}
	defer rows.Close()
	result := make([]AccountingBook, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapGLPostgresError(err)
		}
		book, err := readAccountingBook(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, book)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGLPostgresError(err)
	}
	sortAccountingBooks(result)
	return result, nil
}

func (repository *PostgresConfigurationRepository) CommitLedgerMutation(ctx context.Context, mutation LedgerMutation) error {
	return repository.commitLedger(ctx, mutation, nil)
}

func (repository *PostgresConfigurationRepository) CommitLedgerMutationWithIdempotency(ctx context.Context, mutation LedgerMutation, durable DurableLedgerMutationCommit) error {
	return repository.commitLedger(ctx, mutation, &durable)
}

func (repository *PostgresConfigurationRepository) CommitAccountingBookMutation(ctx context.Context, mutation AccountingBookMutation) error {
	return repository.commitAccountingBook(ctx, mutation, nil)
}

func (repository *PostgresConfigurationRepository) CommitAccountingBookMutationWithIdempotency(ctx context.Context, mutation AccountingBookMutation, durable DurableAccountingBookMutationCommit) error {
	return repository.commitAccountingBook(ctx, mutation, &durable)
}

func (repository *PostgresConfigurationRepository) commitLedger(ctx context.Context, mutation LedgerMutation, durable *DurableLedgerMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidLedgerService
	}
	if repository.ledgerAuditWriter == nil {
		return ErrLedgerAuditUnavailable
	}
	if err := validateLedgerMutation(mutation); err != nil {
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
		return mapGLPostgresError(err)
	}
	if err := repository.ensureNoLedgerOverlap(ctx, tx, mutation.After); err != nil {
		return err
	}
	approval, err := approvalJSON(mutation.After.Approval)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO gl.ledger (ledger_id, accounting_scope_id, legal_entity_id, ledger_type, functional_currency, fiscal_calendar_id, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, mutation.After.ID, mutation.After.AccountingScopeID, mutation.After.LegalEntityID, mutation.After.LedgerType, mutation.After.FunctionalCurrency, mutation.After.FiscalCalendarID, mutation.After.LifecycleStatus, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE gl.ledger SET legal_entity_id=$1, ledger_type=$2, functional_currency=$3, fiscal_calendar_id=$4, lifecycle_status=$5, effective_from=$6, effective_to=$7, approval_reference=$8, aggregate_version=$9, revision_number=$10, updated_at=$11 WHERE ledger_id=$12 AND accounting_scope_id=$13 AND aggregate_version=$14`, mutation.After.LegalEntityID, mutation.After.LedgerType, mutation.After.FunctionalCurrency, mutation.After.FiscalCalendarID, mutation.After.LifecycleStatus, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.After.AccountingScopeID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrLedgerVersionConflict
		}
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	if err := insertLedgerRevision(ctx, tx, mutation.After); err != nil {
		return err
	}
	auditReference, err := repository.ledgerAuditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrLedgerAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE gl.ledger SET last_audit_reference=$1 WHERE ledger_id=$2`, auditReference, mutation.After.ID); err != nil {
		return mapGLPostgresError(err)
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidLedgerService
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

func (repository *PostgresConfigurationRepository) commitAccountingBook(ctx context.Context, mutation AccountingBookMutation, durable *DurableAccountingBookMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidAccountingBookService
	}
	if repository.accountingBookAuditWriter == nil {
		return ErrAccountingBookAuditUnavailable
	}
	if err := validateAccountingBookMutation(mutation); err != nil {
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
		return mapGLPostgresError(err)
	}
	if err := ensureAccountingBookParent(ctx, tx, mutation.After); err != nil {
		return err
	}
	if err := repository.ensureNoAccountingBookOverlap(ctx, tx, mutation.After); err != nil {
		return err
	}
	approval, err := approvalJSON(mutation.After.Approval)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO gl.accounting_book (accounting_book_id, accounting_scope_id, ledger_id, book_type, accounting_basis, posting_policy_version, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, mutation.After.ID, mutation.After.AccountingScopeID, mutation.After.LedgerID, mutation.After.BookType, mutation.After.AccountingBasis, mutation.After.PostingPolicyVersion, mutation.After.LifecycleStatus, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE gl.accounting_book SET ledger_id=$1, book_type=$2, accounting_basis=$3, posting_policy_version=$4, lifecycle_status=$5, effective_from=$6, effective_to=$7, approval_reference=$8, aggregate_version=$9, revision_number=$10, updated_at=$11 WHERE accounting_book_id=$12 AND accounting_scope_id=$13 AND aggregate_version=$14`, mutation.After.LedgerID, mutation.After.BookType, mutation.After.AccountingBasis, mutation.After.PostingPolicyVersion, mutation.After.LifecycleStatus, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.After.AccountingScopeID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrAccountingBookVersionConflict
		}
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	if err := insertAccountingBookRevision(ctx, tx, mutation.After); err != nil {
		return err
	}
	auditReference, err := repository.accountingBookAuditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrAccountingBookAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE gl.accounting_book SET last_audit_reference=$1 WHERE accounting_book_id=$2`, auditReference, mutation.After.ID); err != nil {
		return mapGLPostgresError(err)
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidAccountingBookService
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

type glQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readLedger(ctx context.Context, queryer glQueryer, id uuid.UUID) (Ledger, error) {
	var ledger Ledger
	var effectiveTo *time.Time
	var approvalBytes []byte
	var lastAudit *uuid.UUID
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT ledger_id, accounting_scope_id, legal_entity_id, ledger_type, functional_currency, fiscal_calendar_id, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at, last_audit_reference FROM gl.ledger WHERE ledger_id=$1`, id).Scan(&ledger.ID, &ledger.AccountingScopeID, &ledger.LegalEntityID, &ledger.LedgerType, &ledger.FunctionalCurrency, &ledger.FiscalCalendarID, &ledger.LifecycleStatus, &ledger.EffectiveDateFrom, &effectiveTo, &approvalBytes, &version, &revision, &ledger.CreatedAt, &ledger.UpdatedAt, &lastAudit)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ledger{}, ErrLedgerNotFound
	}
	if err != nil {
		return Ledger{}, mapGLPostgresError(err)
	}
	ledger.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return Ledger{}, err
	}
	ledger.RevisionNumber = revision
	ledger.EffectiveDateFrom = dateOnly(ledger.EffectiveDateFrom)
	ledger.EffectiveDateTo = cloneDate(effectiveTo)
	if lastAudit != nil {
		ledger.LastAuditReference = *lastAudit
	}
	if len(approvalBytes) > 0 && string(approvalBytes) != "null" {
		var approval ApprovalDecisionReference
		if err := json.Unmarshal(approvalBytes, &approval); err != nil {
			return Ledger{}, err
		}
		ledger.Approval = &approval
	}
	ledger.Revisions, err = queryLedgerRevisions(ctx, queryer, id)
	if err != nil {
		return Ledger{}, err
	}
	return ledger, nil
}

func readAccountingBook(ctx context.Context, queryer glQueryer, id uuid.UUID) (AccountingBook, error) {
	var book AccountingBook
	var effectiveTo *time.Time
	var approvalBytes []byte
	var lastAudit *uuid.UUID
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT accounting_book_id, accounting_scope_id, ledger_id, book_type, accounting_basis, posting_policy_version, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at, last_audit_reference FROM gl.accounting_book WHERE accounting_book_id=$1`, id).Scan(&book.ID, &book.AccountingScopeID, &book.LedgerID, &book.BookType, &book.AccountingBasis, &book.PostingPolicyVersion, &book.LifecycleStatus, &book.EffectiveDateFrom, &effectiveTo, &approvalBytes, &version, &revision, &book.CreatedAt, &book.UpdatedAt, &lastAudit)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountingBook{}, ErrAccountingBookNotFound
	}
	if err != nil {
		return AccountingBook{}, mapGLPostgresError(err)
	}
	book.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return AccountingBook{}, err
	}
	book.RevisionNumber = revision
	book.EffectiveDateFrom = dateOnly(book.EffectiveDateFrom)
	book.EffectiveDateTo = cloneDate(effectiveTo)
	if lastAudit != nil {
		book.LastAuditReference = *lastAudit
	}
	if len(approvalBytes) > 0 && string(approvalBytes) != "null" {
		var approval ApprovalDecisionReference
		if err := json.Unmarshal(approvalBytes, &approval); err != nil {
			return AccountingBook{}, err
		}
		book.Approval = &approval
	}
	book.Revisions, err = queryAccountingBookRevisions(ctx, queryer, id)
	if err != nil {
		return AccountingBook{}, err
	}
	return book, nil
}

func queryLedgerRevisions(ctx context.Context, queryer glQueryer, id uuid.UUID) ([]LedgerRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM gl.ledger_revision WHERE ledger_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, mapGLPostgresError(err)
	}
	defer rows.Close()
	result := make([]LedgerRevision, 0)
	for rows.Next() {
		var revision LedgerRevision
		var version int64
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &version, &snapshot, &revision.CreatedAt); err != nil {
			return nil, mapGLPostgresError(err)
		}
		var err error
		revision.Version, err = aggregateversion.FromInt64(version)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snapshot, &revision.Snapshot); err != nil {
			return nil, err
		}
		if revision.Snapshot.EffectiveDateTo != nil {
			revision.Snapshot.EffectiveDateTo = cloneDate(revision.Snapshot.EffectiveDateTo)
		}
		result = append(result, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGLPostgresError(err)
	}
	return result, nil
}

func queryAccountingBookRevisions(ctx context.Context, queryer glQueryer, id uuid.UUID) ([]AccountingBookRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM gl.accounting_book_revision WHERE accounting_book_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, mapGLPostgresError(err)
	}
	defer rows.Close()
	result := make([]AccountingBookRevision, 0)
	for rows.Next() {
		var revision AccountingBookRevision
		var version int64
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &version, &snapshot, &revision.CreatedAt); err != nil {
			return nil, mapGLPostgresError(err)
		}
		var err error
		revision.Version, err = aggregateversion.FromInt64(version)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snapshot, &revision.Snapshot); err != nil {
			return nil, err
		}
		if revision.Snapshot.EffectiveDateTo != nil {
			revision.Snapshot.EffectiveDateTo = cloneDate(revision.Snapshot.EffectiveDateTo)
		}
		result = append(result, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGLPostgresError(err)
	}
	return result, nil
}

func insertLedgerRevision(ctx context.Context, tx pgx.Tx, ledger Ledger) error {
	snapshot, err := json.Marshal(ledger.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO gl.ledger_revision (ledger_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, ledger.ID, ledger.RevisionNumber, ledger.Version.Value(), snapshot, ledger.EffectiveDateFrom, ledger.EffectiveDateTo, ledger.UpdatedAt)
	return mapGLPostgresError(err)
}

func insertAccountingBookRevision(ctx context.Context, tx pgx.Tx, book AccountingBook) error {
	snapshot, err := json.Marshal(book.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO gl.accounting_book_revision (accounting_book_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, book.ID, book.RevisionNumber, book.Version.Value(), snapshot, book.EffectiveDateFrom, book.EffectiveDateTo, book.UpdatedAt)
	return mapGLPostgresError(err)
}

func (repository *PostgresConfigurationRepository) ensureNoLedgerOverlap(ctx context.Context, tx pgx.Tx, candidate Ledger) error {
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT ledger_id FROM gl.ledger WHERE accounting_scope_id=$1 AND legal_entity_id=$2 AND ledger_type=$3 AND functional_currency=$4 AND fiscal_calendar_id=$5 AND ledger_id<>$6 AND effective_from <= COALESCE($7::date, DATE '9999-12-31') AND COALESCE(effective_to, DATE '9999-12-31') >= $8::date LIMIT 1 FOR UPDATE`, candidate.AccountingScopeID, candidate.LegalEntityID, candidate.LedgerType, candidate.FunctionalCurrency, candidate.FiscalCalendarID, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	return ErrLedgerDuplicate
}

func (repository *PostgresConfigurationRepository) ensureNoAccountingBookOverlap(ctx context.Context, tx pgx.Tx, candidate AccountingBook) error {
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT accounting_book_id FROM gl.accounting_book WHERE accounting_scope_id=$1 AND ledger_id=$2 AND book_type=$3 AND accounting_basis=$4 AND accounting_book_id<>$5 AND effective_from <= COALESCE($6::date, DATE '9999-12-31') AND COALESCE(effective_to, DATE '9999-12-31') >= $7::date LIMIT 1 FOR UPDATE`, candidate.AccountingScopeID, candidate.LedgerID, candidate.BookType, candidate.AccountingBasis, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	return ErrAccountingBookDuplicate
}

func ensureAccountingBookParent(ctx context.Context, tx pgx.Tx, candidate AccountingBook) error {
	var parentScopeID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT accounting_scope_id FROM gl.ledger WHERE ledger_id=$1 FOR SHARE`, candidate.LedgerID).Scan(&parentScopeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAccountingBookReferenceInvalid
		}
		return mapGLPostgresError(err)
	}
	if parentScopeID != candidate.AccountingScopeID {
		return ErrAccountingBookReferenceInvalid
	}
	return nil
}

func approvalJSON(value *ApprovalDecisionReference) (any, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func mapGLPostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "ledger_effective_identity_unique":
			return ErrLedgerDuplicate
		case "accounting_book_effective_identity_unique":
			return ErrAccountingBookDuplicate
		case "accounting_book_ledger_fk":
			return ErrAccountingBookReferenceInvalid
		case "chart_of_accounts_effective_identity_unique":
			return ErrChartOfAccountsDuplicate
		case "chart_of_accounts_ledger_fk":
			return ErrChartOfAccountsReferenceInvalid
		case "account_effective_identity_unique":
			return ErrAccountDuplicate
		case "account_chart_fk":
			return ErrAccountReferenceInvalid
		case "ledger_status_check", "ledger_currency_check", "ledger_effective_check", "ledger_type_check":
			return ErrInvalidLedger
		case "accounting_book_status_check", "accounting_book_effective_check", "accounting_book_type_check":
			return ErrInvalidAccountingBook
		case "chart_of_accounts_status_check", "chart_of_accounts_policy_check", "chart_of_accounts_effective_check":
			return ErrInvalidChartOfAccounts
		case "account_status_check", "account_normal_balance_check", "account_code_check", "account_name_check", "account_type_check", "account_currency_policy_check", "account_effective_check", "account_restrictions_array_check", "account_reporting_mappings_array_check":
			return ErrInvalidAccount
		}
	}
	return err
}

var _ LedgerRepository = (*PostgresConfigurationRepository)(nil)
var _ AccountingBookRepository = (*PostgresConfigurationRepository)(nil)
var _ DurableLedgerRepository = (*PostgresConfigurationRepository)(nil)
var _ DurableAccountingBookRepository = (*PostgresConfigurationRepository)(nil)
