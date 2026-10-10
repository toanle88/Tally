package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/platform/money"
)

func TestGLPostingHandlerEstablishesReplaysAndMapsTypedConflicts(t *testing.T) {
	service, err := newHTTPPostingService(t, true)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{PostingService: service}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	request := newHTTPPostingRequest()
	params := generated.GlSubmitPostingRequestParams{IdempotencyKey: "posting-http-1", XCorrelationID: generated.NewOptUUID(generated.UUID(uuid.New()))}

	response, err := handler.GlSubmitPostingRequest(actorContext, request, params)
	if err != nil {
		t.Fatal(err)
	}
	established, ok := response.(*generated.GlSubmitPostingRequestEstablishedResult)
	if !ok || established.Status != "established" || established.Data.Outcome != generated.GlSubmitPostingRequestResultDataOutcomeJournalEntryPosted || !established.Data.JournalNumber.Set {
		t.Fatalf("established response = %#v", response)
	}
	if established.Data.GateEvidence.FiscalPeriodId != request.Data.FiscalPeriodId || established.Data.SourceReference == "" {
		t.Fatalf("established evidence = %#v", established.Data)
	}

	replay, err := handler.GlSubmitPostingRequest(actorContext, request, params)
	if err != nil {
		t.Fatal(err)
	}
	replayed, ok := replay.(*generated.GlSubmitPostingRequestEstablishedResult)
	if !ok || !replayed.Data.Replayed {
		t.Fatalf("replay response = %#v", replay)
	}

	changed := *request
	changed.Data.Description = generated.NewOptString("changed business content")
	conflict, err := handler.GlSubmitPostingRequest(actorContext, &changed, params)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := conflict.(*generated.GlSubmitPostingRequestConflict); !ok || value.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("idempotency response = %#v", conflict)
	}

	invalid := *request
	invalid.Data.Lines = invalid.Data.Lines[:1]
	invalidResponse, err := handler.GlSubmitPostingRequest(actorContext, &invalid, generated.GlSubmitPostingRequestParams{IdempotencyKey: "posting-http-invalid"})
	if err != nil {
		t.Fatal(err)
	}
	invalidProblem, ok := invalidResponse.(*generated.GlSubmitPostingRequestUnprocessableEntity)
	if !ok || invalidProblem.Code != "VALIDATION_FAILED" || len(invalidProblem.FieldErrors) == 0 {
		t.Fatalf("validation response = %#v", invalidResponse)
	}
}

func TestGLPostingHandlerRequiresActorAndFailsClosedWithoutService(t *testing.T) {
	request := newHTTPPostingRequest()
	params := generated.GlSubmitPostingRequestParams{IdempotencyKey: "posting-http-auth"}
	if response, err := (IdentityHandler{PostingService: nil}).GlSubmitPostingRequest(context.Background(), request, params); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlSubmitPostingRequestServiceUnavailable); !ok || problem.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("unavailable response = %#v", response)
	}

	service, err := newHTTPPostingService(t, true)
	if err != nil {
		t.Fatal(err)
	}
	if response, err := (IdentityHandler{PostingService: service}).GlSubmitPostingRequest(context.Background(), request, params); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlSubmitPostingRequestForbidden); !ok || problem.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("authorization response = %#v", response)
	}
}

func newHTTPPostingService(t *testing.T, allowed bool) (*gl.PostingService, error) {
	t.Helper()
	registry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{{Code: "USD", Scale: 2}})
	if err != nil {
		return nil, err
	}
	return gl.NewPostingService(
		gl.NewMemoryPostingRepository(),
		gl.MemoryPostingAuthorizer{Decision: gl.PostingAuthorizationDecision{Allowed: allowed, PolicyReference: "http-policy", PolicyVersion: "http-v1"}},
		gl.AllowAllPostingReferenceValidator{},
		gl.MemoryPostingGateValidator{},
		gl.AllowAllPostingApprovalValidator{},
		&gl.MemoryPostingAuditRecorder{},
		registry,
		func() time.Time { return time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC) },
	)
}

func newHTTPPostingRequest() *generated.GlSubmitPostingRequestCommandRequest {
	tenantID, legalEntityID, ledgerID, bookID, scopeID, periodID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	accountOne, accountTwo, combinationOne, combinationTwo := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	return &generated.GlSubmitPostingRequestCommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.UUID(scopeID),
		Data: generated.GlSubmitPostingRequestCommandData{
			ContractVersion: 2, RequestId: generated.UUID(uuid.New()), SourceContext: "manual-entry", SourceAggregateType: "posting-request", SourceAggregateId: generated.UUID(uuid.New()), SourceVersion: 1,
			TenantId: generated.UUID(tenantID), LegalEntityId: generated.UUID(legalEntityID), LedgerId: generated.UUID(ledgerID), AccountingBookId: generated.UUID(bookID), FunctionalCurrency: generated.CurrencyCode("USD"), PostingDate: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), FiscalPeriodId: generated.UUID(periodID), PeriodStateVersion: 1, PostingGateVersion: 1,
			PostingPurpose: generated.GlSubmitPostingRequestCommandDataPostingPurposeOrdinary, TransactionCurrency: generated.CurrencyCode("USD"), Description: generated.NewOptString("HTTP posting"),
			Lines: []generated.GlPostingRequestLine{
				{AccountId: generated.UUID(accountOne), DebitOrCredit: generated.GlPostingRequestLineDebitOrCreditDebit, LineCurrencyMode: generated.GlPostingRequestLineLineCurrencyModeTransactionAndFunctional, TransactionAmount: generated.Money("10.00"), FunctionalAmount: generated.Money("10.00"), SegmentCombinationId: generated.UUID(combinationOne)},
				{AccountId: generated.UUID(accountTwo), DebitOrCredit: generated.GlPostingRequestLineDebitOrCreditCredit, LineCurrencyMode: generated.GlPostingRequestLineLineCurrencyModeTransactionAndFunctional, TransactionAmount: generated.Money("10.00"), FunctionalAmount: generated.Money("10.00"), SegmentCombinationId: generated.UUID(combinationTwo)},
			},
		},
	}
}
