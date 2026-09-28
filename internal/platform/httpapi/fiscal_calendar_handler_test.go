package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestFiscalCalendarHandlerReturnsEstablishedResultWithUnavailableImpact(t *testing.T) {
	scopeID := uuid.New()
	service, err := organization.NewFiscalCalendarService(
		organization.NewMemoryFiscalCalendarRepository(),
		organization.MemoryFiscalCalendarAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.FiscalCalendarManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.AllowAllFiscalCalendarApprovalValidator{}, &organization.MemoryFiscalCalendarAuditRecorder{}, organization.UnavailableFiscalCalendarImpactReader{},
		func() time.Time { return time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	request := &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action":        mustRaw("create"),
			"calendarType":  mustRaw("gregorian"),
			"periodPattern": mustRaw("quarterly"),
			"effectiveFrom": mustRaw("2026-01-01"),
			"periods": mustRaw([]map[string]any{
				{"reference": "q1", "ordinal": 1, "startDate": "2026-01-01", "endDate": "2026-03-31"},
				{"reference": "q2", "ordinal": 2, "startDate": "2026-04-01", "endDate": "2026-06-30"},
			}),
		},
	}
	response, err := (IdentityHandler{FiscalCalendarService: service}).OmdMaintainFiscalCalendars(ctx, request, generated.OmdMaintainFiscalCalendarsParams{IdempotencyKey: "calendar-http-create"})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := response.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("response = %T, want established result", response)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "gregorian") || !strings.Contains(string(body), "q1") || !strings.Contains(string(body), "unavailable") {
		t.Fatalf("fiscal calendar result = %s", body)
	}
	if result.AggregateVersion != 1 {
		t.Fatalf("aggregate version = %d, want 1", result.AggregateVersion)
	}
}

func TestFiscalCalendarHandlerRejectsOverlappingPeriodDefinitions(t *testing.T) {
	scopeID := uuid.New()
	service, err := organization.NewFiscalCalendarService(
		organization.NewMemoryFiscalCalendarRepository(),
		organization.MemoryFiscalCalendarAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.FiscalCalendarManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.AllowAllFiscalCalendarApprovalValidator{}, &organization.MemoryFiscalCalendarAuditRecorder{}, organization.UnavailableFiscalCalendarImpactReader{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	request := &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action": mustRaw("create"), "calendarType": mustRaw("gregorian"), "periodPattern": mustRaw("quarterly"), "effectiveFrom": mustRaw("2026-01-01"),
			"periods": mustRaw([]map[string]any{
				{"reference": "q1", "ordinal": 1, "startDate": "2026-01-01", "endDate": "2026-04-01"},
				{"reference": "q2", "ordinal": 2, "startDate": "2026-03-15", "endDate": "2026-06-30"},
			}),
		},
	}
	response, err := (IdentityHandler{FiscalCalendarService: service}).OmdMaintainFiscalCalendars(ctx, request, generated.OmdMaintainFiscalCalendarsParams{IdempotencyKey: "calendar-http-overlap"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.OmdMaintainFiscalCalendarsBadRequest); !ok {
		t.Fatalf("response = %T, want bad request", response)
	}
}
