package fiscalperiod

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
)

type us3TaskStub struct {
	results []CloseTaskResult
	err     error
	runs    []uuid.UUID
}

func (stub *us3TaskStub) Execute(_ context.Context, _ Actor, run CloseRun) (CloseTaskExecution, error) {
	stub.runs = append(stub.runs, run.ID)
	if stub.err != nil {
		return CloseTaskExecution{}, stub.err
	}
	return CloseTaskExecution{Accepted: true, Results: append([]CloseTaskResult(nil), stub.results...)}, nil
}

type us3DecisionStub struct {
	err  error
	seen []CloseApprovalDecision
}

func (stub *us3DecisionStub) Validate(_ context.Context, _ Actor, decision CloseApprovalDecision) error {
	stub.seen = append(stub.seen, decision)
	if stub.err != nil {
		return stub.err
	}
	return decision.Validate()
}

type us3SealStub struct {
	result AuditSealResult
	err    error
	calls  int
}

func (stub *us3SealStub) RequestSeal(_ context.Context, _ Actor, _ CloseRun) (AuditSealResult, error) {
	stub.calls++
	return stub.result, stub.err
}

func TestResumeCloseRunPreservesIdentityAndRecordsTaskProgress(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 8, 0, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{}
	service, repository, gate, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub}, clock)
	for _, step := range defaultCloseSteps() {
		done := clock()
		taskStub.results = append(taskStub.results, CloseTaskResult{StepID: step.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: step.Owner, ImmutableEvidenceRef: "evidence:" + step.ID, LedgerWatermark: 25, CompletedAt: &done})
	}

	command := resumeFixture(state, "resume-1")
	result, err := service.ResumeCloseRun(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if len(taskStub.runs) != 1 || taskStub.runs[0] != state.CloseRun.ID {
		t.Fatalf("task identities = %#v", taskStub.runs)
	}
	if len(gate.calls) != 1 {
		t.Fatalf("resume acquired a second barrier: %#v", gate.calls)
	}
	if result.State.CloseRun.ID != state.CloseRun.ID || result.State.CloseRun.Status != CloseRunAwaitingApproval || result.State.CloseRun.FinalLedgerWatermark == nil || *result.State.CloseRun.FinalLedgerWatermark != 25 {
		t.Fatalf("resumed state = %#v", result.State.CloseRun)
	}
	if len(result.State.CloseRun.Steps) != len(defaultCloseSteps()) || result.State.CloseRun.Steps[0].ImmutableEvidenceRef == "" {
		t.Fatalf("step evidence = %#v", result.State.CloseRun.Steps)
	}
	replay, err := service.ResumeCloseRun(context.Background(), actor, command)
	if err != nil || !replay.Replayed || len(taskStub.runs) != 1 {
		t.Fatalf("resume replay = %#v, err = %v, calls = %#v", replay, err, taskStub.runs)
	}
	_ = repository
}

func TestApplyCloseApprovalFinalizesGLAndLeavesSealPending(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{}
	decisionStub := &us3DecisionStub{}
	sealStub := &us3SealStub{result: AuditSealResult{Status: CloseRunSealPending, RequestID: uuid.New()}}
	service, _, _, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub, Workflow: decisionStub, Seal: sealStub}, clock)
	for _, step := range defaultCloseSteps() {
		done := clock()
		taskStub.results = append(taskStub.results, CloseTaskResult{StepID: step.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: step.Owner, ImmutableEvidenceRef: "evidence:" + step.ID, LedgerWatermark: 25, CompletedAt: &done})
	}
	resumed, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(state, "resume-finalize"))
	if err != nil {
		t.Fatal(err)
	}
	approval := finalApprovalFixture(resumed.State, "final-approval-1")
	result, err := service.ApplyCloseApprovalDecision(context.Background(), actor, approval)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateHardClosed || result.State.CloseRun.Status != CloseRunCompleted || result.State.CloseRun.SealStatus != CloseRunSealPending {
		t.Fatalf("finalized state = %#v / %#v", result.State.Period, result.State.CloseRun)
	}
	if result.State.CloseRun.FinalLedgerWatermark == nil || *result.State.CloseRun.FinalLedgerWatermark != 25 || result.GateResult.Gate.GateMode != gl.PostingGateHardClosed {
		t.Fatalf("finalization evidence = %#v / %#v", result.State.CloseRun, result.GateResult.Gate)
	}
	if sealStub.calls != 1 || len(decisionStub.seen) != 1 {
		t.Fatalf("seal/decision calls = %d/%d", sealStub.calls, len(decisionStub.seen))
	}
}

