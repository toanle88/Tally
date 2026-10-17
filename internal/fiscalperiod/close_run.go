package fiscalperiod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	CloseStepPending   = "Pending"
	CloseStepRunning   = "Running"
	CloseStepCompleted = "Completed"
	CloseStepFailed    = "Failed"
	CloseStepBlocked   = "Blocked"

	CloseExceptionUnresolved = "Unresolved"
	CloseExceptionApproved   = "Approved"
	CloseExceptionRejected   = "Rejected"
	CloseExceptionExpired    = "Expired"

	CloseApprovalApproved = "approved"
	CloseApprovalRejected = "rejected"
)

// CloseStep is the FPM-owned progress record for one mandatory or optional
// close task. The owning capability remains responsible for the task result;
// FPM only retains the immutable evidence and current process projection.
type CloseStep struct {
	ID                   string     `json:"id"`
	Stage                string     `json:"stage"`
	Owner                string     `json:"owner"`
	Mandatory            bool       `json:"mandatory"`
	Status               string     `json:"status"`
	DependencyOutcome    string     `json:"dependencyOutcome,omitempty"`
	ProgressCount        int64      `json:"progressCount"`
	ProgressTotal        int64      `json:"progressTotal"`
	Attempts             int64      `json:"attempts"`
	StartedAt            *time.Time `json:"startedAt,omitempty"`
	CompletedAt          *time.Time `json:"completedAt,omitempty"`
	LastAttemptAt        *time.Time `json:"lastAttemptAt,omitempty"`
	Blocker              string     `json:"blocker,omitempty"`
	ImmutableEvidenceRef string     `json:"immutableEvidenceRef,omitempty"`
}

func (step CloseStep) Validate() error {
	if strings.TrimSpace(step.ID) == "" || strings.TrimSpace(step.Stage) == "" || strings.TrimSpace(step.Owner) == "" {
		return ErrFiscalPeriodPolicyConflict
	}
	if step.Status != CloseStepPending && step.Status != CloseStepRunning && step.Status != CloseStepCompleted && step.Status != CloseStepFailed && step.Status != CloseStepBlocked {
		return ErrFiscalPeriodStateConflict
	}
	if step.ProgressCount < 0 || step.ProgressTotal < 0 || step.Attempts < 0 || (step.ProgressTotal > 0 && step.ProgressCount > step.ProgressTotal) {
		return ErrFiscalPeriodVersionConflict
	}
	if step.Status == CloseStepCompleted && strings.TrimSpace(step.ImmutableEvidenceRef) == "" {
		return ErrFiscalPeriodPolicyConflict
	}
	return nil
}

type CloseException struct {
	ID                uuid.UUID              `json:"id"`
	CloseRunID        uuid.UUID              `json:"closeRunId"`
	Code              string                 `json:"code"`
	Status            string                 `json:"status"`
	ApprovalRequestID uuid.UUID              `json:"approvalRequestId"`
	ApprovalDecision  *CloseApprovalDecision `json:"approvalDecision,omitempty"`
	Blocker           string                 `json:"blocker,omitempty"`
	ExpiresAt         *time.Time             `json:"expiresAt,omitempty"`
}

func (exception CloseException) Validate() error {
	if exception.ID == uuid.Nil || exception.CloseRunID == uuid.Nil || strings.TrimSpace(exception.Code) == "" || exception.ApprovalRequestID == uuid.Nil {
		return ErrFiscalPeriodPolicyConflict
	}
	if exception.Status != CloseExceptionUnresolved && exception.Status != CloseExceptionApproved && exception.Status != CloseExceptionRejected && exception.Status != CloseExceptionExpired {
		return ErrFiscalPeriodStateConflict
	}
	if exception.ApprovalDecision != nil {
		if err := exception.ApprovalDecision.Validate(); err != nil {
			return err
		}
		if exception.ApprovalDecision.RequestID != exception.ApprovalRequestID {
			return ErrFiscalPeriodScopeConflict
		}
	} else if exception.Status == CloseExceptionApproved || exception.Status == CloseExceptionRejected {
		return ErrFiscalPeriodStateConflict
	}
	return nil
}

// CloseApprovalDecision is immutable evidence returned by Workflow and
// applied by FPM. FPM never creates or edits a Workflow decision.
type CloseApprovalDecision struct {
	RequestID                   uuid.UUID `json:"requestId"`
	DecisionID                  uuid.UUID `json:"decisionId"`
	DecisionVersion             int64     `json:"decisionVersion"`
	Outcome                     string    `json:"outcome"`
	Reason                      string    `json:"reason,omitempty"`
	PolicyVersion               string    `json:"policyVersion"`
	SubjectVersion              int64     `json:"subjectVersion"`
	CandidateFingerprint        string    `json:"candidateFingerprint"`
	ApproverUserID              uuid.UUID `json:"approverUserId"`
	DecisionReference           uuid.UUID `json:"decisionReference"`
	SeparationOfDutiesReference string    `json:"separationOfDutiesReference,omitempty"`
	RecordedAt                  time.Time `json:"recordedAt"`
}

func (decision CloseApprovalDecision) Validate() error {
	if decision.RequestID == uuid.Nil || decision.DecisionID == uuid.Nil || decision.DecisionReference == uuid.Nil || decision.ApproverUserID == uuid.Nil || decision.DecisionVersion < 1 || decision.SubjectVersion < 1 || strings.TrimSpace(decision.PolicyVersion) == "" || strings.TrimSpace(decision.CandidateFingerprint) == "" || strings.TrimSpace(decision.Outcome) == "" || decision.RecordedAt.IsZero() {
		return ErrCloseDecisionRejected
	}
	if decision.Outcome != CloseApprovalApproved && decision.Outcome != CloseApprovalRejected {
		return ErrCloseDecisionRejected
	}
	return nil
}

type CloseTaskResult struct {
	StepID               string
	Status               string
	DependencyOutcome    string
	ProgressCount        int64
	ProgressTotal        int64
	Owner                string
	Blocker              string
	ImmutableEvidenceRef string
	LedgerWatermark      int64
	CompletedAt          *time.Time
}

type CloseTaskExecution struct {
	Accepted bool
	Results  []CloseTaskResult
}

type CloseTaskPort interface {
	Execute(context.Context, Actor, CloseRun) (CloseTaskExecution, error)
}

type CloseDecisionPort interface {
	Validate(context.Context, Actor, CloseApprovalDecision) error
}

// ReopenDecisionPort is the FPM-side boundary for immutable Workflow
// approval evidence. Workflow owns the approval request and decision; FPM
// only asks the owning context to validate the reference against the current
// request and candidate fingerprint.
type ReopenDecisionPort interface {
	Validate(context.Context, Actor, ReopenRequest, ReopenApprovalDecision) error
}

type AuditSealResult struct {
	Status         string
	RequestID      uuid.UUID
	ProofReference string
	FailureCode    string
	FailureDetail  string
}

type AuditSealPort interface {
	RequestSeal(context.Context, Actor, CloseRun) (AuditSealResult, error)
}

type CloseRunDependencies struct {
	CloseTasks     CloseTaskPort
	Workflow       CloseDecisionPort
	ReopenWorkflow ReopenDecisionPort
	Seal           AuditSealPort
}

func (dependencies CloseRunDependencies) withDefaults() CloseRunDependencies {
	if dependencies.CloseTasks == nil {
		dependencies.CloseTasks = PendingCloseTaskPort{}
	}
	if dependencies.Workflow == nil {
		dependencies.Workflow = EvidenceCloseDecisionPort{}
	}
	if dependencies.ReopenWorkflow == nil {
		dependencies.ReopenWorkflow = UnavailableReopenDecisionPort{}
	}
	if dependencies.Seal == nil {
		dependencies.Seal = PendingAuditSealPort{}
	}
	return dependencies
}

// PendingCloseTaskPort preserves the asynchronous boundary until the owning
// close-task contexts are available. It reports progress as pending rather
// than treating unavailable downstream work as a successful close.
type PendingCloseTaskPort struct{}

func (PendingCloseTaskPort) Execute(_ context.Context, _ Actor, run CloseRun) (CloseTaskExecution, error) {
	steps := append([]CloseStep(nil), run.Steps...)
	if len(steps) == 0 {
		steps = defaultCloseSteps()
	}
	results := make([]CloseTaskResult, 0, len(steps))
	for _, step := range steps {
		if step.Status == CloseStepCompleted {
			continue
		}
		results = append(results, CloseTaskResult{StepID: step.ID, Status: CloseStepPending, DependencyOutcome: "pending", Owner: step.Owner, ProgressCount: step.ProgressCount, ProgressTotal: step.ProgressTotal, Blocker: "Awaiting owning close-task capability"})
	}
	return CloseTaskExecution{Accepted: true, Results: results}, nil
}

type EvidenceCloseDecisionPort struct{}

func (EvidenceCloseDecisionPort) Validate(_ context.Context, _ Actor, decision CloseApprovalDecision) error {
	return decision.Validate()
}

type PendingAuditSealPort struct{}

func (PendingAuditSealPort) RequestSeal(_ context.Context, _ Actor, run CloseRun) (AuditSealResult, error) {
	requestID := run.SealRequestID
	if requestID == uuid.Nil {
		requestID = uuid.New()
	}
	return AuditSealResult{Status: CloseRunSealPending, RequestID: requestID}, nil
}

type UnavailableCloseTaskPort struct{}

func (UnavailableCloseTaskPort) Execute(context.Context, Actor, CloseRun) (CloseTaskExecution, error) {
	return CloseTaskExecution{}, ErrCloseTaskUnavailable
}

type UnavailableCloseDecisionPort struct{}

func (UnavailableCloseDecisionPort) Validate(context.Context, Actor, CloseApprovalDecision) error {
	return ErrCloseDecisionUnavailable
}

type UnavailableReopenDecisionPort struct{}

func (UnavailableReopenDecisionPort) Validate(context.Context, Actor, ReopenRequest, ReopenApprovalDecision) error {
	return ErrReopenDecisionUnavailable
}

type UnavailableAuditSealPort struct{}

func (UnavailableAuditSealPort) RequestSeal(context.Context, Actor, CloseRun) (AuditSealResult, error) {
	return AuditSealResult{}, ErrCloseSealUnavailable
}

