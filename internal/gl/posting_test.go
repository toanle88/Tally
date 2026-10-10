package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/accountingscope"
	"github.com/toanle88/Tally/internal/platform/money"
)

func TestPostingServicePostsBalancedRequestExactlyOnce(t *testing.T) {
	service, repository, _ := testPostingService(t, MemoryPostingGateValidator{})
	request := validPostingRequest(t, "posting-1")
	actor := Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}

	first, err := service.Execute(context.Background(), actor, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome != PostingOutcomeJournalEntryPosted || first.LifecycleStatus() != PostingStatusPosted {
		t.Fatalf("first result = %#v", first)
	}
	second, err := service.Execute(context.Background(), actor, request)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || second.AggregateID() != first.AggregateID() {
		t.Fatalf("retry = %#v, want replay of %s", second, first.AggregateID())
	}
	if got := len(repository.Journals()); got != 1 {
		t.Fatalf("journal count = %d, want 1", got)
	}
	if got := len(repository.Events()); got != 1 || repository.Events()[0].Type != PostingOutcomeJournalEntryPosted {
		t.Fatalf("events = %#v, want one posted event", repository.Events())
	}
}

func TestPostingServiceRejectsChangedIdempotencyFingerprint(t *testing.T) {
	service, repository, _ := testPostingService(t, MemoryPostingGateValidator{})
	actor := Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}
	firstRequest := validPostingRequest(t, "posting-1")
	if _, err := service.Execute(context.Background(), actor, firstRequest); err != nil {
		t.Fatal(err)
	}
	changed := firstRequest
	changed.Description = "changed content"
	if _, err := service.Execute(context.Background(), actor, changed); !errors.Is(err, ErrPostingIdempotencyConflict) {
		t.Fatalf("changed retry error = %v, want idempotency conflict", err)
	}
	if got := len(repository.Journals()); got != 1 {
		t.Fatalf("journal count = %d, want 1", got)
	}
	if attempts := repository.Attempts(); len(attempts) != 2 || attempts[1].Outcome != PostingOutcomeIdempotencyConflict {
		t.Fatalf("attempts = %#v, want posted and idempotency-conflict outcomes", attempts)
	}
}

func TestPostingServiceReplaysBeforeCurrentGateRejection(t *testing.T) {
	repository := NewMemoryPostingRepository()
	audit := &MemoryPostingAuditRecorder{}
	gate := &replayGateValidator{}
	service, err := NewPostingService(
		repository,
		MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true}},
		AllowAllPostingReferenceValidator{}, gate, AllowAllPostingApprovalValidator{}, audit, testPostingCurrencyRegistry(t), time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	request := validPostingRequest(t, "posting-replay-after-gate-change")
	actor := Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}
	first, err := service.Execute(context.Background(), actor, request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.Execute(context.Background(), actor, request)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.AggregateID() != first.AggregateID() {
		t.Fatalf("replay = %#v, want existing result %s", replay, first.AggregateID())
	}
	if gate.calls != 1 {
		t.Fatalf("gate calls = %d, want one call for the initial admission", gate.calls)
	}
}

func TestPostingValidationRetainsRejectedAttemptWithoutJournal(t *testing.T) {
	service, repository, _ := testPostingService(t, MemoryPostingGateValidator{})
	request := validPostingRequest(t, "posting-invalid")
	request.Lines = request.Lines[:1]
	_, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, request)
	var validation *PostingValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want validation error", err)
	}
	if got := len(repository.Journals()); got != 0 {
		t.Fatalf("journal count = %d, want 0", got)
	}
	attempts := repository.Attempts()
	if len(attempts) != 1 || attempts[0].Outcome != PostingOutcomePostingRejected {
		t.Fatalf("attempts = %#v, want one rejected attempt", attempts)
	}
}

func TestPostingServiceFailsClosedWhenRejectedAttemptAuditFails(t *testing.T) {
	repository := NewMemoryPostingRepository()
	auditError := errors.New("rejected-attempt audit unavailable")
	audit := &MemoryPostingAuditRecorder{Err: auditError}
	service, err := NewPostingService(
		repository,
		MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true}},
		AllowAllPostingReferenceValidator{}, MemoryPostingGateValidator{}, AllowAllPostingApprovalValidator{}, audit, testPostingCurrencyRegistry(t), time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	request := validPostingRequest(t, "posting-rejected-audit-failure")
	request.Lines = request.Lines[:1]
	_, err = service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, request)
	if !errors.Is(err, auditError) {
		t.Fatalf("error = %v, want rejected-attempt audit error", err)
	}
	if len(repository.Journals()) != 0 || len(repository.Attempts()) != 0 || len(repository.Events()) != 0 {
		t.Fatalf("repository retained state after audit failure: journals=%d attempts=%d events=%d", len(repository.Journals()), len(repository.Attempts()), len(repository.Events()))
	}
}

