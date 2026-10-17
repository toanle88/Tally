package fiscalperiod

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/gl"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	PermissionStartHardClose              = "finance.fpm.start.hard.close"
	PermissionApplyPostingGateResult      = "finance.fpm.apply.posting.gate.result"
	PermissionResumeCloseRun              = "finance.fpm.resume.close.run"
	PermissionAbortCloseRun               = "finance.fpm.abort.close.run"
	PermissionApplyCloseExceptionApproval = "finance.fpm.apply.close.exception.approval.decision"
	PermissionApplyCloseApproval          = "finance.fpm.apply.close.approval.decision"
	PermissionRequestReopen               = "finance.fpm.request.reopen"
	PermissionApplyReopenApprovalDecision = "finance.fpm.apply.reopen.approval.decision"
	PermissionStartReclose                = "finance.fpm.start.reclose"
	StartHardCloseOperation               = "start-hard-close"
	ApplyPostingGateResultOperation       = "apply-posting-gate-result"
	ResumeCloseRunOperation               = "resume-close-run"
	AbortCloseRunOperation                = "abort-close-run"
	ApplyCloseExceptionApprovalOperation  = "apply-close-exception-approval-decision"
	ApplyCloseApprovalOperation           = "apply-close-approval-decision"
	RequestReopenOperation                = "request-reopen"
	ApplyReopenApprovalDecisionOperation  = "apply-reopen-approval-decision"
	StartRecloseOperation                 = "start-reclose"
	ReconcileReopenExpiryOperation        = "reconcile-reopen-expiry"
	StartHardCloseCommandIdentity         = "start-hard-close.v1"
	ApplyPostingGateResultCommandIdentity = "apply-posting-gate-result.v1"
	ResumeCloseRunCommandIdentity         = "resume-close-run.v1"
	AbortCloseRunCommandIdentity          = "abort-close-run.v1"
	ApplyCloseExceptionApprovalIdentity   = "apply-close-exception-approval-decision.v1"
	ApplyCloseApprovalIdentity            = "apply-close-approval-decision.v1"
	RequestReopenCommandIdentity          = "request-reopen.v1"
	ApplyReopenApprovalDecisionIdentity   = "apply-reopen-approval-decision.v1"
	StartRecloseCommandIdentity           = "start-reclose.v1"
	ReconcileReopenExpiryCommandIdentity  = "reconcile-reopen-expiry.v1"

	PeriodStateOpen       = "Open"
	PeriodStateSoftClosed = "SoftClosed"
	PeriodStateClosing    = "Closing"
	PeriodStateHardClosed = "HardClosed"
	PeriodStateReopening  = "Reopening"

	SoftCloseRunActive         = "Active"
	SoftCloseRunHandoffPending = "HandoffPending"
	SoftCloseRunCompleted      = "Completed"
	SoftCloseRunSuperseded     = "Superseded"

	CloseRunInitiating       = "Initiating"
	CloseRunBarrierAcquired  = "BarrierAcquired"
	CloseRunInProgress       = "InProgress"
	CloseRunAwaitingApproval = "AwaitingApproval"
	CloseRunFinalizing       = "Finalizing"
	CloseRunCompleted        = "Completed"
	CloseRunFailed           = "Failed"
	CloseRunAborting         = "Aborting"
	CloseRunAborted          = "Aborted"

	CloseRunSealNotRequested = "NotRequested"
	CloseRunSealPending      = "SealPending"
	CloseRunSealSealed       = "Sealed"
	CloseRunSealFailed       = "SealFailed"

	ControlEpochActive         = "Active"
	ControlEpochHandoffPending = "HandoffPending"
)

var (
	ErrInvalidFiscalPeriodService           = errors.New("invalid fiscal-period service")
	ErrFiscalPeriodNotFound                 = errors.New("fiscal period not found")
	ErrFiscalPeriodStateConflict            = errors.New("fiscal period state conflict")
	ErrFiscalPeriodScopeConflict            = errors.New("fiscal period scope conflict")
	ErrFiscalPeriodOwnerConflict            = errors.New("fiscal period control owner conflict")
	ErrFiscalPeriodVersionConflict          = errors.New("fiscal period version conflict")
	ErrFiscalPeriodPolicyConflict           = errors.New("fiscal period policy conflict")
	ErrFiscalPeriodEpochConflict            = errors.New("fiscal period control epoch conflict")
	ErrFiscalPeriodIdempotencyConflict      = errors.New("fiscal period idempotency conflict")
	ErrFiscalPeriodAuthorizationDenied      = errors.New("fiscal period authorization denied")
	ErrFiscalPeriodAuthorizationUnavailable = errors.New("fiscal period authorization unavailable")
	ErrFiscalPeriodAuditUnavailable         = errors.New("fiscal period audit unavailable")
	ErrFiscalPeriodOutboxUnavailable        = errors.New("fiscal period outbox unavailable")
	ErrFiscalPeriodCommandInProgress        = errors.New("fiscal-period command is still in progress")
	ErrFiscalPeriodDurableCommandFailed     = errors.New("fiscal-period command durably failed")
	ErrPostingBarrierResultConflict         = errors.New("posting barrier result conflict")
	ErrPostingBarrierUnavailable            = errors.New("posting barrier dependency unavailable")
	ErrPostingBarrierRejected               = errors.New("posting barrier request rejected")
	ErrCloseTaskUnavailable                 = errors.New("close task dependency unavailable")
	ErrCloseTaskFailed                      = errors.New("close task failed")
	ErrCloseDecisionUnavailable             = errors.New("close approval decision dependency unavailable")
	ErrCloseDecisionRejected                = errors.New("close approval decision rejected")
	ErrReopenDecisionUnavailable            = errors.New("reopen approval decision dependency unavailable")
	ErrReopenDecisionRejected               = errors.New("reopen approval decision rejected")
	ErrReopenGateUnavailable                = errors.New("reopen gate dependency unavailable")
	ErrReopenGateRejected                   = errors.New("reopen gate request rejected")
	ErrCloseSealUnavailable                 = errors.New("audit seal dependency unavailable")
	ErrCloseSealFailed                      = errors.New("audit seal failed")
	ErrCloseFinalizationPending             = errors.New("close finalization is pending")
	ErrInvalidFiscalPeriodCommand           = errors.New("invalid fiscal-period command")
)

type Actor struct {
	UserID           uuid.UUID
	SubjectReference string
}

func (actor Actor) Validate() error {
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return ErrFiscalPeriodAuthorizationDenied
	}
	return nil
}

type AuthorizationDecision struct {
	Allowed           bool
	Permission        string
	PolicyReference   string
	PolicyVersion     string
	DecisionReference uuid.UUID
	Reason            string
}

type Authorizer interface {
	Authorize(context.Context, Actor, string, uuid.UUID) (AuthorizationDecision, error)
}

type AllowAllAuthorizer struct{}

func (AllowAllAuthorizer) Authorize(_ context.Context, actor Actor, permission string, scopeID uuid.UUID) (AuthorizationDecision, error) {
	if err := actor.Validate(); err != nil || scopeID == uuid.Nil {
		return AuthorizationDecision{}, ErrFiscalPeriodAuthorizationDenied
	}
	return AuthorizationDecision{
		Allowed:           true,
		Permission:        permission,
		PolicyReference:   "development-allow-all",
		PolicyVersion:     "development-v1",
		DecisionReference: uuid.New(),
	}, nil
}