func defaultCloseSteps() []CloseStep {
	return []CloseStep{
		{ID: "fx-revaluation", Stage: "Revaluation", Owner: "FX", Mandatory: true, Status: CloseStepPending},
		{ID: "fixed-assets-depreciation", Stage: "Depreciation", Owner: "FixedAssets", Mandatory: true, Status: CloseStepPending},
		{ID: "revenue-recognition", Stage: "RevenueRecognition", Owner: "Revenue", Mandatory: true, Status: CloseStepPending},
		{ID: "intercompany-controls", Stage: "Intercompany", Owner: "Intercompany", Mandatory: true, Status: CloseStepPending},
		{ID: "reporting-reconciliation", Stage: "ReportingReconciliation", Owner: "Reporting", Mandatory: true, Status: CloseStepPending},
	}
}

type ResumeCloseRunCommand struct {
	CommandID               uuid.UUID
	AccountingScopeID       uuid.UUID
	FiscalPeriodID          uuid.UUID
	SoftCloseRunID          uuid.UUID
	CloseRunID              uuid.UUID
	ExpectedPeriodVersion   int64
	ExpectedCloseRunVersion int64
	ExpectedGateVersion     int64
	SoftCloseControlEpoch   int64
	SoftClosePolicyVersion  string
	IdempotencyKey          string
	CorrelationID           string
	CausationID             string
}

type AbortCloseRunCommand struct {
	ResumeCloseRunCommand
	Reason string
}

type ApplyCloseExceptionApprovalDecisionCommand struct {
	ResumeCloseRunCommand
	ExceptionID                 uuid.UUID
	ApprovalRequestID           uuid.UUID
	DecisionID                  uuid.UUID
	DecisionVersion             int64
	Outcome                     string
	DecisionReason              string
	PolicyVersion               string
	SubjectVersion              int64
	CandidateFingerprint        string
	DecisionReference           uuid.UUID
	ApproverUserID              uuid.UUID
	SeparationOfDutiesReference string
	RecordedAt                  time.Time
}

type ApplyCloseApprovalDecisionCommand struct {
	ResumeCloseRunCommand
	ApprovalRequestID           uuid.UUID
	DecisionID                  uuid.UUID
	DecisionVersion             int64
	Outcome                     string
	DecisionReason              string
	PolicyVersion               string
	SubjectVersion              int64
	CandidateFingerprint        string
	DecisionReference           uuid.UUID
	ApproverUserID              uuid.UUID
	SeparationOfDutiesReference string
	FinalLedgerWatermark        int64
	RecordedAt                  time.Time
}

func (command ResumeCloseRunCommand) canonical() ResumeCloseRunCommand {
	command.SoftClosePolicyVersion = strings.TrimSpace(command.SoftClosePolicyVersion)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command ResumeCloseRunCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil || command.SoftCloseRunID == uuid.Nil || command.CloseRunID == uuid.Nil || command.ExpectedPeriodVersion < 1 || command.ExpectedCloseRunVersion < 1 || command.ExpectedGateVersion < 1 || command.SoftCloseControlEpoch < 1 || command.SoftClosePolicyVersion == "" || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

func (command AbortCloseRunCommand) Validate() error {
	if err := command.ResumeCloseRunCommand.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(command.Reason) == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

func (command ApplyCloseExceptionApprovalDecisionCommand) decision() CloseApprovalDecision {
	policyVersion := strings.TrimSpace(command.PolicyVersion)
	if policyVersion == "" {
		policyVersion = command.SoftClosePolicyVersion
	}
	return CloseApprovalDecision{RequestID: command.ApprovalRequestID, DecisionID: command.DecisionID, DecisionVersion: command.DecisionVersion, Outcome: strings.TrimSpace(command.Outcome), Reason: strings.TrimSpace(command.DecisionReason), PolicyVersion: policyVersion, SubjectVersion: command.SubjectVersion, CandidateFingerprint: strings.TrimSpace(command.CandidateFingerprint), ApproverUserID: command.ApproverUserID, DecisionReference: command.DecisionReference, SeparationOfDutiesReference: strings.TrimSpace(command.SeparationOfDutiesReference), RecordedAt: command.RecordedAt}
}

func (command ApplyCloseApprovalDecisionCommand) decision() CloseApprovalDecision {
	policyVersion := strings.TrimSpace(command.PolicyVersion)
	if policyVersion == "" {
		policyVersion = command.SoftClosePolicyVersion
	}
	return CloseApprovalDecision{RequestID: command.ApprovalRequestID, DecisionID: command.DecisionID, DecisionVersion: command.DecisionVersion, Outcome: strings.TrimSpace(command.Outcome), Reason: strings.TrimSpace(command.DecisionReason), PolicyVersion: policyVersion, SubjectVersion: command.SubjectVersion, CandidateFingerprint: strings.TrimSpace(command.CandidateFingerprint), ApproverUserID: command.ApproverUserID, DecisionReference: command.DecisionReference, SeparationOfDutiesReference: strings.TrimSpace(command.SeparationOfDutiesReference), RecordedAt: command.RecordedAt}
}

func (command ApplyCloseExceptionApprovalDecisionCommand) Validate() error {
	if err := command.ResumeCloseRunCommand.Validate(); err != nil || command.ExceptionID == uuid.Nil {
		return ErrInvalidFiscalPeriodCommand
	}
	if command.ApprovalRequestID == uuid.Nil || command.DecisionID == uuid.Nil || command.DecisionReference == uuid.Nil || command.DecisionVersion < 1 || command.SubjectVersion < 1 || command.ApproverUserID == uuid.Nil || command.RecordedAt.IsZero() || strings.TrimSpace(command.Outcome) == "" || strings.TrimSpace(command.PolicyVersion) == "" || strings.TrimSpace(command.CandidateFingerprint) == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

func (command ApplyCloseApprovalDecisionCommand) Validate() error {
	if err := command.ResumeCloseRunCommand.Validate(); err != nil {
		return err
	}
	if command.ApprovalRequestID == uuid.Nil || command.DecisionID == uuid.Nil || command.DecisionReference == uuid.Nil || command.DecisionVersion < 1 || command.SubjectVersion < 1 || command.ApproverUserID == uuid.Nil || command.RecordedAt.IsZero() || strings.TrimSpace(command.Outcome) == "" || strings.TrimSpace(command.PolicyVersion) == "" || strings.TrimSpace(command.CandidateFingerprint) == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	if command.FinalLedgerWatermark < 0 || (strings.TrimSpace(command.Outcome) == CloseApprovalApproved && command.FinalLedgerWatermark < 1) {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

func (service *Service) ResumeCloseRun(ctx context.Context, actor Actor, command ResumeCloseRunCommand) (CommandResult, error) {
	command = command.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.postingGate == nil || service.closeTasks == nil || service.seal == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	fingerprint := resumeCloseRunFingerprint(command)
	durable, replay, unlock, err := service.beginCloseCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, ResumeCloseRunOperation)
	if err != nil || replay != nil {
		return closeCommandReturn(replay, err)
	}
	if unlock != nil {
		defer unlock()
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionResumeCloseRun, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionResumeCloseRun); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := validateCloseRunCommand(state, command, true); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}

	status, statusErr := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if statusErr != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierUnavailable)
		return CommandResult{}, ErrPostingBarrierUnavailable
	}
	if err := status.Validate(); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierResultConflict)
		return CommandResult{}, ErrPostingBarrierResultConflict
	}
	if status.GateMode == gl.PostingGateHardClosed && status.FinalizedCloseRunID != nil && *status.FinalizedCloseRunID == command.CloseRunID {
		result, applyErr := service.applyAuthoritativeFinalization(ctx, actor, decision, state, closeRunFinalizationResult(status, command), command.CorrelationID, command.CausationID, durable)
		if applyErr != nil {
			return CommandResult{}, applyErr
		}
		service.rememberCommand(actor, command.AccountingScopeID, ResumeCloseRunOperation, command.IdempotencyKey, fingerprint, result)
		return result, nil
	}
	if state.CloseRun.Status == CloseRunFinalizing && state.CloseRun.FinalApproval != nil {
		if state.CloseRun.FinalLedgerWatermark == nil || *state.CloseRun.FinalLedgerWatermark < 1 {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrCloseFinalizationPending)
			return CommandResult{}, ErrCloseFinalizationPending
		}
		if status.GateVersion != command.ExpectedGateVersion || status.ActiveCloseRunID == nil || *status.ActiveCloseRunID != command.CloseRunID || status.GateMode != gl.PostingGateCloseOnly {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierResultConflict)
			return CommandResult{}, ErrPostingBarrierResultConflict
		}
		finalized, finalizeErr := service.postingGate.FinalizePostingGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}, gl.FinalizePostingGateCommand{CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, ExpectedGateVersion: command.ExpectedGateVersion, CloseRunID: command.CloseRunID, FinalLedgerWatermark: *state.CloseRun.FinalLedgerWatermark, IdempotencyKey: command.IdempotencyKey + ":finalize-recovery", CorrelationID: command.CorrelationID, CausationID: command.CausationID})
		if finalizeErr != nil {
			if errors.Is(finalizeErr, gl.ErrPostingBarrierFinalized) {
				reconciled, statusErr := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
				if statusErr != nil || reconciled.FinalizedCloseRunID == nil || *reconciled.FinalizedCloseRunID != command.CloseRunID {
					service.finalizeDurableFailureIfNeeded(ctx, durable, ErrCloseFinalizationPending)
					return CommandResult{}, ErrCloseFinalizationPending
				}
				finalized = closeRunFinalizationResult(reconciled, command)
			} else {
				classified := classifyPostingBarrierError(finalizeErr)
				service.finalizeDurableFailureIfNeeded(ctx, durable, classified)
				return CommandResult{}, classified
			}
		}
		result, applyErr := service.applyAuthoritativeFinalization(ctx, actor, decision, state, finalized, command.CorrelationID, command.CausationID, durable)
		if applyErr != nil {
			return CommandResult{}, applyErr
		}
		service.rememberCommand(actor, command.AccountingScopeID, ResumeCloseRunOperation, command.IdempotencyKey, fingerprint, result)
		return result, nil
	}

	if state.CloseRun.Status == CloseRunAwaitingApproval || state.CloseRun.Status == CloseRunAborting || state.CloseRun.Status == CloseRunAborted || strings.HasPrefix(state.CloseRun.LastFailureCode, "ABORT_RELEASE_") {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodStateConflict)
		return CommandResult{}, ErrFiscalPeriodStateConflict
	}
	if state.CloseRun.Status == CloseRunInitiating || (state.CloseRun.Status == CloseRunFailed && state.CloseRun.BarrierEvidence == nil) {
		if state.Period.State == PeriodStateSoftClosed && status.GateMode == gl.PostingGateSoftClosePolicy {
			if status.ActiveSoftCloseRunID == nil || *status.ActiveSoftCloseRunID != command.SoftCloseRunID || status.ActiveSoftCloseControlEpoch != command.SoftCloseControlEpoch || status.GateVersion != command.ExpectedGateVersion {
				service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
				return CommandResult{}, ErrFiscalPeriodOwnerConflict
			}
			acquired, acquireErr := service.postingGate.AcquirePostingBarrier(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}, gl.AcquirePostingBarrierCommand{
				CommandID: state.CloseRun.BarrierCommandID, AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
				ExpectedGateVersion: command.ExpectedGateVersion, SoftCloseRunID: command.SoftCloseRunID, SoftCloseControlEpoch: command.SoftCloseControlEpoch,
				SoftClosePolicyVersion: command.SoftClosePolicyVersion, CloseRunID: command.CloseRunID, CloseType: state.CloseRun.CloseType,
				IdempotencyKey: state.CloseRun.BarrierIdempotencyKey, CorrelationID: command.CorrelationID, CausationID: command.CausationID,
			})
			if acquireErr != nil {
				classified := classifyPostingBarrierError(acquireErr)
				if recordErr := service.recordCloseRunFailure(ctx, actor, decision, state, command, classified); recordErr != nil {
					service.finalizeDurableFailureIfNeeded(ctx, durable, recordErr)
					return CommandResult{}, recordErr
				}
				service.finalizeDurableFailureIfNeeded(ctx, durable, classified)
				return CommandResult{}, classified
			}
			barrierResult := acquired
			applyCommand := ApplyPostingGateResultCommand{CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, SoftCloseRunID: command.SoftCloseRunID, CloseRunID: command.CloseRunID, ExpectedPeriodVersion: command.ExpectedPeriodVersion, ExpectedGateVersion: command.ExpectedGateVersion, SoftCloseControlEpoch: command.SoftCloseControlEpoch, SoftClosePolicyVersion: command.SoftClosePolicyVersion, Result: barrierResult, IdempotencyKey: command.IdempotencyKey + ":barrier-recovery", CorrelationID: command.CorrelationID, CausationID: command.CausationID}
			applied, applyErr := service.applyPostingGateResult(ctx, actor, decision, applyCommand, false, nil)
			if applyErr != nil {
				service.finalizeDurableFailureIfNeeded(ctx, durable, applyErr)
				return CommandResult{}, applyErr
			}
			state, err = service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
			if err != nil {
				service.finalizeDurableFailureIfNeeded(ctx, durable, err)
				return CommandResult{}, err
			}
			status = applied.GateResult.Gate
		} else if status.GateMode == gl.PostingGateCloseOnly && status.ActiveCloseRunID != nil && *status.ActiveCloseRunID == command.CloseRunID && state.Period.State == PeriodStateSoftClosed {
			barrierResult := gl.SoftCloseGateResult{Operation: gl.PostingBarrierAcquireOperation, Gate: status, ValidationOutcome: "reconciled", NextAction: "continue-close", PolicyReference: "reconciled-from-gl-status", AuthorizationPolicyVersion: command.SoftClosePolicyVersion}
			applyCommand := ApplyPostingGateResultCommand{CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, SoftCloseRunID: command.SoftCloseRunID, CloseRunID: command.CloseRunID, ExpectedPeriodVersion: command.ExpectedPeriodVersion, ExpectedGateVersion: command.ExpectedGateVersion, SoftCloseControlEpoch: command.SoftCloseControlEpoch, SoftClosePolicyVersion: command.SoftClosePolicyVersion, Result: barrierResult, IdempotencyKey: command.IdempotencyKey + ":barrier-recovery", CorrelationID: command.CorrelationID, CausationID: command.CausationID}
			applied, applyErr := service.applyPostingGateResult(ctx, actor, decision, applyCommand, false, nil)
			if applyErr != nil {
				service.finalizeDurableFailureIfNeeded(ctx, durable, applyErr)
				return CommandResult{}, applyErr
			}
			state, err = service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
			if err != nil {
				service.finalizeDurableFailureIfNeeded(ctx, durable, err)
				return CommandResult{}, err
			}
			status = applied.GateResult.Gate
		}
	}
	expectedActiveGateVersion := command.ExpectedGateVersion
	if state.CloseRun.BarrierEvidence != nil {
		expectedActiveGateVersion = state.CloseRun.BarrierEvidence.GateVersion
	}
	if status.ActiveCloseRunID == nil || *status.ActiveCloseRunID != command.CloseRunID || status.GateMode != gl.PostingGateCloseOnly || status.GateVersion != expectedActiveGateVersion {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}

	result, err := service.resumeCloseTasks(ctx, actor, decision, state, command, durable)
	if err != nil {
		if result.State.CloseRun != nil {
			service.rememberCommand(actor, command.AccountingScopeID, ResumeCloseRunOperation, command.IdempotencyKey, fingerprint, result)
			return result, err
		}
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	service.rememberCommand(actor, command.AccountingScopeID, ResumeCloseRunOperation, command.IdempotencyKey, fingerprint, result)
	return result, nil
}

