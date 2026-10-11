package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestScopedReopenGateOpenAndCloseNoChangeRestoresHardClosed(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scopeID, periodID, requestID := uuid.New(), uuid.New(), uuid.New()
	audit := &MemorySoftCloseGateAuditRecorder{}
	repository := NewMemorySoftCloseGateRepository(audit)
	gate := NewOpenMemorySoftCloseGate(scopeID, periodID, 4)
	gate.GateMode = PostingGateHardClosed
	gate.GateVersion = 7
	gate.AggregateVersion = 7
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, allowScopedReopenForTest(scopeID), AllowAllSoftClosePolicyEvaluator{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "controller"}
	command := validOpenScopedReopenCommand(scopeID, periodID, requestID, 7, now.Add(time.Hour))

	opened, err := service.OpenScopedReopenGate(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Gate.GateMode != PostingGateScopedReopen || opened.Gate.ActiveReopenRequestID == nil || *opened.Gate.ActiveReopenRequestID != requestID || opened.Gate.GateVersion != 8 {
		t.Fatalf("opened gate = %#v", opened.Gate)
	}
	if opened.Gate.ActiveAdmissionCounters.ControlOwnerType != ScopedReopenGateOwnerType || opened.Gate.ActiveAdmissionCounters.ControlOwnerID != requestID || opened.Gate.ActiveAdmissionCounters.ControlOwnerEpoch == nil || *opened.Gate.ActiveAdmissionCounters.ControlOwnerEpoch != 9 || opened.Gate.ActiveAdmissionCounters.AdmittedPostingCount != 0 {
		t.Fatalf("opened counters = %#v", opened.Gate.ActiveAdmissionCounters)
	}
	if opened.Gate.ReopenAuthorizationExpiry == nil || !opened.Gate.ReopenAuthorizationExpiry.Equal(now.Add(time.Hour)) || opened.Gate.ReopenControlAuthorityEpoch == nil || *opened.Gate.ReopenControlAuthorityEpoch != 9 {
		t.Fatalf("opened authorization = %#v", opened.Gate)
	}

	closed, err := service.CloseScopedReopenGate(context.Background(), actor, CloseScopedReopenGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 8,
		ReopenRequestID: requestID, IdempotencyKey: "close-no-change", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Gate.GateMode != PostingGateHardClosed || closed.Gate.ActiveReopenRequestID != nil || closed.Gate.ActiveAdmissionCounters.ControlOwnerType != "" {
		t.Fatalf("closed gate = %#v", closed.Gate)
	}
	if closed.Gate.LastFrozenAdmissionSummary == nil || closed.Gate.LastFrozenAdmissionSummary.AdmittedPostingCount != 0 || len(closed.Gate.FrozenAdmissionSummaryHistory) != 1 {
		t.Fatalf("closed summary = %#v", closed.Gate.LastFrozenAdmissionSummary)
	}
	if events := repository.Events(); len(events) != 2 || events[0].EventType != ScopedReopenGateOpenedEventType || events[1].EventType != ScopedReopenGateClosedEventType {
		t.Fatalf("events = %#v", events)
	}
	if len(audit.Records) != 2 || audit.Records[0].ApprovalRequestID != command.Approval.ApprovalRequestID || audit.Records[0].ApprovalDecisionID != command.Approval.DecisionID || audit.Records[1].ApproverUserID != command.Approval.ApproverUserID {
		t.Fatalf("approval audit evidence = %#v", audit.Records)
	}
}

func TestScopedReopenGateAdmittedCorrectionRequiresScopeAndClosesToMandatoryReclose(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scopeID, periodID, requestID := uuid.New(), uuid.New(), uuid.New()
	accountID := uuid.New()
	repository := NewMemorySoftCloseGateRepository()
	gate := NewOpenMemorySoftCloseGate(scopeID, periodID, 4)
	gate.GateMode = PostingGateHardClosed
	gate.GateVersion = 3
	gate.AggregateVersion = 3
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, allowScopedReopenForTest(scopeID), AllowAllSoftClosePolicyEvaluator{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "controller"}
	openCommand := validOpenScopedReopenCommand(scopeID, periodID, requestID, 3, now.Add(time.Hour))
	openCommand.AuthorizationScope.AccountIDs = []uuid.UUID{accountID}
	openCommand.NoChangeClosePermitted = true
	if _, err := service.OpenScopedReopenGate(context.Background(), actor, openCommand); err != nil {
		t.Fatal(err)
	}

	posting := PostingRequest{FiscalPeriodID: periodID, PeriodStateVersion: 4, PostingGateVersion: 4, PostingPurpose: PostingPurposeReopenCorrection, ReopenRequestID: &requestID, ControlAuthorityEpoch: int64Pointer(9), SourceAggregateType: "manual-journal", Lines: []PostingLine{{AccountID: accountID}, {AccountID: accountID}}}
	evidence := PostingGateEvidence{FiscalPeriodID: periodID, PeriodStateVersion: 4, PostingGateVersion: 4, GateMode: PostingGateScopedReopen, ReopenRequestID: &requestID, AuthorizationScope: jsonScopeForTest(t, ScopedReopenScope{AccountIDs: []uuid.UUID{accountID}}), AuthorizationExpiry: timePointer(now.Add(time.Hour)), ControlAuthorityEpoch: int64Pointer(9)}
	if err := validatePostingGateEvidenceAt(posting, evidence, now); err != nil {
		t.Fatalf("in-scope posting evidence rejected: %v", err)
	}
	outOfScope := posting
	outOfScope.Lines = []PostingLine{{AccountID: uuid.New()}, {AccountID: uuid.New()}}
	if err := validatePostingGateEvidenceAt(outOfScope, evidence, now); !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("out-of-scope posting error = %v", err)
	}
	expiredEvidence := evidence
	expiredEvidence.AuthorizationExpiry = timePointer(now)
	if err := validatePostingGateEvidenceAt(posting, expiredEvidence, now); !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("expired posting error = %v", err)
	}

	active, err := repository.GetSoftCloseGate(context.Background(), scopeID, periodID)
	if err != nil {
		t.Fatal(err)
	}
	active.ActiveAdmissionCounters.AdmittedPostingCount = 1
	active.ActiveAdmissionCounters.ReopenPostingAdmitted = true
	active.ActiveAdmissionCounters.FirstAdmittedLedgerPosition = 1
	active.ActiveAdmissionCounters.LastAdmittedLedgerPosition = 1
	if err := repository.Seed(active); err != nil {
		t.Fatal(err)
	}
	closed, err := service.CloseScopedReopenGate(context.Background(), actor, CloseScopedReopenGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 4,
		ReopenRequestID: requestID, IdempotencyKey: "close-with-correction", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Gate.GateMode != PostingGateCloseOnly || closed.Gate.ActiveReopenRequestID == nil || *closed.Gate.ActiveReopenRequestID != requestID {
		t.Fatalf("mandatory reclose gate = %#v", closed.Gate)
	}
	if closed.Gate.LastFrozenAdmissionSummary == nil || closed.Gate.LastFrozenAdmissionSummary.AdmittedPostingCount != 1 || !closed.Gate.LastFrozenAdmissionSummary.ReopenPostingAdmitted {
		t.Fatalf("mandatory reclose summary = %#v", closed.Gate.LastFrozenAdmissionSummary)
	}
}

func TestScopedReopenGateReplaysAndRejectsChangedFingerprint(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scopeID, periodID, requestID := uuid.New(), uuid.New(), uuid.New()
	repository := NewMemorySoftCloseGateRepository()
	gate := NewOpenMemorySoftCloseGate(scopeID, periodID, 4)
	gate.GateMode = PostingGateHardClosed
	gate.GateVersion = 5
	gate.AggregateVersion = 5
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, allowScopedReopenForTest(scopeID), AllowAllSoftClosePolicyEvaluator{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "controller"}
	command := validOpenScopedReopenCommand(scopeID, periodID, requestID, 5, now.Add(time.Hour))
	first, err := service.OpenScopedReopenGate(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.OpenScopedReopenGate(context.Background(), actor, command)
	if err != nil || !replay.Replayed || replay.Gate.GateVersion != first.Gate.GateVersion {
		t.Fatalf("replay = %#v, err = %v", replay, err)
	}
	changed := command
	changed.AuthorizationScope.TransactionClasses = []string{"different-class"}
	if _, err := service.OpenScopedReopenGate(context.Background(), actor, changed); !errors.Is(err, ErrSoftCloseGateIdempotencyConflict) {
		t.Fatalf("changed command error = %v", err)
	}
}

func validOpenScopedReopenCommand(scopeID, periodID, requestID uuid.UUID, expectedVersion int64, expiry time.Time) OpenScopedReopenGateCommand {
	return OpenScopedReopenGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: expectedVersion,
		ReopenRequestID: requestID, AuthorizationScope: ScopedReopenScope{AccountIDs: []uuid.UUID{uuid.New()}, TransactionClasses: []string{"manual-journal"}},
		AuthorizationExpiry: expiry, ControlAuthorityEpoch: 9,
		Approval:               ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "reopen-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:approved", ApproverUserID: uuid.New()},
		NoChangeClosePermitted: true, IdempotencyKey: "open-scoped-reopen", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
}

func allowScopedReopenForTest(scopeID uuid.UUID) MemorySoftCloseGateAuthorizer {
	return MemorySoftCloseGateAuthorizer{Decision: AuthorizationDecision{Allowed: true, ApprovedScopeIDs: []uuid.UUID{scopeID}, DecisionReference: uuid.New(), PolicyReference: "iam-policy", PolicyVersion: "iam-policy-v1"}}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func jsonScopeForTest(t *testing.T, scope ScopedReopenScope) []byte {
	t.Helper()
	_, raw, err := canonicalScopedReopenScope(scope)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
