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

type PostgresChartOfAccountsAuditWriter func(context.Context, pgx.Tx, ChartOfAccountsAuditRecord) (uuid.UUID, error)
type PostgresAccountAuditWriter func(context.Context, pgx.Tx, AccountAuditRecord) (uuid.UUID, error)

type PostgresChartAccountRepository struct {
	pool               *pgxpool.Pool
	chartAuditWriter   PostgresChartOfAccountsAuditWriter
	accountAuditWriter PostgresAccountAuditWriter
}

func NewPostgresChartAccountRepository(pool *pgxpool.Pool, chartAuditWriter PostgresChartOfAccountsAuditWriter, accountAuditWriter PostgresAccountAuditWriter) (*PostgresChartAccountRepository, error) {
	if pool == nil {
		return nil, ErrInvalidChartOfAccountsService
	}
	return &PostgresChartAccountRepository{pool: pool, chartAuditWriter: chartAuditWriter, accountAuditWriter: accountAuditWriter}, nil
}

func (repository *PostgresChartAccountRepository) GetChartOfAccounts(ctx context.Context, id uuid.UUID) (ChartOfAccounts, error) {
	if repository == nil || repository.pool == nil {
		return ChartOfAccounts{}, ErrInvalidChartOfAccountsService
	}
	return readChartOfAccounts(ctx, repository.pool, id)
}

func (repository *PostgresChartAccountRepository) ListChartsOfAccounts(ctx context.Context, scopeID *uuid.UUID) ([]ChartOfAccounts, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidChartOfAccountsService
	}
	rows, err := repository.pool.Query(ctx, `SELECT chart_of_accounts_id FROM gl.chart_of_accounts WHERE ($1::uuid IS NULL OR accounting_scope_id = $1) ORDER BY chart_of_accounts_id`, nullableUUID(scopeID))
	if err != nil {
		return nil, mapGLPostgresError(err)
	}
	defer rows.Close()
	result := make([]ChartOfAccounts, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapGLPostgresError(err)
		}
		chart, err := readChartOfAccounts(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, chart)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGLPostgresError(err)
	}
	sortChartsOfAccounts(result)
	return result, nil
}

func (repository *PostgresChartAccountRepository) CommitChartOfAccountsMutation(ctx context.Context, mutation ChartOfAccountsMutation) error {
	return repository.commitChartOfAccounts(ctx, mutation, nil)
}

func (repository *PostgresChartAccountRepository) CommitChartOfAccountsMutationWithIdempotency(ctx context.Context, mutation ChartOfAccountsMutation, durable DurableChartOfAccountsMutationCommit) error {
	return repository.commitChartOfAccounts(ctx, mutation, &durable)
}

func (repository *PostgresChartAccountRepository) GetAccount(ctx context.Context, id uuid.UUID) (Account, error) {
	if repository == nil || repository.pool == nil {
		return Account{}, ErrInvalidAccountService
	}
	return readAccount(ctx, repository.pool, id)
}

func (repository *PostgresChartAccountRepository) ListAccounts(ctx context.Context, scopeID *uuid.UUID) ([]Account, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidAccountService
	}
	rows, err := repository.pool.Query(ctx, `SELECT account_id FROM gl.account WHERE ($1::uuid IS NULL OR accounting_scope_id = $1) ORDER BY account_id`, nullableUUID(scopeID))
	if err != nil {
		return nil, mapGLPostgresError(err)
	}
	defer rows.Close()
	result := make([]Account, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapGLPostgresError(err)
		}
		account, err := readAccount(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, account)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGLPostgresError(err)
	}
	sortAccounts(result)
	return result, nil
}

func (repository *PostgresChartAccountRepository) CommitAccountMutation(ctx context.Context, mutation AccountMutation) error {
	return repository.commitAccount(ctx, mutation, nil)
}

func (repository *PostgresChartAccountRepository) CommitAccountMutationWithIdempotency(ctx context.Context, mutation AccountMutation, durable DurableAccountMutationCommit) error {
	return repository.commitAccount(ctx, mutation, &durable)
}