type FiscalPeriod struct {
	ID                   uuid.UUID  `json:"id"`
	AccountingScopeID    uuid.UUID  `json:"accountingScopeId"`
	State                string     `json:"state"`
	StateVersion         int64      `json:"stateVersion"`
	AggregateVersion     int64      `json:"aggregateVersion"`
	ActiveSoftCloseRunID *uuid.UUID `json:"activeSoftCloseRunId,omitempty"`
	ActiveCloseRunID     *uuid.UUID `json:"activeCloseRunId,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func (period FiscalPeriod) Validate() error {
	if period.ID == uuid.Nil || period.AccountingScopeID == uuid.Nil {
		return ErrFiscalPeriodNotFound
	}
	if period.State != PeriodStateOpen && period.State != PeriodStateSoftClosed && period.State != PeriodStateClosing && period.State != PeriodStateHardClosed && period.State != PeriodStateReopening {
		return fmt.Errorf("%w: unsupported period state %q", ErrFiscalPeriodStateConflict, period.State)
	}
	if period.StateVersion < 1 || period.AggregateVersion < 1 {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

type SoftCloseRun struct {
	ID                uuid.UUID `json:"id"`
	AccountingScopeID uuid.UUID `json:"accountingScopeId"`
	FiscalPeriodID    uuid.UUID `json:"fiscalPeriodId"`
	Status            string    `json:"status"`
	ControlEpoch      int64     `json:"controlEpoch"`
	OpenedGateVersion int64     `json:"openedGateVersion"`
	PolicyVersion     string    `json:"policyVersion"`
	AggregateVersion  int64     `json:"aggregateVersion"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (run SoftCloseRun) Validate() error {
	if run.ID == uuid.Nil || run.AccountingScopeID == uuid.Nil || run.FiscalPeriodID == uuid.Nil {
		return ErrFiscalPeriodNotFound
	}
	if run.Status != SoftCloseRunActive && run.Status != SoftCloseRunHandoffPending && run.Status != SoftCloseRunCompleted && run.Status != SoftCloseRunSuperseded {
		return fmt.Errorf("%w: unsupported soft-close status %q", ErrFiscalPeriodStateConflict, run.Status)
	}
	if run.ControlEpoch < 1 || run.OpenedGateVersion < 1 || run.AggregateVersion < 1 || strings.TrimSpace(run.PolicyVersion) == "" {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

type SoftCloseControlEpoch struct {
	SoftCloseRunID    uuid.UUID                      `json:"softCloseRunId"`
	FiscalPeriodID    uuid.UUID                      `json:"fiscalPeriodId"`
	Number            int64                          `json:"number"`
	OpenedGateVersion int64                          `json:"openedGateVersion"`
	PolicyVersion     string                         `json:"policyVersion"`
	Status            string                         `json:"status"`
	HandoffCloseRunID *uuid.UUID                     `json:"handoffCloseRunId,omitempty"`
	FrozenSummary     *gl.FrozenGateAdmissionSummary `json:"frozenSummary,omitempty"`
	AggregateVersion  int64                          `json:"aggregateVersion"`
	CreatedAt         time.Time                      `json:"createdAt"`
	UpdatedAt         time.Time                      `json:"updatedAt"`
}

func (epoch SoftCloseControlEpoch) Validate() error {
	if epoch.SoftCloseRunID == uuid.Nil || epoch.FiscalPeriodID == uuid.Nil || epoch.Number < 1 || epoch.OpenedGateVersion < 1 || epoch.AggregateVersion < 1 {
		return ErrFiscalPeriodEpochConflict
	}
	if strings.TrimSpace(epoch.PolicyVersion) == "" {
		return ErrFiscalPeriodPolicyConflict
	}
	if epoch.Status != ControlEpochActive && epoch.Status != ControlEpochHandoffPending {
		return fmt.Errorf("%w: unsupported control-epoch status %q", ErrFiscalPeriodEpochConflict, epoch.Status)
	}
	return nil
}

type BarrierEvidence struct {
	Operation                     string                        `json:"operation"`
	GateMode                      string                        `json:"gateMode"`
	GateVersion                   int64                         `json:"gateVersion"`
	BarrierLedgerPosition         int64                         `json:"barrierLedgerPosition"`
	PriorControlProcessID         *uuid.UUID                    `json:"priorControlProcessId,omitempty"`
	PriorControlProcessType       string                        `json:"priorControlProcessType"`
	PriorControlEpoch             *int64                        `json:"priorControlEpoch,omitempty"`
	FrozenSummary                 gl.FrozenGateAdmissionSummary `json:"frozenSummary"`
	DecisionReference             uuid.UUID                     `json:"decisionReference"`
	PolicyReference               string                        `json:"policyReference"`
	AuthorizationPolicyVersion    string                        `json:"authorizationPolicyVersion"`
	AuditReference                uuid.UUID                     `json:"auditReference"`
	CommandFingerprint            string                        `json:"commandFingerprint"`
	CommandType                   string                        `json:"commandType"`
	ApplicationCommandFingerprint string                        `json:"applicationCommandFingerprint"`
	ValidationOutcome             string                        `json:"validationOutcome"`
	NextAction                    string                        `json:"nextAction"`
	AuthoritativeResult           gl.SoftCloseGateResult        `json:"authoritativeResult"`
}

type CloseRun struct {
	ID                     uuid.UUID               `json:"id"`
	AccountingScopeID      uuid.UUID               `json:"accountingScopeId"`
	FiscalPeriodID         uuid.UUID               `json:"fiscalPeriodId"`
	CloseType              string                  `json:"closeType"`
	Status                 string                  `json:"status"`
	AggregateVersion       int64                   `json:"aggregateVersion"`
	CommandFingerprint     string                  `json:"commandFingerprint"`
	BarrierCommandID       uuid.UUID               `json:"barrierCommandId"`
	BarrierIdempotencyKey  string                  `json:"barrierIdempotencyKey"`
	BarrierEvidence        *BarrierEvidence        `json:"barrierEvidence,omitempty"`
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
	LastFailureCode        string                  `json:"lastFailureCode,omitempty"`
	LastFailureDetail      string                  `json:"lastFailureDetail,omitempty"`
	CreatedAt              time.Time               `json:"createdAt"`
	UpdatedAt              time.Time               `json:"updatedAt"`
}

func (run CloseRun) Validate() error {
	if run.ID == uuid.Nil || run.AccountingScopeID == uuid.Nil || run.FiscalPeriodID == uuid.Nil {
		return ErrFiscalPeriodNotFound
	}
	if run.CloseType != gl.CloseTypeInitialHardClose && run.CloseType != gl.CloseTypeReclose {
		return ErrFiscalPeriodPolicyConflict
	}
	if run.Status != CloseRunInitiating && run.Status != CloseRunBarrierAcquired && run.Status != CloseRunInProgress && run.Status != CloseRunAwaitingApproval && run.Status != CloseRunFinalizing && run.Status != CloseRunCompleted && run.Status != CloseRunFailed && run.Status != CloseRunAborting && run.Status != CloseRunAborted {
		return fmt.Errorf("%w: unsupported close-run status %q", ErrFiscalPeriodStateConflict, run.Status)
	}
	if run.AggregateVersion < 1 || strings.TrimSpace(run.CommandFingerprint) == "" || run.BarrierCommandID == uuid.Nil || strings.TrimSpace(run.BarrierIdempotencyKey) == "" {
		return ErrFiscalPeriodVersionConflict
	}
	if run.SealStatus != "" && run.SealStatus != CloseRunSealNotRequested && run.SealStatus != CloseRunSealPending && run.SealStatus != CloseRunSealSealed && run.SealStatus != CloseRunSealFailed {
		return ErrFiscalPeriodStateConflict
	}
	if run.FinalLedgerWatermark != nil && *run.FinalLedgerWatermark < 1 {
		return ErrFiscalPeriodVersionConflict
	}
	if run.Status == CloseRunCompleted && (run.FinalizationGateResult == nil || run.FinalLedgerWatermark == nil) {
		return ErrFiscalPeriodStateConflict
	}
	if run.Status == CloseRunAborted && run.ReleaseGateResult == nil {
		return ErrFiscalPeriodStateConflict
	}
	stepIDs := make(map[string]struct{}, len(run.Steps))
	for _, step := range run.Steps {
		if _, exists := stepIDs[step.ID]; exists {
			return ErrFiscalPeriodPolicyConflict
		}
		stepIDs[step.ID] = struct{}{}
		if err := step.Validate(); err != nil {
			return err
		}
	}
	exceptionIDs := make(map[uuid.UUID]struct{}, len(run.Exceptions))
	for _, exception := range run.Exceptions {
		if _, exists := exceptionIDs[exception.ID]; exists {
			return ErrFiscalPeriodPolicyConflict
		}
		if exception.CloseRunID != run.ID {
			return ErrFiscalPeriodScopeConflict
		}
		exceptionIDs[exception.ID] = struct{}{}
		if err := exception.Validate(); err != nil {
			return err
		}
	}
	for _, approval := range run.ApprovalDecisions {
		if err := approval.Validate(); err != nil {
			return err
		}
	}
	if run.FinalApproval != nil {
		if err := run.FinalApproval.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type ControlState struct {
	Period        FiscalPeriod          `json:"period"`
	SoftClose     SoftCloseRun          `json:"softClose"`
	ControlEpoch  SoftCloseControlEpoch `json:"controlEpoch"`
	CloseRun      *CloseRun             `json:"closeRun,omitempty"`
	ReopenRequest *ReopenRequest        `json:"reopenRequest,omitempty"`
}

func (state ControlState) Validate() error {
	if err := state.Period.Validate(); err != nil {
		return err
	}
	if err := state.SoftClose.Validate(); err != nil {
		return err
	}
	if err := state.ControlEpoch.Validate(); err != nil {
		return err
	}
	if state.Period.AccountingScopeID != state.SoftClose.AccountingScopeID || state.Period.ID != state.SoftClose.FiscalPeriodID || state.SoftClose.ID != state.ControlEpoch.SoftCloseRunID || state.ControlEpoch.FiscalPeriodID != state.Period.ID {
		return ErrFiscalPeriodScopeConflict
	}
	if state.CloseRun != nil {
		if err := state.CloseRun.Validate(); err != nil {
			return err
		}
		if state.CloseRun.AccountingScopeID != state.Period.AccountingScopeID || state.CloseRun.FiscalPeriodID != state.Period.ID {
			return ErrFiscalPeriodScopeConflict
		}
	}
	if state.ReopenRequest != nil {
		if err := state.ReopenRequest.Validate(); err != nil {
			return err
		}
		if state.ReopenRequest.AccountingScopeID != state.Period.AccountingScopeID || state.ReopenRequest.FiscalPeriodID != state.Period.ID {
			return ErrFiscalPeriodScopeConflict
		}
	}
	if state.Period.State == PeriodStateReopening {
		if state.ReopenRequest == nil || (state.ReopenRequest.Status != ReopenRequestActive && state.ReopenRequest.Status != ReopenRequestExpiredPendingClosure && state.ReopenRequest.Status != ReopenRequestGateClosing && state.ReopenRequest.Status != ReopenRequestRecloseHandoff) {
			return ErrFiscalPeriodStateConflict
		}
	}
	if state.ReopenRequest != nil && (state.ReopenRequest.Status == ReopenRequestActive || state.ReopenRequest.Status == ReopenRequestExpiredPendingClosure) && state.Period.State != PeriodStateReopening {
		return ErrFiscalPeriodStateConflict
	}
	if state.ReopenRequest != nil {
		switch state.ReopenRequest.Status {
		case ReopenRequestRecloseInProgress:
			if state.Period.State != PeriodStateClosing || state.CloseRun == nil || state.CloseRun.CloseType != gl.CloseTypeReclose {
				return ErrFiscalPeriodStateConflict
			}
		case ReopenRequestCompletedNoChange, ReopenRequestCompleted:
			if state.Period.State != PeriodStateHardClosed {
				return ErrFiscalPeriodStateConflict
			}
		}
	}
	return nil
}

type Event struct {
	EventType         string
	AggregateID       uuid.UUID
	AggregateVersion  int64
	AccountingScopeID uuid.UUID
	CorrelationID     string
	CausationID       string
	OccurredAt        time.Time
	Payload           []byte
}

type AuditRecord struct {
	AggregateID           uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Action                string
	AccountingScopeID     uuid.UUID
	Permission            string
	PolicyReference       string
	PolicyVersion         string
	DecisionReference     uuid.UUID
	RevisionVersion       int64
	BeforeFingerprint     string
	AfterFingerprint      string
	CorrelationID         string
	CausationID           string
}

type Mutation struct {
	Before             ControlState
	After              ControlState
	Operation          string
	CommandFingerprint string
	Audit              AuditRecord
	Events             []Event
}

type Repository interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (ControlState, error)
	Commit(context.Context, Mutation) error
}

type DurableRepository interface {
	CommitWithIdempotency(context.Context, Mutation, DurableMutationCommit) error
}

type DurableMutationCommit struct {
	Coordinator  platformidempotency.DurableCoordinator
	Acquisitions []platformidempotency.DurableAcquisition
	Results      []platformidempotency.CommandResultMetadata
}

type DurableServiceConfig struct {
	Database                               platformidempotency.TxBeginner
	Coordinator                            platformidempotency.DurableCoordinator
	Policy                                 platformidempotency.IdempotencyPolicy
	StartOperationID                       string
	ApplyOperationID                       string
	ResumeOperationID                      string
	AbortOperationID                       string
	ApplyCloseExceptionApprovalOperationID string
	ApplyCloseApprovalOperationID          string
	RequestReopenOperationID               string
	ApplyReopenApprovalOperationID         string
	StartRecloseOperationID                string
}

type PostingGateApplicationPort interface {
	AcquirePostingBarrier(context.Context, gl.Actor, gl.AcquirePostingBarrierCommand) (gl.SoftCloseGateResult, error)
	ReleasePostingBarrier(context.Context, gl.Actor, gl.ReleasePostingBarrierCommand) (gl.SoftCloseGateResult, error)
	FinalizePostingGate(context.Context, gl.Actor, gl.FinalizePostingGateCommand) (gl.SoftCloseGateResult, error)
	OpenScopedReopenGate(context.Context, gl.Actor, gl.OpenScopedReopenGateCommand) (gl.SoftCloseGateResult, error)
	CloseScopedReopenGate(context.Context, gl.Actor, gl.CloseScopedReopenGateCommand) (gl.SoftCloseGateResult, error)
	OpenOperationalReopenGate(context.Context, gl.Actor, gl.OpenOperationalReopenGateCommand) (gl.SoftCloseGateResult, error)
	CloseOperationalReopenGate(context.Context, gl.Actor, gl.CloseOperationalReopenGateCommand) (gl.SoftCloseGateResult, error)
	BeginRecloseGate(context.Context, gl.Actor, gl.BeginRecloseGateCommand) (gl.SoftCloseGateResult, error)
	GetPostingGateStatus(context.Context, uuid.UUID, uuid.UUID) (gl.PostingGateStatus, error)
}

type StartHardCloseCommand struct {
	CommandID              uuid.UUID
	AccountingScopeID      uuid.UUID
	FiscalPeriodID         uuid.UUID
	SoftCloseRunID         uuid.UUID
	ExpectedPeriodVersion  int64
	ExpectedGateVersion    int64
	SoftCloseControlEpoch  int64
	SoftClosePolicyVersion string
	IdempotencyKey         string
	CorrelationID          string
	CausationID            string
}

func (command StartHardCloseCommand) canonical() StartHardCloseCommand {
	command.SoftClosePolicyVersion = strings.TrimSpace(command.SoftClosePolicyVersion)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command StartHardCloseCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil || command.SoftCloseRunID == uuid.Nil {
		return ErrInvalidFiscalPeriodCommand
	}
	if command.ExpectedPeriodVersion < 1 || command.ExpectedGateVersion < 1 || command.SoftCloseControlEpoch < 1 || command.SoftClosePolicyVersion == "" || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

type ApplyPostingGateResultCommand struct {
	CommandID                    uuid.UUID
	AccountingScopeID            uuid.UUID
	FiscalPeriodID               uuid.UUID
	SoftCloseRunID               uuid.UUID
	CloseRunID                   uuid.UUID
	ExpectedPeriodVersion        int64
	ExpectedGateVersion          int64
	SoftCloseControlEpoch        int64
	SoftClosePolicyVersion       string
	ReopenRequestID              uuid.UUID
	ExpectedReopenRequestVersion int64
	Result                       gl.SoftCloseGateResult
	IdempotencyKey               string
	CorrelationID                string
	CausationID                  string
}

func (command ApplyPostingGateResultCommand) canonical() ApplyPostingGateResultCommand {
	command.SoftClosePolicyVersion = strings.TrimSpace(command.SoftClosePolicyVersion)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command ApplyPostingGateResultCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil {
		return ErrInvalidFiscalPeriodCommand
	}
	if command.ExpectedPeriodVersion < 1 || command.ExpectedGateVersion < 1 || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	switch command.Result.Operation {
	case gl.PostingBarrierAcquireOperation:
		if command.SoftCloseRunID == uuid.Nil || command.CloseRunID == uuid.Nil || command.SoftCloseControlEpoch < 1 || command.SoftClosePolicyVersion == "" {
			return ErrInvalidFiscalPeriodCommand
		}
	case gl.ScopedReopenGateCloseOperation, gl.OperationalReopenGateCloseOperation:
		if command.ReopenRequestID == uuid.Nil || command.ExpectedReopenRequestVersion < 1 {
			return ErrInvalidFiscalPeriodCommand
		}
	default:
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

type CommandResult struct {
	State      ControlState
	GateResult gl.SoftCloseGateResult
	Decision   AuthorizationDecision
	Replayed   bool
}

type storedResult struct {
	Fingerprint string
	Result      CommandResult
}

type durableCommand struct {
	acquisition platformidempotency.DurableAcquisition
	identity    platformidempotency.IdempotencyIdentity
	operationID string
}

type Service struct {
	repository     Repository
	authorizer     Authorizer
	postingGate    PostingGateApplicationPort
	closeTasks     CloseTaskPort
	workflow       CloseDecisionPort
	reopenWorkflow ReopenDecisionPort
	seal           AuditSealPort
	clock          func() time.Time
	durable        *DurableServiceConfig

	mu          sync.Mutex
	idempotency map[string]storedResult
}

func NewService(repository Repository, authorizer Authorizer, postingGate PostingGateApplicationPort, clock func() time.Time) (*Service, error) {
	return newService(repository, authorizer, postingGate, clock, nil, CloseRunDependencies{})
}

func NewServiceWithDependencies(repository Repository, authorizer Authorizer, postingGate PostingGateApplicationPort, dependencies CloseRunDependencies, clock func() time.Time) (*Service, error) {
	return newService(repository, authorizer, postingGate, clock, nil, dependencies)
}

func NewServiceWithDurableIdempotency(repository Repository, authorizer Authorizer, postingGate PostingGateApplicationPort, clock func() time.Time, durable DurableServiceConfig) (*Service, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.StartOperationID) == "" || strings.TrimSpace(durable.ApplyOperationID) == "" || strings.TrimSpace(durable.ResumeOperationID) == "" || strings.TrimSpace(durable.AbortOperationID) == "" || strings.TrimSpace(durable.ApplyCloseExceptionApprovalOperationID) == "" || strings.TrimSpace(durable.ApplyCloseApprovalOperationID) == "" || strings.TrimSpace(durable.RequestReopenOperationID) == "" || strings.TrimSpace(durable.ApplyReopenApprovalOperationID) == "" || strings.TrimSpace(durable.StartRecloseOperationID) == "" {
		return nil, ErrInvalidFiscalPeriodService
	}
	return newService(repository, authorizer, postingGate, clock, &durable, CloseRunDependencies{})
}

func newService(repository Repository, authorizer Authorizer, postingGate PostingGateApplicationPort, clock func() time.Time, durable *DurableServiceConfig, dependencies CloseRunDependencies) (*Service, error) {
	if repository == nil || authorizer == nil || postingGate == nil || clock == nil {
		return nil, ErrInvalidFiscalPeriodService
	}
	dependencies = dependencies.withDefaults()
	return &Service{repository: repository, authorizer: authorizer, postingGate: postingGate, closeTasks: dependencies.CloseTasks, workflow: dependencies.Workflow, reopenWorkflow: dependencies.ReopenWorkflow, seal: dependencies.Seal, clock: clock, durable: durable, idempotency: make(map[string]storedResult)}, nil
}

// BindReopenDecisionPort connects the FPM service to the owning Workflow
// context after both module runtimes have been composed. The port is kept
// behind FPM's narrow validation boundary so FPM never reads Workflow data or
// storage directly.
func (service *Service) BindReopenDecisionPort(port ReopenDecisionPort) error {
	if service == nil || port == nil {
		return ErrInvalidFiscalPeriodService
	}
	service.reopenWorkflow = port
	return nil
}

func (service *Service) acquireDurableCommand(ctx context.Context, actor Actor, scopeID uuid.UUID, key, commandFingerprint, operation string) (durableCommand, *CommandResult, error) {
	if service == nil || service.durable == nil {
		return durableCommand{}, nil, ErrInvalidFiscalPeriodService
	}
	identity, err := newDurableIdentity(operation, actor, scopeID, key)
	if err != nil {
		return durableCommand{}, nil, err
	}
	operationID := service.durableOperationID(operation)
	acquisition, err := service.durable.Coordinator.Acquire(ctx, service.durable.Database, identity, platformidempotency.Fingerprint(commandFingerprint), operationID, service.durable.Policy)
	if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
		return durableCommand{}, nil, ErrFiscalPeriodIdempotencyConflict
	}
	if err != nil {
		return durableCommand{}, nil, err
	}
	if acquisition.Decision() == platformidempotency.DecisionReturn {
		switch acquisition.Result().State() {
		case platformidempotency.StateInProgress:
			return durableCommand{}, nil, ErrFiscalPeriodCommandInProgress
		case platformidempotency.StateFailed:
			return durableCommand{}, nil, ErrFiscalPeriodDurableCommandFailed
		}
		result, err := decodeDurableCommandResult(acquisition.Result().ResultBody())
		if err != nil {
			return durableCommand{}, nil, err
		}
		result.Replayed = true
		return durableCommand{}, &result, nil
	}
	return durableCommand{acquisition: acquisition, identity: identity, operationID: operationID}, nil, nil
}

func (service *Service) durableOperationID(operation string) string {
	if service == nil || service.durable == nil {
		return operation
	}
	if operation == StartHardCloseOperation {
		return service.durable.StartOperationID
	}
	if operation == ApplyPostingGateResultOperation {
		return service.durable.ApplyOperationID
	}
	if operation == ResumeCloseRunOperation {
		return service.durable.ResumeOperationID
	}
	if operation == AbortCloseRunOperation {
		return service.durable.AbortOperationID
	}
	if operation == ApplyCloseExceptionApprovalOperation {
		return service.durable.ApplyCloseExceptionApprovalOperationID
	}
	if operation == ApplyCloseApprovalOperation {
		return service.durable.ApplyCloseApprovalOperationID
	}
	if operation == RequestReopenOperation {
		return service.durable.RequestReopenOperationID
	}
	if operation == ApplyReopenApprovalDecisionOperation {
		return service.durable.ApplyReopenApprovalOperationID
	}
	if operation == StartRecloseOperation {
		return service.durable.StartRecloseOperationID
	}
	return operation
}

func newDurableIdentity(operation string, actor Actor, scopeID uuid.UUID, key string) (platformidempotency.IdempotencyIdentity, error) {
	scope, err := json.Marshal(struct {
		Module    string    `json:"module"`
		Operation string    `json:"operation"`
		ActorID   uuid.UUID `json:"actorId"`
		ScopeID   uuid.UUID `json:"scopeId"`
	}{"fiscal-period", operation, actor.UserID, scopeID})
	if err != nil {
		return platformidempotency.IdempotencyIdentity{}, err
	}
	return platformidempotency.NewOpaqueIdentity(string(scope), key)
}

func decodeDurableCommandResult(body []byte) (CommandResult, error) {
	var result CommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return CommandResult{}, ErrFiscalPeriodCommandInProgress
	}
	return result, nil
}

func (service *Service) establishedMetadata(command durableCommand, result CommandResult) (platformidempotency.CommandResultMetadata, error) {
	stored := result
	stored.Replayed = false
	body, err := json.Marshal(stored)
	if err != nil {
		return platformidempotency.CommandResultMetadata{}, err
	}
	status := 200
	aggregateID := stored.State.Period.ID
	var processIDPointer *uuid.UUID
	if stored.State.CloseRun != nil && stored.State.CloseRun.ID != uuid.Nil {
		processID := stored.State.CloseRun.ID
		processIDPointer = &processID
	}
	return platformidempotency.NewCommandResultMetadata(command.identity, command.acquisition.Result().Fingerprint(), command.operationID, platformidempotency.StateEstablished, &status, body, &aggregateID, processIDPointer)
}

func (service *Service) finalizeDurableResult(ctx context.Context, command *durableCommand, result CommandResult) error {
	if service == nil || service.durable == nil || command == nil || command.acquisition.Decision() != platformidempotency.DecisionExecute {
		return nil
	}
	metadata, err := service.establishedMetadata(*command, result)
	if err != nil {
		return err
	}
	tx, err := service.durable.Database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := service.durable.Coordinator.Finalize(ctx, tx, command.acquisition, metadata); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (service *Service) finalizeDurableFailureIfNeeded(ctx context.Context, command *durableCommand, commandErr error) {
	if service == nil || service.durable == nil || command == nil || command.acquisition.Decision() != platformidempotency.DecisionExecute || shouldLeaveFiscalPeriodCommandInProgress(commandErr) {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{fiscalPeriodFailureCode(commandErr)})
	if err != nil {
		return
	}
	initial := command.acquisition.Result()
	metadata, err := platformidempotency.NewCommandResultMetadata(initial.Identity(), initial.Fingerprint(), initial.OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
	if err != nil {
		return
	}
	tx, err := service.durable.Database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := service.durable.Coordinator.Finalize(ctx, tx, command.acquisition, metadata); err != nil {
		return
	}
	if err := tx.Commit(ctx); err != nil {
		return
	}
	committed = true
}

func shouldLeaveFiscalPeriodCommandInProgress(err error) bool {
	return errors.Is(err, ErrPostingBarrierUnavailable) || errors.Is(err, ErrPostingBarrierRejected) || errors.Is(err, ErrCloseTaskUnavailable) || errors.Is(err, ErrCloseDecisionUnavailable) || errors.Is(err, ErrReopenDecisionUnavailable) || errors.Is(err, ErrReopenGateUnavailable) || errors.Is(err, ErrCloseSealUnavailable) || errors.Is(err, ErrFiscalPeriodAuditUnavailable) || errors.Is(err, ErrFiscalPeriodOutboxUnavailable) || errors.Is(err, ErrFiscalPeriodVersionConflict)
}

func fiscalPeriodFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrFiscalPeriodAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrFiscalPeriodIdempotencyConflict):
		return "IDEMPOTENCY_CONFLICT"
	case errors.Is(err, ErrPostingBarrierResultConflict):
		return "POSTING_BARRIER_RESULT_CONFLICT"
	case errors.Is(err, ErrFiscalPeriodOwnerConflict):
		return "OWNER_CONFLICT"
	case errors.Is(err, ErrCloseTaskFailed):
		return "CLOSE_TASK_FAILED"
	case errors.Is(err, ErrCloseDecisionRejected):
		return "CLOSE_APPROVAL_REJECTED"
	case errors.Is(err, ErrReopenDecisionRejected):
		return "REOPEN_APPROVAL_REJECTED"
	case errors.Is(err, ErrReopenGateRejected):
		return "REOPEN_GATE_REJECTED"
	case errors.Is(err, ErrCloseSealFailed):
		return "AUDIT_SEAL_FAILED"
	default:
		return "FISCAL_PERIOD_COMMAND_FAILED"
	}
}

func (service *Service) StartHardClose(ctx context.Context, actor Actor, command StartHardCloseCommand) (CommandResult, error) {
	command = command.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.postingGate == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	fingerprint := startHardCloseFingerprint(command)
	identity := service.idempotencyIdentity(StartHardCloseOperation, actor.UserID, command.AccountingScopeID, command.IdempotencyKey)
	var durable *durableCommand
	if service.durable != nil {
		acquisition, replay, err := service.acquireDurableCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, StartHardCloseOperation)
		if err != nil {
			return CommandResult{}, err
		}
		if replay != nil {
			return *replay, nil
		}
		durable = &acquisition
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if existing, ok := service.idempotency[identity]; ok {
			if existing.Fingerprint != fingerprint {
				return CommandResult{}, ErrFiscalPeriodIdempotencyConflict
			}
			existing.Result.Replayed = true
			return existing.Result, nil
		}
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionStartHardClose, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionStartHardClose); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, ErrFiscalPeriodAuthorizationDenied
	}

	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := state.Validate(); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if state.CloseRun != nil && state.CloseRun.Status == CloseRunBarrierAcquired && state.CloseRun.BarrierEvidence != nil {
		if state.CloseRun.CommandFingerprint != fingerprint {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
			return CommandResult{}, ErrFiscalPeriodOwnerConflict
		}
		result := CommandResult{State: state, GateResult: evidenceResult(*state.CloseRun), Decision: decision, Replayed: true}
		if durable != nil {
			if err := service.finalizeDurableResult(ctx, durable, result); err != nil {
				return CommandResult{}, err
			}
		} else {
			service.idempotency[identity] = storedResult{Fingerprint: fingerprint, Result: result}
		}
		return result, nil
	}
	if err := validateStartPreconditions(state, command); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}

	now := service.clock().UTC()
	working := cloneControlState(state)
	closeRun := working.CloseRun
	if closeRun == nil || closeRun.Status == CloseRunAborted {
		closeRunID := uuid.New()
		barrierCommandID := uuid.New()
		closeRun = &CloseRun{
			ID: closeRunID, AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
			CloseType: gl.CloseTypeInitialHardClose, Status: CloseRunInitiating, AggregateVersion: 1,
			CommandFingerprint: fingerprint, BarrierCommandID: barrierCommandID,
			BarrierIdempotencyKey: command.IdempotencyKey + ":posting-barrier",
			Steps:                 defaultCloseSteps(), SealStatus: CloseRunSealNotRequested,
			CreatedAt: now, UpdatedAt: now,
		}
		working.CloseRun = closeRun
		if err := service.repository.Commit(ctx, Mutation{
			Before: state, After: working, Operation: StartHardCloseOperation, CommandFingerprint: fingerprint,
			Audit: service.auditRecord(actor, decision, command.AccountingScopeID, closeRun.ID, StartHardCloseOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID),
		}); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		state = working
	} else {
		if closeRun.CommandFingerprint != fingerprint {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
			return CommandResult{}, ErrFiscalPeriodOwnerConflict
		}
	}

	var gateResult gl.SoftCloseGateResult
	if closeRun != nil {
		gateResult, _ = service.recoverPostingBarrierResult(ctx, command, *closeRun)
	}
	if gateResult.Operation == "" {
		gateResult, err = service.postingGate.AcquirePostingBarrier(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}, gl.AcquirePostingBarrierCommand{
			CommandID: closeRun.BarrierCommandID, AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
			ExpectedGateVersion: command.ExpectedGateVersion, SoftCloseRunID: command.SoftCloseRunID,
			SoftCloseControlEpoch: command.SoftCloseControlEpoch, SoftClosePolicyVersion: command.SoftClosePolicyVersion,
			CloseRunID: closeRun.ID, CloseType: gl.CloseTypeInitialHardClose, IdempotencyKey: closeRun.BarrierIdempotencyKey,
			CorrelationID: command.CorrelationID, CausationID: command.CausationID,
		})
	}
	if err != nil {
		failureState := cloneControlState(state)
		failureState.CloseRun.Status = CloseRunFailed
		failureState.CloseRun.LastFailureCode = barrierErrorCode(err)
		failureState.CloseRun.LastFailureDetail = err.Error()
		failureState.CloseRun.AggregateVersion++
		failureState.CloseRun.UpdatedAt = service.clock().UTC()
		if commitErr := service.repository.Commit(ctx, Mutation{Before: state, After: failureState, Operation: StartHardCloseOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, closeRun.ID, StartHardCloseOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID)}); commitErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, commitErr)
			return CommandResult{}, commitErr
		}
		classified := classifyPostingBarrierError(err)
		service.finalizeDurableFailureIfNeeded(ctx, durable, classified)
		return CommandResult{}, classified
	}

	applyCommand := ApplyPostingGateResultCommand{
		CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		SoftCloseRunID: command.SoftCloseRunID, CloseRunID: closeRun.ID, ExpectedPeriodVersion: command.ExpectedPeriodVersion,
		ExpectedGateVersion: command.ExpectedGateVersion, SoftCloseControlEpoch: command.SoftCloseControlEpoch,
		SoftClosePolicyVersion: command.SoftClosePolicyVersion, Result: gateResult,
		IdempotencyKey: command.IdempotencyKey + ":apply", CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
	result, err := service.applyPostingGateResult(ctx, actor, decision, applyCommand, false, durable)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	result.Decision = decision
	if durable == nil {
		service.idempotency[identity] = storedResult{Fingerprint: fingerprint, Result: result}
	}
	return result, nil
}

