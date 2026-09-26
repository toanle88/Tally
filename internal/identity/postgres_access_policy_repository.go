package identity

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toanle88/Tally/internal/identity/identitydb"
)

// PostgresAccessPolicyStore reads only the current identity-owned policy
// revisions. Revision writes remain outside this story and are protected by
// the database immutability trigger.
type PostgresAccessPolicyStore struct {
	pool *pgxpool.Pool
}

func NewPostgresAccessPolicyStore(pool *pgxpool.Pool) (*PostgresAccessPolicyStore, error) {
	if pool == nil {
		return nil, ErrPolicyUnavailable
	}
	return &PostgresAccessPolicyStore{pool: pool}, nil
}

func (store *PostgresAccessPolicyStore) List(ctx context.Context) ([]AccessPolicy, error) {
	if store == nil || store.pool == nil {
		return nil, ErrPolicyUnavailable
	}
	rows, err := identitydb.New(store.pool).ListCurrentIdentityAccessPolicies(ctx)
	if err != nil {
		return nil, err
	}
	policies := make([]AccessPolicy, 0, len(rows))
	for _, row := range rows {
		var actorIDs, roleIDs []string
		var permissions []string
		if err := json.Unmarshal(row.SubjectActorIds, &actorIDs); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(row.SubjectRoleIds, &roleIDs); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(row.Permissions, &permissions); err != nil {
			return nil, err
		}
		var rules []AccessRule
		if err := json.Unmarshal(row.Rules, &rules); err != nil {
			return nil, err
		}
		policy := AccessPolicy{
			ID:            rowUUID(row.PolicyID),
			Version:       row.PolicyVersion,
			Status:        row.Status,
			EffectiveFrom: row.EffectiveFrom.Time,
			Rules:         rules,
			Permissions:   permissions,
		}
		for _, value := range actorIDs {
			id, err := parsePolicyUUID(value)
			if err != nil {
				return nil, err
			}
			policy.SubjectActorIDs = append(policy.SubjectActorIDs, id)
		}
		for _, value := range roleIDs {
			id, err := parsePolicyUUID(value)
			if err != nil {
				return nil, err
			}
			policy.SubjectRoleIDs = append(policy.SubjectRoleIDs, id)
		}
		if row.EffectiveTo.Valid {
			effectiveTo := row.EffectiveTo.Time
			policy.EffectiveTo = &effectiveTo
		}
		if err := policy.Validate(); err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

func parsePolicyUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

var _ AccessPolicyStore = (*PostgresAccessPolicyStore)(nil)
