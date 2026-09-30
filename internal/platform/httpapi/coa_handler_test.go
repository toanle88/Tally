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
