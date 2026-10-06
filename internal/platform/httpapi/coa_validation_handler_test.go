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

func TestCoaHandlerValidatesSegmentCombinationReadOnlyAndReplays(t *testing.T) {
	scopeID := uuid.New()
	repository := coa.NewMemorySegmentDefinitionRepository()
	audit := &coa.MemoryAuditRecorder{}
	definitionService, err := coa.NewSegmentDefinitionService(repository, coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{
		Allowed: true, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}}, audit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	valueService, err := coa.NewSegmentValueService(repository, coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{
		Allowed: true, Permission: coa.SegmentValueManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}}, audit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	validationService, err := coa.NewSegmentCombinationValidationService(repository, coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{
		Allowed: true, Permission: coa.SegmentCombinationValidationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}})
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SegmentDefinitionService: definitionService, SegmentValueService: valueService, SegmentCombinationValidationService: validationService}
	definitionResponse, err := handler.CoaMaintainSegmentDefinitions(ctx, &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action": mustRaw("create"), "segmentType": mustRaw("department"), "code": mustRaw("D-001"), "name": mustRaw("Operations"), "status": mustRaw("active"), "effectiveDateFrom": mustRaw("2026-01-01"),
		},
	}, generated.CoaMaintainSegmentDefinitionsParams{IdempotencyKey: "validation-definition"})
	if err != nil {
		t.Fatal(err)
	}
	definitionEstablished, ok := definitionResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("definition response = %T", definitionResponse)
	}
	definitionID := uuid.UUID(definitionEstablished.AggregateId)
	valueResponse, err := handler.CoaMaintainSegmentValues(ctx, &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), ExpectedVersion: generated.NewOptInt(1), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action": mustRaw("create"), "segmentDefinitionId": mustRaw(definitionID.String()), "value": mustRaw("1000"), "description": mustRaw("Operations"), "status": mustRaw("active"), "effectiveDateFrom": mustRaw("2026-01-01"),
		},
	}, generated.CoaMaintainSegmentValuesParams{IdempotencyKey: "validation-value", IfMatch: generated.NewOptString(`"1"`)})
	if err != nil {
		t.Fatal(err)
	}
	valueEstablished, ok := valueResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("value response = %T", valueResponse)
	}
	var valueProjection struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(valueEstablished.Data["segmentValue"], &valueProjection); err != nil {
		t.Fatal(err)
	}
	valueID, err := uuid.Parse(valueProjection.ID)
	if err != nil {
		t.Fatal(err)
	}

	request := &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		BusinessDate:      generated.NewOptDate(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
		Data: generated.CommandRequestData{
			"segmentValues": mustRaw([]map[string]string{{"segmentDefinitionId": definitionID.String(), "segmentValueId": valueID.String()}}),
		},
	}
	response, err := handler.CoaValidateSegmentCombinations(ctx, request, generated.CoaValidateSegmentCombinationsParams{IdempotencyKey: "validation-1"})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := response.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("validation response = %T", response)
	}
	if result.AggregateId != generated.UUID(uuid.Nil) || result.AggregateVersion != 0 || !strings.Contains(string(result.Data["validationStatus"]), "valid") || !strings.Contains(string(result.Data["effectiveDateResult"]), "effective") {
		t.Fatalf("validation result = %#v", result)
	}
	if strings.Contains(string(result.Data["sourceVersions"]), "revisions") {
		t.Fatalf("validation result leaked revision history: %s", result.Data["sourceVersions"])
	}

	request.CommandId = generated.UUID(uuid.New())
	replayedResponse, err := handler.CoaValidateSegmentCombinations(ctx, request, generated.CoaValidateSegmentCombinationsParams{IdempotencyKey: "validation-1"})
	if err != nil {
		t.Fatal(err)
	}
	replayed, ok := replayedResponse.(*generated.EstablishedResult)
	if !ok || !strings.Contains(string(replayed.Data["replayed"]), "true") {
		t.Fatalf("replayed response = %#v", replayedResponse)
	}

	request.Data["segmentValues"] = mustRaw([]map[string]string{{"segmentDefinitionId": definitionID.String(), "segmentValueId": uuid.New().String()}})
	conflictResponse, err := handler.CoaValidateSegmentCombinations(ctx, request, generated.CoaValidateSegmentCombinationsParams{IdempotencyKey: "validation-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := conflictResponse.(*generated.CoaValidateSegmentCombinationsConflict); !ok {
		t.Fatalf("changed-content response = %T", conflictResponse)
	}
	definitionAfter, err := definitionService.GetSafe(ctx, definitionID, &scopeID)
	if err != nil {
		t.Fatal(err)
	}
	if definitionAfter.Version.Value() != 2 {
		t.Fatalf("validation changed definition version to %d", definitionAfter.Version.Value())
	}
}

func TestCoaHandlerMapsSegmentCombinationAuthorizationProblem(t *testing.T) {
	scopeID := uuid.New()
	validationService, err := coa.NewSegmentCombinationValidationService(
		coa.NewMemorySegmentDefinitionRepository(),
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{
			Allowed: false, Permission: coa.SegmentCombinationValidationPermission, DecisionReference: uuid.New(),
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	correlationID := uuid.New()
	response, err := (IdentityHandler{SegmentCombinationValidationService: validationService}).CoaValidateSegmentCombinations(ctx, &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		BusinessDate:      generated.NewOptDate(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
		Data: generated.CommandRequestData{
			"segmentValues": mustRaw([]map[string]string{{"segmentDefinitionId": uuid.New().String(), "segmentValueId": uuid.New().String()}}),
		},
	}, generated.CoaValidateSegmentCombinationsParams{
		XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID)), IdempotencyKey: "validation-denied-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	problem, ok := response.(*generated.CoaValidateSegmentCombinationsForbidden)
	if !ok || problem.Code != "AUTHORIZATION_DENIED" || problem.CorrelationId != generated.UUID(correlationID) {
		t.Fatalf("authorization response = %#v", response)
	}
}