func (service *Service) recordCloseRunFailure(ctx context.Context, actor Actor, decision AuthorizationDecision, before ControlState, command ResumeCloseRunCommand, failure error) error {
	if before.CloseRun == nil {
		return ErrFiscalPeriodStateConflict
	}
	code := fiscalPeriodFailureCode(failure)
	if code == "FISCAL_PERIOD_COMMAND_FAILED" {
		code = barrierErrorCode(failure)
	}
	detail := failure.Error()
	if before.CloseRun.Status == CloseRunFailed && before.CloseRun.LastFailureCode == code && before.CloseRun.LastFailureDetail == detail {
		return nil
	}
	after := cloneControlState(before)
	after.CloseRun.Status = CloseRunFailed
	after.CloseRun.LastFailureCode = code
	after.CloseRun.LastFailureDetail = detail
	after.CloseRun.AggregateVersion++
	after.CloseRun.UpdatedAt = service.clock().UTC()
	fingerprint := resumeCloseRunFingerprint(command)
	return service.repository.Commit(ctx, Mutation{
		Before: before, After: after, Operation: ResumeCloseRunOperation, CommandFingerprint: fingerprint,
		Audit: service.auditRecord(actor, decision, command.AccountingScopeID, command.CloseRunID, ResumeCloseRunOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID),
	})
}

func (service *Service) AbortCloseRun(ctx context.Context, actor Actor, command AbortCloseRunCommand) (CommandResult, error) {
	command.ResumeCloseRunCommand = command.ResumeCloseRunCommand.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.postingGate == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	fingerprint := abortCloseRunFingerprint(command)
	durable, replay, unlock, err := service.beginCloseCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, AbortCloseRunOperation)
	if err != nil || replay != nil {
		return closeCommandReturn(replay, err)
	}
	if unlock != nil {
		defer unlock()
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionAbortCloseRun, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionAbortCloseRun); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := validateCloseRunCommand(state, command.ResumeCloseRunCommand, false); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if state.CloseRun.CloseType != gl.CloseTypeInitialHardClose || state.CloseRun.Status == CloseRunCompleted || state.CloseRun.Status == CloseRunAborted || state.CloseRun.Status == CloseRunFinalizing {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierRejected)
		return CommandResult{}, ErrPostingBarrierRejected
	}
	status, err := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierUnavailable)
		return CommandResult{}, ErrPostingBarrierUnavailable
	}
	if err := status.Validate(); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierResultConflict)
		return CommandResult{}, ErrPostingBarrierResultConflict
	}
	if status.GateMode == gl.PostingGateHardClosed || status.FinalizedCloseRunID != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierRejected)
		return CommandResult{}, ErrPostingBarrierRejected
	}
	if (state.CloseRun.Status == CloseRunAborting || state.CloseRun.Status == CloseRunFailed) && releasedAbortStatusMatches(state, command, status) {
		released := reconciledReleaseResult(status, command.SoftClosePolicyVersion)
		result, completeErr := service.completeAbortedCloseRun(ctx, actor, decision, state, command, released, durable, fingerprint)
		if completeErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, completeErr)
			return CommandResult{}, completeErr
		}
		service.rememberCommand(actor, command.AccountingScopeID, AbortCloseRunOperation, command.IdempotencyKey, fingerprint, result)
		return result, nil
	}
	expectedGateVersion := command.ExpectedGateVersion
	if state.CloseRun.BarrierEvidence != nil {
		if state.CloseRun.BarrierEvidence.GateVersion != command.ExpectedGateVersion {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodVersionConflict)
			return CommandResult{}, ErrFiscalPeriodVersionConflict
		}
		expectedGateVersion = state.CloseRun.BarrierEvidence.GateVersion
	}
	if status.GateVersion != expectedGateVersion || status.ActiveCloseRunID == nil || *status.ActiveCloseRunID != command.CloseRunID || status.GateMode != gl.PostingGateCloseOnly {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}
	if status.ActiveCloseType != gl.CloseTypeInitialHardClose || status.ActiveAdmissionCounters.ClosePostingAdmitted || status.ActiveAdmissionCounters.AdmittedPostingCount > 0 {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierRejected)
		return CommandResult{}, ErrPostingBarrierRejected
	}

	working := cloneControlState(state)
	working.CloseRun.Status = CloseRunAborting
	working.CloseRun.AggregateVersion++
	working.CloseRun.LastFailureCode = "ABORT_REQUESTED"
	working.CloseRun.LastFailureDetail = strings.TrimSpace(command.Reason)
	working.CloseRun.UpdatedAt = service.clock().UTC()
	if err := service.repository.Commit(ctx, Mutation{Before: state, After: working, Operation: AbortCloseRunOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, command.CloseRunID, AbortCloseRunOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID)}); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}

	released, err := service.postingGate.ReleasePostingBarrier(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}, gl.ReleasePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, ExpectedGateVersion: expectedGateVersion, CloseRunID: command.CloseRunID, IdempotencyKey: command.IdempotencyKey + ":release", CorrelationID: command.CorrelationID, CausationID: command.CausationID})
	if err != nil {
		failed := cloneControlState(working)
		failed.CloseRun.Status = CloseRunFailed
		failed.CloseRun.LastFailureCode = "ABORT_RELEASE_" + barrierErrorCode(err)
		failed.CloseRun.LastFailureDetail = err.Error()
		failed.CloseRun.AggregateVersion++
		failed.CloseRun.UpdatedAt = service.clock().UTC()
		if commitErr := service.repository.Commit(ctx, Mutation{Before: working, After: failed, Operation: AbortCloseRunOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, command.CloseRunID, AbortCloseRunOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID)}); commitErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, commitErr)
			return CommandResult{}, commitErr
		}
		classified := classifyPostingBarrierError(err)
		service.finalizeDurableFailureIfNeeded(ctx, durable, classified)
		return CommandResult{}, classified
	}

	result, err := service.completeAbortedCloseRun(ctx, actor, decision, working, command, released, durable, fingerprint)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	service.rememberCommand(actor, command.AccountingScopeID, AbortCloseRunOperation, command.IdempotencyKey, fingerprint, result)
	return result, nil
}

