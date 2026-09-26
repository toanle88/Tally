package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toanle88/Tally/internal/identity/identitydb"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

type PostgresEmergencyAccessAuditWriter func(context.Context, pgx.Tx, EmergencyAccessAuditRecord) (uuid.UUID, error)

type PostgresEmergencyAccessRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresEmergencyAccessAuditWriter
}

func NewPostgresEmergencyAccessRepository(pool *pgxpool.Pool) (*PostgresEmergencyAccessRepository, error) {
	return NewPostgresEmergencyAccessRepositoryWithAudit(pool, nil)
}

func NewPostgresEmergencyAccessRepositoryWithAudit(pool *pgxpool.Pool, auditWriter PostgresEmergencyAccessAuditWriter) (*PostgresEmergencyAccessRepository, error) {
	if pool == nil {
		return nil, ErrInvalidEmergencyAccessService
	}
	return &PostgresEmergencyAccessRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresEmergencyAccessRepository) CommitEmergencyAccessMutation(ctx context.Context, mutation EmergencyAccessMutation) error {
	return repository.commitEmergencyAccessMutation(ctx, mutation, nil)
}

func (repository *PostgresEmergencyAccessRepository) CommitEmergencyAccessMutationWithIdempotency(ctx context.Context, mutation EmergencyAccessMutation, commit DurableEmergencyAccessMutationCommit) error {
	return repository.commitEmergencyAccessMutation(ctx, mutation, &commit)
}

