package identity

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/identity/identitydb"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

// PostgresAuditWriter is supplied by the audit owner. Identity passes the
// open transaction; it never writes the audit schema itself.
type PostgresAuditWriter func(context.Context, pgx.Tx, AuditRecord) (uuid.UUID, error)

// PostgresUserRepository persists the identity aggregate and its assignment
// joins inside the identity-owned schema. Assignment replacement is committed
// atomically with the aggregate version update.
type PostgresUserRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresAuditWriter
}

func NewPostgresUserRepository(pool *pgxpool.Pool) (*PostgresUserRepository, error) {
	return NewPostgresUserRepositoryWithAudit(pool, nil)
}

func NewPostgresUserRepositoryWithAudit(pool *pgxpool.Pool, auditWriter PostgresAuditWriter) (*PostgresUserRepository, error) {
	if pool == nil {
		return nil, ErrInvalidUserService
	}
	return &PostgresUserRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresUserRepository) CommitUserMutation(ctx context.Context, mutation UserMutation) error {
	return repository.commitUserMutation(ctx, mutation, nil)
}

func (repository *PostgresUserRepository) CommitUserMutationWithIdempotency(ctx context.Context, mutation UserMutation, commit DurableUserMutationCommit) error {
	return repository.commitUserMutation(ctx, mutation, &commit)
}

