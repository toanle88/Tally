package httpapi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/invoicing"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestBillingScheduleHandlerEstablishesReplaysAndReadsSchedule(t *testing.T) {
	scopeID := uuid.New()
	service := newHTTPBillingScheduleService(t, scopeID, true)
	ctx := httpInvoiceTemplateActorContext(t, uuid.New())
	request := newHTTPBillingScheduleRequest(scopeID, uuid.New())
	params := generated.InvConfigureBillingSchedulesParams{IdempotencyKey: "http-schedule-create"}

	response, err := (IdentityHandler{BillingScheduleService: service}).InvConfigureBillingSchedules(ctx, request, params)
	if err != nil {
		t.Fatalf("create handler error = %v", err)
	}
	created, ok := response.(*generated.EstablishedResult)
	if !ok || created.Status != "established" || created.AggregateVersion != 1 || created.Data["billingScheduleId"] == nil {
		t.Fatalf("create response = %#v", response)
	}
	var scheduleID uuid.UUID
	if err := jsonUnmarshalRaw(created.Data["billingScheduleId"], &scheduleID); err != nil {
		t.Fatalf("schedule id result error = %v", err)
	}

	replayed, err := (IdentityHandler{BillingScheduleService: service}).InvConfigureBillingSchedules(ctx, request, params)
	if err != nil {
		t.Fatalf("replay handler error = %v", err)
	}
	replayedResult, ok := replayed.(*generated.EstablishedResult)
	if !ok || string(replayedResult.Data["replayed"]) != "true" {
		t.Fatalf("replay response = %#v", replayed)
	}

	request.Data.ContractReference = "changed-contract"
	changed, err := (IdentityHandler{BillingScheduleService: service}).InvConfigureBillingSchedules(ctx, request, params)
	if err != nil {
		t.Fatalf("changed duplicate handler error = %v", err)
	}
	conflict, ok := changed.(*generated.InvConfigureBillingSchedulesConflict)
	if !ok || conflict.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("changed duplicate response = %#v", changed)
	}

	listResponse, err := (IdentityHandler{BillingScheduleService: service}).InvListBillingSchedules(ctx, generated.InvListBillingSchedulesParams{XAccountingScopeID: generated.NewOptUUID(generated.UUID(scopeID))})
	if err != nil {
		t.Fatalf("list handler error = %v", err)
	}
	list, ok := listResponse.(*generated.EstablishedResult)
	if !ok || list.Data["billingSchedules"] == nil || list.Data["page"] == nil {
		t.Fatalf("list response = %#v", listResponse)
	}

	getResponse, err := (IdentityHandler{BillingScheduleService: service}).InvGetBillingSchedule(ctx, generated.InvGetBillingScheduleParams{BillingScheduleId: generated.UUID(scheduleID), XAccountingScopeID: generated.NewOptUUID(generated.UUID(scopeID))})
	if err != nil {
		t.Fatalf("get handler error = %v", err)
	}
	get, ok := getResponse.(*generated.EstablishedResult)
	if !ok || get.AggregateId != generated.UUID(scheduleID) || get.Data["billingSchedule"] == nil {
		t.Fatalf("get response = %#v", getResponse)
	}
}

