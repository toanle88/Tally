package httpapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestGLHandlersUseTypedCommandsAndMapEstablishedResults(t *testing.T) {
	scopeID := uuid.New()
	ledgerRepository := gl.NewMemoryLedgerRepository()
	ledgerService, err := gl.NewLedgerService(ledgerRepository, gl.MemoryLedgerAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "gl-policy"}}, gl.AllowAllReferenceValidator{}, gl.AllowAllApprovalValidator{}, &gl.MemoryLedgerAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	bookRepository := gl.NewMemoryAccountingBookRepository(ledgerRepository)
	bookService, err := gl.NewAccountingBookService(bookRepository, gl.MemoryAccountingBookAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, Permission: gl.AccountingBookManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "gl-policy"}}, gl.AllowAllReferenceValidator{}, gl.AllowAllApprovalValidator{}, &gl.MemoryAccountingBookAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{LedgerService: ledgerService, AccountingBookService: bookService}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	request := &generated.GlMaintainLedgersCommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.UUID(scopeID), Data: generated.GlMaintainLedgersCommandData{Action: generated.GlMaintainLedgersCommandDataActionCreate, LegalEntityId: generated.UUID(uuid.New()), LedgerType: "primary", FunctionalCurrency: generated.CurrencyCode("USD"), FiscalCalendarId: generated.UUID(uuid.New()), LifecycleStatus: generated.GlMaintainLedgersCommandDataLifecycleStatusDraft, EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
	response, err := handler.GlMaintainLedgers(actorContext, request, generated.GlMaintainLedgersParams{IdempotencyKey: "http-ledger-create"})
	if err != nil {
		t.Fatal(err)
	}
	created, ok := response.(*generated.GlMaintainLedgersEstablishedResult)
	if !ok || created.Status != "established" || created.AggregateVersion != 1 || created.Data.Ledger.LifecycleStatus != gl.LedgerStatusDraft {
		t.Fatalf("response = %#v", response)
	}
	if created.Data.Ledger.EffectiveDateTo.Null != true {
		t.Fatalf("open-ended effective date = %#v, want explicit null", created.Data.Ledger.EffectiveDateTo)
	}
	replayResponse, err := handler.GlMaintainLedgers(actorContext, request, generated.GlMaintainLedgersParams{IdempotencyKey: "http-ledger-create"})
	if err != nil {
		t.Fatal(err)
	}
	replayed, ok := replayResponse.(*generated.GlMaintainLedgersEstablishedResult)
	if !ok || !replayed.Data.Replayed.Set || !replayed.Data.Replayed.Value {
		t.Fatalf("replay response = %#v", replayResponse)
	}

	update := *request
	update.CommandId = generated.UUID(uuid.New())
	update.ExpectedVersion = generated.NewOptInt(1)
	update.Data.Action = generated.GlMaintainLedgersCommandDataActionUpdate
	update.Data.LedgerId = generated.NewOptUUID(created.AggregateId)
	update.Data.LifecycleStatus = generated.GlMaintainLedgersCommandDataLifecycleStatusActive
	updatedResponse, err := handler.GlMaintainLedgers(actorContext, &update, generated.GlMaintainLedgersParams{IdempotencyKey: "http-ledger-update", IfMatch: generated.NewOptString("1")})
	if err != nil {
		t.Fatal(err)
	}
	updated, ok := updatedResponse.(*generated.GlMaintainLedgersEstablishedResult)
	if !ok || updated.AggregateVersion != 2 || updated.Data.Ledger.LifecycleStatus != gl.LedgerStatusActive {
		t.Fatalf("updated response = %#v", updatedResponse)
	}

	bookRequest := &generated.GlMaintainAccountingBooksCommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.UUID(scopeID), Data: generated.GlMaintainAccountingBooksCommandData{Action: generated.GlMaintainAccountingBooksCommandDataActionCreate, LedgerId: created.AggregateId, BookType: "statutory", AccountingBasis: "accrual", PostingPolicyVersion: "posting-v1", LifecycleStatus: generated.GlMaintainAccountingBooksCommandDataLifecycleStatusActive, EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
	bookResponse, err := handler.GlMaintainAccountingBooks(actorContext, bookRequest, generated.GlMaintainAccountingBooksParams{IdempotencyKey: "http-book-create"})
	if err != nil {
		t.Fatal(err)
	}
	bookCreated, ok := bookResponse.(*generated.GlMaintainAccountingBooksEstablishedResult)
	if !ok || bookCreated.Status != "established" || bookCreated.Data.AccountingBook.LedgerId != created.AggregateId {
		t.Fatalf("book response = %#v", bookResponse)
	}
}

func TestGLHandlersMapReferenceDependencyFailure(t *testing.T) {
	scopeID := uuid.New()
	service, err := gl.NewLedgerService(
		gl.NewMemoryLedgerRepository(),
		gl.MemoryLedgerAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		gl.MemoryReferenceValidator{LedgerError: gl.ErrLedgerReferenceUnavailable},
		gl.AllowAllApprovalValidator{},
		&gl.MemoryLedgerAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{LedgerService: service}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	request := &generated.GlMaintainLedgersCommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.UUID(scopeID), Data: generated.GlMaintainLedgersCommandData{Action: generated.GlMaintainLedgersCommandDataActionCreate, LegalEntityId: generated.UUID(uuid.New()), LedgerType: "primary", FunctionalCurrency: generated.CurrencyCode("USD"), FiscalCalendarId: generated.UUID(uuid.New()), LifecycleStatus: generated.GlMaintainLedgersCommandDataLifecycleStatusDraft, EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
	response, err := handler.GlMaintainLedgers(actorContext, request, generated.GlMaintainLedgersParams{IdempotencyKey: "http-reference-unavailable"})
	if err != nil {
		t.Fatal(err)
	}
	problem, ok := response.(*generated.GlMaintainLedgersServiceUnavailable)
	if !ok || problem.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGLHandlersPropagateCorrelationAndMapAuditAndIdempotencyFailures(t *testing.T) {
	scopeID := uuid.New()
	audit := &gl.MemoryLedgerAuditRecorder{}
	service, err := gl.NewLedgerService(
		gl.NewMemoryLedgerRepository(),
		gl.MemoryLedgerAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		gl.AllowAllReferenceValidator{},
		gl.AllowAllApprovalValidator{},
		audit,
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{LedgerService: service}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	commandID := uuid.New()
	correlationID := uuid.New()
	request := &generated.GlMaintainLedgersCommandRequest{CommandId: generated.UUID(commandID), AccountingScopeId: generated.UUID(scopeID), Data: generated.GlMaintainLedgersCommandData{Action: generated.GlMaintainLedgersCommandDataActionCreate, LegalEntityId: generated.UUID(uuid.New()), LedgerType: "primary", FunctionalCurrency: generated.CurrencyCode("USD"), FiscalCalendarId: generated.UUID(uuid.New()), LifecycleStatus: generated.GlMaintainLedgersCommandDataLifecycleStatusDraft, EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
	params := generated.GlMaintainLedgersParams{IdempotencyKey: "http-correlation-1", XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID))}
	response, err := handler.GlMaintainLedgers(actorContext, request, params)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.GlMaintainLedgersEstablishedResult); !ok {
		t.Fatalf("established response = %#v", response)
	}
	if len(audit.Records) != 1 || audit.Records[0].CorrelationID != correlationID.String() || audit.Records[0].CausationID != commandID.String() {
		t.Fatalf("audit records = %#v", audit.Records)
	}

	changed := *request
	changed.Data.LedgerType = "secondary"
	conflictResponse, err := handler.GlMaintainLedgers(actorContext, &changed, params)
	if err != nil {
		t.Fatal(err)
	}
	conflict, ok := conflictResponse.(*generated.GlMaintainLedgersConflict)
	if !ok || conflict.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("idempotency response = %#v", conflictResponse)
	}

	failingAudit := &gl.MemoryLedgerAuditRecorder{Err: errors.New("audit unavailable")}
	failingService, err := gl.NewLedgerService(
		gl.NewMemoryLedgerRepository(),
		gl.MemoryLedgerAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		gl.AllowAllReferenceValidator{},
		gl.AllowAllApprovalValidator{},
		failingAudit,
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	failingHandler := IdentityHandler{LedgerService: failingService}
	failingResponse, err := failingHandler.GlMaintainLedgers(actorContext, request, generated.GlMaintainLedgersParams{IdempotencyKey: "http-audit-failure"})
	if err != nil {
		t.Fatal(err)
	}
	if problem, ok := failingResponse.(*generated.GlMaintainLedgersServiceUnavailable); !ok || problem.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("audit failure response = %#v", failingResponse)
	}
}

func TestGLHandlersMapScopeAndAuthorizationFailures(t *testing.T) {
	scopeID := uuid.New()
	service, err := gl.NewLedgerService(gl.NewMemoryLedgerRepository(), gl.MemoryLedgerAuthorizer{Decision: gl.AuthorizationDecision{Allowed: false, Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, gl.AllowAllReferenceValidator{}, gl.AllowAllApprovalValidator{}, &gl.MemoryLedgerAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{LedgerService: service}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	request := &generated.GlMaintainLedgersCommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.UUID(scopeID), Data: generated.GlMaintainLedgersCommandData{Action: generated.GlMaintainLedgersCommandDataActionCreate, LegalEntityId: generated.UUID(uuid.New()), LedgerType: "primary", FunctionalCurrency: generated.CurrencyCode("USD"), FiscalCalendarId: generated.UUID(uuid.New()), LifecycleStatus: generated.GlMaintainLedgersCommandDataLifecycleStatusDraft, EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
	deniedResponse, err := handler.GlMaintainLedgers(actorContext, request, generated.GlMaintainLedgersParams{IdempotencyKey: "http-denied"})
	if err != nil {
		t.Fatal(err)
	}
	denied, ok := deniedResponse.(*generated.GlMaintainLedgersForbidden)
	if !ok || denied.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("denied response = %#v", deniedResponse)
	}
	// The mismatch is checked before authorization and is therefore a typed conflict.
	response, err := handler.GlMaintainLedgers(actorContext, request, generated.GlMaintainLedgersParams{IdempotencyKey: "http-scope-mismatch-2", XAccountingScopeID: generated.NewOptUUID(generated.UUID(uuid.New()))})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.GlMaintainLedgersConflict); !ok {
		t.Fatalf("scope mismatch response = %#v", response)
	}

	invalidScopeRequest := *request
	invalidScopeRequest.AccountingScopeId = generated.UUID(uuid.Nil)
	response, err = handler.GlMaintainLedgers(actorContext, &invalidScopeRequest, generated.GlMaintainLedgersParams{IdempotencyKey: "http-invalid-scope"})
	if err != nil {
		t.Fatal(err)
	}
	invalidScope, ok := response.(*generated.GlMaintainLedgersBadRequest)
	if !ok || invalidScope.Code != "INVALID_REQUEST" {
		t.Fatalf("invalid scope response = %#v", response)
	}

	request.ExpectedVersion = generated.NewOptInt(1)
	response, err = handler.GlMaintainLedgers(actorContext, request, generated.GlMaintainLedgersParams{IdempotencyKey: "http-version-header-mismatch", IfMatch: generated.NewOptString("2")})
	if err != nil {
		t.Fatal(err)
	}
	versionMismatch, ok := response.(*generated.GlMaintainLedgersConflict)
	if !ok || versionMismatch.Code != "VERSION_CONFLICT" {
		t.Fatalf("version mismatch response = %#v", response)
	}
}