func (repository *PostgresUserRepository) commitUserMutation(ctx context.Context, mutation UserMutation, commit *DurableUserMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidUserService
	}
	if repository.auditWriter == nil {
		return ErrAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() {
			return ErrVersionConflict
		}
	} else {
		if mutation.Before.ID != mutation.After.ID || mutation.ExpectedVersion == nil ||
			mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() ||
			mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 {
			return ErrVersionConflict
		}
		if mutation.Before.AuthenticationSubject != mutation.After.AuthenticationSubject {
			return ErrAuthenticationImmutable
		}
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
		err = queries.CreateIdentityUser(ctx, identitydb.CreateIdentityUserParams{
			ID:                       toPGUUID(mutation.After.ID),
			AuthenticationSubjectOid: mutation.After.AuthenticationSubject.OID,
			AuthenticationSubjectTid: mutation.After.AuthenticationSubject.TID,
			AuthenticationSubjectSub: mutation.After.AuthenticationSubject.Sub,
			Status:                   string(mutation.After.Status),
			AggregateVersion:         mutation.After.Version.Value(),
			CreatedAt:                toPGTimestamp(mutation.After.CreatedAt),
			UpdatedAt:                toPGTimestamp(mutation.After.UpdatedAt),
			LastAuditReference:       pgtype.UUID{},
		})
		if err != nil {
			return mapPostgresUserError(err)
		}
	} else {
		rows, updateErr := queries.UpdateIdentityUser(ctx, identitydb.UpdateIdentityUserParams{
			Status:                   string(mutation.After.Status),
			NextAggregateVersion:     mutation.After.Version.Value(),
			UpdatedAt:                toPGTimestamp(mutation.After.UpdatedAt),
			LastAuditReference:       pgtype.UUID{},
			ID:                       toPGUUID(mutation.After.ID),
			ExpectedAggregateVersion: mutation.ExpectedVersion.Value(),
		})
		if updateErr != nil {
			return mapPostgresUserError(updateErr)
		}
		if rows != 1 {
			return ErrVersionConflict
		}
		if err := queries.DeleteIdentityUserAssignments(ctx, toPGUUID(mutation.After.ID)); err != nil {
			return err
		}
	}
	if err := createIdentityUserAssignments(ctx, queries, mutation.After); err != nil {
		return mapPostgresUserError(err)
	}

	auditReference, err := repository.auditWriter(ctx, transaction, mutation.Audit)
	if err != nil {
		return err
	}
	if auditReference == uuid.Nil {
		return ErrAuditUnavailable
	}
	if err := queries.SetIdentityUserAuditReference(ctx, identitydb.SetIdentityUserAuditReferenceParams{
		ID:                 toPGUUID(mutation.After.ID),
		LastAuditReference: toPGUUID(auditReference),
	}); err != nil {
		return err
	}

	if commit != nil {
		if commit.Coordinator == nil {
			return ErrInvalidUserService
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
func (repository *PostgresUserRepository) Get(ctx context.Context, id uuid.UUID) (User, error) {
	if repository == nil || repository.pool == nil {
		return User{}, ErrInvalidUserService
	}
	queries := identitydb.New(repository.pool)
	row, err := queries.GetIdentityUser(ctx, toPGUUID(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return repository.userFromRow(ctx, queries, row)
}

func (repository *PostgresUserRepository) FindByAuthenticationSubject(ctx context.Context, subject AuthenticationSubject) (User, error) {
	if repository == nil || repository.pool == nil {
		return User{}, ErrInvalidUserService
	}
	queries := identitydb.New(repository.pool)
	row, err := queries.FindIdentityUserBySubject(ctx, identitydb.FindIdentityUserBySubjectParams{
		AuthenticationSubjectOid: subject.OID,
		AuthenticationSubjectTid: subject.TID,
		AuthenticationSubjectSub: subject.Sub,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return repository.userFromRow(ctx, queries, row)
}

func (repository *PostgresUserRepository) List(ctx context.Context) ([]User, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidUserService
	}
	queries := identitydb.New(repository.pool)
	rows, err := queries.ListIdentityUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(rows))
	for _, row := range rows {
		user, err := repository.userFromRow(ctx, queries, row)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func createIdentityUserAssignments(ctx context.Context, queries *identitydb.Queries, user User) error {
	for _, assignment := range user.Assignments {
		if err := queries.CreateIdentityUserRoleAssignment(ctx, identitydb.CreateIdentityUserRoleAssignmentParams{
			UserID: toPGUUID(user.ID),
			RoleID: toPGUUID(assignment.RoleID),
		}); err != nil {
			return err
		}
		for _, scope := range assignment.Scopes {
			if err := queries.CreateIdentityUserRoleAssignmentScope(ctx, identitydb.CreateIdentityUserRoleAssignmentScopeParams{
				UserID:  toPGUUID(user.ID),
				RoleID:  toPGUUID(assignment.RoleID),
				ScopeID: scope.ScopeID,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (repository *PostgresUserRepository) userFromRow(ctx context.Context, queries *identitydb.Queries, row identitydb.IdentityUserAccount) (User, error) {
	version, err := aggregateversion.FromInt64(row.AggregateVersion)
	if err != nil {
		return User{}, err
	}
	status := UserStatus(row.Status)
	if err := status.Validate(); err != nil {
		return User{}, err
	}
	assignmentRows, err := queries.ListIdentityUserRoleAssignments(ctx, row.ID)
	if err != nil {
		return User{}, err
	}
	byRole := make(map[uuid.UUID][]EntityAccessScope)
	for _, assignment := range assignmentRows {
		roleID := uuid.UUID(assignment.RoleID.Bytes)
		byRole[roleID] = append(byRole[roleID], EntityAccessScope{ScopeID: assignment.ScopeID})
	}
	roleIDs := make([]uuid.UUID, 0, len(byRole))
	for roleID := range byRole {
		roleIDs = append(roleIDs, roleID)
	}
	sort.Slice(roleIDs, func(left, right int) bool { return roleIDs[left].String() < roleIDs[right].String() })
	assignments := make([]RoleAssignment, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		assignments = append(assignments, RoleAssignment{RoleID: roleID, Scopes: byRole[roleID]})
	}
	normalized, err := normalizeAssignments(assignments)
	if err != nil {
		return User{}, err
	}
	user := User{
		ID: rowUUID(row.ID),
		AuthenticationSubject: AuthenticationSubject{
			OID: row.AuthenticationSubjectOid,
			TID: row.AuthenticationSubjectTid,
			Sub: row.AuthenticationSubjectSub,
		},
		Status:      status,
		Assignments: normalized,
		Version:     version,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
	if err := user.Validate(); err != nil {
		return User{}, err
	}
	return user, nil
}

func toPGUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: value != uuid.Nil}
}

func rowUUID(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	return uuid.UUID(value.Bytes)
}

func toPGTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func mapPostgresUserError(err error) error {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) || pgError.Code != "23505" {
		return err
	}
	switch pgError.ConstraintName {
	case "user_account_subject_unique":
		return ErrDuplicateAuthenticationSubject
	case "user_role_assignment_pk", "user_role_assignment_scope_pk":
		return ErrDuplicateAssignment
	default:
		return err
	}
}

var _ UserRepository = (*PostgresUserRepository)(nil)
var _ AtomicUserMutationRepository = (*PostgresUserRepository)(nil)
var _ DurableUserMutationRepository = (*PostgresUserRepository)(nil)
