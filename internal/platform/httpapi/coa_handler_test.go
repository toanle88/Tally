package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/coa"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestCoaHandlerMaintainsSegmentDefinitionAndMapsSafeResult(t *testing.T) {
	scopeID := uuid.New()
	service, err := coa.NewSegmentDefinitionService(
		coa.NewMemorySegmentDefinitionRepository(),
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&coa.MemoryAuditRecorder{},
		func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SegmentDefinitionService: service}
	correlationID := uuid.New()
	request := &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action":            mustRaw("create"),
			"segmentType":       mustRaw("department"),
			"code":              mustRaw("D-001"),
			"name":              mustRaw("Operations"),
			"status":            mustRaw("draft"),
			"effectiveDateFrom": mustRaw("2026-01-01"),
			"effectiveDateTo":   mustRaw("2026-12-31"),
		},
	}
	response, err := handler.CoaMaintainSegmentDefinitions(ctx, request, generated.CoaMaintainSegmentDefinitionsParams{
		XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID)),
		IdempotencyKey: "coa-create-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	created, ok := response.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("create response = %T, want established result", response)
	}
	if created.CorrelationId != generated.UUID(correlationID) || created.AggregateVersion != 1 {
		t.Fatalf("created result = %#v", created)
	}
	if !strings.Contains(string(created.Data["approvalStatus"]), "not-required") || !strings.Contains(string(created.Data["validationOutcome"]), "valid") {
		t.Fatalf("safe state data = %#v", created.Data)
	}
	body, err := json.Marshal(created)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "revisions") {
		t.Fatalf("safe result leaked revision history: %s", body)
	}

	update := &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		ExpectedVersion:   generated.NewOptInt(1),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action":              mustRaw("update"),
			"segmentDefinitionId": mustRaw(uuid.UUID(created.AggregateId).String()),
			"segmentType":         mustRaw("department"),
			"code":                mustRaw("D-001"),
			"name":                mustRaw("Operations and Shared Services"),
			"status":              mustRaw("active"),
			"effectiveDateFrom":   mustRaw("2026-01-01"),
			"effectiveDateTo":     mustRaw("2026-12-31"),
		},
	}
	updatedResponse, err := handler.CoaMaintainSegmentDefinitions(ctx, update, generated.CoaMaintainSegmentDefinitionsParams{
		IdempotencyKey: "coa-update-1",
		IfMatch:        generated.NewOptString("W/\"1\""),
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, ok := updatedResponse.(*generated.EstablishedResult)
	if !ok || updated.AggregateVersion != 2 {
		t.Fatalf("updated response = %#v", updatedResponse)
	}
}

func TestCoaHandlerMapsDenialAndValidation(t *testing.T) {
	scopeID := uuid.New()
	service, err := coa.NewSegmentDefinitionService(
		coa.NewMemorySegmentDefinitionRepository(),
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: false, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New()}},
		&coa.MemoryAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SegmentDefinitionService: service}
	request := &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action":            mustRaw("create"),
			"segmentType":       mustRaw("department"),
			"code":              mustRaw("D-001"),
			"name":              mustRaw("Operations"),
			"status":            mustRaw("draft"),
			"effectiveDateFrom": mustRaw("2026-01-01"),
		},
	}
	response, err := handler.CoaMaintainSegmentDefinitions(ctx, request, generated.CoaMaintainSegmentDefinitionsParams{IdempotencyKey: "coa-denied-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.CoaMaintainSegmentDefinitionsForbidden); !ok {
		t.Fatalf("denied response = %T, want forbidden", response)
	}

	invalidService, err := coa.NewSegmentDefinitionService(
		coa.NewMemorySegmentDefinitionRepository(),
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&coa.MemoryAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	invalidHandler := IdentityHandler{SegmentDefinitionService: invalidService}
	request.Data["status"] = mustRaw("retired")
	response, err = invalidHandler.CoaMaintainSegmentDefinitions(ctx, request, generated.CoaMaintainSegmentDefinitionsParams{IdempotencyKey: "coa-invalid-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.CoaMaintainSegmentDefinitionsBadRequest); !ok {
		t.Fatalf("invalid response = %T, want bad request", response)
	}
}

func TestCoaHandlerMaintainsSegmentValueWithParentVersion(t *testing.T) {
	scopeID := uuid.New()
	repository := coa.NewMemorySegmentDefinitionRepository()
	definitionService, err := coa.NewSegmentDefinitionService(
		repository,
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&coa.MemoryAuditRecorder{},
		func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	valueService, err := coa.NewSegmentValueService(
		repository,
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentValueManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&coa.MemoryAuditRecorder{},
		func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SegmentDefinitionService: definitionService, SegmentValueService: valueService}
	definitionRequest := &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action":            mustRaw("create"),
			"segmentType":       mustRaw("department"),
			"code":              mustRaw("D-001"),
			"name":              mustRaw("Operations"),
			"status":            mustRaw("draft"),
			"effectiveDateFrom": mustRaw("2026-01-01"),
		},
	}
	definitionResponse, err := handler.CoaMaintainSegmentDefinitions(ctx, definitionRequest, generated.CoaMaintainSegmentDefinitionsParams{IdempotencyKey: "definition-create"})
	if err != nil {
		t.Fatal(err)
	}
	definition, ok := definitionResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("definition response = %T, want established result", definitionResponse)
	}
	valueResponse, err := handler.CoaMaintainSegmentValues(ctx, &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		ExpectedVersion:   generated.NewOptInt(1),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action":              mustRaw("create"),
			"segmentDefinitionId": mustRaw(uuid.UUID(definition.AggregateId).String()),
			"value":               mustRaw("1000"),
			"description":         mustRaw("Operations"),
			"status":              mustRaw("active"),
			"effectiveDateFrom":   mustRaw("2026-01-01"),
		},
	}, generated.CoaMaintainSegmentValuesParams{IdempotencyKey: "value-create", IfMatch: generated.NewOptString(`"1"`)})
	if err != nil {
		t.Fatal(err)
	}
	value, ok := valueResponse.(*generated.EstablishedResult)
	if !ok || value.AggregateId != definition.AggregateId || value.AggregateVersion != 2 {
		t.Fatalf("value response = %#v", valueResponse)
	}
	if !strings.Contains(string(value.Data["segmentValue"]), "1000") {
		t.Fatalf("value safe projection = %#v", value.Data)
	}
}

