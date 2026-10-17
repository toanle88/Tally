package httpapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/invoicing"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestInvoiceTemplateHandlerEstablishesAndReplaysCommand(t *testing.T) {
	scopeID := uuid.New()
	service := newHTTPInvoiceTemplateService(t, scopeID, true)
	actorID := uuid.New()
	ctx := httpInvoiceTemplateActorContext(t, actorID)
	correlationID := uuid.New()
	request := newHTTPInvoiceTemplateRequest(scopeID, uuid.New(), "http-template-create")
	params := generated.InvConfigureInvoiceTemplatesParams{IdempotencyKey: "http-template-create", XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID))}

	response, err := (IdentityHandler{InvoiceTemplateService: service}).InvConfigureInvoiceTemplates(ctx, request, params)
	if err != nil {
		t.Fatalf("create handler error = %v", err)
	}
	created, ok := response.(*generated.EstablishedResult)
	if !ok || created.Status != "established" || created.AggregateVersion != 1 || created.CorrelationId != generated.UUID(correlationID) {
		t.Fatalf("create response = %#v", response)
	}
	if created.Links.Self == "" || created.Data["templateId"] == nil {
		t.Fatalf("create result envelope = %#v", created)
	}

	replayed, err := (IdentityHandler{InvoiceTemplateService: service}).InvConfigureInvoiceTemplates(ctx, request, params)
	if err != nil {
		t.Fatalf("replay handler error = %v", err)
	}
	replayedResult, ok := replayed.(*generated.EstablishedResult)
	if !ok || string(replayedResult.Data["replayed"]) != "true" {
		t.Fatalf("replay response = %#v", replayed)
	}

	request.Data.Name = "changed duplicate"
	changed, err := (IdentityHandler{InvoiceTemplateService: service}).InvConfigureInvoiceTemplates(ctx, request, params)
	if err != nil {
		t.Fatalf("changed duplicate handler error = %v", err)
	}
	problem, ok := changed.(*generated.InvConfigureInvoiceTemplatesConflict)
	if !ok || problem.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("changed duplicate response = %#v", changed)
	}
}

func TestInvoiceTemplateHandlerMapsAuthorizationAndVersionFailures(t *testing.T) {
	scopeID := uuid.New()
	deniedService := newHTTPInvoiceTemplateServiceWithDecision(t, scopeID, invoicing.AuthorizationDecision{Allowed: false, Permission: invoicing.InvoiceTemplateManagementPermission, DecisionReference: uuid.New()})
	ctx := httpInvoiceTemplateActorContext(t, uuid.New())
	request := newHTTPInvoiceTemplateRequest(scopeID, uuid.New(), "http-denied")
	response, err := (IdentityHandler{InvoiceTemplateService: deniedService}).InvConfigureInvoiceTemplates(ctx, request, generated.InvConfigureInvoiceTemplatesParams{IdempotencyKey: "http-denied"})
	if err != nil {
		t.Fatalf("denied handler error = %v", err)
	}
	denied, ok := response.(*generated.InvConfigureInvoiceTemplatesForbidden)
	if !ok || denied.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("denied response = %#v", response)
	}

	service := newHTTPInvoiceTemplateService(t, scopeID, true)
	createdResponse, err := (IdentityHandler{InvoiceTemplateService: service}).InvConfigureInvoiceTemplates(ctx, newHTTPInvoiceTemplateRequest(scopeID, uuid.New(), "http-stale-create"), generated.InvConfigureInvoiceTemplatesParams{IdempotencyKey: "http-stale-create"})
	if err != nil {
		t.Fatalf("setup create handler error = %v", err)
	}
	created := createdResponse.(*generated.EstablishedResult)
	var templateID uuid.UUID
	if err := jsonUnmarshalRaw(created.Data["templateId"], &templateID); err != nil {
		t.Fatalf("template id result error = %v", err)
	}
	update := newHTTPInvoiceTemplateRequest(scopeID, uuid.New(), "http-stale-update")
	update.Data.Action = generated.InvConfigureInvoiceTemplatesCommandDataActionUpdate
	update.Data.TemplateId = generated.NewOptUUID(generated.UUID(templateID))
	update.ExpectedVersion = generated.NewOptInt(99)
	update.Data.Name = "stale update"
	staleResponse, err := (IdentityHandler{InvoiceTemplateService: service}).InvConfigureInvoiceTemplates(ctx, update, generated.InvConfigureInvoiceTemplatesParams{IdempotencyKey: "http-stale-update"})
	if err != nil {
		t.Fatalf("stale handler error = %v", err)
	}
	stale, ok := staleResponse.(*generated.InvConfigureInvoiceTemplatesConflict)
	if !ok || stale.Code != "VERSION_CONFLICT" {
		t.Fatalf("stale response = %#v", staleResponse)
	}
}