func (repository *PostgresEmergencyAccessRepository) commitEmergencyAccessMutation(ctx context.Context, mutation EmergencyAccessMutation, commit *DurableEmergencyAccessMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidEmergencyAccessService
	}
	if repository.auditWriter == nil {
		return ErrEmergencyAccessAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() {
			return ErrEmergencyAccessVersionConflict
		}
	} else if mutation.ExpectedVersion == nil || !mutation.Before.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 {
		return ErrEmergencyAccessVersionConflict
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
		err = queries.CreateIdentityEmergencyAccessGrant(ctx, identitydb.CreateIdentityEmergencyAccessGrantParams{
			ID: toPGUUID(mutation.After.ID), TargetActorID: toPGUUID(mutation.After.TargetActorID),
			CurrentVersion: mutation.After.Version.Value(), Status: mutation.After.Status,
			CreatedAt: toPGTimestamp(mutation.After.CreatedAt), UpdatedAt: toPGTimestamp(mutation.After.UpdatedAt),
			LastAuditReference: pgtype.UUID{},
		})
	} else {
		var rows int64
		rows, err = queries.UpdateIdentityEmergencyAccessGrant(ctx, identitydb.UpdateIdentityEmergencyAccessGrantParams{
			ID: toPGUUID(mutation.After.ID), NextCurrentVersion: mutation.After.Version.Value(), Status: mutation.After.Status,
			UpdatedAt: toPGTimestamp(mutation.After.UpdatedAt), LastAuditReference: pgtype.UUID{},
			ExpectedCurrentVersion: mutation.ExpectedVersion.Value(),
		})
		if err == nil && rows != 1 {
			return ErrEmergencyAccessVersionConflict
		}
	}
	if err != nil {
		return mapPostgresEmergencyAccessError(err)
	}
	auditReference, err := repository.auditWriter(ctx, transaction, mutation.Audit)
	if err != nil {
		return err
	}
	if auditReference == uuid.Nil {
		return ErrEmergencyAccessAuditUnavailable
	}
	approval := mutation.After.Approval
	if err := queries.CreateIdentityEmergencyAccessGrantRevision(ctx, identitydb.CreateIdentityEmergencyAccessGrantRevisionParams{
		GrantID: toPGUUID(mutation.After.ID), RevisionVersion: mutation.After.Version.Value(), TargetActorID: toPGUUID(mutation.After.TargetActorID),
		Status: mutation.After.Status, ReasonCode: mutation.After.ReasonCode, GrantingActorID: toPGUUID(mutation.After.GrantingActorID),
		ApproverUserID: toPGUUID(approval.ApproverUserID), ApprovalRequestID: toPGUUID(approval.ApprovalRequestID), ApprovalDecisionID: toPGUUID(approval.DecisionID),
		PolicyVersion: mutation.After.PolicyVersion, DecisionVersion: approval.DecisionVersion, SubjectVersion: approval.SubjectVersion,
		CandidateFingerprint: approval.CandidateFingerprint, StartAt: toPGTimestamp(mutation.After.StartAt), ExpiresAt: toPGTimestamp(mutation.After.ExpiresAt),
		ReviewStatus: mutation.After.ReviewStatus, ReviewOutcomeCode: optionalPGText(mutation.After.ReviewOutcomeCode), ReviewReference: optionalPGText(mutation.After.ReviewReference),
		ReviewDueAt: toPGTimestamp(mutation.After.ReviewDueAt), RevokedAt: optionalPGTimestamp(mutation.After.RevokedAt),
		RevokedBy: optionalPGUUID(mutation.After.RevokedBy), RevocationReason: optionalPGText(mutation.After.RevocationReason),
		AuditReference: toPGUUID(auditReference), CreatedAt: toPGTimestamp(mutation.After.UpdatedAt),
	}); err != nil {
		return mapPostgresEmergencyAccessError(err)
	}
	for _, permission := range mutation.After.Permissions {
		if err := queries.CreateIdentityEmergencyAccessGrantPermission(ctx, identitydb.CreateIdentityEmergencyAccessGrantPermissionParams{GrantID: toPGUUID(mutation.After.ID), GrantVersion: mutation.After.Version.Value(), Permission: permission}); err != nil {
			return mapPostgresEmergencyAccessError(err)
		}
	}
	for _, scopeID := range mutation.After.ScopeIDs {
		if err := queries.CreateIdentityEmergencyAccessGrantScope(ctx, identitydb.CreateIdentityEmergencyAccessGrantScopeParams{GrantID: toPGUUID(mutation.After.ID), GrantVersion: mutation.After.Version.Value(), ScopeID: scopeID}); err != nil {
			return mapPostgresEmergencyAccessError(err)
		}
	}
	if err := queries.SetIdentityEmergencyAccessGrantAuditReference(ctx, identitydb.SetIdentityEmergencyAccessGrantAuditReferenceParams{ID: toPGUUID(mutation.After.ID), LastAuditReference: toPGUUID(auditReference)}); err != nil {
		return err
	}
	if commit != nil {
		if commit.Coordinator == nil {
			return ErrInvalidEmergencyAccessService
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

func (repository *PostgresEmergencyAccessRepository) Get(ctx context.Context, id uuid.UUID) (EmergencyAccessGrant, error) {
	if repository == nil || repository.pool == nil {
		return EmergencyAccessGrant{}, ErrInvalidEmergencyAccessService
	}
	queries := identitydb.New(repository.pool)
	row, err := queries.GetIdentityEmergencyAccessGrant(ctx, toPGUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return EmergencyAccessGrant{}, ErrEmergencyAccessNotFound
	}
	if err != nil {
		return EmergencyAccessGrant{}, err
	}
	return repository.grantFromRow(ctx, queries, identitydb.ListIdentityEmergencyAccessGrantsRow{
		ID: row.ID, TargetActorID: row.TargetActorID, CurrentVersion: row.CurrentVersion, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		LastAuditReference: row.LastAuditReference, ReasonCode: row.ReasonCode, GrantingActorID: row.GrantingActorID, ApproverUserID: row.ApproverUserID,
		ApprovalRequestID: row.ApprovalRequestID, ApprovalDecisionID: row.ApprovalDecisionID, PolicyVersion: row.PolicyVersion, DecisionVersion: row.DecisionVersion,
		SubjectVersion: row.SubjectVersion, CandidateFingerprint: row.CandidateFingerprint, StartAt: row.StartAt, ExpiresAt: row.ExpiresAt, ReviewStatus: row.ReviewStatus,
		ReviewOutcomeCode: row.ReviewOutcomeCode, ReviewReference: row.ReviewReference, ReviewDueAt: row.ReviewDueAt, RevokedAt: row.RevokedAt, RevokedBy: row.RevokedBy,
		RevocationReason: row.RevocationReason, AuditReference: row.AuditReference, RevisionCreatedAt: row.RevisionCreatedAt,
	})
}

func (repository *PostgresEmergencyAccessRepository) List(ctx context.Context) ([]EmergencyAccessGrant, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidEmergencyAccessService
	}
	queries := identitydb.New(repository.pool)
	rows, err := queries.ListIdentityEmergencyAccessGrants(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]EmergencyAccessGrant, 0, len(rows))
	for _, row := range rows {
		grant, err := repository.grantFromRow(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		result = append(result, grant)
	}
	return result, nil
}

func (repository *PostgresEmergencyAccessRepository) grantFromRow(ctx context.Context, queries *identitydb.Queries, row identitydb.ListIdentityEmergencyAccessGrantsRow) (EmergencyAccessGrant, error) {
	version, err := aggregateversion.FromInt64(row.CurrentVersion)
	if err != nil {
		return EmergencyAccessGrant{}, err
	}
	permissions, err := queries.ListIdentityEmergencyAccessGrantPermissions(ctx, identitydb.ListIdentityEmergencyAccessGrantPermissionsParams{GrantID: row.ID, GrantVersion: row.CurrentVersion})
	if err != nil {
		return EmergencyAccessGrant{}, err
	}
	scopes, err := queries.ListIdentityEmergencyAccessGrantScopes(ctx, identitydb.ListIdentityEmergencyAccessGrantScopesParams{GrantID: row.ID, GrantVersion: row.CurrentVersion})
	if err != nil {
		return EmergencyAccessGrant{}, err
	}
	grant := EmergencyAccessGrant{
		ID: rowUUID(row.ID), TargetActorID: rowUUID(row.TargetActorID), Status: row.Status, Permissions: permissions, ScopeIDs: scopes,
		ReasonCode: row.ReasonCode, GrantingActorID: rowUUID(row.GrantingActorID), Approval: ApprovalDecisionReference{ApprovalRequestID: rowUUID(row.ApprovalRequestID), DecisionID: rowUUID(row.ApprovalDecisionID), ApproverUserID: rowUUID(row.ApproverUserID), PolicyVersion: row.PolicyVersion, DecisionVersion: row.DecisionVersion, SubjectVersion: row.SubjectVersion, CandidateFingerprint: row.CandidateFingerprint},
		PolicyVersion: row.PolicyVersion, StartAt: row.StartAt.Time, ExpiresAt: row.ExpiresAt.Time, ReviewStatus: row.ReviewStatus, ReviewDueAt: row.ReviewDueAt.Time,
		Version: version, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time, AuditReference: rowUUID(row.AuditReference),
	}
	if row.ReviewOutcomeCode.Valid {
		grant.ReviewOutcomeCode = row.ReviewOutcomeCode.String
	}
	if row.ReviewReference.Valid {
		grant.ReviewReference = row.ReviewReference.String
	}
	if row.RevokedAt.Valid {
		value := row.RevokedAt.Time
		grant.RevokedAt = &value
	}
	if row.RevokedBy.Valid {
		value := rowUUID(row.RevokedBy)
		grant.RevokedBy = &value
	}
	if row.RevocationReason.Valid {
		grant.RevocationReason = row.RevocationReason.String
	}
	if err := grant.Validate(); err != nil {
		return EmergencyAccessGrant{}, err
	}
	return grant, nil
}

func mapPostgresEmergencyAccessError(err error) error {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) {
		return err
	}
	switch pgError.ConstraintName {
	case "emergency_access_grant_pk", "emergency_access_grant_revision_pk", "emergency_access_grant_permission_pk", "emergency_access_grant_scope_pk":
		return ErrEmergencyAccessIdempotencyConflict
	case "emergency_access_grant_revision_interval_check", "emergency_access_grant_revision_review_status_check", "emergency_access_grant_revision_review_outcome_check", "emergency_access_grant_revision_revocation_check":
		return ErrInvalidEmergencyAccessGrant
	default:
		return err
	}
}

func optionalPGText(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func optionalPGTimestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return toPGTimestamp(*value)
}

func optionalPGUUID(value *uuid.UUID) pgtype.UUID {
	if value == nil || *value == uuid.Nil {
		return pgtype.UUID{}
	}
	return toPGUUID(*value)
}

var _ EmergencyAccessRepository = (*PostgresEmergencyAccessRepository)(nil)
var _ AtomicEmergencyAccessMutationRepository = (*PostgresEmergencyAccessRepository)(nil)
var _ DurableEmergencyAccessMutationRepository = (*PostgresEmergencyAccessRepository)(nil)