func TestPostingServiceRejectsUnbalancedFunctionalTotals(t *testing.T) {
	service, _, _ := testPostingService(t, MemoryPostingGateValidator{})
	request := validPostingRequest(t, "posting-unbalanced")
	request.Lines[1].FunctionalAmount = "99.99"
	_, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, request)
	if !errors.Is(err, ErrPostingValidation) {
		t.Fatalf("error = %v, want validation error", err)
	}
}

func TestPostingServiceRejectsMissingConversionEvidence(t *testing.T) {
	service, _, _ := testPostingService(t, MemoryPostingGateValidator{})
	request := validPostingRequest(t, "posting-fx")
	request.TransactionCurrency = "EUR"
	_, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, request)
	if !errors.Is(err, ErrPostingValidation) {
		t.Fatalf("error = %v, want validation error", err)
	}
	var validation *PostingValidationError
	if !errors.As(err, &validation) || !hasPostingIssue(validation.Issues, "conversion_evidence_required") {
		t.Fatalf("issues = %#v, want conversion evidence issue", validation)
	}
}

func TestPostingServiceRejectsFunctionalOnlyLineWithoutAuthorizationAndEvidence(t *testing.T) {
	service, repository, _ := testPostingService(t, MemoryPostingGateValidator{})
	request := validPostingRequest(t, "posting-functional-only")
	request.Lines = []PostingLine{
		{AccountID: uuid.New(), DebitOrCredit: PostingDebit, LineCurrencyMode: PostingLineFunctionalOnlyAdjustment, TransactionAmount: "0.00", FunctionalAmount: "10.00", SegmentCombinationID: uuid.New()},
		{AccountID: uuid.New(), DebitOrCredit: PostingDebit, LineCurrencyMode: PostingLineTransactionAndFunctional, TransactionAmount: "100.00", FunctionalAmount: "90.00", SegmentCombinationID: uuid.New()},
		{AccountID: uuid.New(), DebitOrCredit: PostingCredit, LineCurrencyMode: PostingLineTransactionAndFunctional, TransactionAmount: "100.00", FunctionalAmount: "100.00", SegmentCombinationID: uuid.New()},
	}
	_, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, request)
	var validation *PostingValidationError
	if !errors.As(err, &validation) || !hasPostingIssue(validation.Issues, "functional_only_authorization_required") || !hasPostingIssue(validation.Issues, "functional_only_evidence_required") {
		t.Fatalf("error = %#v, want functional-only authorization and evidence issues", err)
	}
	if len(repository.Journals()) != 0 {
		t.Fatal("invalid functional-only request created a journal")
	}
}

func TestPostingServiceRejectsStaleGateAndHardClose(t *testing.T) {
	request := validPostingRequest(t, "posting-gate")
	scope := request.AccountingScope
	staleGate := MemoryPostingGateValidator{Evidence: PostingGateEvidence{FiscalPeriodID: request.FiscalPeriodID, PeriodStateVersion: request.PeriodStateVersion, PostingGateVersion: request.PostingGateVersion + 1, GateMode: PostingGateOpen}}
	service, repository, _ := testPostingService(t, staleGate)
	_, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, request)
	if !errors.Is(err, ErrPostingGateConflict) {
		t.Fatalf("stale gate error = %v, want gate conflict", err)
	}
	if len(repository.Journals()) != 0 {
		t.Fatal("stale gate created a journal")
	}

	hardClosed := MemoryPostingGateValidator{Evidence: PostingGateEvidence{FiscalPeriodID: request.FiscalPeriodID, PeriodStateVersion: request.PeriodStateVersion, PostingGateVersion: request.PostingGateVersion, GateMode: PostingGateHardClosed}}
	service, repository, _ = testPostingService(t, hardClosed)
	request.AccountingScope = scope
	_, err = service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, request)
	if !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("hard-close error = %v, want gate closed", err)
	}
	if len(repository.Journals()) != 0 {
		t.Fatal("hard-close request created a journal")
	}
}

func TestPostingServiceRequiresPurposeSpecificGateMode(t *testing.T) {
	request := validPostingRequest(t, "posting-close-gate")
	closeRunID := uuid.New()
	request.PostingPurpose = PostingPurposeClose
	request.CloseRunID = &closeRunID
	service, repository, _ := testPostingService(t, MemoryPostingGateValidator{Evidence: PostingGateEvidence{FiscalPeriodID: request.FiscalPeriodID, PeriodStateVersion: request.PeriodStateVersion, PostingGateVersion: request.PostingGateVersion, GateMode: PostingGateOpen}})
	_, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, request)
	if !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("close gate error = %v, want gate closed", err)
	}
	if len(repository.Journals()) != 0 {
		t.Fatal("close posting admitted on open gate")
	}

	authorizationID := uuid.New()
	policyAdjustment := validPostingRequest(t, "posting-policy-adjustment-gate")
	policyAdjustment.PostingPurpose = PostingPurposePolicyAdjustment
	policyAdjustment.PostingAuthorizationID = &authorizationID
	service, repository, _ = testPostingService(t, MemoryPostingGateValidator{Evidence: PostingGateEvidence{FiscalPeriodID: policyAdjustment.FiscalPeriodID, PeriodStateVersion: policyAdjustment.PeriodStateVersion, PostingGateVersion: policyAdjustment.PostingGateVersion, GateMode: PostingGateHardClosed}})
	_, err = service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, policyAdjustment)
	if !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("policy-adjustment gate error = %v, want gate closed", err)
	}
	if len(repository.Journals()) != 0 {
		t.Fatal("policy adjustment created a journal in a hard-closed gate")
	}
}