func TestApplyCloseApprovalDecisionRecordsRejectionWithoutWatermark(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 9, 15, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{}
	service, _, gate, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub}, clock)
	for _, step := range defaultCloseSteps() {
		done := clock()
		taskStub.results = append(taskStub.results, CloseTaskResult{StepID: step.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: step.Owner, ImmutableEvidenceRef: "evidence:" + step.ID, LedgerWatermark: 25, CompletedAt: &done})
	}
	resumed, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(state, "resume-rejected-approval"))
	if err != nil {
		t.Fatal(err)
	}
	approval := finalApprovalFixture(resumed.State, "reject-final-approval")
	approval.Outcome = CloseApprovalRejected
	approval.FinalLedgerWatermark = 0
	approval.DecisionReason = "Final close evidence requires correction."
	result, err := service.ApplyCloseApprovalDecision(context.Background(), actor, approval)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.CloseRun.Status != CloseRunFailed || result.State.CloseRun.FinalApproval == nil || result.State.CloseRun.FinalApproval.Outcome != CloseApprovalRejected || gate.finalizeCalls != 0 {
		t.Fatalf("rejected approval state = %#v, finalizeCalls = %d", result.State.CloseRun, gate.finalizeCalls)
	}
}

func TestApplyCloseApprovalRejectsAnExpiredApprovedException(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 9, 20, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{}
	service, repository, gate, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub}, clock)
	approvalRequestID := uuid.New()
	state.CloseRun.Exceptions = []CloseException{{
		ID: uuid.New(), CloseRunID: state.CloseRun.ID, Code: "FX_RATE_EXCEPTION", Status: CloseExceptionApproved,
		ApprovalRequestID: approvalRequestID, ExpiresAt: func() *time.Time { value := clock().Add(-time.Minute); return &value }(),
		ApprovalDecision: &CloseApprovalDecision{RequestID: approvalRequestID, DecisionID: uuid.New(), DecisionVersion: 1, Outcome: CloseApprovalApproved, PolicyVersion: "close-policy-v1", SubjectVersion: state.CloseRun.AggregateVersion, CandidateFingerprint: state.CloseRun.CommandFingerprint, ApproverUserID: uuid.New(), DecisionReference: uuid.New(), SeparationOfDutiesReference: "sod:expired", RecordedAt: clock().Add(-2 * time.Minute)},
	}}
	state.CloseRun.ApprovalPolicyVersion = "close-policy-v1"
	state.CloseRun.AggregateVersion++
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	for _, step := range defaultCloseSteps() {
		done := clock()
		taskStub.results = append(taskStub.results, CloseTaskResult{StepID: step.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: step.Owner, ImmutableEvidenceRef: "evidence:" + step.ID, LedgerWatermark: 25, CompletedAt: &done})
	}
	resumed, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(state, "resume-expired-exception"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyCloseApprovalDecision(context.Background(), actor, finalApprovalFixture(resumed.State, "approve-expired-exception")); !errors.Is(err, ErrFiscalPeriodPolicyConflict) {
		t.Fatalf("expired exception error = %v", err)
	}
	if gate.finalizeCalls != 0 {
		t.Fatalf("expired exception called GL finalization %d times", gate.finalizeCalls)
	}
}

