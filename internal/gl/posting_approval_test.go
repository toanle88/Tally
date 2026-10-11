package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPostingApprovalDecisionApprovedPostsPendingJournalExactlyOnce(t *testing.T) {
	posting, apply, repository, audit, actor := testPostingApprovalServices(t)
	request := validPostingRequest(t, "approval-approved")
	requestActor := Actor{UserID: uuid.New(), SubjectReference: "submitter-1"}
	pending, err := posting.Execute(context.Background(), requestActor, request)
	if err != nil {
		t.Fatal(err)
	}

	command := approvalDecisionCommand(t, pending, request.AccountingScopeID, PostingApprovalOutcomeApproved, "apply-approved")
	first, err := apply.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome != PostingOutcomeJournalEntryPosted || first.LifecycleStatus() != PostingStatusPosted || first.ApplicationStatus != PostingApprovalApplicationApplied {
		t.Fatalf("first approval result = %#v", first)
	}
	if first.AggregateVersion() != 2 || first.Journal == nil || first.Journal.LedgerPosition != 1 {
		t.Fatalf("posted journal = %#v, want version 2 and ledger position 1", first.Journal)
	}
	if first.PolicyReference != "policy-1" {
		t.Fatalf("policy reference = %q, want the immutable approval policy", first.PolicyReference)
	}

	replay, err := apply.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.AggregateID() != first.AggregateID() || replay.AggregateVersion() != first.AggregateVersion() {
		t.Fatalf("replay = %#v, want the established result", replay)
	}
	if got := len(repository.Journals()); got != 1 {
		t.Fatalf("journal count = %d, want 1", got)
	}
	if got := len(repository.Events()); got != 2 {
		t.Fatalf("event count = %d, want submit and apply", got)
	}
	if got := len(audit.Records); got != 2 {
		t.Fatalf("audit count = %d, want submit and apply", got)
	}
	if attempts := repository.Attempts(); len(attempts) != 2 || attempts[1].Operation != PostingApprovalAction {
		t.Fatalf("attempts = %#v, want one apply attempt", attempts)
	}
}

func TestPostingApprovalDecisionEvaluatesSoftClosePolicyBeforeApplication(t *testing.T) {
	request := validPostingRequest(t, "approval-soft-close-policy")
	gate := MemoryPostingGateValidator{Evidence: PostingGateEvidence{
		FiscalPeriodID: request.FiscalPeriodID, PeriodStateVersion: request.PeriodStateVersion, PostingGateVersion: request.PostingGateVersion,
		GateMode: PostingGateSoftClosePolicy, SoftClosePolicyVersion: "policy-v1", SoftCloseRestrictions: []byte(`{"ordinaryPosting":"review"}`),
	}}
	evaluator := &recordingSoftClosePostingPolicyEvaluator{}
	repository := NewMemoryPostingRepository()
	audit := &MemoryPostingAuditRecorder{}
	currency := testPostingCurrencyRegistry(t)
	postingAuthorizer := MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true, ApprovalRequired: true, PolicyReference: "policy-1", PolicyVersion: "1"}}
	posting, err := NewPostingService(
		repository, postingAuthorizer, AllowAllPostingReferenceValidator{}, gate,
		requiredPostingApprovalValidator{approval: PostingApproval{Required: true, ApprovalRequestID: uuid.New(), PolicyReference: "policy-1", PolicyVersion: "1"}},
		audit, currency, time.Now, evaluator,
	)
	if err != nil {
		t.Fatal(err)
	}
	apply, err := NewPostingApprovalDecisionService(
		repository, MemoryPostingApprovalDecisionVerifier{}, MemoryPostingApprovalDecisionAuthorizer{Decision: PostingApprovalDecisionAuthorization{Allowed: true}},
		postingAuthorizer, AllowAllPostingReferenceValidator{}, gate, currency, time.Now, evaluator,
	)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := posting.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "submitter-1"}, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := apply.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "approver-1"}, approvalDecisionCommand(t, pending, request.AccountingScopeID, PostingApprovalOutcomeApproved, "approval-soft-close-policy-apply")); err != nil {
		t.Fatal(err)
	}
	if evaluator.calls != 2 {
		t.Fatalf("policy evaluator calls = %d, want submission and approval application", evaluator.calls)
	}
}