func (service *Service) completeAbortedCloseRun(ctx context.Context, actor Actor, decision AuthorizationDecision, before ControlState, command AbortCloseRunCommand, released gl.SoftCloseGateResult, durable *durableCommand, fingerprint string) (CommandResult, error) {
	if err := validateReleasedAbortResult(before, command, released); err != nil {
		return CommandResult{}, err
	}
	after := cloneControlState(before)
	closeRun := after.CloseRun
	closeRun.Status = CloseRunAborted
	closeRun.LastFailureCode = "ABORTED"
	closeRun.LastFailureDetail = strings.TrimSpace(command.Reason)
	closeRun.ReleaseGateResult = &released
	closeRun.AggregateVersion++
	now := service.clock().UTC()
	closeRun.UpdatedAt = now
	after.Period.State = PeriodStateSoftClosed
	after.Period.StateVersion++
	after.Period.AggregateVersion++
	after.Period.ActiveSoftCloseRunID = cloneUUID(&after.SoftClose.ID)
	after.Period.ActiveCloseRunID = nil
	after.Period.UpdatedAt = now
	after.SoftClose.Status = SoftCloseRunActive
	after.SoftClose.ControlEpoch = released.Gate.ActiveSoftCloseControlEpoch
	after.SoftClose.OpenedGateVersion = released.Gate.GateVersion
	after.SoftClose.AggregateVersion++
	after.SoftClose.UpdatedAt = now
	after.ControlEpoch = SoftCloseControlEpoch{SoftCloseRunID: after.SoftClose.ID, FiscalPeriodID: after.Period.ID, Number: released.Gate.ActiveSoftCloseControlEpoch, OpenedGateVersion: released.Gate.GateVersion, PolicyVersion: after.SoftClose.PolicyVersion, Status: ControlEpochActive, AggregateVersion: 1, CreatedAt: now, UpdatedAt: now}
	payload := closeStatePayload(after, released)
	events := []Event{{EventType: "GateAdmissionSummaryRecorded", AggregateID: closeRun.ID, AggregateVersion: closeRun.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}, {EventType: "CloseRunAborted", AggregateID: closeRun.ID, AggregateVersion: closeRun.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}, {EventType: "SoftCloseResumed", AggregateID: after.SoftClose.ID, AggregateVersion: after.SoftClose.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}, {EventType: "PeriodStateChanged", AggregateID: after.Period.ID, AggregateVersion: after.Period.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}}
	result := CommandResult{State: after, GateResult: released, Decision: decision}
	if err := service.commitCloseMutation(ctx, Mutation{Before: before, After: after, Operation: AbortCloseRunOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, closeRun.ID, AbortCloseRunOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID), Events: events}, durable, result); err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func releasedAbortStatusMatches(state ControlState, command AbortCloseRunCommand, status gl.PostingGateStatus) bool {
	if state.CloseRun == nil || status.AccountingScopeID != command.AccountingScopeID || status.FiscalPeriodID != command.FiscalPeriodID || status.GateMode != gl.PostingGateSoftClosePolicy || status.GateVersion != command.ExpectedGateVersion+1 || status.ActiveCloseRunID != nil || status.ActiveCloseType != "" || status.FinalizedCloseRunID != nil || status.ActiveSoftCloseRunID == nil || *status.ActiveSoftCloseRunID != state.SoftClose.ID || status.ActiveSoftCloseControlEpoch <= state.ControlEpoch.Number || (status.SoftClosePolicyVersion != "" && status.SoftClosePolicyVersion != state.SoftClose.PolicyVersion) {
		return false
	}
	if status.ActiveAdmissionCounters.ControlOwnerType != gl.SoftCloseGateOwnerType || status.ActiveAdmissionCounters.ControlOwnerID != state.SoftClose.ID {
		return false
	}
	if status.ActiveAdmissionCounters.ControlOwnerEpoch != nil && *status.ActiveAdmissionCounters.ControlOwnerEpoch != status.ActiveSoftCloseControlEpoch {
		return false
	}
	return true
}

func reconciledReleaseResult(status gl.PostingGateStatus, policyVersion string) gl.SoftCloseGateResult {
	return gl.SoftCloseGateResult{Operation: gl.PostingBarrierReleaseOperation, Gate: status, PolicyReference: "reconciled-from-gl-status", AuthorizationPolicyVersion: policyVersion, ValidationOutcome: "reconciled", NextAction: "resume-soft-close"}
}

func validateReleasedAbortResult(state ControlState, command AbortCloseRunCommand, released gl.SoftCloseGateResult) error {
	if state.CloseRun == nil || state.CloseRun.ID != command.CloseRunID || released.Operation != gl.PostingBarrierReleaseOperation {
		return ErrPostingBarrierResultConflict
	}
	if err := released.Gate.Validate(); err != nil {
		return ErrPostingBarrierResultConflict
	}
	if released.Gate.AccountingScopeID != command.AccountingScopeID || released.Gate.FiscalPeriodID != command.FiscalPeriodID || released.Gate.GateMode != gl.PostingGateSoftClosePolicy || released.Gate.GateVersion != command.ExpectedGateVersion+1 || released.Gate.ActiveCloseRunID != nil || released.Gate.ActiveCloseType != "" || released.Gate.FinalizedCloseRunID != nil || released.Gate.ActiveSoftCloseRunID == nil || *released.Gate.ActiveSoftCloseRunID != state.SoftClose.ID || released.Gate.ActiveSoftCloseControlEpoch <= state.ControlEpoch.Number || (released.Gate.SoftClosePolicyVersion != "" && released.Gate.SoftClosePolicyVersion != state.SoftClose.PolicyVersion) {
		return ErrPostingBarrierResultConflict
	}
	if released.Gate.ActiveAdmissionCounters.ControlOwnerType != gl.SoftCloseGateOwnerType || released.Gate.ActiveAdmissionCounters.ControlOwnerID != state.SoftClose.ID {
		return ErrPostingBarrierResultConflict
	}
	counters := released.Gate.ActiveAdmissionCounters
	if counters.ClosePostingAdmitted || counters.ReopenPostingAdmitted || counters.AdmittedPostingCount != 0 || counters.FirstAdmittedLedgerPosition != 0 || counters.LastAdmittedLedgerPosition != 0 {
		return ErrPostingBarrierResultConflict
	}
	if err := validateFrozenCloseSummary(released.Gate, state.CloseRun.ID, gl.PostingBarrierReleaseCommandIdentity); err != nil {
		return err
	}
	return nil
}

func (service *Service) ApplyCloseExceptionApprovalDecision(ctx context.Context, actor Actor, command ApplyCloseExceptionApprovalDecisionCommand) (CommandResult, error) {
	command.ResumeCloseRunCommand = command.ResumeCloseRunCommand.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.workflow == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	fingerprint := exceptionApprovalFingerprint(command)
	durable, replay, unlock, err := service.beginCloseCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, ApplyCloseExceptionApprovalOperation)
	if err != nil || replay != nil {
		return closeCommandReturn(replay, err)
	}
	if unlock != nil {
		defer unlock()
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionApplyCloseExceptionApproval, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionApplyCloseExceptionApproval); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := validateCloseRunCommand(state, command.ResumeCloseRunCommand, false); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if state.CloseRun.Status == CloseRunCompleted || state.CloseRun.Status == CloseRunAborted || state.CloseRun.Status == CloseRunFinalizing || state.Period.State == PeriodStateHardClosed {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodStateConflict)
		return CommandResult{}, ErrFiscalPeriodStateConflict
	}
	approval := command.decision()
	if err := service.workflow.Validate(ctx, actor, approval); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		if errors.Is(err, ErrCloseDecisionUnavailable) {
			return CommandResult{}, ErrCloseDecisionUnavailable
		}
		return CommandResult{}, ErrCloseDecisionRejected
	}
	if approval.CandidateFingerprint != state.CloseRun.CommandFingerprint || approval.SubjectVersion != state.CloseRun.AggregateVersion || (state.CloseRun.ApprovalPolicyVersion != "" && approval.PolicyVersion != state.CloseRun.ApprovalPolicyVersion) {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrCloseDecisionRejected)
		return CommandResult{}, ErrCloseDecisionRejected
	}
	if strings.TrimSpace(approval.SeparationOfDutiesReference) == "" || approval.ApproverUserID == actor.UserID {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrCloseDecisionRejected)
		return CommandResult{}, ErrCloseDecisionRejected
	}
	working := cloneControlState(state)
	var target *CloseException
	for index := range working.CloseRun.Exceptions {
		if working.CloseRun.Exceptions[index].ID == command.ExceptionID {
			target = &working.CloseRun.Exceptions[index]
			break
		}
	}
	if target == nil || target.CloseRunID != command.CloseRunID || target.ApprovalRequestID != command.ApprovalRequestID {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}
	if target.Status == CloseExceptionExpired || (target.ExpiresAt != nil && !service.clock().UTC().Before(target.ExpiresAt.UTC())) {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrCloseDecisionRejected)
		return CommandResult{}, ErrCloseDecisionRejected
	}
	if target.ApprovalDecision != nil {
		if sameCloseApprovalDecision(*target.ApprovalDecision, approval) {
			result := CommandResult{State: state, Decision: decision, Replayed: true}
			if err := service.finalizeDurableResult(ctx, durable, result); err != nil {
				return CommandResult{}, err
			}
			return result, nil
		}
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodIdempotencyConflict)
		return CommandResult{}, ErrFiscalPeriodIdempotencyConflict
	}
	target.ApprovalDecision = &approval
	if working.CloseRun.ApprovalPolicyVersion == "" {
		working.CloseRun.ApprovalPolicyVersion = approval.PolicyVersion
	}
	appendCloseApprovalDecision(working.CloseRun, approval)
	if approval.Outcome == CloseApprovalApproved {
		target.Status = CloseExceptionApproved
		target.Blocker = "Approved exception retained as close evidence"
	} else {
		target.Status = CloseExceptionRejected
		target.Blocker = "Workflow decision rejected the exception"
	}
	working.CloseRun.AggregateVersion++
	working.CloseRun.UpdatedAt = service.clock().UTC()
	working.CloseRun.LastFailureCode = ""
	working.CloseRun.LastFailureDetail = ""
	result := CommandResult{State: working, Decision: decision}
	if err := service.commitCloseMutation(ctx, Mutation{Before: state, After: working, Operation: ApplyCloseExceptionApprovalOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, command.CloseRunID, ApplyCloseExceptionApprovalOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID)}, durable, result); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	service.rememberCommand(actor, command.AccountingScopeID, ApplyCloseExceptionApprovalOperation, command.IdempotencyKey, fingerprint, result)
	return result, nil
}