func TestApplyCloseExceptionApprovalDecisionStoresImmutableWorkflowEvidence(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 9, 30, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{}
	decisionStub := &us3DecisionStub{}
	service, repository, _, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub, Workflow: decisionStub}, clock)
	for _, step := range defaultCloseSteps() {
		done := clock()
		taskStub.results = append(taskStub.results, CloseTaskResult{StepID: step.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: step.Owner, ImmutableEvidenceRef: "evidence:" + step.ID, LedgerWatermark: 25, CompletedAt: &done})
	}
	exceptionID := uuid.New()
	approvalRequestID := uuid.New()
	state.CloseRun.Exceptions = []CloseException{{ID: exceptionID, CloseRunID: state.CloseRun.ID, Code: "FX_RATE_EXCEPTION", Status: CloseExceptionUnresolved, ApprovalRequestID: approvalRequestID}}
	state.CloseRun.AggregateVersion++
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	resumed, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(state, "resume-exception"))
	if err != nil {
		t.Fatal(err)
	}
	approval := ApplyCloseExceptionApprovalDecisionCommand{
		ResumeCloseRunCommand:       resumeFixture(resumed.State, "approve-exception"),
		ExceptionID:                 exceptionID,
		ApprovalRequestID:           approvalRequestID,
		DecisionID:                  uuid.New(),
		DecisionVersion:             1,
		Outcome:                     CloseApprovalApproved,
		DecisionReason:              "Exception evidence reviewed",
		PolicyVersion:               "close-policy-v1",
		SubjectVersion:              resumed.State.CloseRun.AggregateVersion,
		CandidateFingerprint:        resumed.State.CloseRun.CommandFingerprint,
		DecisionReference:           uuid.New(),
		ApproverUserID:              uuid.New(),
		SeparationOfDutiesReference: "sod:close-review",
		RecordedAt:                  clock(),
	}
	result, err := service.ApplyCloseExceptionApprovalDecision(context.Background(), actor, approval)
	if err != nil {
		t.Fatal(err)
	}
	stored := result.State.CloseRun.Exceptions[0]
	if stored.Status != CloseExceptionApproved || stored.ApprovalDecision == nil || stored.ApprovalDecision.DecisionReference != approval.DecisionReference || stored.ApprovalDecision.CandidateFingerprint != state.CloseRun.CommandFingerprint {
		t.Fatalf("exception approval evidence = %#v", stored)
	}
	if len(decisionStub.seen) != 1 {
		t.Fatalf("workflow validation calls = %d", len(decisionStub.seen))
	}

	changed := approval
	changed.DecisionReference = uuid.New()
	if _, err := service.ApplyCloseExceptionApprovalDecision(context.Background(), actor, changed); !errors.Is(err, ErrFiscalPeriodIdempotencyConflict) {
		t.Fatalf("changed duplicate error = %v", err)
	}
	changedKey := approval
	changedKey.IdempotencyKey = "approve-exception-changed-content"
	changedKey.ResumeCloseRunCommand = resumeFixture(result.State, changedKey.IdempotencyKey)
	changedKey.SubjectVersion = result.State.CloseRun.AggregateVersion
	changedKey.DecisionReference = uuid.New()
	if _, err := service.ApplyCloseExceptionApprovalDecision(context.Background(), actor, changedKey); !errors.Is(err, ErrFiscalPeriodIdempotencyConflict) {
		t.Fatalf("changed-content duplicate error = %v", err)
	}
	stale := approval
	stale.IdempotencyKey = "approve-exception-stale"
	stale.ResumeCloseRunCommand = resumeFixture(result.State, stale.IdempotencyKey)
	stale.SubjectVersion = result.State.CloseRun.AggregateVersion
	stale.CandidateFingerprint = "sha256:stale"
	if _, err := service.ApplyCloseExceptionApprovalDecision(context.Background(), actor, stale); !errors.Is(err, ErrCloseDecisionRejected) {
		t.Fatalf("stale decision error = %v", err)
	}
}