func TestPostingApprovalDecisionRejectedCancelsWithoutLedgerEffect(t *testing.T) {
	posting, apply, repository, _, actor := testPostingApprovalServices(t)
	request := validPostingRequest(t, "approval-rejected")
	pending, err := posting.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "submitter-1"}, request)
	if err != nil {
		t.Fatal(err)
	}

	decision := approvalDecisionCommand(t, pending, request.AccountingScopeID, PostingApprovalOutcomeRejected, "apply-rejected")
	decision.Decision.Reason = "Rejected by the approval policy"
	result, err := apply.Execute(context.Background(), actor, decision)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != PostingOutcomePostingRejected || result.LifecycleStatus() != PostingStatusCancelled || result.ApplicationStatus != PostingApprovalApplicationApplied {
		t.Fatalf("rejection result = %#v", result)
	}
	if result.Journal == nil || result.Journal.LedgerPosition != 0 || result.Journal.Version != 2 || result.Journal.Approval == nil || result.Journal.Approval.DecisionOutcome != PostingApprovalOutcomeRejected || result.Journal.Approval.DecisionReason != decision.Decision.Reason {
		t.Fatalf("cancelled journal = %#v, want no ledger effect and applied rejection", result.Journal)
	}
	if got := len(repository.Events()); got != 2 {
		t.Fatalf("event count = %d, want submit and rejection", got)
	}
}

func TestPostingApprovalDecisionUsesDecisionIdentityAndRejectsChangedDelivery(t *testing.T) {
	posting, apply, _, _, actor := testPostingApprovalServices(t)
	request := validPostingRequest(t, "approval-decision-identity")
	pending, err := posting.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "submitter-1"}, request)
	if err != nil {
		t.Fatal(err)
	}

	firstCommand := approvalDecisionCommand(t, pending, request.AccountingScopeID, PostingApprovalOutcomeApproved, "apply-first")
	if _, err := apply.Execute(context.Background(), actor, firstCommand); err != nil {
		t.Fatal(err)
	}
	secondCommand := firstCommand
	secondCommand.IdempotencyKey = "apply-second-delivery"
	unchanged, err := apply.Execute(context.Background(), actor, secondCommand)
	if err != nil {
		t.Fatal(err)
	}
	if !unchanged.Replayed || unchanged.ApplicationStatus != PostingApprovalApplicationUnchanged {
		t.Fatalf("same decision delivery = %#v, want unchanged replay", unchanged)
	}

	changed := firstCommand
	changed.IdempotencyKey = "apply-changed-content"
	changed.Decision.Reason = "changed after approval"
	if _, err := apply.Execute(context.Background(), actor, changed); !errors.Is(err, ErrPostingApprovalDecisionIdempotencyConflict) {
		t.Fatalf("changed delivery error = %v, want idempotency conflict", err)
	}
}

func TestPostingApprovalDecisionDoesNotMutateOnStaleVersionOrAuditFailure(t *testing.T) {
	posting, apply, repository, audit, actor := testPostingApprovalServices(t)
	request := validPostingRequest(t, "approval-stale")
	pending, err := posting.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "submitter-1"}, request)
	if err != nil {
		t.Fatal(err)
	}
	stale := approvalDecisionCommand(t, pending, request.AccountingScopeID, PostingApprovalOutcomeApproved, "apply-stale")
	stale.ExpectedJournalVersion = 2
	if _, err := apply.Execute(context.Background(), actor, stale); !errors.Is(err, ErrPostingApprovalDecisionVersionConflict) {
		t.Fatalf("stale version error = %v, want version conflict", err)
	}
	if repository.Journals()[0].Status != PostingStatusPendingApproval {
		t.Fatal("stale version changed the pending journal")
	}

	audit.Err = errors.New("approval audit unavailable")
	valid := approvalDecisionCommand(t, pending, request.AccountingScopeID, PostingApprovalOutcomeApproved, "apply-audit-failure")
	if _, err := apply.Execute(context.Background(), actor, valid); !errors.Is(err, audit.Err) {
		t.Fatalf("audit failure error = %v, want audit error", err)
	}
	if journal := repository.Journals()[0]; journal.Status != PostingStatusPendingApproval || journal.Version != 1 || journal.LedgerPosition != 0 {
		t.Fatalf("journal after audit failure = %#v, want unchanged pending journal", journal)
	}
}

