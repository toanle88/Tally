package coa

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestSegmentChangeApprovalDecisionAppliesDefinitionAndReplaysSafely(t *testing.T) {
	ctx := context.Background()
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "approval-subject"}
	audit := &MemoryAuditRecorder{}
	definitions := NewMemorySegmentDefinitionRepository()
	definitionService, err := NewSegmentDefinitionService(definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	created, err := definitionService.Execute(ctx, actor, testCreateCommand(scopeID, "approval-definition-create"))
	if err != nil {
		t.Fatal(err)
	}
	requestRepository := NewMemorySegmentChangeRequestRepository(definitions)
	requestService, err := NewSegmentChangeRequestService(requestRepository, definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	requestResult, err := requestService.Execute(ctx, actor, SegmentChangeRequestCommand{
		ChangeType: SegmentChangeTypeDefinition, SubjectID: created.SegmentDefinition.ID, SubjectVersion: created.SegmentDefinition.Version, ScopeID: scopeID,
		RequestedEffectiveDate: created.SegmentDefinition.EffectiveDateFrom, ApprovalRequestID: uuid.New(), IdempotencyKey: "approval-request",
		ProposedChange: SegmentChangeProposal{Action: SegmentChangeRequestAction, SegmentDefinitionID: created.SegmentDefinition.ID, SegmentType: created.SegmentDefinition.SegmentType, Code: created.SegmentDefinition.Code, Name: "Operations and Shared Services", Status: SegmentStatusActive, EffectiveDateFrom: created.SegmentDefinition.EffectiveDateFrom, EffectiveDateTo: created.SegmentDefinition.EffectiveDateTo},
	})
	if err != nil {
		t.Fatal(err)
	}
	approvalService, err := NewSegmentChangeApprovalDecisionService(requestRepository, requestRepository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeApprovalDecisionPermission, DecisionReference: uuid.New(), PolicyReference: "coa-apply-v1", ApprovedScopeIDs: []uuid.UUID{scopeID}}}, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	decision := SegmentChangeApprovalDecisionReference{ApprovalRequestID: requestResult.SegmentChangeRequest.ApprovalRequestID, DecisionID: uuid.New(), PolicyVersion: "coa-apply-v1", DecisionVersion: 1, SubjectVersion: mustAggregateVersion(requestResult.SegmentChangeRequest.SubjectVersion), CandidateFingerprint: requestResult.SegmentChangeRequest.ProposedFingerprint, ApproverUserID: uuid.New(), DecidedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	command := SegmentChangeApprovalDecisionCommand{ScopeID: scopeID, SegmentChangeRequestID: requestResult.SegmentChangeRequest.ID, Outcome: SegmentChangeApprovalOutcomeApproved, Approval: decision, IdempotencyKey: "approval-apply", CorrelationID: "approval-correlation"}
	result, err := approvalService.Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.ApplicationStatus != SegmentChangeRequestApplicationApplied || result.ResultingSubjectVersion != 2 || result.SegmentChangeRequest.ApprovalStatus != SegmentChangeRequestApprovalApproved {
		t.Fatalf("approval result = %#v", result)
	}
	stored, err := definitions.Get(ctx, created.SegmentDefinition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version.Value() != 2 || stored.Name != "Operations and Shared Services" || stored.Status != SegmentStatusActive || len(stored.Revisions) != 2 {
		t.Fatalf("applied definition = %#v", stored)
	}
	replayed, err := approvalService.Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.SegmentChangeRequest.ID != result.SegmentChangeRequest.ID || len(audit.Records) != 3 {
		t.Fatalf("replay = %#v, audit count = %d", replayed, len(audit.Records))
	}
	conflicting := command
	conflicting.Outcome = SegmentChangeApprovalOutcomeRejected
	if _, err := approvalService.Execute(ctx, actor, conflicting); !errors.Is(err, ErrSegmentChangeApprovalDecisionIdempotencyConflict) {
		t.Fatalf("reused idempotency key error = %v, want conflict", err)
	}
}

func TestSegmentChangeApprovalDecisionRejectsWithoutChangingSubject(t *testing.T) {
	ctx := context.Background()
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "approval-subject"}
	audit := &MemoryAuditRecorder{}
	definitions := NewMemorySegmentDefinitionRepository()
	definitionService, err := NewSegmentDefinitionService(definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	created, err := definitionService.Execute(ctx, actor, testCreateCommand(scopeID, "rejection-definition-create"))
	if err != nil {
		t.Fatal(err)
	}
	requestRepository := NewMemorySegmentChangeRequestRepository(definitions)
	requestService, err := NewSegmentChangeRequestService(requestRepository, definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	requestResult, err := requestService.Execute(ctx, actor, SegmentChangeRequestCommand{
		ChangeType: SegmentChangeTypeDefinition, SubjectID: created.SegmentDefinition.ID, SubjectVersion: created.SegmentDefinition.Version, ScopeID: scopeID,
		RequestedEffectiveDate: created.SegmentDefinition.EffectiveDateFrom, ApprovalRequestID: uuid.New(), IdempotencyKey: "rejection-request",
		ProposedChange: SegmentChangeProposal{Action: SegmentChangeRequestAction, SegmentDefinitionID: created.SegmentDefinition.ID, SegmentType: created.SegmentDefinition.SegmentType, Code: created.SegmentDefinition.Code, Name: "Rejected", Status: SegmentStatusActive, EffectiveDateFrom: created.SegmentDefinition.EffectiveDateFrom, EffectiveDateTo: created.SegmentDefinition.EffectiveDateTo},
	})
	if err != nil {
		t.Fatal(err)
	}
	approvalService, err := NewSegmentChangeApprovalDecisionService(requestRepository, requestRepository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeApprovalDecisionPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	decision := SegmentChangeApprovalDecisionReference{ApprovalRequestID: requestResult.SegmentChangeRequest.ApprovalRequestID, DecisionID: uuid.New(), PolicyVersion: "coa-apply-v1", DecisionVersion: 1, SubjectVersion: mustAggregateVersion(1), CandidateFingerprint: requestResult.SegmentChangeRequest.ProposedFingerprint, ApproverUserID: uuid.New(), DecidedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	invalidReference := decision
	invalidReference.CandidateFingerprint = "sha256:wrong"
	if _, err := approvalService.Execute(ctx, actor, SegmentChangeApprovalDecisionCommand{ScopeID: scopeID, SegmentChangeRequestID: requestResult.SegmentChangeRequest.ID, Outcome: SegmentChangeApprovalOutcomeRejected, Approval: invalidReference, IdempotencyKey: "invalid-reference"}); !errors.Is(err, ErrSegmentChangeApprovalDecisionReferenceInvalid) {
		t.Fatalf("invalid approval reference error = %v, want invalid reference", err)
	}
	result, err := approvalService.Execute(ctx, actor, SegmentChangeApprovalDecisionCommand{ScopeID: scopeID, SegmentChangeRequestID: requestResult.SegmentChangeRequest.ID, Outcome: SegmentChangeApprovalOutcomeRejected, Approval: decision, IdempotencyKey: "rejection-apply"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != SegmentChangeApprovalOutcomeRejected || result.ApplicationStatus != SegmentChangeRequestApplicationUnchanged || result.RejectionReason != "APPROVAL_REJECTED" {
		t.Fatalf("rejected result = %#v", result)
	}
	stored, err := definitions.Get(ctx, created.SegmentDefinition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version.Value() != 1 || stored.Name != "Operations" || stored.Status != SegmentStatusDraft {
		t.Fatalf("rejected subject = %#v", stored)
	}
}

func TestSegmentChangeApprovalDecisionRecordsUnchangedApprovedOutcome(t *testing.T) {
	ctx := context.Background()
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "approval-subject"}
	audit := &MemoryAuditRecorder{}
	definitions := NewMemorySegmentDefinitionRepository()
	definitionService, err := NewSegmentDefinitionService(definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	created, err := definitionService.Execute(ctx, actor, testCreateCommand(scopeID, "unchanged-definition-create"))
	if err != nil {
		t.Fatal(err)
	}
	requestRepository := NewMemorySegmentChangeRequestRepository(definitions)
	requestService, err := NewSegmentChangeRequestService(requestRepository, definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	requestResult, err := requestService.Execute(ctx, actor, SegmentChangeRequestCommand{
		ChangeType: SegmentChangeTypeDefinition, SubjectID: created.SegmentDefinition.ID, SubjectVersion: created.SegmentDefinition.Version, ScopeID: scopeID,
		RequestedEffectiveDate: created.SegmentDefinition.EffectiveDateFrom, ApprovalRequestID: uuid.New(), IdempotencyKey: "unchanged-request",
		ProposedChange: SegmentChangeProposal{Action: SegmentChangeRequestAction, SegmentDefinitionID: created.SegmentDefinition.ID, SegmentType: created.SegmentDefinition.SegmentType, Code: created.SegmentDefinition.Code, Name: created.SegmentDefinition.Name, Status: created.SegmentDefinition.Status, EffectiveDateFrom: created.SegmentDefinition.EffectiveDateFrom, EffectiveDateTo: created.SegmentDefinition.EffectiveDateTo},
	})
	if err != nil {
		t.Fatal(err)
	}
	approvalService, err := NewSegmentChangeApprovalDecisionService(requestRepository, requestRepository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeApprovalDecisionPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	result, err := approvalService.Execute(ctx, actor, SegmentChangeApprovalDecisionCommand{
		ScopeID: scopeID, SegmentChangeRequestID: requestResult.SegmentChangeRequest.ID, Outcome: SegmentChangeApprovalOutcomeApproved,
		Approval:       SegmentChangeApprovalDecisionReference{ApprovalRequestID: requestResult.SegmentChangeRequest.ApprovalRequestID, DecisionID: uuid.New(), PolicyVersion: "coa-apply-v1", DecisionVersion: 1, SubjectVersion: mustAggregateVersion(1), CandidateFingerprint: requestResult.SegmentChangeRequest.ProposedFingerprint, ApproverUserID: uuid.New(), DecidedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		IdempotencyKey: "unchanged-apply",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ApplicationStatus != SegmentChangeRequestApplicationUnchanged || result.ResultingSubjectVersion != 1 || result.AppliedSubjectVersion != 1 {
		t.Fatalf("unchanged result = %#v", result)
	}
	stored, err := definitions.Get(ctx, created.SegmentDefinition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version.Value() != 1 || len(stored.Revisions) != 1 || stored.Name != created.SegmentDefinition.Name {
		t.Fatalf("unchanged subject = %#v", stored)
	}
}

func TestSegmentChangeApprovalDecisionRecordsStaleConflict(t *testing.T) {
	ctx := context.Background()
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "approval-subject"}
	audit := &MemoryAuditRecorder{}
	definitions := NewMemorySegmentDefinitionRepository()
	definitionService, err := NewSegmentDefinitionService(definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	created, err := definitionService.Execute(ctx, actor, testCreateCommand(scopeID, "stale-definition-create"))
	if err != nil {
		t.Fatal(err)
	}
	requestRepository := NewMemorySegmentChangeRequestRepository(definitions)
	requestService, err := NewSegmentChangeRequestService(requestRepository, definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	requestResult, err := requestService.Execute(ctx, actor, SegmentChangeRequestCommand{ChangeType: SegmentChangeTypeDefinition, SubjectID: created.SegmentDefinition.ID, SubjectVersion: created.SegmentDefinition.Version, ScopeID: scopeID, RequestedEffectiveDate: created.SegmentDefinition.EffectiveDateFrom, ApprovalRequestID: uuid.New(), IdempotencyKey: "stale-request", ProposedChange: SegmentChangeProposal{Action: SegmentChangeRequestAction, SegmentDefinitionID: created.SegmentDefinition.ID, SegmentType: created.SegmentDefinition.SegmentType, Code: created.SegmentDefinition.Code, Name: "Stale", Status: SegmentStatusActive, EffectiveDateFrom: created.SegmentDefinition.EffectiveDateFrom, EffectiveDateTo: created.SegmentDefinition.EffectiveDateTo}})
	if err != nil {
		t.Fatal(err)
	}
	version := created.SegmentDefinition.Version
	_, err = definitionService.Execute(ctx, actor, SegmentDefinitionCommand{Action: SegmentDefinitionActionUpdate, SegmentDefinitionID: created.SegmentDefinition.ID, ScopeID: scopeID, SegmentType: created.SegmentDefinition.SegmentType, Code: created.SegmentDefinition.Code, Name: "Independent update", Status: SegmentStatusActive, EffectiveDateFrom: created.SegmentDefinition.EffectiveDateFrom, EffectiveDateTo: created.SegmentDefinition.EffectiveDateTo, ExpectedVersion: &version, IdempotencyKey: "stale-independent-update"})
	if err != nil {
		t.Fatal(err)
	}
	approvalService, err := NewSegmentChangeApprovalDecisionService(requestRepository, requestRepository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeApprovalDecisionPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	decision := SegmentChangeApprovalDecisionReference{ApprovalRequestID: requestResult.SegmentChangeRequest.ApprovalRequestID, DecisionID: uuid.New(), PolicyVersion: "coa-apply-v1", DecisionVersion: 1, SubjectVersion: mustAggregateVersion(1), CandidateFingerprint: requestResult.SegmentChangeRequest.ProposedFingerprint, ApproverUserID: uuid.New(), DecidedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	result, err := approvalService.Execute(ctx, actor, SegmentChangeApprovalDecisionCommand{ScopeID: scopeID, SegmentChangeRequestID: requestResult.SegmentChangeRequest.ID, Outcome: SegmentChangeApprovalOutcomeApproved, Approval: decision, IdempotencyKey: "stale-apply"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ApplicationStatus != SegmentChangeRequestApplicationConflict || result.ConflictCode != "STALE_SUBJECT" || result.SegmentChangeRequest.NextAction != SegmentChangeRequestNextActionResolveConflict {
		t.Fatalf("stale result = %#v", result)
	}
}

func mustAggregateVersion(value int64) aggregateversion.AggregateVersion {
	return aggregateversion.AggregateVersion(value)
}