func TestPostingServiceEstablishesPendingApprovalWithoutPostedEffect(t *testing.T) {
	repository := NewMemoryPostingRepository()
	audit := &MemoryPostingAuditRecorder{}
	currency := testPostingCurrencyRegistry(t)
	service, err := NewPostingService(
		repository,
		MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true, ApprovalRequired: true, PolicyReference: "policy-1", PolicyVersion: "1"}},
		AllowAllPostingReferenceValidator{}, MemoryPostingGateValidator{}, requiredPostingApprovalValidator{approval: PostingApproval{Required: true, ApprovalRequestID: uuid.New(), PolicyReference: "policy-1", PolicyVersion: "1"}}, audit, currency, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "accountant-1"}, validPostingRequest(t, "posting-pending"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != PostingOutcomePostingPendingApproval || result.LifecycleStatus() != PostingStatusPendingApproval || result.NextAction == "" {
		t.Fatalf("pending result = %#v", result)
	}
	if len(repository.Journals()) != 1 || repository.Journals()[0].Status != PostingStatusPendingApproval {
		t.Fatalf("journals = %#v, want one pending journal", repository.Journals())
	}
	if journal := repository.Journals()[0]; journal.LedgerPosition != 0 || journal.Number[:4] != "PJE-" {
		t.Fatalf("pending journal = %#v, want no ledger position", journal)
	}
}

type requiredPostingApprovalValidator struct {
	approval PostingApproval
}

func (validator requiredPostingApprovalValidator) EvaluatePostingApproval(context.Context, Actor, PostingRequest, PostingAuthorizationDecision) (PostingApproval, error) {
	return validator.approval, nil
}

type replayGateValidator struct {
	calls int
}

func (validator *replayGateValidator) ValidatePostingGate(_ context.Context, request PostingRequest) (PostingGateEvidence, error) {
	validator.calls++
	if validator.calls > 1 {
		return PostingGateEvidence{}, ErrPostingGateClosed
	}
	return PostingGateEvidence{FiscalPeriodID: request.FiscalPeriodID, PeriodStateVersion: request.PeriodStateVersion, PostingGateVersion: request.PostingGateVersion, GateMode: PostingGateOpen}, nil
}

func testPostingService(t *testing.T, gate PostingGateValidator) (*PostingService, *MemoryPostingRepository, *MemoryPostingAuditRecorder) {
	t.Helper()
	repository := NewMemoryPostingRepository()
	audit := &MemoryPostingAuditRecorder{}
	service, err := NewPostingService(
		repository,
		MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true}},
		AllowAllPostingReferenceValidator{}, gate, AllowAllPostingApprovalValidator{}, audit, testPostingCurrencyRegistry(t), time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, audit
}

func validPostingRequest(t *testing.T, idempotencyKey string) PostingRequest {
	t.Helper()
	tenantID, legalEntityID, ledgerID, bookID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	scope, err := accountingscope.New(tenantID, legalEntityID, ledgerID, bookID, "USD")
	if err != nil {
		t.Fatal(err)
	}
	return PostingRequest{
		ContractVersion: PostingContractVersion, RequestID: uuid.New(), SourceContext: "accounts-payable", SourceAggregateType: "vendor-invoice", SourceAggregateID: uuid.New(), SourceVersion: 1,
		AccountingScope: scope, AccountingScopeID: uuid.New(), PostingDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), FiscalPeriodID: uuid.New(), PeriodStateVersion: 2, PostingGateVersion: 4, PostingPurpose: PostingPurposeOrdinary,
		TransactionCurrency: "USD", IdempotencyKey: idempotencyKey, CorrelationID: uuid.NewString(), CausationID: uuid.NewString(), Description: "vendor invoice",
		Lines: []PostingLine{
			{AccountID: uuid.New(), DebitOrCredit: PostingDebit, LineCurrencyMode: PostingLineTransactionAndFunctional, TransactionAmount: "100.00", FunctionalAmount: "100.00", SegmentCombinationID: uuid.New()},
			{AccountID: uuid.New(), DebitOrCredit: PostingCredit, LineCurrencyMode: PostingLineTransactionAndFunctional, TransactionAmount: "100.00", FunctionalAmount: "100.00", SegmentCombinationID: uuid.New()},
		},
	}
}

func testPostingCurrencyRegistry(t *testing.T) money.CurrencyRegistry {
	t.Helper()
	registry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{{Code: "USD", Scale: 2}, {Code: "EUR", Scale: 2}, {Code: "JPY", Scale: 0}})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func hasPostingIssue(issues []PostingIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
