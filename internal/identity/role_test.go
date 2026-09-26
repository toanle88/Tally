package identity

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestRoleLifecycleReplacesRetiresAndAuditsEachRevision(t *testing.T) {
	service, repository, audit := testRoleService(t, []string{"entity-1"})
	actor := testActor(t)
	grant := testRoleGrant("entity-1")
	create := roleCreateCommand("role-1", "Finance operator", []PermissionGrant{grant}, actor)
	created, err := service.Execute(context.Background(), actor, create)
	if err != nil {
		t.Fatal(err)
	}
	if created.Role.Version.Value() != 1 || created.Role.Status != RoleStatusActive {
		t.Fatalf("created role = %#v", created.Role)
	}

	update := roleUpdateCommand("role-2", created.Role, "Finance operator v2", []PermissionGrant{grant}, actor)
	updated, err := service.Execute(context.Background(), actor, update)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Role.Version.Value() != 2 || updated.Role.Name != "Finance operator v2" {
		t.Fatalf("updated role = %#v", updated.Role)
	}

	retire := roleRetireCommand("role-3", updated.Role, actor)
	retired, err := service.Execute(context.Background(), actor, retire)
	if err != nil {
		t.Fatal(err)
	}
	if retired.Role.Status != RoleStatusRetired || len(retired.Role.Grants) != 0 || retired.Role.Version.Value() != 3 {
		t.Fatalf("retired role = %#v", retired.Role)
	}
	current, err := repository.Get(context.Background(), created.Role.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != RoleStatusRetired {
		t.Fatalf("stored role status = %q, want retired", current.Status)
	}
	if len(audit.Records) != 3 || audit.Records[0].RoleVersion != 1 || audit.Records[2].BeforeFingerprint == "" {
		t.Fatalf("audit records = %#v", audit.Records)
	}
}

func TestRoleGrantValidationRejectsUnknownDuplicateAndInvalidDates(t *testing.T) {
	now := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	unknown := PermissionGrant{Permission: "finance.unknown.future.permission", ScopeIDs: []string{"entity-1"}, EffectiveFrom: now}
	if !errors.Is(unknown.Validate(), ErrInvalidPermissionGrant) {
		t.Fatalf("unknown permission error = %v", unknown.Validate())
	}
	duplicateScope := PermissionGrant{Permission: "finance.gl.submit.posting.request", ScopeIDs: []string{"entity-1", "entity-1"}, EffectiveFrom: now}
	if !errors.Is(duplicateScope.Validate(), ErrDuplicatePermissionGrant) {
		t.Fatalf("duplicate scope error = %v", duplicateScope.Validate())
	}
	end := now
	invalidDates := PermissionGrant{Permission: "finance.gl.submit.posting.request", ScopeIDs: []string{"entity-1"}, EffectiveFrom: now, EffectiveTo: &end}
	if !errors.Is(invalidDates.Validate(), ErrInvalidPermissionGrant) {
		t.Fatalf("invalid dates error = %v", invalidDates.Validate())
	}
}

func TestRoleServiceUsesDefaultDenyAndActorScopeContainment(t *testing.T) {
	repository := NewMemoryRoleRepository()
	audit := &MemoryRoleAuditRecorder{}
	authorizer := MemoryRoleAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: RoleManagementPermission, ApprovedScopeIDs: []string{"entity-1"}, PolicyReference: "test-policy", DecisionReference: uuid.New(),
	}}
	service, err := NewRoleService(repository, authorizer, AllowAllRoleApprovalPort{}, AllowAllRoleSegregationPort{}, audit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor(t)
	command := roleCreateCommand("outside-scope", "Scoped role", []PermissionGrant{testRoleGrant("entity-2")}, actor)
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrRoleAuthorizationDenied) {
		t.Fatalf("outside scope error = %v, want ErrRoleAuthorizationDenied", err)
	}
	if _, err := repository.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(audit.Records) != 0 {
		t.Fatalf("audit records after denial = %d, want 0", len(audit.Records))
	}
}

func TestRoleServiceRejectsApprovalMismatchAndPreservesIdempotentReplay(t *testing.T) {
	service, repository, audit := testRoleService(t, []string{"entity-1"})
	actor := testActor(t)
	command := roleCreateCommand("approval-1", "Approved role", []PermissionGrant{testRoleGrant("entity-1")}, actor)
	command.Approval.CandidateFingerprint = "sha256:wrong"
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrApprovalRejected) {
		t.Fatalf("approval mismatch error = %v, want ErrApprovalRejected", err)
	}

	command = roleCreateCommand("approval-2", "Approved role", []PermissionGrant{testRoleGrant("entity-1")}, actor)
	result, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Role.ID != result.Role.ID || len(audit.Records) != 1 {
		t.Fatalf("replay = %#v, audit = %#v", replay, audit.Records)
	}
	command.Name = "changed"
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrRoleIdempotencyConflict) {
		t.Fatalf("changed replay error = %v, want ErrRoleIdempotencyConflict", err)
	}
	if _, err := repository.Get(context.Background(), result.Role.ID); err != nil {
		t.Fatal(err)
	}
}