func TestSealFailureDoesNotReopenHardClosedPeriod(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 9, 45, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{}
	sealStub := &us3SealStub{result: AuditSealResult{Status: CloseRunSealFailed, FailureCode: "SEAL_PROVIDER_REJECTED", FailureDetail: "audit provider rejected the seal request"}}
	service, _, _, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub, Seal: sealStub}, clock)
	for _, step := range defaultCloseSteps() {
		done := clock()
		taskStub.results = append(taskStub.results, CloseTaskResult{StepID: step.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: step.Owner, ImmutableEvidenceRef: "evidence:" + step.ID, LedgerWatermark: 25, CompletedAt: &done})
	}
	resumed, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(state, "resume-seal-failure"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ApplyCloseApprovalDecision(context.Background(), actor, finalApprovalFixture(resumed.State, "approve-seal-failure"))
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateHardClosed || result.State.CloseRun.Status != CloseRunCompleted || result.State.CloseRun.SealStatus != CloseRunSealFailed || result.State.CloseRun.SealFailureCode != "SEAL_PROVIDER_REJECTED" {
		t.Fatalf("seal failure state = %#v / %#v", result.State.Period, result.State.CloseRun)
	}
}

func TestAbortCloseRunReleasesOnlyAnUnadmittedInitialBarrier(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC) }
	service, repository, _, actor, state := startedUS3Service(t, CloseRunDependencies{}, clock)
	command := resumeFixture(state, "abort-1")
	command.ExpectedGateVersion = state.CloseRun.BarrierEvidence.GateVersion
	result, err := service.AbortCloseRun(context.Background(), actor, AbortCloseRunCommand{ResumeCloseRunCommand: command, Reason: "Task dependency cannot complete before cutoff"})
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateSoftClosed || result.State.CloseRun.Status != CloseRunAborted || result.State.SoftClose.Status != SoftCloseRunActive || result.State.ControlEpoch.Number != 2 {
		t.Fatalf("aborted state = %#v", result.State)
	}
	if len(repository.Events()) != 7 {
		t.Fatalf("abort events = %#v", repository.Events())
	}
}

func TestAbortCloseRunRejectsAdmittedPostingWithoutChangingFPMState(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 11, 0, 0, 0, time.UTC) }
	state, start := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{result: hardCloseResult(start, uuid.New())}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, clock)
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor()
	started, err := service.StartHardClose(context.Background(), actor, start)
	if err != nil {
		t.Fatal(err)
	}
	gate.result.Gate.ActiveAdmissionCounters.ClosePostingAdmitted = true
	gate.result.Gate.ActiveAdmissionCounters.AdmittedPostingCount = 1
	current := started.State
	command := resumeFixture(current, "abort-admitted")
	if _, err := service.AbortCloseRun(context.Background(), actor, AbortCloseRunCommand{ResumeCloseRunCommand: command, Reason: "Abort after admitted posting"}); !errors.Is(err, ErrPostingBarrierRejected) {
		t.Fatalf("admitted abort error = %v", err)
	}
	after, err := repository.Get(context.Background(), start.AccountingScopeID, start.FiscalPeriodID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Period.State != PeriodStateClosing || after.CloseRun.Status != CloseRunBarrierAcquired {
		t.Fatalf("admitted abort mutated state = %#v", after)
	}
}