func (repository *PostgresChartAccountRepository) commitChartOfAccounts(ctx context.Context, mutation ChartOfAccountsMutation, durable *DurableChartOfAccountsMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidChartOfAccountsService
	}
	if repository.chartAuditWriter == nil {
		return ErrChartOfAccountsAuditUnavailable
	}
	if err := validateChartOfAccountsMutation(mutation); err != nil {
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
	if err := ensureChartOfAccountsParent(ctx, tx, mutation.After); err != nil {
		return err
	}
	if err := ensureNoChartOfAccountsOverlap(ctx, tx, mutation.After); err != nil {
		return err
	}
	approval, err := approvalJSON(mutation.After.Approval)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO gl.chart_of_accounts (chart_of_accounts_id, accounting_scope_id, ledger_id, account_code_policy, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, mutation.After.ID, mutation.After.AccountingScopeID, mutation.After.LedgerID, mutation.After.AccountCodePolicy, mutation.After.LifecycleStatus, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE gl.chart_of_accounts SET account_code_policy=$1, lifecycle_status=$2, effective_from=$3, effective_to=$4, approval_reference=$5, aggregate_version=$6, revision_number=$7, updated_at=$8 WHERE chart_of_accounts_id=$9 AND accounting_scope_id=$10 AND aggregate_version=$11`, mutation.After.AccountCodePolicy, mutation.After.LifecycleStatus, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.After.AccountingScopeID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrChartOfAccountsVersionConflict
		}
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	if err := insertChartOfAccountsRevision(ctx, tx, mutation.After); err != nil {
		return err
	}
	auditReference, err := repository.chartAuditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrChartOfAccountsAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE gl.chart_of_accounts SET last_audit_reference=$1 WHERE chart_of_accounts_id=$2`, auditReference, mutation.After.ID); err != nil {
		return mapGLPostgresError(err)
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidChartOfAccountsService
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

func (repository *PostgresChartAccountRepository) commitAccount(ctx context.Context, mutation AccountMutation, durable *DurableAccountMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidAccountService
	}
	if repository.accountAuditWriter == nil {
		return ErrAccountAuditUnavailable
	}
	if err := validateAccountMutation(mutation); err != nil {
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
	if err := ensureAccountParent(ctx, tx, mutation.After); err != nil {
		return err
	}
	if err := ensureNoAccountOverlap(ctx, tx, mutation.After); err != nil {
		return err
	}
	restrictions, err := json.Marshal(mutation.After.Restrictions)
	if err != nil {
		return err
	}
	mappings, err := json.Marshal(mutation.After.ReportingMappings)
	if err != nil {
		return err
	}
	approval, err := approvalJSON(mutation.After.Approval)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO gl.account (account_id, accounting_scope_id, chart_of_accounts_id, account_code, account_name, account_type, normal_balance, lifecycle_status, restrictions, currency_policy, reporting_mappings, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`, mutation.After.ID, mutation.After.AccountingScopeID, mutation.After.ChartOfAccountsID, mutation.After.AccountCode, mutation.After.AccountName, mutation.After.AccountType, mutation.After.NormalBalance, mutation.After.LifecycleStatus, restrictions, mutation.After.CurrencyPolicy, mappings, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE gl.account SET account_code=$1, account_name=$2, account_type=$3, normal_balance=$4, lifecycle_status=$5, restrictions=$6, currency_policy=$7, reporting_mappings=$8, effective_from=$9, effective_to=$10, approval_reference=$11, aggregate_version=$12, revision_number=$13, updated_at=$14 WHERE account_id=$15 AND accounting_scope_id=$16 AND aggregate_version=$17`, mutation.After.AccountCode, mutation.After.AccountName, mutation.After.AccountType, mutation.After.NormalBalance, mutation.After.LifecycleStatus, restrictions, mutation.After.CurrencyPolicy, mappings, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.After.AccountingScopeID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrAccountVersionConflict
		}
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	if err := insertAccountRevision(ctx, tx, mutation.After); err != nil {
		return err
	}
	auditReference, err := repository.accountAuditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrAccountAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE gl.account SET last_audit_reference=$1 WHERE account_id=$2`, auditReference, mutation.After.ID); err != nil {
		return mapGLPostgresError(err)
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidAccountService
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

func readChartOfAccounts(ctx context.Context, queryer glQueryer, id uuid.UUID) (ChartOfAccounts, error) {
	var chart ChartOfAccounts
	var effectiveTo *time.Time
	var approvalBytes []byte
	var lastAudit *uuid.UUID
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT chart_of_accounts_id, accounting_scope_id, ledger_id, account_code_policy, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at, last_audit_reference FROM gl.chart_of_accounts WHERE chart_of_accounts_id=$1`, id).Scan(&chart.ID, &chart.AccountingScopeID, &chart.LedgerID, &chart.AccountCodePolicy, &chart.LifecycleStatus, &chart.EffectiveDateFrom, &effectiveTo, &approvalBytes, &version, &revision, &chart.CreatedAt, &chart.UpdatedAt, &lastAudit)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChartOfAccounts{}, ErrChartOfAccountsNotFound
	}
	if err != nil {
		return ChartOfAccounts{}, mapGLPostgresError(err)
	}
	chart.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return ChartOfAccounts{}, err
	}
	chart.RevisionNumber = revision
	chart.EffectiveDateFrom = dateOnly(chart.EffectiveDateFrom)
	chart.EffectiveDateTo = cloneDate(effectiveTo)
	if lastAudit != nil {
		chart.LastAuditReference = *lastAudit
	}
	if len(approvalBytes) > 0 && string(approvalBytes) != "null" {
		var approval ApprovalDecisionReference
		if err := json.Unmarshal(approvalBytes, &approval); err != nil {
			return ChartOfAccounts{}, err
		}
		chart.Approval = &approval
	}
	chart.Revisions, err = queryChartOfAccountsRevisions(ctx, queryer, id)
	if err != nil {
		return ChartOfAccounts{}, err
	}
	return chart, nil
}