func (service *Service) ApplyCloseApprovalDecision(ctx context.Context, actor Actor, command ApplyCloseApprovalDecisionCommand) (CommandResult, error) {
	command.ResumeCloseRunCommand = command.ResumeCloseRunCommand.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.workflow == nil || service.postingGate == nil || service.seal == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	fingerprint := finalApprovalFingerprint(command)
	durable, replay, unlock, err := service.beginCloseCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, ApplyCloseApprovalOperation)
	if err != nil || replay != nil {
		return closeCommandReturn(replay, err)
	}
	if unlock != nil {
		defer unlock()
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionApplyCloseApproval, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionApplyCloseApproval); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := validateCloseRunCommand(state, command.ResumeCloseRunCommand, false); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if state.CloseRun.Status != CloseRunAwaitingApproval {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodStateConflict)
		return CommandResult{}, ErrFiscalPeriodStateConflict
	}
	approval := command.decision()
	if err := service.workflow.Validate(ctx, actor, approval); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		if errors.Is(err, ErrCloseDecisionUnavailable) {
			return CommandResult{}, ErrCloseDecisionUnavailable
		}
		return CommandResult{}, ErrCloseDecisionRejected
	}
	if approval.CandidateFingerprint != state.CloseRun.CommandFingerprint || approval.SubjectVersion != state.CloseRun.AggregateVersion || (state.CloseRun.ApprovalPolicyVersion != "" && approval.PolicyVersion != state.CloseRun.ApprovalPolicyVersion) {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrCloseDecisionRejected)
		return CommandResult{}, ErrCloseDecisionRejected
	}
	if strings.TrimSpace(approval.SeparationOfDutiesReference) == "" || approval.ApproverUserID == actor.UserID {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrCloseDecisionRejected)
		return CommandResult{}, ErrCloseDecisionRejected
	}
	if approval.Outcome != CloseApprovalApproved {
		working := cloneControlState(state)
		working.CloseRun.FinalApproval = &approval
		working.CloseRun.ApprovalPolicyVersion = approval.PolicyVersion
		appendCloseApprovalDecision(working.CloseRun, approval)
		working.CloseRun.Status = CloseRunFailed
		working.CloseRun.LastFailureCode = "FINAL_APPROVAL_REJECTED"
		working.CloseRun.LastFailureDetail = approval.Reason
		working.CloseRun.AggregateVersion++
		working.CloseRun.UpdatedAt = service.clock().UTC()
		result := CommandResult{State: working, Decision: decision}
		if err := service.commitCloseMutation(ctx, Mutation{Before: state, After: working, Operation: ApplyCloseApprovalOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, command.CloseRunID, ApplyCloseApprovalOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID)}, durable, result); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		service.rememberCommand(actor, command.AccountingScopeID, ApplyCloseApprovalOperation, command.IdempotencyKey, fingerprint, result)
		return result, nil
	}
	if err := validateFinalApprovalReadiness(state, command, service.clock().UTC()); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	status, err := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierUnavailable)
		return CommandResult{}, ErrPostingBarrierUnavailable
	}
	if status.GateVersion != command.ExpectedGateVersion || status.ActiveCloseRunID == nil || *status.ActiveCloseRunID != command.CloseRunID || status.GateMode != gl.PostingGateCloseOnly || state.CloseRun.BarrierEvidence == nil || status.GateVersion != state.CloseRun.BarrierEvidence.GateVersion {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierResultConflict)
		return CommandResult{}, ErrPostingBarrierResultConflict
	}
	working := cloneControlState(state)
	working.CloseRun.FinalApproval = &approval
	appendCloseApprovalDecision(working.CloseRun, approval)
	working.CloseRun.ApprovalPolicyVersion = approval.PolicyVersion
	working.CloseRun.Status = CloseRunFinalizing
	working.CloseRun.AggregateVersion++
	working.CloseRun.FinalLedgerWatermark = cloneInt64(&command.FinalLedgerWatermark)
	working.CloseRun.UpdatedAt = service.clock().UTC()
	if err := service.repository.Commit(ctx, Mutation{Before: state, After: working, Operation: ApplyCloseApprovalOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, command.CloseRunID, ApplyCloseApprovalOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID)}); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}

	finalized, err := service.postingGate.FinalizePostingGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}, gl.FinalizePostingGateCommand{CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, ExpectedGateVersion: command.ExpectedGateVersion, CloseRunID: command.CloseRunID, FinalLedgerWatermark: command.FinalLedgerWatermark, IdempotencyKey: command.IdempotencyKey + ":finalize", CorrelationID: command.CorrelationID, CausationID: command.CausationID})
	if err != nil {
		if errors.Is(err, gl.ErrPostingBarrierFinalized) {
			status, statusErr := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
			if statusErr != nil || status.FinalizedCloseRunID == nil || *status.FinalizedCloseRunID != command.CloseRunID {
				service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierUnavailable)
				return CommandResult{}, ErrPostingBarrierUnavailable
			}
			finalized = closeRunFinalizationResult(status, command.ResumeCloseRunCommand)
		} else {
			service.finalizeDurableFailureIfNeeded(ctx, durable, classifyPostingBarrierError(err))
			return CommandResult{}, classifyPostingBarrierError(err)
		}
	}
	result, err := service.applyAuthoritativeFinalization(ctx, actor, decision, working, finalized, command.CorrelationID, command.CausationID, durable)
	if err != nil {
		return CommandResult{}, err
	}
	service.rememberCommand(actor, command.AccountingScopeID, ApplyCloseApprovalOperation, command.IdempotencyKey, fingerprint, result)
	return result, nil
}