func TestResumeCloseRunReconcilesGLFinalizationBeforeFPMPersistence(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC) }
	service, repository, gate, actor, state := startedUS3Service(t, CloseRunDependencies{}, clock)
	state.CloseRun.Status = CloseRunFinalizing
	state.CloseRun.AggregateVersion++
	watermark := int64(25)
	state.CloseRun.FinalLedgerWatermark = &watermark
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	finalized := gate.result.Gate
	finalized.GateMode = gl.PostingGateHardClosed
	finalized.ActiveCloseRunID = nil
	finalized.ActiveCloseType = ""
	finalized.FinalizedCloseRunID = cloneUUID(&state.CloseRun.ID)
	finalized.FinalLedgerWatermark = &watermark
	finalized.ActiveAdmissionCounters = gl.ActiveGateAdmissionCounters{}
	finalized.LastFrozenAdmissionSummary = &gl.FrozenGateAdmissionSummary{ControlOwnerType: gl.PostingBarrierOwnerType, ControlOwnerID: state.CloseRun.ID, FrozenAtGateVersion: finalized.GateVersion, FrozenByCommandFingerprint: "sha256:recovered-finalize", FrozenByCommandType: gl.PostingBarrierFinalizeCommandIdentity}
	finalized.FinalizationCommandFingerprint = "sha256:recovered-finalize"
	gate.result.Gate = finalized
	command := resumeFixture(state, "resume-finalized")
	result, err := service.ResumeCloseRun(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateHardClosed || result.State.CloseRun.Status != CloseRunCompleted || !result.Replayed && result.GateResult.Gate.GateMode != gl.PostingGateHardClosed {
		t.Fatalf("reconciled finalization = %#v / %#v", result.State.Period, result.State.CloseRun)
	}
}

func TestResumeCloseRunRetriesAFailedBarrierWithTheExistingIdentity(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 12, 30, 0, 0, time.UTC) }
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{result: hardCloseResult(command, uuid.New()), err: errors.New("GL temporarily unavailable")}
	softOwner := state.SoftClose.ID
	gate.result.Gate.GateMode = gl.PostingGateSoftClosePolicy
	gate.result.Gate.GateVersion = command.ExpectedGateVersion
	gate.result.Gate.AggregateVersion = command.ExpectedGateVersion
	gate.result.Gate.ActiveCloseRunID = nil
	gate.result.Gate.ActiveCloseType = ""
	gate.result.Gate.ActiveSoftCloseRunID = &softOwner
	gate.result.Gate.ActiveSoftCloseControlEpoch = state.ControlEpoch.Number
	gate.result.Gate.SoftClosePolicyVersion = state.SoftClose.PolicyVersion
	softOwnerEpoch := state.ControlEpoch.Number
	gate.result.Gate.ActiveAdmissionCounters = gl.ActiveGateAdmissionCounters{ControlOwnerType: gl.SoftCloseGateOwnerType, ControlOwnerID: softOwner, ControlOwnerEpoch: &softOwnerEpoch}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, clock)
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor()
	if _, err := service.StartHardClose(context.Background(), actor, command); !errors.Is(err, ErrPostingBarrierUnavailable) {
		t.Fatalf("initial barrier error = %v", err)
	}
	failed, err := repository.Get(context.Background(), command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.CloseRun == nil || failed.CloseRun.Status != CloseRunFailed {
		t.Fatalf("failed close-run state = %#v", failed.CloseRun)
	}
	closeRunID := failed.CloseRun.ID
	gate.err = nil
	resume := ResumeCloseRunCommand{CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, SoftCloseRunID: command.SoftCloseRunID, CloseRunID: closeRunID, ExpectedPeriodVersion: failed.Period.AggregateVersion, ExpectedCloseRunVersion: failed.CloseRun.AggregateVersion, ExpectedGateVersion: command.ExpectedGateVersion, SoftCloseControlEpoch: command.SoftCloseControlEpoch, SoftClosePolicyVersion: command.SoftClosePolicyVersion, IdempotencyKey: "resume-failed-barrier", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	result, err := service.ResumeCloseRun(context.Background(), actor, resume)
	if err != nil {
		t.Fatal(err)
	}
	if len(gate.calls) != 2 || gate.calls[1].CloseRunID != closeRunID || result.State.CloseRun.ID != closeRunID || result.State.CloseRun.Status != CloseRunInProgress || result.State.Period.State != PeriodStateClosing {
		t.Fatalf("barrier retry = calls=%#v state=%#v", gate.calls, result.State)
	}
}

func TestResumeCloseRunPersistsTaskFailureAndCanRetry(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 13, 0, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{err: errors.New("task coordinator failed")}
	service, repository, _, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub}, clock)
	firstCommand := resumeFixture(state, "resume-task-failure")
	if _, err := service.ResumeCloseRun(context.Background(), actor, firstCommand); !errors.Is(err, ErrCloseTaskFailed) {
		t.Fatalf("task failure = %v", err)
	}
	failed, err := repository.Get(context.Background(), state.Period.AccountingScopeID, state.Period.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.CloseRun.Status != CloseRunFailed || failed.CloseRun.LastFailureCode != "CLOSE_TASK_FAILED" || failed.CloseRun.Steps[0].Status != CloseStepFailed {
		t.Fatalf("persisted task failure = %#v", failed.CloseRun)
	}
	for _, step := range defaultCloseSteps() {
		done := clock()
		taskStub.results = append(taskStub.results, CloseTaskResult{StepID: step.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: step.Owner, ImmutableEvidenceRef: "retry-evidence:" + step.ID, LedgerWatermark: 25, CompletedAt: &done})
	}
	taskStub.err = nil
	result, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(failed, "resume-task-retry"))
	if err != nil {
		t.Fatal(err)
	}
	if result.State.CloseRun.Status != CloseRunAwaitingApproval || len(taskStub.runs) != 2 || taskStub.runs[0] != taskStub.runs[1] {
		t.Fatalf("task retry = runs=%#v state=%#v", taskStub.runs, result.State.CloseRun)
	}
}

