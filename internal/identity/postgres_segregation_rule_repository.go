package identity

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/toanle88/Tally/internal/identity/identitydb"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

type PostgresSegregationRuleAuditWriter func(context.Context, pgx.Tx, SegregationRuleAuditRecord) (uuid.UUID, error)

type PostgresSegregationRuleRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresSegregationRuleAuditWriter
}

func NewPostgresSegregationRuleRepository(pool *pgxpool.Pool) (*PostgresSegregationRuleRepository, error) {
	if pool == nil {
		return nil, ErrSegregationRuleUnavailable
	}
	return &PostgresSegregationRuleRepository{pool: pool}, nil
}

func NewPostgresSegregationRuleRepositoryWithAudit(pool *pgxpool.Pool, auditWriter PostgresSegregationRuleAuditWriter) (*PostgresSegregationRuleRepository, error) {
	repository, err := NewPostgresSegregationRuleRepository(pool)
	if err != nil {
		return nil, err
	}
	repository.auditWriter = auditWriter
	return repository, nil
}

func (repository *PostgresSegregationRuleRepository) CommitSegregationRuleMutation(ctx context.Context, mutation SegregationRuleMutation) error {
	return repository.commitSegregationRuleMutation(ctx, mutation, nil)
}

func (repository *PostgresSegregationRuleRepository) CommitSegregationRuleMutationWithIdempotency(ctx context.Context, mutation SegregationRuleMutation, commit SegregationRuleMutationCommit) error {
	return repository.commitSegregationRuleMutation(ctx, mutation, &commit)
}

func (repository *PostgresSegregationRuleRepository) commitSegregationRuleMutation(ctx context.Context, mutation SegregationRuleMutation, commit *SegregationRuleMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrSegregationRuleUnavailable
	}
	if repository.auditWriter == nil {
		return ErrSegregationRuleAudit
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() {
			return ErrSegregationRuleVersion
		}
	} else if mutation.ExpectedVersion == nil || !mutation.Before.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 {
		return ErrSegregationRuleVersion
	}
	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback(ctx)
		}
	}()
	queries := identitydb.New(transaction)
	if mutation.Before.ID == uuid.Nil {
		err = queries.CreateIdentitySegregationRule(ctx, identitydb.CreateIdentitySegregationRuleParams{ID: toPGUUID(mutation.After.ID), Code: mutation.After.Code, CurrentVersion: mutation.After.Version.Value(), Status: string(mutation.After.Status), CreatedAt: toPGTimestamp(mutation.After.CreatedAt), UpdatedAt: toPGTimestamp(mutation.After.UpdatedAt), LastAuditReference: pgtype.UUID{}})
	} else {
		var rows int64
		rows, err = queries.UpdateIdentitySegregationRule(ctx, identitydb.UpdateIdentitySegregationRuleParams{ID: toPGUUID(mutation.After.ID), Code: mutation.After.Code, NextCurrentVersion: mutation.After.Version.Value(), Status: string(mutation.After.Status), UpdatedAt: toPGTimestamp(mutation.After.UpdatedAt), LastAuditReference: pgtype.UUID{}, ExpectedCurrentVersion: mutation.ExpectedVersion.Value()})
		if err == nil && rows != 1 {
			return ErrSegregationRuleVersion
		}
	}
	if err != nil {
		return mapPostgresSegregationRuleError(err)
	}
	auditReference, err := repository.auditWriter(ctx, transaction, mutation.Audit)
	if err != nil {
		return err
	}
	if auditReference == uuid.Nil {
		return ErrSegregationRuleAudit
	}
	approval := mutation.After.Approval
	effectiveTo := pgtype.Timestamptz{}
	if mutation.After.EffectiveTo != nil {
		effectiveTo = toPGTimestamp(*mutation.After.EffectiveTo)
	}
	if err := queries.CreateIdentitySegregationRuleRevision(ctx, identitydb.CreateIdentitySegregationRuleRevisionParams{
		RuleID: toPGUUID(mutation.After.ID), RevisionVersion: mutation.After.Version.Value(), Code: mutation.After.Code, Name: mutation.After.Name, Status: string(mutation.After.Status),
		ConflictingPermissions: append([]string(nil), mutation.After.ConflictingPermissions...), EnforcementMode: string(mutation.After.EnforcementMode), ScopeIds: append([]string(nil), mutation.After.ScopeIDs...), AmountThreshold: decimalToNumeric(mutation.After.AmountThreshold), CoolingOffSeconds: int64(mutation.After.CoolingOff.Seconds()),
		EffectiveFrom: toPGTimestamp(mutation.After.EffectiveFrom), EffectiveTo: effectiveTo, ApprovalRequestID: toPGUUID(approval.ApprovalRequestID), ApprovalDecisionID: toPGUUID(approval.DecisionID), ApproverUserID: toPGUUID(approval.ApproverUserID), PolicyVersion: approval.PolicyVersion, DecisionVersion: approval.DecisionVersion, SubjectVersion: approval.SubjectVersion, CandidateFingerprint: approval.CandidateFingerprint, AuditReference: toPGUUID(auditReference), CreatedAt: toPGTimestamp(mutation.After.UpdatedAt),
	}); err != nil {
		return mapPostgresSegregationRuleError(err)
	}
	if err := queries.SetIdentitySegregationRuleAuditReference(ctx, identitydb.SetIdentitySegregationRuleAuditReferenceParams{ID: toPGUUID(mutation.After.ID), LastAuditReference: toPGUUID(auditReference)}); err != nil {
		return err
	}
	if commit != nil {
		if commit.Coordinator == nil {
			return ErrSegregationRuleUnavailable
		}
		if err := commit.Coordinator.Finalize(ctx, transaction, commit.Acquisition, commit.Result); err != nil {
			return err
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (repository *PostgresSegregationRuleRepository) Get(ctx context.Context, id uuid.UUID) (SegregationRule, error) {
	if repository == nil || repository.pool == nil {
		return SegregationRule{}, ErrSegregationRuleUnavailable
	}
	row, err := identitydb.New(repository.pool).GetCurrentIdentitySegregationRule(ctx, toPGUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return SegregationRule{}, ErrSegregationRuleNotFound
	}
	if err != nil {
		return SegregationRule{}, err
	}
	return segregationRuleFromRow(row)
}

func (repository *PostgresSegregationRuleRepository) List(ctx context.Context) ([]SegregationRule, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrSegregationRuleUnavailable
	}
	rows, err := identitydb.New(repository.pool).ListCurrentIdentitySegregationRules(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]SegregationRule, 0, len(rows))
	for _, row := range rows {
		rule, err := segregationRuleFromListRow(row)
		if err != nil {
			return nil, err
		}
		result = append(result, rule)
	}
	return result, nil
}