func readAccount(ctx context.Context, queryer glQueryer, id uuid.UUID) (Account, error) {
	var account Account
	var effectiveTo *time.Time
	var approvalBytes, restrictionsBytes, mappingsBytes []byte
	var lastAudit *uuid.UUID
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT account_id, accounting_scope_id, chart_of_accounts_id, account_code, account_name, account_type, normal_balance, lifecycle_status, restrictions, currency_policy, reporting_mappings, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at, last_audit_reference FROM gl.account WHERE account_id=$1`, id).Scan(&account.ID, &account.AccountingScopeID, &account.ChartOfAccountsID, &account.AccountCode, &account.AccountName, &account.AccountType, &account.NormalBalance, &account.LifecycleStatus, &restrictionsBytes, &account.CurrencyPolicy, &mappingsBytes, &account.EffectiveDateFrom, &effectiveTo, &approvalBytes, &version, &revision, &account.CreatedAt, &account.UpdatedAt, &lastAudit)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, mapGLPostgresError(err)
	}
	account.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return Account{}, err
	}
	account.RevisionNumber = revision
	account.EffectiveDateFrom = dateOnly(account.EffectiveDateFrom)
	account.EffectiveDateTo = cloneDate(effectiveTo)
	if lastAudit != nil {
		account.LastAuditReference = *lastAudit
	}
	if len(approvalBytes) > 0 && string(approvalBytes) != "null" {
		var approval ApprovalDecisionReference
		if err := json.Unmarshal(approvalBytes, &approval); err != nil {
			return Account{}, err
		}
		account.Approval = &approval
	}
	if len(restrictionsBytes) > 0 {
		if err := json.Unmarshal(restrictionsBytes, &account.Restrictions); err != nil {
			return Account{}, err
		}
	}
	if len(mappingsBytes) > 0 {
		if err := json.Unmarshal(mappingsBytes, &account.ReportingMappings); err != nil {
			return Account{}, err
		}
	}
	account.Revisions, err = queryAccountRevisions(ctx, queryer, id)
	if err != nil {
		return Account{}, err
	}
	return account, nil
}

func queryChartOfAccountsRevisions(ctx context.Context, queryer glQueryer, id uuid.UUID) ([]ChartOfAccountsRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM gl.chart_of_accounts_revision WHERE chart_of_accounts_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, mapGLPostgresError(err)
	}
	defer rows.Close()
	result := make([]ChartOfAccountsRevision, 0)
	for rows.Next() {
		var revision ChartOfAccountsRevision
		var version int64
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &version, &snapshot, &revision.CreatedAt); err != nil {
			return nil, mapGLPostgresError(err)
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
	if err := rows.Err(); err != nil {
		return nil, mapGLPostgresError(err)
	}
	return result, nil
}

func queryAccountRevisions(ctx context.Context, queryer glQueryer, id uuid.UUID) ([]AccountRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM gl.account_revision WHERE account_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, mapGLPostgresError(err)
	}
	defer rows.Close()
	result := make([]AccountRevision, 0)
	for rows.Next() {
		var revision AccountRevision
		var version int64
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &version, &snapshot, &revision.CreatedAt); err != nil {
			return nil, mapGLPostgresError(err)
		}
		revision.Version, err = aggregateversion.FromInt64(version)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snapshot, &revision.Snapshot); err != nil {
			return nil, err
		}
		revision.Snapshot.Restrictions = cloneAccountRestrictions(revision.Snapshot.Restrictions)
		revision.Snapshot.ReportingMappings = cloneAccountReportingMappings(revision.Snapshot.ReportingMappings)
		revision.Snapshot.EffectiveDateTo = cloneDate(revision.Snapshot.EffectiveDateTo)
		result = append(result, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGLPostgresError(err)
	}
	return result, nil
}

func insertChartOfAccountsRevision(ctx context.Context, tx pgx.Tx, chart ChartOfAccounts) error {
	snapshot, err := json.Marshal(chart.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO gl.chart_of_accounts_revision (chart_of_accounts_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, chart.ID, chart.RevisionNumber, chart.Version.Value(), snapshot, chart.EffectiveDateFrom, chart.EffectiveDateTo, chart.UpdatedAt)
	return mapGLPostgresError(err)
}

func insertAccountRevision(ctx context.Context, tx pgx.Tx, account Account) error {
	snapshot, err := json.Marshal(account.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO gl.account_revision (account_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, account.ID, account.RevisionNumber, account.Version.Value(), snapshot, account.EffectiveDateFrom, account.EffectiveDateTo, account.UpdatedAt)
	return mapGLPostgresError(err)
}

func ensureChartOfAccountsParent(ctx context.Context, tx pgx.Tx, candidate ChartOfAccounts) error {
	var scopeID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT accounting_scope_id FROM gl.ledger WHERE ledger_id=$1 FOR SHARE`, candidate.LedgerID).Scan(&scopeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrChartOfAccountsReferenceInvalid
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	if scopeID != candidate.AccountingScopeID {
		return ErrChartOfAccountsReferenceInvalid
	}
	return nil
}