func TestResumeCloseRunRejectsChangedEvidenceForCompletedStep(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 13, 15, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{}
	service, repository, _, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub}, clock)
	firstStep := defaultCloseSteps()[0]
	taskStub.results = []CloseTaskResult{{StepID: firstStep.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: firstStep.Owner, ImmutableEvidenceRef: "immutable:evidence", LedgerWatermark: 25}}
	progressed, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(state, "resume-progressed"))
	if err != nil {
		t.Fatal(err)
	}
	if progressed.State.CloseRun.Status != CloseRunInProgress {
		t.Fatalf("progressed state = %#v", progressed.State.CloseRun)
	}
	taskStub.results = []CloseTaskResult{{StepID: firstStep.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: firstStep.Owner, ImmutableEvidenceRef: "changed:evidence", LedgerWatermark: 25}}
	if _, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(progressed.State, "resume-changed-evidence")); !errors.Is(err, ErrCloseTaskFailed) {
		t.Fatalf("changed evidence error = %v", err)
	}
	failed, err := repository.Get(context.Background(), state.Period.AccountingScopeID, state.Period.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.CloseRun.Status != CloseRunFailed || failed.CloseRun.Steps[0].ImmutableEvidenceRef != "immutable:evidence" {
		t.Fatalf("changed evidence state = %#v", failed.CloseRun)
	}
}

