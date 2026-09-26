//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/idempotency"
)

func TestIdentityRoleRepositoryPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	auditReferences := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	var auditRecords []identity.RoleAuditRecord
	auditWriter := identity.PostgresRoleAuditWriter(func(_ context.Context, _ pgx.Tx, record identity.RoleAuditRecord) (uuid.UUID, error) {
		auditRecords = append(auditRecords, record)
		return auditReferences[len(auditRecords)-1], nil
	})
	repository, err := identity.NewPostgresRoleRepositoryWithAudit(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	approval := func(seed string) identity.ApprovalDecisionReference {
		return identity.ApprovalDecisionReference{
			ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), ApproverUserID: uuid.New(),
			PolicyVersion: "iam-policy-v1", DecisionVersion: 1, SubjectVersion: 1,
			CandidateFingerprint: seed,
		}
	}
	createGrant := identity.PermissionGrant{
		Permission:    "finance.iam.manage.roles",
		ScopeIDs:      []string{"entity-1", "entity-2"},
		EffectiveFrom: now,
	}
	role, err := identity.NewRole(uuid.New(), "Role Administrators", []identity.PermissionGrant{createGrant}, approval("create-fingerprint"), now)
	if err != nil {
		t.Fatal(err)
	}

	if err := repository.CommitRoleMutation(ctx, identity.RoleMutation{
		After: role,
		Audit: identity.RoleAuditRecord{
			RoleID: role.ID, Action: identity.RoleActionCreate, RoleVersion: 1,
			ApprovalRequestID:  role.Approval.ApprovalRequestID,
			ApprovalDecisionID: role.Approval.DecisionID, ApproverUserID: role.Approval.ApproverUserID,
			AfterFingerprint: role.Approval.CandidateFingerprint,
		},
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := repository.Get(ctx, role.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version.Value() != 1 || stored.Status != identity.RoleStatusActive ||
		len(stored.Grants) != 1 || len(stored.Grants[0].ScopeIDs) != 2 {
		t.Fatalf("stored role = %#v, want version 1 with two scopes", stored)
	}
	if stored.Approval.CandidateFingerprint != "create-fingerprint" || stored.AuditReference != auditReferences[0] {
		t.Fatalf("stored approval/audit = %q/%s, want create-fingerprint/%s", stored.Approval.CandidateFingerprint, stored.AuditReference, auditReferences[0])
	}

	updated := stored
	updateGrant := identity.PermissionGrant{
		Permission:    "finance.iam.manage.users",
		ScopeIDs:      []string{"entity-3"},
		EffectiveFrom: now.Add(24 * time.Hour),
	}
	if err := updated.Replace("Role Administrators v2", []identity.PermissionGrant{updateGrant}, approval("update-fingerprint"), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	nextVersion, err := updated.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	updated.Version = nextVersion
	updated.Approval.SubjectVersion = updated.Version.Value()
	if err := repository.CommitRoleMutation(ctx, identity.RoleMutation{
		Before: stored, After: updated, ExpectedVersion: &stored.Version,
		Audit: identity.RoleAuditRecord{
			RoleID: updated.ID, Action: identity.RoleActionUpdate, RoleVersion: 2,
			ApprovalRequestID:  updated.Approval.ApprovalRequestID,
			ApprovalDecisionID: updated.Approval.DecisionID, ApproverUserID: updated.Approval.ApproverUserID,
			BeforeFingerprint: stored.Approval.CandidateFingerprint,
			AfterFingerprint:  updated.Approval.CandidateFingerprint,
		},
	}); err != nil {
		t.Fatal(err)
	}

	stale := stored
	stale.Name = "stale"
	stale.Version = updated.Version
	if err := repository.CommitRoleMutation(ctx, identity.RoleMutation{
		Before: stored, After: stale, ExpectedVersion: &stored.Version,
		Audit: identity.RoleAuditRecord{RoleID: role.ID, Action: identity.RoleActionUpdate, RoleVersion: 2},
	}); !errors.Is(err, identity.ErrVersionConflict) {
		t.Fatalf("stale update error = %v, want ErrVersionConflict", err)
	}

	retired := updated
	if err := retired.Retire(approval("retire-fingerprint"), now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	retiredVersion, err := retired.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	retired.Version = retiredVersion
	retired.Approval.SubjectVersion = retired.Version.Value()
	if err := repository.CommitRoleMutation(ctx, identity.RoleMutation{
		Before: updated, After: retired, ExpectedVersion: &updated.Version,
		Audit: identity.RoleAuditRecord{
			RoleID: retired.ID, Action: identity.RoleActionRetire, RoleVersion: 3,
			ApprovalRequestID:  retired.Approval.ApprovalRequestID,
			ApprovalDecisionID: retired.Approval.DecisionID, ApproverUserID: retired.Approval.ApproverUserID,
			BeforeFingerprint: updated.Approval.CandidateFingerprint,
			AfterFingerprint:  retired.Approval.CandidateFingerprint,
		},
	}); err != nil {
		t.Fatal(err)
	}

	current, err := repository.Get(ctx, role.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != identity.RoleStatusRetired || len(current.Grants) != 0 || current.Version.Value() != 3 || current.AuditReference != auditReferences[2] {
		t.Fatalf("retired role = %#v, want retired version 3 with no active grants", current)
	}
	if err := repository.ValidateRoleAssignments(ctx, []identity.RoleAssignment{{RoleID: role.ID}}); !errors.Is(err, identity.ErrRoleRetired) {
		t.Fatalf("retired assignment validation = %v, want ErrRoleRetired", err)
	}
	if err := repository.ValidateRoleAssignments(ctx, []identity.RoleAssignment{{RoleID: uuid.New()}}); !errors.Is(err, identity.ErrRoleNotFound) {
		t.Fatalf("unknown assignment validation = %v, want ErrRoleNotFound", err)
	}

	var revisionCount, grantCount, auditReferenceCount int
	if err := fixture.pool.QueryRow(ctx, "SELECT COUNT(*) FROM identity.role_revision WHERE role_id = $1", role.ID).Scan(&revisionCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, "SELECT COUNT(*) FROM identity.role_permission_grant WHERE role_id = $1", role.ID).Scan(&grantCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, "SELECT COUNT(DISTINCT audit_reference) FROM identity.role_revision WHERE role_id = $1", role.ID).Scan(&auditReferenceCount); err != nil {
		t.Fatal(err)
	}
	if revisionCount != 3 || grantCount != 3 || auditReferenceCount != 3 {
		t.Fatalf("historical rows = revisions %d grants %d audit references %d, want 3/3/3", revisionCount, grantCount, auditReferenceCount)
	}

	var lastAuditReference uuid.UUID
	if err := fixture.pool.QueryRow(ctx, "SELECT last_audit_reference FROM identity.role WHERE id = $1", role.ID).Scan(&lastAuditReference); err != nil {
		t.Fatal(err)
	}
	if lastAuditReference != auditReferences[2] || len(auditRecords) != 3 {
		t.Fatalf("audit linkage = %s records=%d, want %s and 3 records", lastAuditReference, len(auditRecords), auditReferences[2])
	}

	roles, err := repository.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 || roles[0].ID != role.ID {
		t.Fatalf("listed roles = %#v, want the persisted role", roles)
	}
	assertNoAcquiredConnections(t, fixture.pool)
}

type integrationRoleApprovalPort struct{}

func (integrationRoleApprovalPort) ValidateRoleApproval(context.Context, identity.ApplicationActor, identity.RoleCommand, *identity.Role, string) error {
	return nil
}

func TestIdentityRoleServiceDurableIdempotencyReplaysAfterRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	repository, err := identity.NewPostgresRoleRepositoryWithAudit(
		fixture.pool,
		func(_ context.Context, _ pgx.Tx, _ identity.RoleAuditRecord) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{
		UserID:  uuid.New(),
		Subject: identity.AuthenticationSubject{OID: "role-durable-oid", TID: "tenant", Sub: "role-durable-sub"},
	}
	command := identity.RoleCommand{
		Action: identity.RoleActionCreate, Name: "Durable role", IdempotencyKey: "role-durable-create",
		Grants: []identity.PermissionGrant{{
			Permission: "finance.iam.manage.roles", ScopeIDs: []string{"entity-1"},
			EffectiveFrom: time.Now().UTC(),
		}},
		Approval: identity.ApprovalDecisionReference{
			ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), ApproverUserID: uuid.New(),
			PolicyVersion: "iam-policy-v1", DecisionVersion: 1, SubjectVersion: 1,
			CandidateFingerprint: "durable-candidate",
		},
	}
	authorizer := identity.MemoryRoleAuthorizer{Decision: identity.AuthorizationDecision{
		Allowed: true, Permission: identity.RoleManagementPermission, ApprovedScopeIDs: []string{"*"},
		PolicyReference: "iam-policy-v1", DecisionReference: uuid.New(),
	}}
	policy := idempotency.IdempotencyPolicy{RecordTTL: time.Hour, LeaseTTL: time.Minute}
	newService := func() *identity.RoleService {
		service, err := identity.NewRoleServiceWithDurableIdempotency(
			repository, authorizer, integrationRoleApprovalPort{}, identity.AllowAllRoleSegregationPort{},
			&identity.MemoryRoleAuditRecorder{}, time.Now,
			identity.DurableRoleServiceConfig{
				Database: fixture.pool, Coordinator: idempotency.NewPostgresCoordinator(),
				Policy: policy, OperationID: "identity.manage-roles.v1",
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		return service
	}

	first, err := newService().Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed {
		t.Fatal("first durable role command was replayed")
	}
	second, err := newService().Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || second.Role.ID != first.Role.ID {
		t.Fatalf("durable replay = %#v, want replay of role %s", second, first.Role.ID)
	}
	assertNoAcquiredConnections(t, fixture.pool)
}
