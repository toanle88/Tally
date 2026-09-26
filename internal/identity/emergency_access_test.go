package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

type emergencyAccessTestAuthorizer struct{}

func (emergencyAccessTestAuthorizer) AuthorizeEmergencyAccess(_ context.Context, _ ApplicationActor, command EmergencyAccessGrantCommand, _ *EmergencyAccessGrant) (AuthorizationDecision, error) {
	permission := EmergencyAccessRevokePermission
	if command.Action == EmergencyAccessActionGrant {
		permission = EmergencyAccessGrantPermission
	}
	return AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: permission, PolicyVersion: "emergency-policy-v1", PolicyReference: "policy-emergency", DecisionReference: uuid.New(), ReasonCode: AuthorizationReasonAllowed}, nil
}

func emergencyAccessTestActor(now time.Time) ApplicationActor {
	return ApplicationActor{
		UserID:  uuid.MustParse("00000000-0000-0000-0000-000000000011"),
		Subject: AuthenticationSubject{OID: "emergency-oid", TID: "tenant", Sub: "emergency-sub", Assurance: AuthenticationAssurance{AuthenticatedAt: now.Add(-time.Minute), AssuranceLevel: "high", Methods: "mfa"}},
	}
}

func emergencyAccessGrantTestCommand(now time.Time, idempotency string) EmergencyAccessGrantCommand {
	command := EmergencyAccessGrantCommand{
		Action: EmergencyAccessActionGrant, TargetActorID: uuid.MustParse("00000000-0000-0000-0000-000000000012"),
		Permissions: []string{"finance.gl.submit.posting.request"}, ScopeIDs: []string{"entity:100"}, ReasonCode: "break-fix",
		StartsAt: now, ExpiresAt: now.Add(2 * time.Hour), Approval: ApprovalDecisionReference{
			ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "emergency-policy-v1", DecisionVersion: 1, SubjectVersion: 1, ApproverUserID: uuid.MustParse("00000000-0000-0000-0000-000000000013"),
		}, IdempotencyKey: idempotency, CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	candidate := EmergencyAccessGrant{ID: uuid.Nil, TargetActorID: command.TargetActorID, Status: EmergencyAccessGrantStatusActive, Permissions: command.Permissions, ScopeIDs: command.ScopeIDs, ReasonCode: command.ReasonCode, GrantingActorID: uuid.MustParse("00000000-0000-0000-0000-000000000011"), Approval: command.Approval, PolicyVersion: "emergency-policy-v1", StartAt: command.StartsAt, ExpiresAt: command.ExpiresAt, ReviewStatus: EmergencyAccessReviewPending, ReviewDueAt: now.Add(24 * time.Hour), Version: aggregateversion.Initial(), CreatedAt: now, UpdatedAt: now}
	command.Approval.CandidateFingerprint = candidateEmergencyAccessFingerprint(command, candidate)
	return command
}

func TestEmergencyAccessGrantRevokeAndReviewLifecycle(t *testing.T) {
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	repository := NewMemoryEmergencyAccessRepository()
	audit := &MemoryEmergencyAccessAuditRecorder{}
	service, err := NewEmergencyAccessService(repository, emergencyAccessTestAuthorizer{}, AllowAllEmergencyAccessApprovalPort{}, audit, WeekdayEmergencyAccessCalendar{}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewEmergencyAccessService() error = %v", err)
	}
	actor := emergencyAccessTestActor(now)
	grant, err := service.Execute(context.Background(), actor, emergencyAccessGrantTestCommand(now, "grant-1"))
	if err != nil {
		t.Fatalf("grant error = %v", err)
	}
	if grant.Grant.Status != EmergencyAccessGrantStatusActive || grant.Grant.ReviewStatus != EmergencyAccessReviewPending || grant.Grant.Version.Value() != 1 {
		t.Fatalf("unexpected grant state: %#v", grant.Grant)
	}
	if !grant.Grant.Allows(now.Add(time.Minute), grant.Grant.TargetActorID, "finance.gl.submit.posting.request", "entity:100") {
		t.Fatal("grant should allow the approved permission and scope")
	}

	expectedVersion := grant.Grant.Version
	revoke := EmergencyAccessGrantCommand{Action: EmergencyAccessActionRevoke, GrantID: grant.Grant.ID, ReasonCode: "incident-contained", ExpectedVersion: &expectedVersion, ReviewStatus: EmergencyAccessReviewPending, IdempotencyKey: "revoke-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	revoked, err := service.Execute(context.Background(), actor, revoke)
	if err != nil {
		t.Fatalf("revoke error = %v", err)
	}
	if revoked.Grant.Status != EmergencyAccessGrantStatusRevoked || revoked.Grant.RevokedAt == nil || revoked.Grant.EffectiveStatus(now) != EmergencyAccessGrantStatusRevoked {
		t.Fatalf("unexpected revoke state: %#v", revoked.Grant)
	}
	if revoked.Grant.Allows(now, revoked.Grant.TargetActorID, "finance.gl.submit.posting.request", "entity:100") {
		t.Fatal("revoked grant must not allow use")
	}
	duplicateVersion := revoked.Grant.Version
	if _, err := service.Execute(context.Background(), actor, EmergencyAccessGrantCommand{Action: EmergencyAccessActionRevoke, GrantID: revoked.Grant.ID, ReasonCode: "duplicate-revoke", ExpectedVersion: &duplicateVersion, ReviewStatus: EmergencyAccessReviewPending, IdempotencyKey: "revoke-duplicate", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}); !errors.Is(err, ErrEmergencyAccessAlreadyRevoked) {
		t.Fatalf("duplicate revoke error = %v, want ErrEmergencyAccessAlreadyRevoked", err)
	}

	reviewVersion := revoked.Grant.Version
	completed, err := service.Execute(context.Background(), actor, EmergencyAccessGrantCommand{Action: EmergencyAccessActionRevoke, GrantID: revoked.Grant.ID, ReasonCode: "post-use-review", ExpectedVersion: &reviewVersion, ReviewStatus: EmergencyAccessReviewCompleted, ReviewOutcomeCode: "review-code-001", ReviewReference: "review-1", IdempotencyKey: "review-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if err != nil {
		t.Fatalf("review completion error = %v", err)
	}
	if completed.Grant.EffectiveReviewStatus(now) != EmergencyAccessReviewCompleted || completed.Grant.ReviewOutcomeCode != "review-code-001" {
		t.Fatalf("unexpected review state: %#v", completed.Grant)
	}
	if len(audit.Records) != 3 {
		t.Fatalf("audit records = %d, want 3", len(audit.Records))
	}
}

func TestEmergencyAccessExpiryAndReviewOverdueAreDerivedWithoutCleanup(t *testing.T) {
	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	repository := NewMemoryEmergencyAccessRepository()
	service, err := NewEmergencyAccessService(repository, emergencyAccessTestAuthorizer{}, AllowAllEmergencyAccessApprovalPort{}, &MemoryEmergencyAccessAuditRecorder{}, WeekdayEmergencyAccessCalendar{}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewEmergencyAccessService() error = %v", err)
	}
	grant, err := service.Execute(context.Background(), emergencyAccessTestActor(now), emergencyAccessGrantTestCommand(now, "grant-expiry"))
	if err != nil {
		t.Fatalf("grant error = %v", err)
	}
	if grant.Grant.EffectiveStatus(grant.Grant.ExpiresAt) != EmergencyAccessGrantStatusExpired {
		t.Fatal("expiry must deny access without a cleanup mutation")
	}
	if grant.Grant.EffectiveReviewStatus(grant.Grant.ReviewDueAt) != EmergencyAccessReviewOverdue {
		t.Fatal("pending review must become overdue at its deadline")
	}
	decision, err := NewEmergencyAccessEvaluator(repository, func() time.Time { return grant.Grant.ExpiresAt })
	if err != nil {
		t.Fatalf("NewEmergencyAccessEvaluator() error = %v", err)
	}
	access, err := decision.Evaluate(context.Background(), grant.Grant.TargetActorID, "finance.gl.submit.posting.request", "entity:100", grant.Grant.ID.String())
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if access.Allowed || access.GrantReference != grant.Grant.ID {
		t.Fatalf("expired access decision = %#v", access)
	}
}

func TestEmergencyAccessRejectsStaleAssuranceAndDuplicatePayload(t *testing.T) {
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	service, err := NewEmergencyAccessService(NewMemoryEmergencyAccessRepository(), emergencyAccessTestAuthorizer{}, AllowAllEmergencyAccessApprovalPort{}, &MemoryEmergencyAccessAuditRecorder{}, WeekdayEmergencyAccessCalendar{}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewEmergencyAccessService() error = %v", err)
	}
	stale := emergencyAccessTestActor(now)
	stale.Subject.Assurance.AuthenticatedAt = now.Add(-EmergencyAccessAssuranceMaxAge - time.Second)
	if _, err := service.Execute(context.Background(), stale, emergencyAccessGrantTestCommand(now, "stale")); !errors.Is(err, ErrEmergencyAccessStepUpRequired) {
		t.Fatalf("stale assurance error = %v, want ErrEmergencyAccessStepUpRequired", err)
	}

	actor := emergencyAccessTestActor(now)
	command := emergencyAccessGrantTestCommand(now, "duplicate")
	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatalf("first grant error = %v", err)
	}
	second, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatalf("duplicate grant error = %v", err)
	}
	if !second.Replayed || second.Grant.ID != first.Grant.ID {
		t.Fatalf("duplicate result = %#v", second)
	}
	command.ReasonCode = "different-reason"
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrEmergencyAccessIdempotencyConflict) {
		t.Fatalf("duplicate payload error = %v, want ErrEmergencyAccessIdempotencyConflict", err)
	}
}

func TestEmergencyAccessRejectsApprovalWithStalePolicyVersion(t *testing.T) {
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	repository := NewMemoryEmergencyAccessRepository()
	service, err := NewEmergencyAccessService(repository, emergencyAccessTestAuthorizer{}, AllowAllEmergencyAccessApprovalPort{}, &MemoryEmergencyAccessAuditRecorder{}, WeekdayEmergencyAccessCalendar{}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewEmergencyAccessService() error = %v", err)
	}
	command := emergencyAccessGrantTestCommand(now, "stale-approval-policy")
	command.Approval.PolicyVersion = "emergency-policy-v0"
	if _, err := service.Execute(context.Background(), emergencyAccessTestActor(now), command); !errors.Is(err, ErrEmergencyAccessApprovalRejected) {
		t.Fatalf("stale approval policy error = %v, want ErrEmergencyAccessApprovalRejected", err)
	}
}