func TestAbortCloseRunReconcilesAnAuthoritativeRelease(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 13, 30, 0, 0, time.UTC) }
	service, repository, gate, actor, started := startedUS3Service(t, CloseRunDependencies{}, clock)
	aborting := cloneControlState(started)
	aborting.CloseRun.Status = CloseRunAborting
	aborting.CloseRun.AggregateVersion++
	if err := repository.Seed(aborting); err != nil {
		t.Fatal(err)
	}
	released := gate.result
	released.Operation = gl.PostingBarrierReleaseOperation
	released.Gate.GateMode = gl.PostingGateSoftClosePolicy
	released.Gate.GateVersion = aborting.CloseRun.BarrierEvidence.GateVersion + 1
	released.Gate.AggregateVersion = released.Gate.GateVersion
	released.Gate.ActiveCloseRunID = nil
	released.Gate.ActiveCloseType = ""
	released.Gate.FinalizedCloseRunID = nil
	released.Gate.LastFrozenAdmissionSummary = &gl.FrozenGateAdmissionSummary{ControlOwnerType: gl.PostingBarrierOwnerType, ControlOwnerID: aborting.CloseRun.ID, FrozenAtGateVersion: released.Gate.GateVersion, FrozenByCommandFingerprint: "sha256:recovered-release", FrozenByCommandType: gl.PostingBarrierReleaseCommandIdentity}
	softOwner := aborting.SoftClose.ID
	released.Gate.ActiveSoftCloseRunID = &softOwner
	released.Gate.ActiveSoftCloseControlEpoch = aborting.ControlEpoch.Number + 1
	released.Gate.SoftClosePolicyVersion = aborting.SoftClose.PolicyVersion
	released.Gate.ActiveAdmissionCounters = gl.ActiveGateAdmissionCounters{ControlOwnerType: gl.SoftCloseGateOwnerType, ControlOwnerID: softOwner, ControlOwnerEpoch: &released.Gate.ActiveSoftCloseControlEpoch}
	gate.result = released
	command := resumeFixture(aborting, "abort-release-recovery")
	result, err := service.AbortCloseRun(context.Background(), actor, AbortCloseRunCommand{ResumeCloseRunCommand: command, Reason: "Recover the acknowledged release"})
	if err != nil {
		t.Fatal(err)
	}
	if gate.releaseCalls != 0 || result.State.CloseRun.Status != CloseRunAborted || result.State.Period.State != PeriodStateSoftClosed || result.State.ControlEpoch.Number != aborting.ControlEpoch.Number+1 {
		t.Fatalf("release recovery = calls=%d state=%#v", gate.releaseCalls, result.State)
	}
}

func TestStartHardCloseAfterAbortCreatesANewRunIdentity(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 14, 0, 0, 0, time.UTC) }
	service, repository, gate, actor, started := startedUS3Service(t, CloseRunDependencies{}, clock)
	abortCommand := resumeFixture(started, "abort-before-new-run")
	abortCommand.ExpectedGateVersion = started.CloseRun.BarrierEvidence.GateVersion
	aborted, err := service.AbortCloseRun(context.Background(), actor, AbortCloseRunCommand{ResumeCloseRunCommand: abortCommand, Reason: "Reopen the soft-close epoch"})
	if err != nil {
		t.Fatal(err)
	}
	newStart := StartHardCloseCommand{CommandID: uuid.New(), AccountingScopeID: aborted.State.Period.AccountingScopeID, FiscalPeriodID: aborted.State.Period.ID, SoftCloseRunID: aborted.State.SoftClose.ID, ExpectedPeriodVersion: aborted.State.Period.AggregateVersion, ExpectedGateVersion: aborted.State.SoftClose.OpenedGateVersion, SoftCloseControlEpoch: aborted.State.ControlEpoch.Number, SoftClosePolicyVersion: aborted.State.SoftClose.PolicyVersion, IdempotencyKey: "hard-close-after-abort", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	gate.result = hardCloseResult(newStart, uuid.New())
	result, err := service.StartHardClose(context.Background(), actor, newStart)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.CloseRun.ID == started.CloseRun.ID || result.State.CloseRun.Status != CloseRunBarrierAcquired {
		t.Fatalf("new close-run identity = old=%s new=%s state=%s", started.CloseRun.ID, result.State.CloseRun.ID, result.State.CloseRun.Status)
	}
	_ = repository
}