func TestPostingApprovalDecisionRejectsMismatchedImmutableEvidence(t *testing.T) {
	posting, _, repository, _, actor := testPostingApprovalServices(t)
	request := validPostingRequest(t, "approval-evidence-mismatch")
	pending, err := posting.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "submitter-1"}, request)
	if err != nil {
		t.Fatal(err)
	}

	badVerifier := mutatingPostingApprovalVerifier{mutate: func(evidence *PostingApprovalDecisionEvidence) {
		evidence.PolicyReference = "different-policy"
	}}
	apply := testPostingApprovalDecisionService(t, repository, badVerifier)
	command := approvalDecisionCommand(t, pending, request.AccountingScopeID, PostingApprovalOutcomeApproved, "apply-evidence-mismatch")
	if _, err := apply.Execute(context.Background(), actor, command); !errors.Is(err, ErrPostingApprovalDecisionReferenceInvalid) {
		t.Fatalf("mismatched evidence error = %v, want invalid reference", err)
	}
	if journal := repository.Journals()[0]; journal.Status != PostingStatusPendingApproval || journal.Version != 1 || journal.LedgerPosition != 0 {
		t.Fatalf("journal after mismatched evidence = %#v, want unchanged pending journal", journal)
	}
}

func testPostingApprovalServices(t *testing.T) (*PostingService, *PostingApprovalDecisionService, *MemoryPostingRepository, *MemoryPostingAuditRecorder, Actor) {
	t.Helper()
	repository := NewMemoryPostingRepository()
	audit := &MemoryPostingAuditRecorder{}
	currency := testPostingCurrencyRegistry(t)
	postingAuthorizer := MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true, ApprovalRequired: true, PolicyReference: "policy-1", PolicyVersion: "1"}}
	gate := MemoryPostingGateValidator{}
	posting, err := NewPostingService(repository, postingAuthorizer, AllowAllPostingReferenceValidator{}, gate, requiredPostingApprovalValidator{approval: PostingApproval{Required: true, ApprovalRequestID: uuid.New(), PolicyReference: "policy-1", PolicyVersion: "1"}}, audit, currency, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	apply, err := NewPostingApprovalDecisionService(repository, MemoryPostingApprovalDecisionVerifier{}, MemoryPostingApprovalDecisionAuthorizer{Decision: PostingApprovalDecisionAuthorization{Allowed: true}}, postingAuthorizer, AllowAllPostingReferenceValidator{}, gate, currency, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return posting, apply, repository, audit, Actor{UserID: uuid.New(), SubjectReference: "approver-1"}
}

func testPostingApprovalDecisionService(t *testing.T, repository *MemoryPostingRepository, verifier PostingApprovalDecisionVerifier) *PostingApprovalDecisionService {
	t.Helper()
	postingAuthorizer := MemoryPostingAuthorizer{Decision: PostingAuthorizationDecision{Allowed: true, ApprovalRequired: true, PolicyReference: "policy-1", PolicyVersion: "1"}}
	service, err := NewPostingApprovalDecisionService(repository, verifier, MemoryPostingApprovalDecisionAuthorizer{Decision: PostingApprovalDecisionAuthorization{Allowed: true}}, postingAuthorizer, AllowAllPostingReferenceValidator{}, MemoryPostingGateValidator{}, testPostingCurrencyRegistry(t), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

type mutatingPostingApprovalVerifier struct {
	mutate func(*PostingApprovalDecisionEvidence)
}

func (verifier mutatingPostingApprovalVerifier) VerifyPostingApprovalDecision(ctx context.Context, actor Actor, subject PostingApprovalSubject, decision PostingApprovalDecision) (PostingApprovalDecisionEvidence, error) {
	evidence, err := (MemoryPostingApprovalDecisionVerifier{}).VerifyPostingApprovalDecision(ctx, actor, subject, decision)
	if err != nil {
		return PostingApprovalDecisionEvidence{}, err
	}
	verifier.mutate(&evidence)
	return evidence, nil
}

func approvalDecisionCommand(t *testing.T, pending PostingResult, scopeID uuid.UUID, outcome, idempotencyKey string) ApplyJournalApprovalDecisionCommand {
	t.Helper()
	if pending.Journal == nil || pending.Journal.Approval == nil {
		t.Fatal("pending posting did not contain approval evidence")
	}
	return ApplyJournalApprovalDecisionCommand{
		CommandID: uuid.New(), JournalID: pending.AggregateID(), AccountingScopeID: scopeID, ExpectedJournalVersion: pending.AggregateVersion(),
		Decision: PostingApprovalDecision{Outcome: outcome, Reference: ApprovalDecisionReference{
			ApprovalRequestID: pending.Journal.Approval.ApprovalRequestID, DecisionID: uuid.New(), PolicyVersion: pending.Journal.Approval.PolicyVersion,
			DecisionVersion: 1, SubjectVersion: pending.AggregateVersion(), CandidateFingerprint: pending.Journal.RequestFingerprint, ApproverUserID: uuid.New(),
		}},
		IdempotencyKey: idempotencyKey, CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
}