func (service *Service) resumeCloseTasks(ctx context.Context, actor Actor, decision AuthorizationDecision, state ControlState, command ResumeCloseRunCommand, durable *durableCommand) (CommandResult, error) {
	before := state
	working := cloneControlState(state)
	working.CloseRun.Status = CloseRunInProgress
	working.CloseRun.LastFailureCode = ""
	working.CloseRun.LastFailureDetail = ""
	if len(working.CloseRun.Steps) == 0 {
		working.CloseRun.Steps = defaultCloseSteps()
	}
	for index := range working.CloseRun.Steps {
		if working.CloseRun.Steps[index].Status == CloseStepPending {
			now := service.clock().UTC()
			working.CloseRun.Steps[index].Status = CloseStepRunning
			working.CloseRun.Steps[index].StartedAt = &now
			working.CloseRun.Steps[index].LastAttemptAt = &now
			working.CloseRun.Steps[index].Attempts++
		}
	}
	execution, err := service.closeTasks.Execute(ctx, actor, *working.CloseRun)
	if err != nil {
		classified := classifyCloseTaskError(err)
		if errors.Is(err, ErrCloseTaskUnavailable) {
			// Keep the durable command retryable and leave the current close-run
			// projection unchanged while the owning capability is unavailable.
			return CommandResult{}, classified
		}
		working.CloseRun.LastFailureCode = "CLOSE_TASK_FAILED"
		result, commitErr := service.commitCloseTaskFailure(ctx, actor, decision, before, working, command, durable, classified)
		if commitErr != nil {
			return CommandResult{}, commitErr
		}
		return result, classified
	}
	if !execution.Accepted {
		failure := ErrCloseTaskFailed
		working.CloseRun.LastFailureCode = "CLOSE_TASK_NOT_ACCEPTED"
		working.CloseRun.LastFailureDetail = "The close-task coordinator did not accept the run."
		result, commitErr := service.commitCloseTaskFailure(ctx, actor, decision, before, working, command, durable, failure)
		if commitErr != nil {
			return CommandResult{}, commitErr
		}
		return result, failure
	}
	completedWatermark := int64(0)
	if working.CloseRun.FinalLedgerWatermark != nil {
		completedWatermark = *working.CloseRun.FinalLedgerWatermark
	}
	now := service.clock().UTC()
	completedStepIDs := make([]string, 0)
	seenStepIDs := make(map[string]struct{}, len(execution.Results))
	var validationFailure error
	for _, outcome := range execution.Results {
		if _, exists := seenStepIDs[outcome.StepID]; exists {
			validationFailure = fmt.Errorf("%w: duplicate result for close step %q", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		seenStepIDs[outcome.StepID] = struct{}{}
		stepIndex := -1
		for index := range working.CloseRun.Steps {
			if working.CloseRun.Steps[index].ID == outcome.StepID {
				stepIndex = index
				break
			}
		}
		if stepIndex < 0 {
			validationFailure = fmt.Errorf("%w: unknown close step %q", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		step := &working.CloseRun.Steps[stepIndex]
		status := outcome.Status
		if status == "" {
			status = step.Status
		}
		if status != CloseStepPending && status != CloseStepRunning && status != CloseStepCompleted && status != CloseStepFailed && status != CloseStepBlocked {
			validationFailure = fmt.Errorf("%w: unsupported status %q for close step %q", ErrCloseTaskFailed, status, outcome.StepID)
			break
		}
		if step.Status == CloseStepCompleted && status != CloseStepCompleted {
			validationFailure = fmt.Errorf("%w: completed close step %q cannot regress", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		if step.Status == CloseStepCompleted && outcome.ImmutableEvidenceRef != "" && outcome.ImmutableEvidenceRef != step.ImmutableEvidenceRef {
			validationFailure = fmt.Errorf("%w: completed close step %q changed immutable evidence", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		if outcome.Owner != "" && outcome.Owner != step.Owner {
			validationFailure = fmt.Errorf("%w: close step %q changed owner", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		if outcome.ProgressCount < 0 || outcome.ProgressTotal < 0 || outcome.ProgressCount > outcome.ProgressTotal || outcome.ProgressCount < step.ProgressCount {
			validationFailure = fmt.Errorf("%w: invalid progress for close step %q", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		progressTotal := outcome.ProgressTotal
		if progressTotal == 0 && step.ProgressTotal > 0 {
			progressTotal = step.ProgressTotal
		}
		if progressTotal > 0 && outcome.ProgressCount > progressTotal {
			validationFailure = fmt.Errorf("%w: invalid progress total for close step %q", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		if outcome.LedgerWatermark < 0 {
			validationFailure = fmt.Errorf("%w: invalid ledger watermark for close step %q", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		if status == CloseStepCompleted && strings.TrimSpace(outcome.ImmutableEvidenceRef) == "" && strings.TrimSpace(step.ImmutableEvidenceRef) == "" {
			validationFailure = fmt.Errorf("%w: completed close step %q is missing immutable evidence", ErrCloseTaskFailed, outcome.StepID)
			break
		}
		wasCompleted := step.Status == CloseStepCompleted
		step.Status = status
		if outcome.DependencyOutcome != "" {
			step.DependencyOutcome = outcome.DependencyOutcome
		}
		step.ProgressCount = outcome.ProgressCount
		step.ProgressTotal = progressTotal
		step.Blocker = outcome.Blocker
		if outcome.ImmutableEvidenceRef != "" {
			step.ImmutableEvidenceRef = outcome.ImmutableEvidenceRef
		}
		step.LastAttemptAt = &now
		if outcome.CompletedAt != nil {
			step.CompletedAt = outcome.CompletedAt
		} else if step.Status == CloseStepCompleted {
			step.CompletedAt = &now
		}
		if step.Status == CloseStepCompleted && !wasCompleted {
			completedStepIDs = append(completedStepIDs, step.ID)
		}
		if outcome.LedgerWatermark > completedWatermark {
			completedWatermark = outcome.LedgerWatermark
		}
	}
	if validationFailure != nil {
		working.CloseRun.LastFailureCode = "CLOSE_TASK_FAILED"
		result, commitErr := service.commitCloseTaskFailure(ctx, actor, decision, before, working, command, durable, validationFailure)
		if commitErr != nil {
			return CommandResult{}, commitErr
		}
		return result, validationFailure
	}
	if completedWatermark > 0 {
		working.CloseRun.FinalLedgerWatermark = &completedWatermark
	}
	if anyMandatoryStepFailed(*working.CloseRun) {
		working.CloseRun.Status = CloseRunFailed
		working.CloseRun.LastFailureCode = "CLOSE_TASK_FAILED"
		working.CloseRun.LastFailureDetail = "A mandatory close step failed or is blocked."
	} else if allMandatoryStepsComplete(*working.CloseRun) {
		working.CloseRun.Status = CloseRunAwaitingApproval
	} else {
		working.CloseRun.Status = CloseRunInProgress
	}
	working.CloseRun.AggregateVersion++
	working.CloseRun.UpdatedAt = now
	payload := closeStatePayload(working, gl.SoftCloseGateResult{})
	events := []Event{{EventType: "CloseRunResumed", AggregateID: working.CloseRun.ID, AggregateVersion: working.CloseRun.AggregateVersion, AccountingScopeID: working.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}}
	for range completedStepIDs {
		stepPayload := closeStatePayload(working, gl.SoftCloseGateResult{})
		events = append(events, Event{EventType: "CloseStepCompleted", AggregateID: working.CloseRun.ID, AggregateVersion: working.CloseRun.AggregateVersion, AccountingScopeID: working.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: stepPayload})
	}
	if working.CloseRun.Status == CloseRunFailed {
		failure := ErrCloseTaskFailed
		result, commitErr := service.commitCloseTaskFailure(ctx, actor, decision, before, working, command, durable, failure, events...)
		if commitErr != nil {
			return CommandResult{}, commitErr
		}
		return result, failure
	}
	result := CommandResult{State: working, Decision: decision}
	if err := service.commitCloseMutation(ctx, Mutation{Before: before, After: working, Operation: ResumeCloseRunOperation, CommandFingerprint: resumeCloseRunFingerprint(command), Audit: service.auditRecord(actor, decision, command.AccountingScopeID, working.CloseRun.ID, ResumeCloseRunOperation, resumeCloseRunFingerprint(command), resumeCloseRunFingerprint(command), command.CorrelationID, command.CausationID), Events: events}, durable, result); err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func classifyCloseTaskError(err error) error {
	if errors.Is(err, ErrCloseTaskUnavailable) {
		return fmt.Errorf("%w: %v", ErrCloseTaskUnavailable, err)
	}
	if errors.Is(err, ErrCloseTaskFailed) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrCloseTaskFailed, err)
}

func (service *Service) commitCloseTaskFailure(ctx context.Context, actor Actor, decision AuthorizationDecision, before, after ControlState, command ResumeCloseRunCommand, durable *durableCommand, failure error, events ...Event) (CommandResult, error) {
	if after.CloseRun == nil {
		return CommandResult{}, ErrFiscalPeriodStateConflict
	}
	now := service.clock().UTC()
	after.CloseRun.Status = CloseRunFailed
	if after.CloseRun.LastFailureCode == "" {
		after.CloseRun.LastFailureCode = fiscalPeriodFailureCode(failure)
	}
	if after.CloseRun.LastFailureDetail == "" {
		after.CloseRun.LastFailureDetail = failure.Error()
	}
	for index := range after.CloseRun.Steps {
		step := &after.CloseRun.Steps[index]
		if step.Status == CloseStepPending || step.Status == CloseStepRunning {
			step.Status = CloseStepFailed
			if strings.TrimSpace(step.Blocker) == "" {
				step.Blocker = after.CloseRun.LastFailureDetail
			}
			step.LastAttemptAt = &now
		}
	}
	after.CloseRun.AggregateVersion++
	after.CloseRun.UpdatedAt = now
	if after.CloseRun.Status == CloseRunFailed {
		payload := closeStatePayload(after, gl.SoftCloseGateResult{})
		events = append(events, Event{EventType: "CloseRunFailed", AggregateID: after.CloseRun.ID, AggregateVersion: after.CloseRun.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload})
	} else if len(events) == 0 {
		payload := closeStatePayload(after, gl.SoftCloseGateResult{})
		events = []Event{{EventType: "CloseRunFailed", AggregateID: after.CloseRun.ID, AggregateVersion: after.CloseRun.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}}
	}
	fingerprint := resumeCloseRunFingerprint(command)
	result := CommandResult{State: after, Decision: decision}
	mutation := Mutation{Before: before, After: after, Operation: ResumeCloseRunOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, after.CloseRun.ID, ResumeCloseRunOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID), Events: events}
	if err := service.commitCloseMutation(ctx, mutation, durable, result); err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func anyMandatoryStepFailed(run CloseRun) bool {
	for _, step := range run.Steps {
		if step.Mandatory && (step.Status == CloseStepFailed || step.Status == CloseStepBlocked) {
			return true
		}
	}
	return false
}

func (service *Service) applyAuthoritativeFinalization(ctx context.Context, actor Actor, decision AuthorizationDecision, state ControlState, finalized gl.SoftCloseGateResult, correlationID, causationID string, durable *durableCommand) (CommandResult, error) {
	if err := validateFinalizationResult(state, finalized); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if state.Period.State == PeriodStateHardClosed && state.CloseRun.Status == CloseRunCompleted {
		if !sameFinalizationFacts(state.CloseRun.FinalizationGateResult, finalized) {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierResultConflict)
			return CommandResult{}, ErrPostingBarrierResultConflict
		}
		before := state
		after := cloneControlState(state)
		after.CloseRun.FinalLedgerWatermark = cloneInt64(finalized.Gate.FinalLedgerWatermark)
		sealResult, sealErr := service.requestAuditSeal(ctx, actor, *after.CloseRun)
		if sealErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, sealErr)
			return CommandResult{}, sealErr
		}
		if sealErr := applySealResult(after.CloseRun, sealResult); sealErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, sealErr)
			return CommandResult{}, sealErr
		}
		if sameSealState(before.CloseRun, after.CloseRun) {
			result := CommandResult{State: after, GateResult: finalized, Decision: decision, Replayed: true}
			if durable != nil {
				if err := service.finalizeDurableResult(ctx, durable, result); err != nil {
					return CommandResult{}, err
				}
			}
			return result, nil
		}
		after.CloseRun.AggregateVersion++
		after.CloseRun.UpdatedAt = service.clock().UTC()
		result := CommandResult{State: after, GateResult: finalized, Decision: decision}
		if err := service.commitCloseMutation(ctx, Mutation{Before: before, After: after, Operation: ApplyPostingGateResultOperation, CommandFingerprint: postingGateResultFingerprint(finalized), Audit: service.auditRecord(actor, decision, after.Period.AccountingScopeID, after.CloseRun.ID, ApplyPostingGateResultOperation, postingGateResultFingerprint(finalized), postingGateResultFingerprint(finalized), correlationID, causationID)}, durable, result); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		return result, nil
	}
	before := state
	after := cloneControlState(state)
	now := service.clock().UTC()
	after.Period.State = PeriodStateHardClosed
	after.Period.StateVersion++
	after.Period.AggregateVersion++
	after.Period.ActiveSoftCloseRunID = nil
	after.Period.ActiveCloseRunID = nil
	after.Period.UpdatedAt = now
	after.SoftClose.Status = SoftCloseRunSuperseded
	after.SoftClose.AggregateVersion++
	after.SoftClose.UpdatedAt = now
	after.ControlEpoch.Status = ControlEpochHandoffPending
	after.ControlEpoch.AggregateVersion++
	after.ControlEpoch.UpdatedAt = now
	after.CloseRun.Status = CloseRunCompleted
	after.CloseRun.FinalizationGateResult = &finalized
	if finalized.Gate.FinalLedgerWatermark != nil {
		after.CloseRun.FinalLedgerWatermark = cloneInt64(finalized.Gate.FinalLedgerWatermark)
	}
	after.CloseRun.AggregateVersion++
	after.CloseRun.LastFailureCode = ""
	after.CloseRun.LastFailureDetail = ""
	after.CloseRun.UpdatedAt = now
	sealResult, sealErr := service.requestAuditSeal(ctx, actor, *after.CloseRun)
	if sealErr != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, sealErr)
		return CommandResult{}, sealErr
	}
	if sealErr := applySealResult(after.CloseRun, sealResult); sealErr != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, sealErr)
		return CommandResult{}, sealErr
	}
	payload := closeStatePayload(after, finalized)
	events := []Event{{EventType: "GateAdmissionSummaryRecorded", AggregateID: after.CloseRun.ID, AggregateVersion: after.CloseRun.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: correlationID, CausationID: causationID, OccurredAt: now, Payload: payload}, {EventType: "SoftCloseSuperseded", AggregateID: after.SoftClose.ID, AggregateVersion: after.SoftClose.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: correlationID, CausationID: causationID, OccurredAt: now, Payload: payload}, {EventType: "PeriodStateChanged", AggregateID: after.Period.ID, AggregateVersion: after.Period.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: correlationID, CausationID: causationID, OccurredAt: now, Payload: payload}}
	result := CommandResult{State: after, GateResult: finalized, Decision: decision}
	if err := service.commitCloseMutation(ctx, Mutation{Before: before, After: after, Operation: ApplyPostingGateResultOperation, CommandFingerprint: postingGateResultFingerprint(finalized), Audit: service.auditRecord(actor, decision, after.Period.AccountingScopeID, after.CloseRun.ID, ApplyPostingGateResultOperation, postingGateResultFingerprint(finalized), postingGateResultFingerprint(finalized), correlationID, causationID), Events: events}, durable, result); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	return result, nil
}

func validateFinalizationResult(state ControlState, finalized gl.SoftCloseGateResult) error {
	if state.CloseRun == nil || state.CloseRun.BarrierEvidence == nil {
		return ErrPostingBarrierResultConflict
	}
	if err := finalized.Gate.Validate(); err != nil {
		return ErrPostingBarrierResultConflict
	}
	if finalized.Operation != gl.PostingBarrierFinalizeOperation {
		return ErrPostingBarrierResultConflict
	}
	if finalized.Gate.AccountingScopeID != state.Period.AccountingScopeID || finalized.Gate.FiscalPeriodID != state.Period.ID || finalized.Gate.GateMode != gl.PostingGateHardClosed || finalized.Gate.FinalizedCloseRunID == nil || *finalized.Gate.FinalizedCloseRunID != state.CloseRun.ID || finalized.Gate.FinalLedgerWatermark == nil || *finalized.Gate.FinalLedgerWatermark < state.CloseRun.BarrierEvidence.BarrierLedgerPosition || finalized.Gate.ActiveCloseRunID != nil || finalized.Gate.ActiveCloseType != "" || finalized.Gate.ActiveSoftCloseRunID != nil || finalized.Gate.ActiveAdmissionCounters != (gl.ActiveGateAdmissionCounters{}) || strings.TrimSpace(finalized.Gate.FinalizationCommandFingerprint) == "" {
		return ErrPostingBarrierResultConflict
	}
	if err := validateFrozenCloseSummary(finalized.Gate, state.CloseRun.ID, gl.PostingBarrierFinalizeCommandIdentity); err != nil {
		return err
	}
	if state.CloseRun.FinalLedgerWatermark != nil && *state.CloseRun.FinalLedgerWatermark != *finalized.Gate.FinalLedgerWatermark {
		return ErrPostingBarrierResultConflict
	}
	return nil
}

func (service *Service) requestAuditSeal(ctx context.Context, actor Actor, run CloseRun) (AuditSealResult, error) {
	if run.SealStatus == CloseRunSealSealed {
		return AuditSealResult{Status: CloseRunSealSealed, RequestID: run.SealRequestID, ProofReference: run.SealProofReference}, nil
	}
	result, err := service.seal.RequestSeal(ctx, actor, run)
	if err != nil {
		result.Status = CloseRunSealFailed
		result.FailureCode = "AUDIT_SEAL_UNAVAILABLE"
		result.FailureDetail = err.Error()
	}
	if result.Status == "" {
		result.Status = CloseRunSealPending
	}
	if result.Status != CloseRunSealPending && result.Status != CloseRunSealSealed && result.Status != CloseRunSealFailed {
		result.Status = CloseRunSealFailed
		result.FailureCode = "AUDIT_SEAL_INVALID_STATUS"
		result.FailureDetail = "The audit-seal provider returned an unsupported status."
	}
	if run.SealRequestID != uuid.Nil {
		if result.RequestID != uuid.Nil && result.RequestID != run.SealRequestID {
			return AuditSealResult{}, ErrFiscalPeriodIdempotencyConflict
		}
		result.RequestID = run.SealRequestID
	} else if result.RequestID == uuid.Nil {
		result.RequestID = uuid.New()
	}
	if result.Status == CloseRunSealSealed && strings.TrimSpace(result.ProofReference) == "" {
		result.Status = CloseRunSealFailed
		result.FailureCode = "AUDIT_SEAL_PROOF_MISSING"
		result.FailureDetail = "The audit-seal provider did not return immutable proof."
	}
	if result.Status == CloseRunSealFailed && strings.TrimSpace(result.FailureCode) == "" {
		result.FailureCode = "AUDIT_SEAL_FAILED"
	}
	return result, nil
}

func applySealResult(run *CloseRun, result AuditSealResult) error {
	if run == nil {
		return ErrFiscalPeriodStateConflict
	}
	if result.Status == "" {
		result.Status = CloseRunSealPending
	}
	if result.Status != CloseRunSealPending && result.Status != CloseRunSealSealed && result.Status != CloseRunSealFailed {
		return ErrCloseSealFailed
	}
	if run.SealRequestID != uuid.Nil && result.RequestID != uuid.Nil && run.SealRequestID != result.RequestID {
		return ErrFiscalPeriodIdempotencyConflict
	}
	if result.RequestID == uuid.Nil {
		if run.SealRequestID != uuid.Nil {
			result.RequestID = run.SealRequestID
		} else {
			result.RequestID = uuid.New()
		}
	}
	if run.SealStatus == CloseRunSealSealed {
		if result.Status == CloseRunSealSealed && result.ProofReference != "" && result.ProofReference != run.SealProofReference {
			return ErrFiscalPeriodIdempotencyConflict
		}
		return nil
	}
	run.SealStatus = result.Status
	run.SealRequestID = result.RequestID
	if result.ProofReference != "" {
		run.SealProofReference = result.ProofReference
	}
	run.SealFailureCode = result.FailureCode
	run.SealFailureDetail = result.FailureDetail
	return nil
}

func sameFinalizationFacts(existing *gl.SoftCloseGateResult, current gl.SoftCloseGateResult) bool {
	if existing == nil || existing.Gate.GateMode != current.Gate.GateMode || existing.Gate.GateVersion != current.Gate.GateVersion || existing.Gate.AccountingScopeID != current.Gate.AccountingScopeID || existing.Gate.FiscalPeriodID != current.Gate.FiscalPeriodID {
		return false
	}
	if existing.Gate.FinalizedCloseRunID == nil || current.Gate.FinalizedCloseRunID == nil || *existing.Gate.FinalizedCloseRunID != *current.Gate.FinalizedCloseRunID {
		return false
	}
	if existing.Gate.FinalLedgerWatermark == nil || current.Gate.FinalLedgerWatermark == nil || *existing.Gate.FinalLedgerWatermark != *current.Gate.FinalLedgerWatermark {
		return false
	}
	return fingerprint(existing.Gate) == fingerprint(current.Gate)
}

func validateFrozenCloseSummary(gate gl.PeriodPostingGate, closeRunID uuid.UUID, commandType string) error {
	summary := gate.LastFrozenAdmissionSummary
	if summary == nil || summary.ControlOwnerType != gl.PostingBarrierOwnerType || summary.ControlOwnerID != closeRunID || summary.ControlOwnerEpoch != nil || summary.FrozenAtGateVersion != gate.GateVersion || summary.FrozenByCommandType != commandType || strings.TrimSpace(summary.FrozenByCommandFingerprint) == "" {
		return ErrPostingBarrierResultConflict
	}
	return nil
}

func sameSealState(left, right *CloseRun) bool {
	return left != nil && right != nil && left.SealStatus == right.SealStatus && left.SealRequestID == right.SealRequestID && left.SealProofReference == right.SealProofReference && left.SealFailureCode == right.SealFailureCode && left.SealFailureDetail == right.SealFailureDetail
}

func validateCloseRunCommand(state ControlState, command ResumeCloseRunCommand, allowHardClosed bool) error {
	if state.Period.AccountingScopeID != command.AccountingScopeID || state.Period.ID != command.FiscalPeriodID || state.SoftClose.ID != command.SoftCloseRunID || state.CloseRun == nil || state.CloseRun.ID != command.CloseRunID {
		return ErrFiscalPeriodScopeConflict
	}
	if !allowHardClosed && state.Period.State == PeriodStateHardClosed {
		return ErrFiscalPeriodStateConflict
	}
	if state.Period.AggregateVersion != command.ExpectedPeriodVersion || state.CloseRun.AggregateVersion != command.ExpectedCloseRunVersion {
		return ErrFiscalPeriodVersionConflict
	}
	if state.CloseRun.BarrierEvidence != nil && state.CloseRun.BarrierEvidence.GateVersion != command.ExpectedGateVersion {
		return ErrFiscalPeriodVersionConflict
	}
	if state.SoftClose.ControlEpoch != command.SoftCloseControlEpoch || state.ControlEpoch.Number != command.SoftCloseControlEpoch {
		return ErrFiscalPeriodEpochConflict
	}
	if state.SoftClose.PolicyVersion != command.SoftClosePolicyVersion || state.ControlEpoch.PolicyVersion != command.SoftClosePolicyVersion {
		return ErrFiscalPeriodPolicyConflict
	}
	return nil
}

func validateFinalApprovalReadiness(state ControlState, command ApplyCloseApprovalDecisionCommand, now time.Time) error {
	if state.CloseRun == nil || state.CloseRun.Status != CloseRunAwaitingApproval {
		return ErrFiscalPeriodStateConflict
	}
	for _, step := range state.CloseRun.Steps {
		if step.Mandatory && step.Status != CloseStepCompleted {
			return ErrFiscalPeriodPolicyConflict
		}
	}
	for _, exception := range state.CloseRun.Exceptions {
		if exception.Status != CloseExceptionApproved || (exception.ExpiresAt != nil && !now.Before(exception.ExpiresAt.UTC())) {
			return ErrFiscalPeriodPolicyConflict
		}
	}
	if len(state.CloseRun.ApprovalRequestIDs) > 0 && !containsUUID(state.CloseRun.ApprovalRequestIDs, command.ApprovalRequestID) {
		return ErrFiscalPeriodOwnerConflict
	}
	if command.FinalLedgerWatermark < 1 || state.CloseRun.BarrierEvidence == nil || command.FinalLedgerWatermark < state.CloseRun.BarrierEvidence.BarrierLedgerPosition {
		return ErrFiscalPeriodPolicyConflict
	}
	return nil
}

func allMandatoryStepsComplete(run CloseRun) bool {
	for _, step := range run.Steps {
		if step.Mandatory && step.Status != CloseStepCompleted {
			return false
		}
	}
	return true
}

func containsUUID(values []uuid.UUID, value uuid.UUID) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func sameCloseApprovalDecision(left, right CloseApprovalDecision) bool {
	return fingerprint(left) == fingerprint(right)
}

func appendCloseApprovalDecision(run *CloseRun, decision CloseApprovalDecision) {
	if run == nil {
		return
	}
	for _, existing := range run.ApprovalDecisions {
		if sameCloseApprovalDecision(existing, decision) {
			return
		}
	}
	run.ApprovalDecisions = append(run.ApprovalDecisions, decision)
}

func closeStatePayload(state ControlState, gate gl.SoftCloseGateResult) []byte {
	payload, err := json.Marshal(struct {
		Period       FiscalPeriod          `json:"period"`
		SoftClose    SoftCloseRun          `json:"softClose"`
		ControlEpoch SoftCloseControlEpoch `json:"controlEpoch"`
		CloseRun     CloseRun              `json:"closeRun"`
		Gate         gl.PeriodPostingGate  `json:"gate"`
	}{state.Period, state.SoftClose, state.ControlEpoch, dereferenceCloseRun(state.CloseRun), gate.Gate})
	if err != nil {
		return []byte(`{"periodState":"unserializable"}`)
	}
	return payload
}

func marshalCloseRunDetails(run *CloseRun) ([]byte, error) {
	if run == nil {
		return nil, nil
	}
	return json.Marshal(struct {
		Steps                  []CloseStep             `json:"steps,omitempty"`
		Exceptions             []CloseException        `json:"exceptions,omitempty"`
		ApprovalRequestIDs     []uuid.UUID             `json:"approvalRequestIds,omitempty"`
		ApprovalDecisions      []CloseApprovalDecision `json:"approvalDecisions,omitempty"`
		FinalApproval          *CloseApprovalDecision  `json:"finalApproval,omitempty"`
		ApprovalPolicyVersion  string                  `json:"approvalPolicyVersion,omitempty"`
		FinalLedgerWatermark   *int64                  `json:"finalLedgerWatermark,omitempty"`
		SealStatus             string                  `json:"sealStatus"`
		SealRequestID          uuid.UUID               `json:"sealRequestId,omitempty"`
		SealProofReference     string                  `json:"sealProofReference,omitempty"`
		SealFailureCode        string                  `json:"sealFailureCode,omitempty"`
		SealFailureDetail      string                  `json:"sealFailureDetail,omitempty"`
		FinalizationGateResult *gl.SoftCloseGateResult `json:"finalizationGateResult,omitempty"`
		ReleaseGateResult      *gl.SoftCloseGateResult `json:"releaseGateResult,omitempty"`
	}{run.Steps, run.Exceptions, run.ApprovalRequestIDs, run.ApprovalDecisions, run.FinalApproval, run.ApprovalPolicyVersion, run.FinalLedgerWatermark, run.SealStatus, run.SealRequestID, run.SealProofReference, run.SealFailureCode, run.SealFailureDetail, run.FinalizationGateResult, run.ReleaseGateResult})
}

func unmarshalCloseRunDetails(run *CloseRun, data []byte) error {
	if run == nil || len(data) == 0 || string(data) == "null" {
		return nil
	}
	var details struct {
		Steps                  []CloseStep             `json:"steps,omitempty"`
		Exceptions             []CloseException        `json:"exceptions,omitempty"`
		ApprovalRequestIDs     []uuid.UUID             `json:"approvalRequestIds,omitempty"`
		ApprovalDecisions      []CloseApprovalDecision `json:"approvalDecisions,omitempty"`
		FinalApproval          *CloseApprovalDecision  `json:"finalApproval,omitempty"`
		ApprovalPolicyVersion  string                  `json:"approvalPolicyVersion,omitempty"`
		FinalLedgerWatermark   *int64                  `json:"finalLedgerWatermark,omitempty"`
		SealStatus             string                  `json:"sealStatus"`
		SealRequestID          uuid.UUID               `json:"sealRequestId,omitempty"`
		SealProofReference     string                  `json:"sealProofReference,omitempty"`
		SealFailureCode        string                  `json:"sealFailureCode,omitempty"`
		SealFailureDetail      string                  `json:"sealFailureDetail,omitempty"`
		FinalizationGateResult *gl.SoftCloseGateResult `json:"finalizationGateResult,omitempty"`
		ReleaseGateResult      *gl.SoftCloseGateResult `json:"releaseGateResult,omitempty"`
	}
	if err := json.Unmarshal(data, &details); err != nil {
		return err
	}
	run.Steps = details.Steps
	run.Exceptions = details.Exceptions
	run.ApprovalRequestIDs = details.ApprovalRequestIDs
	run.ApprovalDecisions = details.ApprovalDecisions
	run.FinalApproval = details.FinalApproval
	run.ApprovalPolicyVersion = details.ApprovalPolicyVersion
	run.FinalLedgerWatermark = details.FinalLedgerWatermark
	run.SealStatus = details.SealStatus
	run.SealRequestID = details.SealRequestID
	run.SealProofReference = details.SealProofReference
	run.SealFailureCode = details.SealFailureCode
	run.SealFailureDetail = details.SealFailureDetail
	run.FinalizationGateResult = details.FinalizationGateResult
	run.ReleaseGateResult = details.ReleaseGateResult
	return nil
}

func dereferenceCloseRun(run *CloseRun) CloseRun {
	if run == nil {
		return CloseRun{}
	}
	return *run
}

func closeRunFinalizationResult(status gl.PostingGateStatus, command ResumeCloseRunCommand) gl.SoftCloseGateResult {
	return gl.SoftCloseGateResult{Operation: gl.PostingBarrierFinalizeOperation, Gate: status, PolicyReference: "reconciled-from-gl-status", AuthorizationPolicyVersion: command.SoftClosePolicyVersion, ValidationOutcome: "reconciled", NextAction: "request-audit-seal"}
}

func (service *Service) beginCloseCommand(ctx context.Context, actor Actor, scopeID uuid.UUID, key, commandFingerprint, operation string) (*durableCommand, *CommandResult, func(), error) {
	if service.durable != nil {
		acquisition, replay, err := service.acquireDurableCommand(ctx, actor, scopeID, key, commandFingerprint, operation)
		if err != nil || replay != nil {
			return nil, replay, nil, err
		}
		return &acquisition, nil, nil, nil
	}
	identity := service.idempotencyIdentity(operation, actor.UserID, scopeID, key)
	service.mu.Lock()
	if existing, ok := service.idempotency[identity]; ok {
		service.mu.Unlock()
		if existing.Fingerprint != commandFingerprint {
			return nil, nil, nil, ErrFiscalPeriodIdempotencyConflict
		}
		existing.Result.Replayed = true
		return nil, &existing.Result, nil, nil
	}
	return nil, nil, func() { service.mu.Unlock() }, nil
}

func (service *Service) rememberCommand(actor Actor, scopeID uuid.UUID, operation, key, fingerprint string, result CommandResult) {
	if service == nil || service.durable != nil {
		return
	}
	service.idempotency[service.idempotencyIdentity(operation, actor.UserID, scopeID, key)] = storedResult{Fingerprint: fingerprint, Result: result}
}

func (service *Service) commitCloseMutation(ctx context.Context, mutation Mutation, durable *durableCommand, result CommandResult) error {
	if durable == nil {
		return service.repository.Commit(ctx, mutation)
	}
	committer, ok := service.repository.(DurableRepository)
	if !ok {
		return ErrInvalidFiscalPeriodService
	}
	metadata, err := service.establishedMetadata(*durable, result)
	if err != nil {
		return err
	}
	return committer.CommitWithIdempotency(ctx, mutation, DurableMutationCommit{Coordinator: service.durable.Coordinator, Acquisitions: []platformidempotency.DurableAcquisition{durable.acquisition}, Results: []platformidempotency.CommandResultMetadata{metadata}})
}

func closeCommandReturn(replay *CommandResult, err error) (CommandResult, error) {
	if err != nil {
		return CommandResult{}, err
	}
	if replay != nil {
		return *replay, nil
	}
	return CommandResult{}, nil
}

func resumeCloseRunFingerprint(command ResumeCloseRunCommand) string {
	return fingerprint(struct {
		Operation                                       string
		Scope, Period, Soft, Close                      uuid.UUID
		PeriodVersion, CloseVersion, GateVersion, Epoch int64
		Policy                                          string
	}{ResumeCloseRunCommandIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.SoftCloseRunID, command.CloseRunID, command.ExpectedPeriodVersion, command.ExpectedCloseRunVersion, command.ExpectedGateVersion, command.SoftCloseControlEpoch, command.SoftClosePolicyVersion})
}

func abortCloseRunFingerprint(command AbortCloseRunCommand) string {
	return fingerprint(struct {
		Operation string
		Resume    ResumeCloseRunCommand
		Reason    string
	}{AbortCloseRunCommandIdentity, command.ResumeCloseRunCommand, strings.TrimSpace(command.Reason)})
}

func exceptionApprovalFingerprint(command ApplyCloseExceptionApprovalDecisionCommand) string {
	return fingerprint(struct {
		Operation string
		Command   ApplyCloseExceptionApprovalDecisionCommand
	}{ApplyCloseExceptionApprovalIdentity, command})
}

func finalApprovalFingerprint(command ApplyCloseApprovalDecisionCommand) string {
	return fingerprint(struct {
		Operation string
		Command   ApplyCloseApprovalDecisionCommand
	}{ApplyCloseApprovalIdentity, command})
}

func closeRunDecisionFingerprint(runID uuid.UUID, version int64) string {
	return fingerprint(struct {
		CloseRunID uuid.UUID
		Version    int64
	}{runID, version})
}

var _ CloseTaskPort = PendingCloseTaskPort{}
var _ CloseDecisionPort = EvidenceCloseDecisionPort{}
var _ AuditSealPort = PendingAuditSealPort{}