func TestBillingScheduleHandlerMapsAuthorizationAndVersionFailures(t *testing.T) {
	scopeID := uuid.New()
	ctx := httpInvoiceTemplateActorContext(t, uuid.New())
	request := newHTTPBillingScheduleRequest(scopeID, uuid.New())
	deniedService := newHTTPBillingScheduleServiceWithDecision(t, scopeID, invoicing.AuthorizationDecision{Allowed: false, Permission: invoicing.BillingScheduleManagementPermission, DecisionReference: uuid.New()})
	response, err := (IdentityHandler{BillingScheduleService: deniedService}).InvConfigureBillingSchedules(ctx, request, generated.InvConfigureBillingSchedulesParams{IdempotencyKey: "http-schedule-denied"})
	if err != nil {
		t.Fatalf("denied handler error = %v", err)
	}
	denied, ok := response.(*generated.InvConfigureBillingSchedulesForbidden)
	if !ok || denied.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("denied response = %#v", response)
	}

	service := newHTTPBillingScheduleService(t, scopeID, true)
	createdResponse, err := (IdentityHandler{BillingScheduleService: service}).InvConfigureBillingSchedules(ctx, newHTTPBillingScheduleRequest(scopeID, uuid.New()), generated.InvConfigureBillingSchedulesParams{IdempotencyKey: "http-schedule-stale-create"})
	if err != nil {
		t.Fatalf("setup create handler error = %v", err)
	}
	created := createdResponse.(*generated.EstablishedResult)
	var scheduleID uuid.UUID
	if err := jsonUnmarshalRaw(created.Data["billingScheduleId"], &scheduleID); err != nil {
		t.Fatalf("schedule id result error = %v", err)
	}
	update := newHTTPBillingScheduleRequest(scopeID, uuid.New())
	update.Data.Action = generated.InvConfigureBillingSchedulesCommandDataActionUpdate
	update.Data.BillingScheduleId = generated.NewOptUUID(generated.UUID(scheduleID))
	update.ExpectedVersion = generated.NewOptInt(99)
	staleResponse, err := (IdentityHandler{BillingScheduleService: service}).InvConfigureBillingSchedules(ctx, update, generated.InvConfigureBillingSchedulesParams{IdempotencyKey: "http-schedule-stale-update"})
	if err != nil {
		t.Fatalf("stale handler error = %v", err)
	}
	stale, ok := staleResponse.(*generated.InvConfigureBillingSchedulesConflict)
	if !ok || stale.Code != "VERSION_CONFLICT" {
		t.Fatalf("stale response = %#v", staleResponse)
	}
}

func newHTTPBillingScheduleService(t *testing.T, scopeID uuid.UUID, allowed bool) *invoicing.BillingScheduleService {
	decision := invoicing.AuthorizationDecision{Allowed: allowed, Permission: invoicing.BillingScheduleManagementPermission, ApprovedScopeIDs: []uuid.UUID{scopeID}, DecisionReference: uuid.New(), PolicyReference: "http-billing-policy", PolicyVersion: "v1"}
	return newHTTPBillingScheduleServiceWithDecision(t, scopeID, decision)
}

func newHTTPBillingScheduleServiceWithDecision(t *testing.T, _ uuid.UUID, decision invoicing.AuthorizationDecision) *invoicing.BillingScheduleService {
	t.Helper()
	service, err := invoicing.NewBillingScheduleService(
		invoicing.NewMemoryBillingScheduleRepository(),
		invoicing.MemoryBillingScheduleAuthorizer{Decision: decision},
		invoicing.AllowAllBillingScheduleReferenceValidator{},
		&invoicing.MemoryBillingScheduleAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatalf("new billing-schedule service error = %v", err)
	}
	return service
}

func newHTTPBillingScheduleRequest(scopeID, commandID uuid.UUID) *generated.InvConfigureBillingSchedulesCommandRequest {
	return &generated.InvConfigureBillingSchedulesCommandRequest{
		CommandId: generated.UUID(commandID), AccountingScopeId: generated.UUID(scopeID),
		Data: generated.InvConfigureBillingSchedulesCommandData{
			Action: generated.InvConfigureBillingSchedulesCommandDataActionCreate, CustomerId: generated.UUID(uuid.MustParse("33333333-3333-4333-8333-333333333333")),
			ContractReference: "contract-http", Charges: []generated.InvScheduledCharge{{
				ID: generated.UUID(uuid.New()), TemplateId: generated.UUID(uuid.New()), TemplateVersion: 1, TemplateLineItemId: generated.UUID(uuid.New()),
				Quantity: generated.ExactDecimal("2"), Amount: generated.ExactDecimal("10.00"), Currency: generated.CurrencyCode("USD"),
			}}, NextBillingDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TimeZone: "Asia/Ho_Chi_Minh",
			Status: generated.InvConfigureBillingSchedulesCommandDataStatusConfigured, EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}