func TestCoaHandlerRequestsSegmentChangeWithApprovalReference(t *testing.T) {
	scopeID := uuid.New()
	repository := coa.NewMemorySegmentDefinitionRepository()
	audit := &coa.MemoryAuditRecorder{}
	definitionService, err := coa.NewSegmentDefinitionService(
		repository,
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		audit,
		func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	requestRepository := coa.NewMemorySegmentChangeRequestRepository(repository)
	requestService, err := coa.NewSegmentChangeRequestService(
		requestRepository,
		repository,
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		audit,
		func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SegmentDefinitionService: definitionService, SegmentChangeRequestService: requestService}
	createdResponse, err := handler.CoaMaintainSegmentDefinitions(ctx, &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action": mustRaw("create"), "segmentType": mustRaw("department"), "code": mustRaw("D-901"), "name": mustRaw("Finance"),
			"status": mustRaw("draft"), "effectiveDateFrom": mustRaw("2026-01-01"), "effectiveDateTo": mustRaw("2026-12-31"),
		},
	}, generated.CoaMaintainSegmentDefinitionsParams{IdempotencyKey: "handler-request-definition"})
	if err != nil {
		t.Fatal(err)
	}
	created, ok := createdResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("created response = %T", createdResponse)
	}
	approvalRequestID := uuid.New()
	correlationID := uuid.New()
	response, err := handler.CoaRequestSegmentChanges(ctx, &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"changeType": mustRaw("definition"), "subjectId": mustRaw(uuid.UUID(created.AggregateId).String()), "subjectVersion": mustRaw(1),
			"requestedEffectiveDate": mustRaw("2026-01-01"), "approvalRequestId": mustRaw(approvalRequestID.String()),
			"proposedChange": mustRaw(map[string]any{
				"action": "request", "segmentDefinitionId": uuid.UUID(created.AggregateId).String(), "segmentType": "department", "code": "D-901",
				"name": "Finance and Shared Services", "status": "active", "effectiveDateFrom": "2026-01-01", "effectiveDateTo": "2026-12-31",
			}),
		},
	}, generated.CoaRequestSegmentChangesParams{IdempotencyKey: "handler-request-1", XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID))})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := response.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("request response = %T", response)
	}
	if result.Status != "established" || result.AggregateVersion != 1 || result.CorrelationId != generated.UUID(correlationID) {
		t.Fatalf("request result = %#v", result)
	}
	if string(result.Data["approvalStatus"]) != `"pending"` || string(result.Data["subjectVersion"]) != "1" || string(result.Data["approvalRequestId"]) != `"`+approvalRequestID.String()+`"` {
		t.Fatalf("request state = %#v", result.Data)
	}
	if !strings.Contains(string(result.Data["segmentChangeRequest"]), "Finance and Shared Services") {
		t.Fatalf("request projection = %#v", result.Data["segmentChangeRequest"])
	}
	unchanged, err := repository.Get(context.Background(), uuid.UUID(created.AggregateId))
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Version.Value() != 1 || unchanged.Name != "Finance" {
		t.Fatalf("subject was mutated by request = %#v", unchanged)
	}

	approvalDecisionID := uuid.New()
	approverUserID := uuid.New()
	approvalService, err := coa.NewSegmentChangeApprovalDecisionService(
		requestRepository,
		requestRepository,
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentChangeApprovalDecisionPermission, DecisionReference: uuid.New(), PolicyReference: "coa-apply-v1", ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		func() time.Time { return time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	handler.SegmentChangeApprovalDecisionService = approvalService
	storedRequest, err := requestRepository.Get(context.Background(), uuid.UUID(result.AggregateId))
	if err != nil {
		t.Fatal(err)
	}
	approvalResponse, err := handler.CoaApplySegmentChangeApprovalDecision(ctx, &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"segmentChangeRequestId": mustRaw(uuid.UUID(result.AggregateId).String()), "outcome": mustRaw("approved"),
			"approvalRequestId": mustRaw(approvalRequestID.String()), "decisionId": mustRaw(approvalDecisionID.String()),
			"policyVersion": mustRaw("coa-apply-v1"), "decisionVersion": mustRaw(1), "subjectVersion": mustRaw(1),
			"candidateFingerprint": mustRaw(storedRequest.ProposedFingerprint), "approverUserId": mustRaw(approverUserID.String()),
			"decidedAt": mustRaw("2026-01-01T01:00:00Z"),
		},
	}, generated.CoaApplySegmentChangeApprovalDecisionParams{IdempotencyKey: "handler-approval-1", XCorrelationID: generated.NewOptUUID(generated.UUID(uuid.New()))})
	if err != nil {
		t.Fatal(err)
	}
	approvalResult, ok := approvalResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("approval response = %T, want established result: %#v", approvalResponse, approvalResponse)
	}
	if string(approvalResult.Data["applicationStatus"]) != `"applied"` || string(approvalResult.Data["resultingSubjectVersion"]) != "2" {
		t.Fatalf("approval state = %#v", approvalResult.Data)
	}
	applied, err := repository.Get(context.Background(), uuid.UUID(created.AggregateId))
	if err != nil {
		t.Fatal(err)
	}
	if applied.Version.Value() != 2 || applied.Name != "Finance and Shared Services" || applied.Status != coa.SegmentStatusActive {
		t.Fatalf("subject after approval = %#v", applied)
	}
}

func TestCoaHandlerMapsUnsupportedSegmentChangeSubject(t *testing.T) {
	scopeID := uuid.New()
	repository := coa.NewMemorySegmentDefinitionRepository()
	service, err := coa.NewSegmentChangeRequestService(
		coa.NewMemorySegmentChangeRequestRepository(repository), repository,
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&coa.MemoryAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	response, err := (IdentityHandler{SegmentChangeRequestService: service}).CoaRequestSegmentChanges(ctx, &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"changeType": mustRaw("combination"), "subjectId": mustRaw(uuid.NewString()), "subjectVersion": mustRaw(1),
			"requestedEffectiveDate": mustRaw("2026-01-01"), "approvalRequestId": mustRaw(uuid.NewString()),
			"proposedChange": mustRaw(map[string]any{"action": "request", "status": "active", "effectiveDateFrom": "2026-01-01"}),
		},
	}, generated.CoaRequestSegmentChangesParams{IdempotencyKey: "handler-unsupported-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.CoaRequestSegmentChangesUnprocessableEntity); !ok {
		t.Fatalf("unsupported response = %T, want unprocessable entity", response)
	}
}