func TestInvoiceTemplateHandlerReadsAuthoritativeProjection(t *testing.T) {
	scopeID := uuid.New()
	service := newHTTPInvoiceTemplateService(t, scopeID, true)
	ctx := httpInvoiceTemplateActorContext(t, uuid.New())
	request := newHTTPInvoiceTemplateRequest(scopeID, uuid.New(), "http-read-create")
	createdResponse, err := (IdentityHandler{InvoiceTemplateService: service}).InvConfigureInvoiceTemplates(ctx, request, generated.InvConfigureInvoiceTemplatesParams{IdempotencyKey: "http-read-create"})
	if err != nil {
		t.Fatalf("create handler error = %v", err)
	}
	created := createdResponse.(*generated.EstablishedResult)
	var templateID uuid.UUID
	if err := jsonUnmarshalRaw(created.Data["templateId"], &templateID); err != nil {
		t.Fatalf("template id result error = %v", err)
	}

	listResponse, err := (IdentityHandler{InvoiceTemplateService: service}).InvListInvoiceTemplates(ctx, generated.InvListInvoiceTemplatesParams{XAccountingScopeID: generated.NewOptUUID(generated.UUID(scopeID))})
	if err != nil {
		t.Fatalf("list handler error = %v", err)
	}
	list, ok := listResponse.(*generated.EstablishedResult)
	if !ok || list.Data["templates"] == nil || list.Data["page"] == nil {
		t.Fatalf("list response = %#v", listResponse)
	}
	getResponse, err := (IdentityHandler{InvoiceTemplateService: service}).InvGetInvoiceTemplate(ctx, generated.InvGetInvoiceTemplateParams{InvoiceTemplateId: generated.UUID(templateID), XAccountingScopeID: generated.NewOptUUID(generated.UUID(scopeID))})
	if err != nil {
		t.Fatalf("get handler error = %v", err)
	}
	get, ok := getResponse.(*generated.EstablishedResult)
	if !ok || get.AggregateId != generated.UUID(templateID) || get.Data["template"] == nil {
		t.Fatalf("get response = %#v", getResponse)
	}
}

func newHTTPInvoiceTemplateService(t *testing.T, scopeID uuid.UUID, allowed bool) *invoicing.InvoiceTemplateService {
	decision := invoicing.AuthorizationDecision{Allowed: allowed, Permission: invoicing.InvoiceTemplateManagementPermission, ApprovedScopeIDs: []uuid.UUID{scopeID}, DecisionReference: uuid.New(), PolicyReference: "http-policy", PolicyVersion: "v1"}
	return newHTTPInvoiceTemplateServiceWithDecision(t, scopeID, decision)
}

func newHTTPInvoiceTemplateServiceWithDecision(t *testing.T, scopeID uuid.UUID, decision invoicing.AuthorizationDecision) *invoicing.InvoiceTemplateService {
	t.Helper()
	service, err := invoicing.NewInvoiceTemplateService(
		invoicing.NewMemoryInvoiceTemplateRepository(),
		invoicing.MemoryInvoiceTemplateAuthorizer{Decision: decision},
		invoicing.AllowAllInvoiceTemplateReferenceValidator{},
		&invoicing.MemoryInvoiceTemplateAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatalf("new invoice-template service error = %v", err)
	}
	return service
}

func httpInvoiceTemplateActorContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: userID, Subject: identity.AuthenticationSubject{OID: "http-oid", TID: "http-tid", Sub: "http-subject"}})
	if err != nil {
		t.Fatalf("actor context error = %v", err)
	}
	return ctx
}

func newHTTPInvoiceTemplateRequest(scopeID, commandID uuid.UUID, _ string) *generated.InvConfigureInvoiceTemplatesCommandRequest {
	return &generated.InvConfigureInvoiceTemplatesCommandRequest{
		CommandId: generated.UUID(commandID), AccountingScopeId: generated.UUID(scopeID),
		Data: generated.InvConfigureInvoiceTemplatesCommandData{
			Action: generated.InvConfigureInvoiceTemplatesCommandDataActionCreate,
			Code:   "HTTP-TEMPLATE", Name: "HTTP template",
			BillingFrequency: generated.InvConfigureInvoiceTemplatesCommandDataBillingFrequencyMonthly,
			LineItems: []generated.InvTemplateLineItem{{
				ID: generated.UUID(uuid.New()), Code: "BASE", Description: "Base service",
				ChargeRule:  generated.InvTemplateChargeRule{Kind: generated.InvTemplateChargeRuleKindFixed, Amount: generated.NewOptExactDecimal(generated.ExactDecimal("10.00")), Currency: generated.NewOptCurrencyCode(generated.CurrencyCode("USD"))},
				TaxCategory: "standard",
			}},
			PaymentMethod: generated.InvTemplatePaymentMethod{Method: generated.InvTemplatePaymentMethodMethodBankTransfer, TermsDays: 30},
			Applicability: generated.InvTemplateApplicability{Mode: generated.InvTemplateApplicabilityModeAll},
			EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func jsonUnmarshalRaw(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}