func (service *Service) ApplyPostingGateResult(ctx context.Context, actor Actor, command ApplyPostingGateResultCommand) (CommandResult, error) {
	command = command.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	if service.durable == nil {
		decision, err := service.authorizer.Authorize(ctx, actor, PermissionApplyPostingGateResult, command.AccountingScopeID)
		if err != nil {
			return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
		}
		if err := authorizeFiscalPeriodDecision(decision, PermissionApplyPostingGateResult); err != nil {
			return CommandResult{}, ErrFiscalPeriodAuthorizationDenied
		}
		return service.applyPostingGateResult(ctx, actor, decision, command, true, nil)
	}
	fingerprint := applyPostingGateResultFingerprint(command)
	durable, replay, err := service.acquireDurableCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, ApplyPostingGateResultOperation)
	if err != nil {
		return CommandResult{}, err
	}
	if replay != nil {
		return *replay, nil
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionApplyPostingGateResult, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, &durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionApplyPostingGateResult); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, &durable, err)
		return CommandResult{}, ErrFiscalPeriodAuthorizationDenied
	}
	return service.applyPostingGateResult(ctx, actor, decision, command, false, &durable)
}

func (service *Service) applyPostingGateResult(ctx context.Context, actor Actor, decision AuthorizationDecision, command ApplyPostingGateResultCommand, checkIdempotency bool, durable *durableCommand) (CommandResult, error) {
	fingerprint := applyPostingGateResultFingerprint(command)
	identity := service.idempotencyIdentity(ApplyPostingGateResultOperation, actor.UserID, command.AccountingScopeID, command.IdempotencyKey)
	if checkIdempotency {
		service.mu.Lock()
		defer service.mu.Unlock()
		if existing, ok := service.idempotency[identity]; ok {
			if existing.Fingerprint != fingerprint {
				return CommandResult{}, ErrFiscalPeriodIdempotencyConflict
			}
			existing.Result.Replayed = true
			return existing.Result, nil
		}
	}

	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := validatePostingGateResult(command, state); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if isReopenGateCloseOperation(command.Result.Operation) {
		return service.applyReopenCloseGateResult(ctx, actor, decision, command, state, checkIdempotency, durable)
	}
	if state.CloseRun.Status == CloseRunBarrierAcquired && state.CloseRun.BarrierEvidence != nil {
		if state.CloseRun.BarrierEvidence.CommandFingerprint != command.Result.Gate.LastFrozenAdmissionSummary.FrozenByCommandFingerprint {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierResultConflict)
			return CommandResult{}, ErrPostingBarrierResultConflict
		}
		result := CommandResult{State: state, GateResult: command.Result, Decision: decision, Replayed: true}
		if durable != nil {
			if err := service.finalizeDurableResult(ctx, durable, result); err != nil {
				return CommandResult{}, err
			}
		} else if checkIdempotency {
			service.idempotency[identity] = storedResult{Fingerprint: fingerprint, Result: result}
		}
		return result, nil
	}

	before := state
	after := cloneControlState(state)
	now := service.clock().UTC()
	resultGate := command.Result.Gate
	summary := resultGate.LastFrozenAdmissionSummary
	if summary == nil {
		return CommandResult{}, ErrPostingBarrierResultConflict
	}
	closeRun := after.CloseRun
	closeRun.Status = CloseRunBarrierAcquired
	closeRun.AggregateVersion++
	closeRun.UpdatedAt = now
	closeRun.LastFailureCode = ""
	closeRun.LastFailureDetail = ""
	closeRun.BarrierEvidence = &BarrierEvidence{
		Operation: command.Result.Operation, GateMode: resultGate.GateMode, GateVersion: resultGate.GateVersion,
		BarrierLedgerPosition: resultGate.BarrierLedgerPosition, PriorControlProcessID: cloneUUID(resultGate.PriorControlProcessID),
		PriorControlProcessType: resultGate.PriorControlProcessType, PriorControlEpoch: cloneInt64(resultGate.PriorControlEpoch),
		FrozenSummary: *summary, DecisionReference: command.Result.DecisionReference, PolicyReference: command.Result.PolicyReference,
		AuthorizationPolicyVersion: command.Result.AuthorizationPolicyVersion, AuditReference: command.Result.AuditReference,
		CommandFingerprint: summary.FrozenByCommandFingerprint, CommandType: summary.FrozenByCommandType,
		ApplicationCommandFingerprint: postingGateResultFingerprint(command.Result), ValidationOutcome: command.Result.ValidationOutcome, NextAction: command.Result.NextAction,
		AuthoritativeResult: canonicalPostingGateResult(command.Result),
	}
	after.SoftClose.Status = SoftCloseRunHandoffPending
	after.SoftClose.AggregateVersion++
	after.SoftClose.UpdatedAt = now
	after.ControlEpoch.Status = ControlEpochHandoffPending
	after.ControlEpoch.HandoffCloseRunID = cloneUUID(&closeRun.ID)
	after.ControlEpoch.FrozenSummary = cloneSummary(summary)
	after.ControlEpoch.AggregateVersion++
	after.ControlEpoch.UpdatedAt = now
	after.Period.State = PeriodStateClosing
	after.Period.StateVersion++
	after.Period.AggregateVersion++
	after.Period.ActiveSoftCloseRunID = nil
	after.Period.ActiveCloseRunID = cloneUUID(&closeRun.ID)
	after.Period.UpdatedAt = now

	payload, err := json.Marshal(struct {
		Period       FiscalPeriod          `json:"period"`
		SoftClose    SoftCloseRun          `json:"softClose"`
		ControlEpoch SoftCloseControlEpoch `json:"controlEpoch"`
		CloseRun     CloseRun              `json:"closeRun"`
		Gate         gl.PeriodPostingGate  `json:"gate"`
	}{after.Period, after.SoftClose, after.ControlEpoch, *after.CloseRun, resultGate})
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	events := []Event{
		{EventType: "GateAdmissionSummaryRecorded", AggregateID: closeRun.ID, AggregateVersion: closeRun.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: append([]byte(nil), payload...)},
		{EventType: "SoftCloseHandoffStarted", AggregateID: after.SoftClose.ID, AggregateVersion: after.SoftClose.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: append([]byte(nil), payload...)},
		{EventType: "PeriodStateChanged", AggregateID: after.Period.ID, AggregateVersion: after.Period.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: append([]byte(nil), payload...)},
	}
	mutation := Mutation{
		Before: before, After: after, Operation: ApplyPostingGateResultOperation, CommandFingerprint: fingerprint,
		Audit: service.auditRecord(actor, decision, command.AccountingScopeID, closeRun.ID, ApplyPostingGateResultOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID), Events: events,
	}
	result := CommandResult{State: after, GateResult: command.Result, Decision: decision}
	if durable != nil {
		committer, ok := service.repository.(DurableRepository)
		if !ok {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrInvalidFiscalPeriodService)
			return CommandResult{}, ErrInvalidFiscalPeriodService
		}
		metadata, metadataErr := service.establishedMetadata(*durable, result)
		if metadataErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, metadataErr)
			return CommandResult{}, metadataErr
		}
		err = committer.CommitWithIdempotency(ctx, mutation, DurableMutationCommit{Coordinator: service.durable.Coordinator, Acquisitions: []platformidempotency.DurableAcquisition{durable.acquisition}, Results: []platformidempotency.CommandResultMetadata{metadata}})
	} else {
		err = service.repository.Commit(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if checkIdempotency {
		service.idempotency[identity] = storedResult{Fingerprint: fingerprint, Result: result}
	}
	return result, nil
}

func validateStartPreconditions(state ControlState, command StartHardCloseCommand) error {
	if state.Period.AccountingScopeID != command.AccountingScopeID || state.Period.ID != command.FiscalPeriodID || state.SoftClose.ID != command.SoftCloseRunID {
		return ErrFiscalPeriodScopeConflict
	}
	if state.Period.State != PeriodStateSoftClosed || state.SoftClose.Status != SoftCloseRunActive || state.ControlEpoch.Status != ControlEpochActive {
		return ErrFiscalPeriodStateConflict
	}
	if state.Period.AggregateVersion != command.ExpectedPeriodVersion {
		return ErrFiscalPeriodVersionConflict
	}
	if state.SoftClose.ControlEpoch != command.SoftCloseControlEpoch || state.ControlEpoch.Number != command.SoftCloseControlEpoch {
		return ErrFiscalPeriodEpochConflict
	}
	if state.SoftClose.OpenedGateVersion != command.ExpectedGateVersion || state.ControlEpoch.OpenedGateVersion != command.ExpectedGateVersion {
		return ErrFiscalPeriodVersionConflict
	}
	if state.SoftClose.PolicyVersion != command.SoftClosePolicyVersion || state.ControlEpoch.PolicyVersion != command.SoftClosePolicyVersion {
		return ErrFiscalPeriodPolicyConflict
	}
	if state.CloseRun != nil {
		switch state.CloseRun.Status {
		case CloseRunFailed, CloseRunInitiating:
			// A non-terminal or failed initial attempt may be retried with its
			// existing process identity.
		case CloseRunAborted:
			// An authorized initial abort opens a new soft-close epoch. A later
			// hard-close attempt must receive a new CloseRun identity while the
			// aborted run remains immutable history.
			if state.Period.State != PeriodStateSoftClosed || state.Period.ActiveCloseRunID != nil || state.SoftClose.Status != SoftCloseRunActive || state.ControlEpoch.Status != ControlEpochActive {
				return ErrFiscalPeriodOwnerConflict
			}
		default:
			return ErrFiscalPeriodOwnerConflict
		}
	}
	return nil
}

func validatePostingGateResult(command ApplyPostingGateResultCommand, state ControlState) error {
	if state.Period.AccountingScopeID != command.AccountingScopeID || state.Period.ID != command.FiscalPeriodID {
		return ErrFiscalPeriodScopeConflict
	}
	if isReopenGateCloseOperation(command.Result.Operation) {
		return validateReopenCloseGateResult(command, state)
	}
	if state.Period.State != PeriodStateSoftClosed || state.SoftClose.Status != SoftCloseRunActive || state.ControlEpoch.Status != ControlEpochActive {
		if state.CloseRun != nil && state.CloseRun.Status == CloseRunBarrierAcquired {
			return validateAppliedPostingGateResult(command, state)
		}
		return ErrFiscalPeriodStateConflict
	}
	if state.Period.AggregateVersion != command.ExpectedPeriodVersion {
		return ErrFiscalPeriodVersionConflict
	}
	if state.SoftClose.ID != command.SoftCloseRunID || state.SoftClose.ControlEpoch != command.SoftCloseControlEpoch || state.ControlEpoch.Number != command.SoftCloseControlEpoch {
		return ErrFiscalPeriodEpochConflict
	}
	if state.SoftClose.PolicyVersion != command.SoftClosePolicyVersion || state.ControlEpoch.PolicyVersion != command.SoftClosePolicyVersion {
		return ErrFiscalPeriodPolicyConflict
	}
	if state.CloseRun == nil || state.CloseRun.ID != command.CloseRunID {
		return ErrFiscalPeriodOwnerConflict
	}
	gate := command.Result.Gate
	if err := gate.Validate(); err != nil {
		return ErrPostingBarrierResultConflict
	}
	if command.Result.Operation != gl.PostingBarrierAcquireOperation || gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID || gate.GateMode != gl.PostingGateCloseOnly || gate.GateVersion != command.ExpectedGateVersion+1 || gate.ActiveCloseRunID == nil || *gate.ActiveCloseRunID != command.CloseRunID || gate.ActiveCloseType != gl.CloseTypeInitialHardClose || gate.ActiveSoftCloseRunID != nil || gate.PriorControlProcessID == nil || *gate.PriorControlProcessID != command.SoftCloseRunID || gate.PriorControlProcessType != gl.SoftCloseGateOwnerType || gate.PriorControlEpoch == nil || *gate.PriorControlEpoch != command.SoftCloseControlEpoch || gate.PriorGateMode != gl.PostingGateSoftClosePolicy {
		return ErrPostingBarrierResultConflict
	}
	if gate.LastFrozenAdmissionSummary == nil || gate.LastFrozenAdmissionSummary.ControlOwnerType != gl.SoftCloseGateOwnerType || gate.LastFrozenAdmissionSummary.ControlOwnerID != command.SoftCloseRunID || gate.LastFrozenAdmissionSummary.ControlOwnerEpoch == nil || *gate.LastFrozenAdmissionSummary.ControlOwnerEpoch != command.SoftCloseControlEpoch || gate.LastFrozenAdmissionSummary.FrozenAtGateVersion != gate.GateVersion || gate.LastFrozenAdmissionSummary.FrozenByCommandType != gl.PostingBarrierAcquireCommandIdentity || strings.TrimSpace(gate.LastFrozenAdmissionSummary.FrozenByCommandFingerprint) == "" {
		return ErrPostingBarrierResultConflict
	}
	if gate.ActiveAdmissionCounters.ControlOwnerType != gl.PostingBarrierOwnerType || gate.ActiveAdmissionCounters.ControlOwnerID != command.CloseRunID {
		return ErrPostingBarrierResultConflict
	}
	return nil
}

func validateAppliedPostingGateResult(command ApplyPostingGateResultCommand, state ControlState) error {
	if state.CloseRun == nil || state.CloseRun.BarrierEvidence == nil || state.CloseRun.ID != command.CloseRunID {
		return ErrFiscalPeriodOwnerConflict
	}
	gate := command.Result.Gate
	if err := gate.Validate(); err != nil {
		return ErrPostingBarrierResultConflict
	}
	if command.Result.Operation != gl.PostingBarrierAcquireOperation || gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID || gate.GateMode != gl.PostingGateCloseOnly || gate.ActiveCloseRunID == nil || *gate.ActiveCloseRunID != command.CloseRunID || gate.ActiveCloseType != gl.CloseTypeInitialHardClose || gate.ActiveSoftCloseRunID != nil || gate.PriorControlProcessID == nil || *gate.PriorControlProcessID != command.SoftCloseRunID || gate.PriorControlProcessType != gl.SoftCloseGateOwnerType || gate.PriorControlEpoch == nil || *gate.PriorControlEpoch != command.SoftCloseControlEpoch || gate.PriorGateMode != gl.PostingGateSoftClosePolicy || gate.LastFrozenAdmissionSummary == nil || gate.LastFrozenAdmissionSummary.ControlOwnerType != gl.SoftCloseGateOwnerType || gate.LastFrozenAdmissionSummary.ControlOwnerID != command.SoftCloseRunID || gate.LastFrozenAdmissionSummary.ControlOwnerEpoch == nil || *gate.LastFrozenAdmissionSummary.ControlOwnerEpoch != command.SoftCloseControlEpoch || gate.LastFrozenAdmissionSummary.FrozenAtGateVersion != gate.GateVersion || gate.LastFrozenAdmissionSummary.FrozenByCommandType != gl.PostingBarrierAcquireCommandIdentity || strings.TrimSpace(gate.LastFrozenAdmissionSummary.FrozenByCommandFingerprint) == "" || gate.ActiveAdmissionCounters.ControlOwnerType != gl.PostingBarrierOwnerType || gate.ActiveAdmissionCounters.ControlOwnerID != command.CloseRunID {
		return ErrPostingBarrierResultConflict
	}
	if gate.GateVersion != state.CloseRun.BarrierEvidence.GateVersion || gate.LastFrozenAdmissionSummary.FrozenByCommandFingerprint != state.CloseRun.BarrierEvidence.CommandFingerprint {
		return ErrPostingBarrierResultConflict
	}
	if state.CloseRun.BarrierEvidence.ApplicationCommandFingerprint != "" && postingGateResultFingerprint(command.Result) != state.CloseRun.BarrierEvidence.ApplicationCommandFingerprint {
		return ErrPostingBarrierResultConflict
	}
	return nil
}

func authorizeFiscalPeriodDecision(decision AuthorizationDecision, permission string) error {
	if !decision.Allowed || decision.Permission != permission || decision.DecisionReference == uuid.Nil {
		return ErrFiscalPeriodAuthorizationDenied
	}
	return nil
}

func (service *Service) auditRecord(actor Actor, decision AuthorizationDecision, scopeID, aggregateID uuid.UUID, action, before, after, correlation, causation string) AuditRecord {
	return AuditRecord{AggregateID: aggregateID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: action, AccountingScopeID: scopeID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, RevisionVersion: 1, BeforeFingerprint: before, AfterFingerprint: after, CorrelationID: correlation, CausationID: causation}
}

func startHardCloseFingerprint(command StartHardCloseCommand) string {
	return fingerprint(struct {
		Operation             string
		AccountingScopeID     uuid.UUID
		FiscalPeriodID        uuid.UUID
		SoftCloseRunID        uuid.UUID
		ExpectedPeriodVersion int64
		ExpectedGateVersion   int64
		ControlEpoch          int64
		PolicyVersion         string
	}{StartHardCloseCommandIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.SoftCloseRunID, command.ExpectedPeriodVersion, command.ExpectedGateVersion, command.SoftCloseControlEpoch, command.SoftClosePolicyVersion})
}

func applyPostingGateResultFingerprint(command ApplyPostingGateResultCommand) string {
	return fingerprint(struct {
		Operation                    string
		AccountingScopeID            uuid.UUID
		FiscalPeriodID               uuid.UUID
		SoftCloseRunID               uuid.UUID
		CloseRunID                   uuid.UUID
		ReopenRequestID              uuid.UUID
		ExpectedPeriodVersion        int64
		ExpectedGateVersion          int64
		ExpectedReopenRequestVersion int64
		ControlEpoch                 int64
		PolicyVersion                string
		GateVersion                  int64
		CommandFingerprint           string
	}{ApplyPostingGateResultCommandIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.SoftCloseRunID, command.CloseRunID, command.ReopenRequestID, command.ExpectedPeriodVersion, command.ExpectedGateVersion, command.ExpectedReopenRequestVersion, command.SoftCloseControlEpoch, command.SoftClosePolicyVersion, command.Result.Gate.GateVersion, frozenFingerprint(command.Result)})
}

func postingGateResultFingerprint(result gl.SoftCloseGateResult) string {
	return fingerprint(canonicalPostingGateResult(result))
}

func canonicalPostingGateResult(result gl.SoftCloseGateResult) gl.SoftCloseGateResult {
	result.Replayed = false
	return result
}

func frozenFingerprint(result gl.SoftCloseGateResult) string {
	if result.Gate.LastFrozenAdmissionSummary == nil {
		return ""
	}
	return result.Gate.LastFrozenAdmissionSummary.FrozenByCommandFingerprint
}

func fingerprint(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "sha256:invalid"
	}
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func (service *Service) idempotencyIdentity(operation string, actorID, scopeID uuid.UUID, key string) string {
	return operation + ":" + actorID.String() + ":" + scopeID.String() + ":" + key
}

func barrierErrorCode(err error) string {
	switch {
	case errors.Is(err, gl.ErrSoftCloseGateVersionConflict):
		return "GATE_VERSION_CONFLICT"
	case errors.Is(err, gl.ErrSoftCloseGateOwnerConflict):
		return "GATE_OWNER_CONFLICT"
	case errors.Is(err, gl.ErrSoftCloseGateEpochConflict):
		return "GATE_EPOCH_CONFLICT"
	case errors.Is(err, gl.ErrSoftCloseGatePolicyConflict):
		return "GATE_POLICY_CONFLICT"
	default:
		return "GATE_DEPENDENCY_UNAVAILABLE"
	}
}

func classifyPostingBarrierError(err error) error {
	if errors.Is(err, gl.ErrSoftCloseGateVersionConflict) || errors.Is(err, gl.ErrSoftCloseGateScopeConflict) || errors.Is(err, gl.ErrSoftCloseGateModeConflict) || errors.Is(err, gl.ErrSoftCloseGateOwnerConflict) || errors.Is(err, gl.ErrSoftCloseGateEpochConflict) || errors.Is(err, gl.ErrSoftCloseGatePolicyConflict) || errors.Is(err, gl.ErrSoftCloseGatePolicyRejected) || errors.Is(err, gl.ErrSoftCloseGateAuthorizationDenied) || errors.Is(err, gl.ErrPostingBarrierNotReleasable) || errors.Is(err, gl.ErrPostingBarrierAdmissionConflict) || errors.Is(err, gl.ErrPostingBarrierWatermarkConflict) || errors.Is(err, gl.ErrPostingBarrierFinalized) {
		return fmt.Errorf("%w: %v", ErrPostingBarrierRejected, err)
	}
	return fmt.Errorf("%w: %v", ErrPostingBarrierUnavailable, err)
}

func evidenceResult(evidence CloseRun) gl.SoftCloseGateResult {
	if evidence.BarrierEvidence == nil {
		return gl.SoftCloseGateResult{}
	}
	if evidence.BarrierEvidence.AuthoritativeResult.Gate.ID != uuid.Nil {
		return canonicalPostingGateResult(evidence.BarrierEvidence.AuthoritativeResult)
	}
	closeType := evidence.CloseType
	if closeType == "" {
		closeType = gl.CloseTypeInitialHardClose
	}
	return gl.SoftCloseGateResult{
		Operation:         evidence.BarrierEvidence.Operation,
		Gate:              gl.PeriodPostingGate{AccountingScopeID: evidence.AccountingScopeID, FiscalPeriodID: evidence.FiscalPeriodID, GateMode: evidence.BarrierEvidence.GateMode, GateVersion: evidence.BarrierEvidence.GateVersion, ActiveCloseRunID: cloneUUID(&evidence.ID), ActiveCloseType: closeType, BarrierLedgerPosition: evidence.BarrierEvidence.BarrierLedgerPosition, PriorControlProcessID: cloneUUID(evidence.BarrierEvidence.PriorControlProcessID), PriorControlProcessType: evidence.BarrierEvidence.PriorControlProcessType, PriorControlEpoch: cloneInt64(evidence.BarrierEvidence.PriorControlEpoch), PriorGateMode: evidence.BarrierEvidence.AuthoritativeResult.Gate.PriorGateMode, LastFrozenAdmissionSummary: cloneSummary(&evidence.BarrierEvidence.FrozenSummary)},
		DecisionReference: evidence.BarrierEvidence.DecisionReference, PolicyReference: evidence.BarrierEvidence.PolicyReference, AuthorizationPolicyVersion: evidence.BarrierEvidence.AuthorizationPolicyVersion, AuditReference: evidence.BarrierEvidence.AuditReference, ValidationOutcome: evidence.BarrierEvidence.ValidationOutcome, NextAction: evidence.BarrierEvidence.NextAction,
	}
}

func (service *Service) recoverPostingBarrierResult(ctx context.Context, command StartHardCloseCommand, closeRun CloseRun) (gl.SoftCloseGateResult, error) {
	if closeRun.Status != CloseRunInitiating && closeRun.Status != CloseRunFailed {
		return gl.SoftCloseGateResult{}, nil
	}
	status, err := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		return gl.SoftCloseGateResult{}, err
	}
	if err := status.Validate(); err != nil {
		return gl.SoftCloseGateResult{}, err
	}
	if status.AccountingScopeID != command.AccountingScopeID || status.FiscalPeriodID != command.FiscalPeriodID || status.GateMode != gl.PostingGateCloseOnly || status.ActiveCloseRunID == nil || *status.ActiveCloseRunID != closeRun.ID || status.ActiveCloseType != gl.CloseTypeInitialHardClose || status.ActiveSoftCloseRunID != nil || status.PriorControlProcessID == nil || *status.PriorControlProcessID != command.SoftCloseRunID || status.PriorControlProcessType != gl.SoftCloseGateOwnerType || status.PriorControlEpoch == nil || *status.PriorControlEpoch != command.SoftCloseControlEpoch || status.PriorGateMode != gl.PostingGateSoftClosePolicy || status.LastFrozenAdmissionSummary == nil || status.LastFrozenAdmissionSummary.ControlOwnerType != gl.SoftCloseGateOwnerType || status.LastFrozenAdmissionSummary.ControlOwnerID != command.SoftCloseRunID || status.LastFrozenAdmissionSummary.ControlOwnerEpoch == nil || *status.LastFrozenAdmissionSummary.ControlOwnerEpoch != command.SoftCloseControlEpoch || status.LastFrozenAdmissionSummary.FrozenAtGateVersion != status.GateVersion || status.LastFrozenAdmissionSummary.FrozenByCommandType != gl.PostingBarrierAcquireCommandIdentity || strings.TrimSpace(status.LastFrozenAdmissionSummary.FrozenByCommandFingerprint) == "" || status.ActiveAdmissionCounters.ControlOwnerType != gl.PostingBarrierOwnerType || status.ActiveAdmissionCounters.ControlOwnerID != closeRun.ID {
		return gl.SoftCloseGateResult{}, nil
	}
	return gl.SoftCloseGateResult{
		Operation:                  gl.PostingBarrierAcquireOperation,
		Gate:                       status,
		PolicyReference:            "reconciled-from-gl-status",
		AuthorizationPolicyVersion: command.SoftClosePolicyVersion,
		ValidationOutcome:          "reconciled",
		NextAction:                 "continue-close",
	}, nil
}