func ensureAccountParent(ctx context.Context, tx pgx.Tx, candidate Account) error {
	var scopeID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT accounting_scope_id FROM gl.chart_of_accounts WHERE chart_of_accounts_id=$1 FOR SHARE`, candidate.ChartOfAccountsID).Scan(&scopeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAccountReferenceInvalid
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	if scopeID != candidate.AccountingScopeID {
		return ErrAccountReferenceInvalid
	}
	return nil
}

func ensureNoChartOfAccountsOverlap(ctx context.Context, tx pgx.Tx, candidate ChartOfAccounts) error {
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT chart_of_accounts_id FROM gl.chart_of_accounts WHERE accounting_scope_id=$1 AND ledger_id=$2 AND account_code_policy=$3 AND chart_of_accounts_id<>$4 AND effective_from <= COALESCE($5::date, DATE '9999-12-31') AND COALESCE(effective_to, DATE '9999-12-31') >= $6::date LIMIT 1 FOR UPDATE`, candidate.AccountingScopeID, candidate.LedgerID, candidate.AccountCodePolicy, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	return ErrChartOfAccountsDuplicate
}

func ensureNoAccountOverlap(ctx context.Context, tx pgx.Tx, candidate Account) error {
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT account_id FROM gl.account WHERE accounting_scope_id=$1 AND chart_of_accounts_id=$2 AND account_code=$3 AND account_id<>$4 AND effective_from <= COALESCE($5::date, DATE '9999-12-31') AND COALESCE(effective_to, DATE '9999-12-31') >= $6::date LIMIT 1 FOR UPDATE`, candidate.AccountingScopeID, candidate.ChartOfAccountsID, candidate.AccountCode, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return mapGLPostgresError(err)
	}
	return ErrAccountDuplicate
}

var _ ChartOfAccountsRepository = (*PostgresChartAccountRepository)(nil)
var _ AccountRepository = (*PostgresChartAccountRepository)(nil)
var _ DurableChartOfAccountsRepository = (*PostgresChartAccountRepository)(nil)
var _ DurableAccountRepository = (*PostgresChartAccountRepository)(nil)