func TestRoleServiceExpectedVersionHasOneWinner(t *testing.T) {
	service, _, _ := testRoleService(t, []string{"entity-1"})
	actor := testActor(t)
	created, err := service.Execute(context.Background(), actor, roleCreateCommand("concurrent-create", "Role", []PermissionGrant{testRoleGrant("entity-1")}, actor))
	if err != nil {
		t.Fatal(err)
	}
	first := roleUpdateCommand("concurrent-a", created.Role, "Role A", []PermissionGrant{testRoleGrant("entity-1")}, actor)
	second := roleUpdateCommand("concurrent-b", created.Role, "Role B", []PermissionGrant{testRoleGrant("entity-1")}, actor)
	results := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		_, commandErr := service.Execute(context.Background(), actor, first)
		results <- commandErr
	}()
	go func() {
		defer group.Done()
		_, commandErr := service.Execute(context.Background(), actor, second)
		results <- commandErr
	}()
	group.Wait()
	close(results)
	var successes, conflicts int
	for commandErr := range results {
		if commandErr == nil {
			successes++
		} else if errors.Is(commandErr, ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent error = %v", commandErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes = %d, conflicts = %d, want one each", successes, conflicts)
	}
}

func testRoleService(t *testing.T, scopes []string) (*RoleService, *MemoryRoleRepository, *MemoryRoleAuditRecorder) {
	t.Helper()
	repository := NewMemoryRoleRepository()
	audit := &MemoryRoleAuditRecorder{}
	authorizer := MemoryRoleAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: RoleManagementPermission, ApprovedScopeIDs: scopes, PolicyReference: "test-role-policy", DecisionReference: uuid.New(),
	}}
	service, err := NewRoleService(repository, authorizer, AllowAllRoleApprovalPort{}, AllowAllRoleSegregationPort{}, audit, func() time.Time {
		return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, audit
}

func testRoleGrant(scope string) PermissionGrant {
	return PermissionGrant{
		Permission:    "finance.gl.submit.posting.request",
		ScopeIDs:      []string{scope},
		EffectiveFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	}
}

func roleApproval() ApprovalDecisionReference {
	return ApprovalDecisionReference{
		ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "wfa-policy-v1",
		DecisionVersion: 1, SubjectVersion: 1, ApproverUserID: uuid.New(),
	}
}

func roleCreateCommand(key, name string, grants []PermissionGrant, actor ApplicationActor) RoleCommand {
	command := RoleCommand{Action: RoleActionCreate, Name: name, Grants: grants, IdempotencyKey: key, CorrelationID: "corr-" + key}
	candidate := Role{ID: uuid.New(), Name: name, Status: RoleStatusActive, Grants: grants, Version: aggregateversion.Initial()}
	command.Approval = roleApproval()
	command.Approval.CandidateFingerprint = candidateRoleFingerprint(command, candidate)
	return command
}

func roleUpdateCommand(key string, current Role, name string, grants []PermissionGrant, actor ApplicationActor) RoleCommand {
	command := RoleCommand{Action: RoleActionUpdate, RoleID: current.ID, Name: name, Grants: grants, ExpectedVersion: versionForTest(current.Version.Value()), IdempotencyKey: key, CorrelationID: "corr-" + key}
	command.Approval = roleApproval()
	after := cloneRole(current)
	next, _ := current.Version.Advance()
	command.Approval.SubjectVersion = next.Value()
	if err := after.Replace(name, grants, command.Approval, current.UpdatedAt.Add(time.Minute)); err != nil {
		panic(err)
	}
	after.Version = next
	command.Approval.CandidateFingerprint = candidateRoleFingerprint(command, after)
	return command
}

func roleRetireCommand(key string, current Role, actor ApplicationActor) RoleCommand {
	command := RoleCommand{Action: RoleActionRetire, RoleID: current.ID, ExpectedVersion: versionForTest(current.Version.Value()), IdempotencyKey: key, CorrelationID: "corr-" + key}
	command.Approval = roleApproval()
	after := cloneRole(current)
	next, _ := current.Version.Advance()
	command.Approval.SubjectVersion = next.Value()
	if err := after.Retire(command.Approval, current.UpdatedAt.Add(time.Minute)); err != nil {
		panic(err)
	}
	after.Version = next
	command.Approval.CandidateFingerprint = candidateRoleFingerprint(command, after)
	return command
}
