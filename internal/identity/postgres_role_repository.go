package identity

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toanle88/Tally/internal/identity/identitydb"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

// PostgresRoleAuditWriter writes role evidence through the existing audit
// boundary while the identity transaction remains open.
type PostgresRoleAuditWriter func(context.Context, pgx.Tx, RoleAuditRecord) (uuid.UUID, error)

type PostgresRoleRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresRoleAuditWriter
}

func NewPostgresRoleRepository(pool *pgxpool.Pool) (*PostgresRoleRepository, error) {
	return NewPostgresRoleRepositoryWithAudit(pool, nil)
}

func NewPostgresRoleRepositoryWithAudit(pool *pgxpool.Pool, auditWriter PostgresRoleAuditWriter) (*PostgresRoleRepository, error) {
	if pool == nil {
		return nil, ErrInvalidRoleService
	}
	return &PostgresRoleRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresRoleRepository) CommitRoleMutation(ctx context.Context, mutation RoleMutation) error {
	return repository.commitRoleMutation(ctx, mutation, nil)
}

func (repository *PostgresRoleRepository) CommitRoleMutationWithIdempotency(ctx context.Context, mutation RoleMutation, commit DurableRoleMutationCommit) error {
	return repository.commitRoleMutation(ctx, mutation, &commit)
}