func segregationRuleFromRow(row identitydb.GetCurrentIdentitySegregationRuleRow) (SegregationRule, error) {
	version, err := aggregateversion.FromInt64(row.CurrentVersion)
	if err != nil {
		return SegregationRule{}, err
	}
	threshold, err := numericToDecimal(row.AmountThreshold)
	if err != nil {
		return SegregationRule{}, err
	}
	rule := SegregationRule{ID: rowUUID(row.ID), Code: row.Code, Name: row.Name, Status: SegregationRuleStatus(row.Status), Version: version, ConflictingPermissions: append([]string(nil), row.ConflictingPermissions...), EnforcementMode: SegregationEnforcementMode(row.EnforcementMode), ScopeIDs: append([]string(nil), row.ScopeIds...), AmountThreshold: threshold, CoolingOff: time.Duration(row.CoolingOffSeconds) * time.Second, EffectiveFrom: row.EffectiveFrom.Time, Approval: ApprovalDecisionReference{ApprovalRequestID: rowUUID(row.ApprovalRequestID), DecisionID: rowUUID(row.ApprovalDecisionID), ApproverUserID: rowUUID(row.ApproverUserID), PolicyVersion: row.PolicyVersion, DecisionVersion: row.DecisionVersion, SubjectVersion: row.SubjectVersion, CandidateFingerprint: row.CandidateFingerprint}, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time, AuditReference: rowUUID(row.AuditReference)}
	if row.EffectiveTo.Valid {
		value := row.EffectiveTo.Time
		rule.EffectiveTo = &value
	}
	if err := rule.Validate(); err != nil {
		return SegregationRule{}, err
	}
	return rule, nil
}

func segregationRuleFromListRow(row identitydb.ListCurrentIdentitySegregationRulesRow) (SegregationRule, error) {
	return segregationRuleFromRow(identitydb.GetCurrentIdentitySegregationRuleRow{ID: row.ID, Code: row.Code, CurrentVersion: row.CurrentVersion, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, LastAuditReference: row.LastAuditReference, Name: row.Name, ConflictingPermissions: row.ConflictingPermissions, EnforcementMode: row.EnforcementMode, ScopeIds: row.ScopeIds, AmountThreshold: row.AmountThreshold, CoolingOffSeconds: row.CoolingOffSeconds, EffectiveFrom: row.EffectiveFrom, EffectiveTo: row.EffectiveTo, ApprovalRequestID: row.ApprovalRequestID, ApprovalDecisionID: row.ApprovalDecisionID, ApproverUserID: row.ApproverUserID, PolicyVersion: row.PolicyVersion, DecisionVersion: row.DecisionVersion, SubjectVersion: row.SubjectVersion, CandidateFingerprint: row.CandidateFingerprint, AuditReference: row.AuditReference, RevisionCreatedAt: row.RevisionCreatedAt})
}

func numericToDecimal(value pgtype.Numeric) (*decimal.Decimal, error) {
	if !value.Valid || value.NaN {
		return nil, nil
	}
	if value.Int == nil {
		return nil, ErrInvalidSegregationRule
	}
	decimalValue := decimal.NewFromBigInt(new(big.Int).Set(value.Int), value.Exp)
	return &decimalValue, nil
}

func decimalToNumeric(value *decimal.Decimal) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{}
	}
	return pgtype.Numeric{Int: value.Coefficient(), Exp: value.Exponent(), Valid: true}
}

func mapPostgresSegregationRuleError(err error) error {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) {
		return err
	}
	switch pgError.ConstraintName {
	case "segregation_rule_code_unique", "segregation_rule_pk", "segregation_rule_revision_pk":
		return ErrSegregationRuleIdempotency
	case "segregation_rule_revision_version_check", "segregation_rule_version_check":
		return ErrInvalidSegregationRule
	default:
		return err
	}
}

var _ SegregationRuleRepository = (*PostgresSegregationRuleRepository)(nil)
