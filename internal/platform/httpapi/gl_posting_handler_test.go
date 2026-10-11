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

func TestGLPostingHandlerAppliesPendingApprovalDecision(t *testing.T) {
	registry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{{Code: "USD", Scale: 2}})
	if err != nil {
		t.Fatal(err)
	}
	repository := gl.NewMemoryPostingRepository()
	audit := &gl.MemoryPostingAuditRecorder{}
	postingAuthorizer := gl.MemoryPostingAuthorizer{Decision: gl.PostingAuthorizationDecision{Allowed: true, ApprovalRequired: true, PolicyReference: "http-policy", PolicyVersion: "http-v1"}}
	clock := func() time.Time { return time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC) }
	posting, err := gl.NewPostingService(repository, postingAuthorizer, gl.AllowAllPostingReferenceValidator{}, gl.MemoryPostingGateValidator{}, httpRequiredApprovalValidator{approval: gl.PostingApproval{Required: true, ApprovalRequestID: uuid.New(), PolicyReference: "http-policy", PolicyVersion: "http-v1"}}, audit, registry, clock)
	if err != nil {
		t.Fatal(err)
	}
	apply, err := gl.NewPostingApprovalDecisionService(repository, gl.MemoryPostingApprovalDecisionVerifier{}, gl.MemoryPostingApprovalDecisionAuthorizer{Decision: gl.PostingApprovalDecisionAuthorization{Allowed: true, PolicyReference: "http-policy", PolicyVersion: "http-v1"}}, postingAuthorizer, gl.AllowAllPostingReferenceValidator{}, gl.MemoryPostingGateValidator{}, registry, clock)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{PostingService: posting, PostingApprovalDecisionService: apply}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	submitRequest := newHTTPPostingRequest()
	submitParams := generated.GlSubmitPostingRequestParams{IdempotencyKey: "posting-http-pending", XCorrelationID: generated.NewOptUUID(generated.UUID(uuid.New()))}
	pendingResponse, err := handler.GlSubmitPostingRequest(actorContext, submitRequest, submitParams)
	if err != nil {
		t.Fatal(err)
	}
	pendingEstablished, ok := pendingResponse.(*generated.GlSubmitPostingRequestEstablishedResult)
	if !ok || !pendingEstablished.Data.PolicyVersion.Set || pendingEstablished.Data.PolicyVersion.Value != "http-v1" {
		t.Fatalf("pending approval policy evidence = %#v", pendingResponse)
	}
	pending := repository.Journals()[0]
	if pending.Approval == nil {
		t.Fatal("pending journal did not retain approval request")
	}
	decisionID, approverID := uuid.New(), uuid.UUID(submitRequest.CommandId)
	applyRequest := &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), ExpectedVersion: generated.NewOptInt(1), AccountingScopeId: generated.NewOptUUID(submitRequest.AccountingScopeId),
		Data: generated.CommandRequestData{
			"journalId": mustRaw(pending.ID.String()), "outcome": mustRaw("approved"), "approvalRequestId": mustRaw(pending.Approval.ApprovalRequestID.String()),
			"decisionId": mustRaw(decisionID.String()), "policyVersion": mustRaw(pending.Approval.PolicyVersion), "decisionVersion": mustRaw(1), "subjectVersion": mustRaw(1),
			"candidateFingerprint": mustRaw(pending.RequestFingerprint), "approverUserId": mustRaw(approverID.String()),
		},
	}
	applyParams := generated.GlApplyJournalApprovalDecisionParams{IdempotencyKey: "approval-http-1", XCorrelationID: generated.NewOptUUID(generated.UUID(uuid.New())), XAccountingScopeID: generated.NewOptUUID(submitRequest.AccountingScopeId)}
	response, err := handler.GlApplyJournalApprovalDecision(actorContext, applyRequest, applyParams)
	if err != nil {
		t.Fatal(err)
	}
	established, ok := response.(*generated.EstablishedResult)
	if !ok || established.AggregateId != generated.UUID(pending.ID) || established.AggregateVersion != 2 {
		t.Fatalf("approval response = %#v", response)
	}
	replay, err := handler.GlApplyJournalApprovalDecision(actorContext, applyRequest, applyParams)
	if err != nil {
		t.Fatal(err)
	}
	if replayResult, ok := replay.(*generated.EstablishedResult); !ok || replayResult.AggregateVersion != 2 {
		t.Fatalf("approval replay = %#v", replay)
	}
}

type httpRequiredApprovalValidator struct {
	approval gl.PostingApproval
}

func (validator httpRequiredApprovalValidator) EvaluatePostingApproval(context.Context, gl.Actor, gl.PostingRequest, gl.PostingAuthorizationDecision) (gl.PostingApproval, error) {
	return validator.approval, nil
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