func (repository *PostgresRoleRepository) commitRoleMutation(ctx context.Context, mutation RoleMutation, commit *DurableRoleMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidRoleService
	}
	if repository.auditWriter == nil {
		return ErrRoleAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() {
			return ErrVersionConflict
		}
	} else if mutation.Before.ID != mutation.After.ID ||
		mutation.ExpectedVersion == nil ||
		mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() ||
		mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 {
		return ErrVersionConflict
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
		err = queries.CreateIdentityRole(ctx, identitydb.CreateIdentityRoleParams{
			ID:                 toPGUUID(mutation.After.ID),
			Name:               mutation.After.Name,
			Status:             string(mutation.After.Status),
			AggregateVersion:   mutation.After.Version.Value(),
			CreatedAt:          toPGTimestamp(mutation.After.CreatedAt),
			UpdatedAt:          toPGTimestamp(mutation.After.UpdatedAt),
			LastAuditReference: pgtype.UUID{},
		})
	} else {
		var rows int64
		rows, err = queries.UpdateIdentityRole(ctx, identitydb.UpdateIdentityRoleParams{
			ID:                       toPGUUID(mutation.After.ID),
			Name:                     mutation.After.Name,
			Status:                   string(mutation.After.Status),
			NextAggregateVersion:     mutation.After.Version.Value(),
			UpdatedAt:                toPGTimestamp(mutation.After.UpdatedAt),
			LastAuditReference:       pgtype.UUID{},
			ExpectedAggregateVersion: mutation.ExpectedVersion.Value(),
		})
		if err == nil && rows != 1 {
			return ErrVersionConflict
		}
	}
	if err != nil {
		return mapPostgresRoleError(err)
	}

	auditReference, err := repository.auditWriter(ctx, transaction, mutation.Audit)
	if err != nil {
		return err
	}
	if auditReference == uuid.Nil {
		return ErrRoleAuditUnavailable
	}

	approval := mutation.After.Approval
	if err := queries.CreateIdentityRoleRevision(ctx, identitydb.CreateIdentityRoleRevisionParams{
		RoleID:               toPGUUID(mutation.After.ID),
		RevisionVersion:      mutation.After.Version.Value(),
		Name:                 mutation.After.Name,
		Status:               string(mutation.After.Status),
		ApprovalRequestID:    toPGUUID(approval.ApprovalRequestID),
		ApprovalDecisionID:   toPGUUID(approval.DecisionID),
		ApproverUserID:       toPGUUID(approval.ApproverUserID),
		PolicyVersion:        approval.PolicyVersion,
		DecisionVersion:      approval.DecisionVersion,
		SubjectVersion:       approval.SubjectVersion,
		CandidateFingerprint: approval.CandidateFingerprint,
		AuditReference:       toPGUUID(auditReference),
		CreatedAt:            toPGTimestamp(mutation.After.UpdatedAt),
	}); err != nil {
		return mapPostgresRoleError(err)
	}
	for _, grant := range mutation.After.Grants {
		for _, scopeID := range grant.ScopeIDs {
			effectiveTo := pgtype.Timestamptz{}
			if grant.EffectiveTo != nil {
				effectiveTo = toPGTimestamp(*grant.EffectiveTo)
			}
			if err := queries.CreateIdentityRoleGrant(ctx, identitydb.CreateIdentityRoleGrantParams{
				RoleID:        toPGUUID(mutation.After.ID),
				RoleVersion:   mutation.After.Version.Value(),
				Permission:    grant.Permission,
				ScopeID:       scopeID,
				EffectiveFrom: toPGTimestamp(grant.EffectiveFrom),
				EffectiveTo:   effectiveTo,
			}); err != nil {
				return mapPostgresRoleError(err)
			}
		}
	}

	if err := queries.SetIdentityRoleAuditReference(ctx, identitydb.SetIdentityRoleAuditReferenceParams{
		ID:                 toPGUUID(mutation.After.ID),
		LastAuditReference: toPGUUID(auditReference),
	}); err != nil {
		return err
	}

	if commit != nil {
		if commit.Coordinator == nil {
			return ErrInvalidRoleService
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

func (repository *PostgresRoleRepository) Get(ctx context.Context, id uuid.UUID) (Role, error) {
	if repository == nil || repository.pool == nil {
		return Role{}, ErrInvalidRoleService
	}
	queries := identitydb.New(repository.pool)
	row, err := queries.GetIdentityRole(ctx, toPGUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Role{}, ErrRoleNotFound
	}
	if err != nil {
		return Role{}, err
	}
	return repository.roleFromRow(ctx, queries, row)
}

func (repository *PostgresRoleRepository) List(ctx context.Context) ([]Role, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidRoleService
	}
	queries := identitydb.New(repository.pool)
	rows, err := queries.ListIdentityRoles(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Role, 0, len(rows))
	for _, row := range rows {
		role, err := repository.roleFromRow(ctx, queries, identitydb.GetIdentityRoleRow{
			ID:                   row.ID,
			Name:                 row.Name,
			Status:               row.Status,
			AggregateVersion:     row.AggregateVersion,
			CreatedAt:            row.CreatedAt,
			UpdatedAt:            row.UpdatedAt,
			LastAuditReference:   row.LastAuditReference,
			ApprovalRequestID:    row.ApprovalRequestID,
			ApprovalDecisionID:   row.ApprovalDecisionID,
			ApproverUserID:       row.ApproverUserID,
			PolicyVersion:        row.PolicyVersion,
			DecisionVersion:      row.DecisionVersion,
			SubjectVersion:       row.SubjectVersion,
			CandidateFingerprint: row.CandidateFingerprint,
			AuditReference:       row.AuditReference,
			RevisionCreatedAt:    row.RevisionCreatedAt,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, role)
	}
	return result, nil
}

func (repository *PostgresRoleRepository) ValidateRoleAssignments(ctx context.Context, assignments []RoleAssignment) error {
	for _, assignment := range assignments {
		role, err := repository.Get(ctx, assignment.RoleID)
		if err != nil {
			return err
		}
		if role.Status != RoleStatusActive {
			return ErrRoleRetired
		}
	}
	return nil
}

func (repository *PostgresRoleRepository) roleFromRow(ctx context.Context, queries *identitydb.Queries, row identitydb.GetIdentityRoleRow) (Role, error) {
	version, err := aggregateversion.FromInt64(row.AggregateVersion)
	if err != nil {
		return Role{}, err
	}
	grants, err := queries.ListIdentityRoleGrants(ctx, identitydb.ListIdentityRoleGrantsParams{
		RoleID:      row.ID,
		RoleVersion: row.AggregateVersion,
	})
	if err != nil {
		return Role{}, err
	}
	byGrant := make(map[string]*PermissionGrant)
	for _, grant := range grants {
		key := grant.Permission + "\x00" + grant.EffectiveFrom.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
		if grant.EffectiveTo.Valid {
			key += "\x00" + grant.EffectiveTo.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
		}
		item := byGrant[key]
		if item == nil {
			item = &PermissionGrant{
				Permission:    grant.Permission,
				EffectiveFrom: grant.EffectiveFrom.Time.UTC(),
			}
			if grant.EffectiveTo.Valid {
				value := grant.EffectiveTo.Time.UTC()
				item.EffectiveTo = &value
			}
			byGrant[key] = item
		}
		item.ScopeIDs = append(item.ScopeIDs, grant.ScopeID)
	}
	normalizedGrants := make([]PermissionGrant, 0, len(byGrant))
	for _, grant := range byGrant {
		normalizedGrants = append(normalizedGrants, *grant)
	}
	normalizedGrants, err = normalizePermissionGrants(normalizedGrants)
	if err != nil {
		return Role{}, err
	}
	approval := ApprovalDecisionReference{
		ApprovalRequestID:    row.ApprovalRequestID.Bytes,
		DecisionID:           row.ApprovalDecisionID.Bytes,
		ApproverUserID:       row.ApproverUserID.Bytes,
		PolicyVersion:        row.PolicyVersion,
		DecisionVersion:      row.DecisionVersion,
		SubjectVersion:       row.SubjectVersion,
		CandidateFingerprint: row.CandidateFingerprint,
	}
	role := Role{
		ID:             rowUUID(row.ID),
		Name:           row.Name,
		Status:         row.Status,
		Grants:         normalizedGrants,
		Version:        version,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
		Approval:       approval,
		AuditReference: rowUUID(row.AuditReference),
	}
	if err := role.Validate(); err != nil {
		return Role{}, err
	}
	return role, nil
}

func mapPostgresRoleError(err error) error {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) {
		return err
	}
	switch pgError.ConstraintName {
	case "role_pk", "role_revision_pk", "role_permission_grant_pk":
		return ErrRoleIdempotencyConflict
	case "role_permission_grant_date_check", "role_revision_version_check", "role_version_check":
		return ErrInvalidRole
	default:
		return err
	}
}

var _ RoleRepository = (*PostgresRoleRepository)(nil)
var _ AtomicRoleMutationRepository = (*PostgresRoleRepository)(nil)
var _ DurableRoleMutationRepository = (*PostgresRoleRepository)(nil)