func TestSealRetryPreservesTheOriginalRequestIdentity(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 10, 17, 14, 30, 0, 0, time.UTC) }
	taskStub := &us3TaskStub{}
	sealStub := &us3SealStub{result: AuditSealResult{Status: CloseRunSealFailed, FailureCode: "SEAL_PROVIDER_REJECTED", FailureDetail: "retryable provider rejection"}}
	service, repository, gate, actor, state := startedUS3Service(t, CloseRunDependencies{CloseTasks: taskStub, Seal: sealStub}, clock)
	for _, step := range defaultCloseSteps() {
		done := clock()
		taskStub.results = append(taskStub.results, CloseTaskResult{StepID: step.ID, Status: CloseStepCompleted, DependencyOutcome: "reconciled", ProgressCount: 1, ProgressTotal: 1, Owner: step.Owner, ImmutableEvidenceRef: "evidence:" + step.ID, LedgerWatermark: 25, CompletedAt: &done})
	}
	resumed, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(state, "resume-seal-retry"))
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := service.ApplyCloseApprovalDecision(context.Background(), actor, finalApprovalFixture(resumed.State, "approve-seal-retry"))
	if err != nil {
		t.Fatal(err)
	}
	requestID := finalized.State.CloseRun.SealRequestID
	if requestID == uuid.Nil || finalized.State.CloseRun.SealStatus != CloseRunSealFailed {
		t.Fatalf("initial seal request = %#v", finalized.State.CloseRun)
	}
	retry, err := service.ResumeCloseRun(context.Background(), actor, resumeFixture(finalized.State, "resume-seal-retry-2"))
	if err != nil {
		t.Fatal(err)
	}
	if !retry.Replayed || retry.State.CloseRun.SealRequestID != requestID || sealStub.calls != 2 || gate.finalizeCalls != 1 {
		t.Fatalf("seal retry = replay=%v request=%s calls=%d finalizeCalls=%d", retry.Replayed, retry.State.CloseRun.SealRequestID, sealStub.calls, gate.finalizeCalls)
	}
	_ = repository
}

func startedUS3Service(t *testing.T, dependencies CloseRunDependencies, clock func() time.Time) (*Service, *MemoryRepository, *hardCloseGateStub, Actor, ControlState) {
	t.Helper()
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{result: hardCloseResult(command, uuid.New())}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, dependencies, clock)
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor()
	if _, err := service.StartHardClose(context.Background(), actor, command); err != nil {
		t.Fatal(err)
	}
	current, err := repository.Get(context.Background(), command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, gate, actor, current
}

func resumeFixture(state ControlState, key string) ResumeCloseRunCommand {
	return ResumeCloseRunCommand{CommandID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, SoftCloseRunID: state.SoftClose.ID, CloseRunID: state.CloseRun.ID, ExpectedPeriodVersion: state.Period.AggregateVersion, ExpectedCloseRunVersion: state.CloseRun.AggregateVersion, ExpectedGateVersion: state.CloseRun.BarrierEvidence.GateVersion, SoftCloseControlEpoch: state.ControlEpoch.Number, SoftClosePolicyVersion: state.SoftClose.PolicyVersion, IdempotencyKey: key, CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
}

func finalApprovalFixture(state ControlState, key string) ApplyCloseApprovalDecisionCommand {
	run := *state.CloseRun
	approver := uuid.New()
	return ApplyCloseApprovalDecisionCommand{ResumeCloseRunCommand: ResumeCloseRunCommand{CommandID: uuid.New(), AccountingScopeID: run.AccountingScopeID, FiscalPeriodID: run.FiscalPeriodID, SoftCloseRunID: state.SoftClose.ID, CloseRunID: run.ID, ExpectedPeriodVersion: state.Period.AggregateVersion, ExpectedCloseRunVersion: run.AggregateVersion, ExpectedGateVersion: run.BarrierEvidence.GateVersion, SoftCloseControlEpoch: state.ControlEpoch.Number, SoftClosePolicyVersion: state.SoftClose.PolicyVersion, IdempotencyKey: key, CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}, ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), DecisionVersion: 1, Outcome: CloseApprovalApproved, DecisionReason: "Final checklist complete", PolicyVersion: "close-policy-v1", SubjectVersion: run.AggregateVersion, CandidateFingerprint: run.CommandFingerprint, DecisionReference: uuid.New(), ApproverUserID: approver, SeparationOfDutiesReference: "sod:" + approver.String(), FinalLedgerWatermark: 25, RecordedAt: time.Date(2026, 10, 17, 9, 1, 0, 0, time.UTC)}
}
