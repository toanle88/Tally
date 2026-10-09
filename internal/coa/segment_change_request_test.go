package coa

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestSegmentChangeRequestCapturesDefinitionWithoutMutatingSubject(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	subjectAudit := &MemoryAuditRecorder{}
	definitions := NewMemorySegmentDefinitionRepository()
	definitionService, err := NewSegmentDefinitionService(
		definitions,
		MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		subjectAudit,
		fixedCoaClock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	created, err := definitionService.Execute(context.Background(), actor, testCreateCommand(scopeID, "definition-create"))
	if err != nil {
		t.Fatal(err)
	}
	definition := created.SegmentDefinition
	approvalRequestID := uuid.New()
	requestRepository := NewMemorySegmentChangeRequestRepository(definitions)
	requestService, err := NewSegmentChangeRequestService(
		requestRepository,
		definitions,
		MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		subjectAudit,
		fixedCoaClock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	command := SegmentChangeRequestCommand{
		ChangeType: SegmentChangeTypeDefinition, SubjectID: definition.ID, SubjectVersion: definition.Version, ScopeID: scopeID,
		RequestedEffectiveDate: definition.EffectiveDateFrom, ApprovalRequestID: approvalRequestID, IdempotencyKey: "request-definition-1",
		CorrelationID: "correlation-1", CausationID: uuid.NewString(),
		ProposedChange: SegmentChangeProposal{
			Action: SegmentChangeRequestAction, SegmentDefinitionID: definition.ID, SegmentType: definition.SegmentType, Code: definition.Code,
			Name: "Operations and Shared Services", Status: SegmentStatusActive, EffectiveDateFrom: definition.EffectiveDateFrom, EffectiveDateTo: definition.EffectiveDateTo,
		},
	}
	result, err := requestService.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.SegmentChangeRequest.ID == uuid.Nil || result.SegmentChangeRequest.SubjectID != definition.ID || result.SegmentChangeRequest.SubjectVersion != 1 || result.SegmentChangeRequest.ApprovalRequestID != approvalRequestID {
		t.Fatalf("request projection = %#v", result.SegmentChangeRequest)
	}
	if result.SegmentChangeRequest.ApprovalStatus != SegmentChangeRequestApprovalPending || result.SegmentChangeRequest.ApplicationStatus != SegmentChangeRequestApplicationOpen || result.SegmentChangeRequest.NextAction != SegmentChangeRequestNextAction {
		t.Fatalf("request lifecycle = %#v", result.SegmentChangeRequest)
	}
	if result.SegmentChangeRequest.ApprovalDecisionID != nil || result.SegmentChangeRequest.ApprovalApproverUserID != nil || result.SegmentChangeRequest.ApprovalDecidedAt != nil || result.SegmentChangeRequest.ApprovalAppliedAt != nil {
		t.Fatalf("pending request exposed unset approval evidence = %#v", result.SegmentChangeRequest)
	}
	unchanged, err := definitions.Get(context.Background(), definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Version.Value() != 1 || unchanged.Status != SegmentStatusDraft || unchanged.Name != "Operations" {
		t.Fatalf("subject mutated while requesting change = %#v", unchanged)
	}
	if len(subjectAudit.Records) != 2 || subjectAudit.Records[1].SegmentChangeRequestID != result.SegmentChangeRequest.ID || subjectAudit.Records[1].ApprovalRequestID != approvalRequestID {
		t.Fatalf("audit records = %#v", subjectAudit.Records)
	}
	replayed, err := requestService.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.SegmentChangeRequest.ID != result.SegmentChangeRequest.ID || len(subjectAudit.Records) != 2 {
		t.Fatalf("replay = %#v, audit count = %d", replayed, len(subjectAudit.Records))
	}
	changed := command
	changed.ProposedChange.Name = "Different proposal"
	if _, err := requestService.Execute(context.Background(), actor, changed); !errors.Is(err, ErrSegmentChangeRequestIdempotencyConflict) {
		t.Fatalf("changed replay error = %v, want idempotency conflict", err)
	}
}

func TestSegmentChangeRequestCapturesValueParentVersion(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	definitions := NewMemorySegmentDefinitionRepository()
	audit := &MemoryAuditRecorder{}
	definitionService, err := NewSegmentDefinitionService(definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	created, err := definitionService.Execute(context.Background(), actor, testCreateCommand(scopeID, "value-definition-create"))
	if err != nil {
		t.Fatal(err)
	}
	valueService, err := NewSegmentValueService(definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentValueManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	parentVersion := aggregateversion.AggregateVersion(created.SegmentDefinition.Version.Value())
	valueResult, err := valueService.Execute(context.Background(), actor, SegmentValueCommand{
		Action: SegmentValueActionCreate, SegmentDefinitionID: created.SegmentDefinition.ID, ScopeID: scopeID, Value: "1000", Description: "Operations",
		Status: SegmentStatusActive, EffectiveDateFrom: created.SegmentDefinition.EffectiveDateFrom, EffectiveDateTo: created.SegmentDefinition.EffectiveDateTo, ExpectedVersion: &parentVersion, IdempotencyKey: "value-create-for-request",
	})
	if err != nil {
		t.Fatal(err)
	}
	requestRepository := NewMemorySegmentChangeRequestRepository(definitions)
	requestService, err := NewSegmentChangeRequestService(requestRepository, definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	currentParent, err := definitions.Get(context.Background(), created.SegmentDefinition.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := requestService.Execute(context.Background(), actor, SegmentChangeRequestCommand{
		ChangeType: SegmentChangeTypeValue, SubjectID: valueResult.SegmentValue.ID, SubjectVersion: currentParent.Version, ScopeID: scopeID,
		RequestedEffectiveDate: valueResult.SegmentValue.EffectiveDateFrom, ApprovalRequestID: uuid.New(), IdempotencyKey: "request-value-1",
		ProposedChange: SegmentChangeProposal{
			Action: SegmentChangeRequestAction, SegmentValueID: valueResult.SegmentValue.ID, SegmentDefinitionID: currentParent.ID, Value: "1000",
			Description: "Operations and shared services", Status: SegmentStatusActive, EffectiveDateFrom: valueResult.SegmentValue.EffectiveDateFrom, EffectiveDateTo: valueResult.SegmentValue.EffectiveDateTo,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SegmentChangeRequest.SubjectVersion != currentParent.Version.Value() || result.SegmentChangeRequest.ChangeType != SegmentChangeTypeValue {
		t.Fatalf("value request projection = %#v", result.SegmentChangeRequest)
	}
	unchanged, err := definitions.Get(context.Background(), currentParent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Version.Value() != currentParent.Version.Value() || unchanged.Values[0].Description != "Operations" {
		t.Fatalf("value subject mutated = %#v", unchanged)
	}
}

func TestSegmentChangeRequestRejectsUnsupportedStaleAndUnauthorizedCommands(t *testing.T) {
	scopeID := uuid.New()
	otherScopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	definitions := NewMemorySegmentDefinitionRepository()
	audit := &MemoryAuditRecorder{}
	definitionService, err := NewSegmentDefinitionService(definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	created, err := definitionService.Execute(context.Background(), actor, testCreateCommand(scopeID, "request-rejection-definition"))
	if err != nil {
		t.Fatal(err)
	}
	base := SegmentChangeRequestCommand{
		ChangeType: SegmentChangeTypeDefinition, SubjectID: created.SegmentDefinition.ID, SubjectVersion: created.SegmentDefinition.Version, ScopeID: scopeID,
		RequestedEffectiveDate: created.SegmentDefinition.EffectiveDateFrom, ApprovalRequestID: uuid.New(), IdempotencyKey: "request-rejection-base",
		ProposedChange: SegmentChangeProposal{Action: SegmentChangeRequestAction, SegmentDefinitionID: created.SegmentDefinition.ID, SegmentType: created.SegmentDefinition.SegmentType, Code: created.SegmentDefinition.Code, Name: "Changed", Status: SegmentStatusActive, EffectiveDateFrom: created.SegmentDefinition.EffectiveDateFrom},
	}
	requestRepository := NewMemorySegmentChangeRequestRepository(definitions)
	service, err := NewSegmentChangeRequestService(requestRepository, definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	stale := base
	stale.SubjectVersion = aggregateversion.AggregateVersion(2)
	if _, err := service.Execute(context.Background(), actor, stale); !errors.Is(err, ErrSegmentChangeRequestVersionConflict) {
		t.Fatalf("stale error = %v, want version conflict", err)
	}
	unauthorized := base
	unauthorized.IdempotencyKey = "request-unauthorized"
	unauthorized.ScopeID = otherScopeID
	if _, err := service.Execute(context.Background(), actor, unauthorized); !errors.Is(err, ErrSegmentChangeRequestAuthorizationDenied) {
		t.Fatalf("unauthorized scope error = %v, want authorization denied", err)
	}
	unsupported := base
	unsupported.ChangeType = "combination"
	unsupported.IdempotencyKey = "request-unsupported"
	if _, err := service.Execute(context.Background(), actor, unsupported); !errors.Is(err, ErrSegmentChangeRequestUnsupportedSubject) {
		t.Fatalf("unsupported error = %v, want unsupported subject", err)
	}
}

func TestSegmentChangeRequestRejectsAuditFailureAtomically(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	definitions := NewMemorySegmentDefinitionRepository()
	definitionAudit := &MemoryAuditRecorder{}
	definitionService, err := NewSegmentDefinitionService(definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, definitionAudit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	created, err := definitionService.Execute(context.Background(), actor, testCreateCommand(scopeID, "request-audit-definition"))
	if err != nil {
		t.Fatal(err)
	}
	failedAudit := &MemoryAuditRecorder{Err: errors.New("audit unavailable")}
	requestRepository := NewMemorySegmentChangeRequestRepository(definitions)
	service, err := NewSegmentChangeRequestService(requestRepository, definitions, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, failedAudit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Execute(context.Background(), actor, SegmentChangeRequestCommand{
		ChangeType: SegmentChangeTypeDefinition, SubjectID: created.SegmentDefinition.ID, SubjectVersion: created.SegmentDefinition.Version, ScopeID: scopeID,
		RequestedEffectiveDate: created.SegmentDefinition.EffectiveDateFrom, ApprovalRequestID: uuid.New(), IdempotencyKey: "request-audit-failure",
		ProposedChange: SegmentChangeProposal{Action: SegmentChangeRequestAction, SegmentDefinitionID: created.SegmentDefinition.ID, SegmentType: created.SegmentDefinition.SegmentType, Code: created.SegmentDefinition.Code, Name: "Changed", Status: SegmentStatusActive, EffectiveDateFrom: created.SegmentDefinition.EffectiveDateFrom},
	})
	if err == nil || !errors.Is(err, failedAudit.Err) {
		t.Fatalf("audit error = %v, want audit failure", err)
	}
	if _, err := requestRepository.Get(context.Background(), uuid.Nil); !errors.Is(err, ErrSegmentChangeRequestSubjectNotFound) {
		t.Fatalf("request lookup after failed audit = %v", err)
	}
}
