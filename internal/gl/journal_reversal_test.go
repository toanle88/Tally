package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJournalReversalCreatesImmutableLinkedOppositeEntry(t *testing.T) {
	repository := NewMemoryPostingRepository()
	audit := &MemoryPostingAuditRecorder{}
	posting, err := NewPostingService(
		repository,
		MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true}},
		AllowAllPostingReferenceValidator{},
		MemoryPostingGateValidator{},
		AllowAllPostingApprovalValidator{},
		audit,
		testPostingCurrencyRegistry(t),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "journal-reversal-actor"}
	originalRequest := validPostingRequest(t, "original-journal")
	original, err := posting.Execute(context.Background(), actor, originalRequest)
	if err != nil {
		t.Fatal(err)
	}

	authorizationID := uuid.New()
	reversal, err := NewJournalReversalService(
		repository,
		MemoryJournalReversalAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true, PolicyReference: "reversal-policy", PolicyVersion: "reversal-v1", DecisionReference: uuid.New()}},
		AllowAllPostingReferenceValidator{},
		MemoryPostingGateValidator{},
		testPostingCurrencyRegistry(t),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	command := reverseCommandForJournal(original, originalRequest.AccountingScopeID, authorizationID, "correction-source")

	result, err := reversal.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != PostingOutcomeJournalEntryReversed || result.Replayed {
		t.Fatalf("result = %#v, want first established reversal", result)
	}
	if result.Original.Journal.ID != original.AggregateID() || result.Original.Journal.Status != PostingStatusPosted {
		t.Fatalf("original snapshot = %#v", result.Original.Journal)
	}
	if result.Reversal.Journal.ID == original.AggregateID() || result.Reversal.Journal.Status != PostingStatusPosted {
		t.Fatalf("reversal snapshot = %#v", result.Reversal.Journal)
	}
	if result.Reversal.Journal.ReversalOfJournalEntryID == nil || *result.Reversal.Journal.ReversalOfJournalEntryID != original.AggregateID() {
		t.Fatalf("reversal link = %#v", result.Reversal.Journal.ReversalOfJournalEntryID)
	}
	if result.Reversal.Journal.PostingPurpose != PostingPurposePolicyAdjustment || result.Reversal.Journal.PostingAuthorizationID == nil || *result.Reversal.Journal.PostingAuthorizationID != authorizationID {
		t.Fatalf("reversal evidence = %#v", result.Reversal.Journal)
	}
	if result.CorrectionReason != command.CorrectionReason || result.AuthorizationReference != authorizationID {
		t.Fatalf("result evidence = %#v", result)
	}
	if len(result.Original.Lines) != len(result.Reversal.Lines) {
		t.Fatalf("line counts = %d/%d", len(result.Original.Lines), len(result.Reversal.Lines))
	}
	for index, originalLine := range result.Original.Lines {
		reversalLine := result.Reversal.Lines[index]
		if reversalLine.AccountID != originalLine.AccountID || reversalLine.TransactionAmount != originalLine.TransactionAmount || reversalLine.FunctionalAmount != originalLine.FunctionalAmount || reversalLine.SegmentCombinationID != originalLine.SegmentCombinationID || reversalLine.LineReference != originalLine.LineReference {
			t.Fatalf("line %d changed amount/reference: original=%#v reversal=%#v", index, originalLine, reversalLine)
		}
		if originalLine.DebitOrCredit == PostingDebit && reversalLine.DebitOrCredit != PostingCredit || originalLine.DebitOrCredit == PostingCredit && reversalLine.DebitOrCredit != PostingDebit {
			t.Fatalf("line %d direction not inverted: original=%#v reversal=%#v", index, originalLine, reversalLine)
		}
	}

	stored := repository.Journals()
	if len(stored) != 2 || stored[0].ID != original.AggregateID() || stored[0].Status != PostingStatusPosted || stored[0].CorrectionReason != "" {
		t.Fatalf("stored journals = %#v", stored)
	}
	if stored[1].ReversalOfJournalEntryID == nil || *stored[1].ReversalOfJournalEntryID != original.AggregateID() {
		t.Fatalf("stored reversal = %#v", stored[1])
	}
	if attempts := repository.Attempts(); len(attempts) != 2 || attempts[1].Operation != JournalReversalOperation || attempts[1].Outcome != PostingOutcomeJournalEntryReversed {
		t.Fatalf("attempts = %#v", attempts)
	}
	if len(audit.Records) != 2 || audit.Records[1].Operation != JournalReversalOperation || audit.Records[1].OriginalJournalID != original.AggregateID() || audit.Records[1].CorrectionReason != command.CorrectionReason {
		t.Fatalf("audit records = %#v", audit.Records)
	}
}

func TestJournalReversalReplaysAndAllowsOnlyOneCorrection(t *testing.T) {
	service, posting, actor, original, scopeID := newMemoryJournalReversalFixture(t)
	command := reverseCommandForJournal(original, scopeID, uuid.New(), "replay-source")

	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.AggregateID() != first.AggregateID() {
		t.Fatalf("replay = %#v", replay)
	}
	changed := command
	changed.CorrectionReason = "different reason"
	if _, err := service.Execute(context.Background(), actor, changed); !errors.Is(err, ErrPostingIdempotencyConflict) {
		t.Fatalf("changed retry error = %v, want idempotency conflict", err)
	}
	duplicate := reverseCommandForJournal(original, scopeID, uuid.New(), "new-source")
	if _, err := service.Execute(context.Background(), actor, duplicate); !errors.Is(err, ErrJournalEntryAlreadyReversed) {
		t.Fatalf("second reversal error = %v, want already reversed", err)
	}
	if got := len(posting.Journals()); got != 2 {
		t.Fatalf("journal count = %d, want original plus one reversal", got)
	}
}