func cloneControlState(state ControlState) ControlState {
	data, err := json.Marshal(state)
	if err != nil {
		return state
	}
	var clone ControlState
	if err := json.Unmarshal(data, &clone); err != nil {
		return state
	}
	return clone
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneSummary(value *gl.FrozenGateAdmissionSummary) *gl.FrozenGateAdmissionSummary {
	if value == nil {
		return nil
	}
	copy := *value
	copy.ControlOwnerEpoch = cloneInt64(value.ControlOwnerEpoch)
	return &copy
}

type MemoryRepository struct {
	mu        sync.Mutex
	states    map[string]ControlState
	events    []Event
	audits    []AuditRecord
	commitErr error
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{states: make(map[string]ControlState)}
}

func (repository *MemoryRepository) Seed(state ControlState) error {
	if repository == nil {
		return ErrInvalidFiscalPeriodService
	}
	if err := state.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.states[stateKey(state.Period.AccountingScopeID, state.Period.ID)] = cloneControlState(state)
	return nil
}

func (repository *MemoryRepository) Get(_ context.Context, scopeID, periodID uuid.UUID) (ControlState, error) {
	if repository == nil {
		return ControlState{}, ErrInvalidFiscalPeriodService
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	state, ok := repository.states[stateKey(scopeID, periodID)]
	if !ok {
		return ControlState{}, ErrFiscalPeriodNotFound
	}
	return cloneControlState(state), nil
}

func (repository *MemoryRepository) Commit(_ context.Context, mutation Mutation) error {
	if repository == nil {
		return ErrInvalidFiscalPeriodService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if repository.commitErr != nil {
		return repository.commitErr
	}
	key := stateKey(mutation.After.Period.AccountingScopeID, mutation.After.Period.ID)
	current, ok := repository.states[key]
	if !ok {
		return ErrFiscalPeriodNotFound
	}
	if !sameVersions(current, mutation.Before) {
		return ErrFiscalPeriodVersionConflict
	}
	if err := validateReopenRequestMutation(mutation.Before.ReopenRequest, mutation.After.ReopenRequest); err != nil {
		return err
	}
	repository.states[key] = cloneControlState(mutation.After)
	repository.audits = append(repository.audits, mutation.Audit)
	repository.events = append(repository.events, cloneEvents(mutation.Events)...)
	return nil
}

func (repository *MemoryRepository) SetCommitError(err error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.commitErr = err
}

func (repository *MemoryRepository) Events() []Event {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return cloneEvents(repository.events)
}

func (repository *MemoryRepository) Audits() []AuditRecord {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return append([]AuditRecord(nil), repository.audits...)
}

func stateKey(scopeID, periodID uuid.UUID) string { return scopeID.String() + ":" + periodID.String() }

func sameVersions(left, right ControlState) bool {
	if left.Period.ID != right.Period.ID || left.Period.AggregateVersion != right.Period.AggregateVersion || left.SoftClose.ID != right.SoftClose.ID || left.SoftClose.AggregateVersion != right.SoftClose.AggregateVersion || left.ControlEpoch.AggregateVersion != right.ControlEpoch.AggregateVersion {
		return false
	}
	if (left.CloseRun == nil) != (right.CloseRun == nil) {
		return false
	}
	if left.CloseRun != nil && (left.CloseRun.ID != right.CloseRun.ID || left.CloseRun.AggregateVersion != right.CloseRun.AggregateVersion) {
		return false
	}
	if (left.ReopenRequest == nil) != (right.ReopenRequest == nil) {
		return false
	}
	if left.ReopenRequest != nil && (left.ReopenRequest.ID != right.ReopenRequest.ID || left.ReopenRequest.AggregateVersion != right.ReopenRequest.AggregateVersion) {
		return false
	}
	return true
}

func cloneEvents(events []Event) []Event {
	result := make([]Event, len(events))
	copy(result, events)
	for index := range result {
		result[index].Payload = append([]byte(nil), result[index].Payload...)
	}
	return result
}

var _ Repository = (*MemoryRepository)(nil)
var _ PostingGateApplicationPort = (gl.PostingGateApplicationPort)(nil)