func TestJournalReversalRejectsStaleVersionAndRestrictedGate(t *testing.T) {
	service, repository, actor, original, scopeID := newMemoryJournalReversalFixture(t)
	stale := reverseCommandForJournal(original, scopeID, uuid.New(), "stale-source")
	stale.ExpectedJournalVersion++
	if _, err := service.Execute(context.Background(), actor, stale); !errors.Is(err, ErrJournalReversalVersionConflict) {
		t.Fatalf("stale version error = %v, want version conflict", err)
	}
	if len(repository.Journals()) != 1 {
		t.Fatalf("stale reversal created state: %#v", repository.Journals())
	}

	restrictedRepository := NewMemoryPostingRepository()
	postingService, err := NewPostingService(restrictedRepository, MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true}}, AllowAllPostingReferenceValidator{}, MemoryPostingGateValidator{}, AllowAllPostingApprovalValidator{}, &MemoryPostingAuditRecorder{}, testPostingCurrencyRegistry(t), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	restrictedRequest := validPostingRequest(t, "restricted-original")
	restrictedOriginal, err := postingService.Execute(context.Background(), actor, restrictedRequest)
	if err != nil {
		t.Fatal(err)
	}
	restrictedService, err := NewJournalReversalService(
		restrictedRepository,
		MemoryJournalReversalAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true}},
		AllowAllPostingReferenceValidator{},
		MemoryPostingGateValidator{Evidence: PostingGateEvidence{FiscalPeriodID: restrictedOriginal.GateEvidence.FiscalPeriodID, PeriodStateVersion: restrictedOriginal.GateEvidence.PeriodStateVersion, PostingGateVersion: restrictedOriginal.GateEvidence.PostingGateVersion, GateMode: PostingGateHardClosed}},
		testPostingCurrencyRegistry(t),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	restrictedCommand := reverseCommandForJournal(restrictedOriginal, restrictedRequest.AccountingScopeID, uuid.New(), "restricted-source")
	if _, err := restrictedService.Execute(context.Background(), actor, restrictedCommand); !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("restricted gate error = %v, want gate closed", err)
	}
}

func TestJournalReversalRejectsPendingJournalAndAuthorization(t *testing.T) {
	repository := NewMemoryPostingRepository()
	actor := Actor{UserID: uuid.New(), SubjectReference: "journal-reversal-actor"}
	audit := &MemoryPostingAuditRecorder{}
	posting, err := NewPostingService(repository, MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true, ApprovalRequired: true}}, AllowAllPostingReferenceValidator{}, MemoryPostingGateValidator{}, requiredPostingApprovalValidator{approval: PostingApproval{Required: true, ApprovalRequestID: uuid.New()}}, audit, testPostingCurrencyRegistry(t), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	pendingRequest := validPostingRequest(t, "pending-original")
	pending, err := posting.Execute(context.Background(), actor, pendingRequest)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewJournalReversalService(repository, MemoryJournalReversalAuthorizer{Decision: PostingAuthorizationDecision{Allowed: false}}, AllowAllPostingReferenceValidator{}, MemoryPostingGateValidator{}, testPostingCurrencyRegistry(t), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	command := reverseCommandForJournal(pending, pendingRequest.AccountingScopeID, uuid.New(), "pending-source")
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrJournalEntryNotPosted) {
		t.Fatalf("pending journal error = %v, want not posted", err)
	}
}

func newMemoryJournalReversalFixture(t *testing.T) (*JournalReversalService, *MemoryPostingRepository, Actor, PostingResult, uuid.UUID) {
	t.Helper()
	repository := NewMemoryPostingRepository()
	actor := Actor{UserID: uuid.New(), SubjectReference: "journal-reversal-actor"}
	posting, err := NewPostingService(repository, MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true}}, AllowAllPostingReferenceValidator{}, MemoryPostingGateValidator{}, AllowAllPostingApprovalValidator{}, &MemoryPostingAuditRecorder{}, testPostingCurrencyRegistry(t), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	originalRequest := validPostingRequest(t, "fixture-original")
	original, err := posting.Execute(context.Background(), actor, originalRequest)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewJournalReversalService(repository, MemoryJournalReversalAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true}}, AllowAllPostingReferenceValidator{}, MemoryPostingGateValidator{}, testPostingCurrencyRegistry(t), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, actor, original, originalRequest.AccountingScopeID
}

func reverseCommandForJournal(original PostingResult, scopeID, authorizationID uuid.UUID, source string) ReverseJournalEntryCommand {
	return ReverseJournalEntryCommand{
		CommandID: uuid.New(), OriginalJournalID: original.AggregateID(), AccountingScopeID: scopeID, ExpectedJournalVersion: original.AggregateVersion(),
		CorrectionReason: "Correct a posted journal classification", EffectivePostingDate: time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC), FiscalPeriodID: original.GateEvidence.FiscalPeriodID,
		PeriodStateVersion: original.GateEvidence.PeriodStateVersion, PostingGateVersion: original.GateEvidence.PostingGateVersion, PostingAuthorizationID: authorizationID,
		SourceContext: "general-ledger", SourceAggregateType: "journal-correction", SourceAggregateID: uuid.New(), SourceVersion: 1,
		IdempotencyKey: source, CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
}
