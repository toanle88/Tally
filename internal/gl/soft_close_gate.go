package gl

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
	platformevents "github.com/toanle88/Tally/internal/platform/events"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	SoftCloseGateEnterPermission     = "finance.gl.enter.soft.close.gate"
	SoftCloseGateExitPermission      = "finance.gl.exit.soft.close.gate"
	PostingBarrierAcquirePermission  = "finance.gl.acquire.posting.barrier"
	PostingBarrierReleasePermission  = "finance.gl.release.posting.barrier"
	PostingBarrierFinalizePermission = "finance.gl.finalize.posting.gate"

	SoftCloseGateEnterOperation     = "enter-soft-close-gate"
	SoftCloseGateExitOperation      = "exit-soft-close-gate"
	PostingBarrierAcquireOperation  = "acquire-posting-barrier"
	PostingBarrierReleaseOperation  = "release-posting-barrier"
	PostingBarrierFinalizeOperation = "finalize-posting-gate"

	SoftCloseGateEnterCommandIdentity     = "enter-soft-close-gate.v1"
	SoftCloseGateExitCommandIdentity      = "exit-soft-close-gate.v1"
	PostingBarrierAcquireCommandIdentity  = "acquire-posting-barrier.v1"
	PostingBarrierReleaseCommandIdentity  = "release-posting-barrier.v1"
	PostingBarrierFinalizeCommandIdentity = "finalize-posting-gate.v1"

	SoftCloseGateEnteredEventType   = "soft_close_gate_entered"
	SoftCloseGateExitedEventType    = "soft_close_gate_exited"
	PostingBarrierAcquiredEventType = "posting_barrier_acquired"
	PostingBarrierReleasedEventType = "posting_barrier_released"
	PostingGateFinalizedEventType   = "posting_gate_finalized"
	SoftCloseGateOwnerType          = "SoftCloseRun"
	PostingBarrierOwnerType         = "CloseRun"
	CloseTypeInitialHardClose       = "InitialHardClose"
	CloseTypeReclose                = "Reclose"
)

var (
	ErrInvalidSoftCloseGateCommand           = errors.New("invalid soft-close gate command")
	ErrInvalidSoftCloseGateService           = errors.New("invalid soft-close gate service")
	ErrSoftCloseGateNotFound                 = errors.New("soft-close gate not found")
	ErrSoftCloseGateReferenceUnavailable     = errors.New("soft-close gate reference unavailable")
	ErrSoftCloseGateVersionConflict          = errors.New("soft-close gate version conflict")
	ErrSoftCloseGateScopeConflict            = errors.New("soft-close gate scope conflict")
	ErrSoftCloseGateModeConflict             = errors.New("soft-close gate mode conflict")
	ErrSoftCloseGateOwnerConflict            = errors.New("soft-close gate owner conflict")
	ErrSoftCloseGateEpochConflict            = errors.New("soft-close gate control epoch conflict")
	ErrSoftCloseGatePolicyConflict           = errors.New("soft-close gate policy conflict")
	ErrSoftCloseGateAuthorizationDenied      = errors.New("soft-close gate authorization denied")
	ErrSoftCloseGateAuthorizationUnavailable = errors.New("soft-close gate authorization unavailable")
	ErrSoftCloseGateAuthorizationStale       = errors.New("soft-close gate authorization is stale")
	ErrSoftCloseGateAuthorizationExpired     = errors.New("soft-close gate authorization has expired")
	ErrSoftCloseGatePolicyUnavailable        = errors.New("soft-close gate policy evaluator unavailable")
	ErrSoftCloseGatePolicyRejected           = errors.New("soft-close gate policy rejected")
	ErrSoftCloseGateAuditUnavailable         = errors.New("soft-close gate audit unavailable")
	ErrSoftCloseGateOutboxUnavailable        = errors.New("soft-close gate outbox unavailable")
	ErrSoftCloseGateIdempotencyConflict      = errors.New("soft-close gate idempotency conflict")
	ErrSoftCloseGateCommandInProgress        = errors.New("soft-close gate command is already in progress")
	ErrSoftCloseGateDurableCommandFailed     = errors.New("durable soft-close gate command failed")
	ErrSoftCloseGateRecloseSummaryConflict   = errors.New("reclose handoff summary conflict")
	ErrPostingBarrierNotReleasable           = errors.New("posting barrier is not releasable")
	ErrPostingBarrierAdmissionConflict       = errors.New("posting barrier has admitted close postings")
	ErrPostingBarrierWatermarkConflict       = errors.New("posting gate watermark conflict")
	ErrPostingBarrierFinalized               = errors.New("posting gate is already finalized")
)

// SoftCloseGateValidationError keeps operation-specific field evidence at the
// GL boundary without introducing a public request schema.
type SoftCloseGateValidationError struct {
	Issues []PostingIssue
}

func (e *SoftCloseGateValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return ErrInvalidSoftCloseGateCommand.Error()
	}
	return fmt.Sprintf("%s: %s", ErrInvalidSoftCloseGateCommand, e.Issues[0].Message)
}

func (e *SoftCloseGateValidationError) Unwrap() error { return ErrInvalidSoftCloseGateCommand }

// ActiveGateAdmissionCounters are the mutable observations for the active
// control owner. GL updates these counters in the same consistency outcome as
// each admitted controlled posting.
type ActiveGateAdmissionCounters struct {
	ControlOwnerType            string    `json:"controlOwnerType"`
	ControlOwnerID              uuid.UUID `json:"controlOwnerId"`
	ControlOwnerEpoch           *int64    `json:"controlOwnerEpoch,omitempty"`
	ClosePostingAdmitted        bool      `json:"closePostingAdmitted"`
	ReopenPostingAdmitted       bool      `json:"reopenPostingAdmitted"`
	AdmittedPostingCount        int64     `json:"admittedPostingCount"`
	FirstAdmittedLedgerPosition int64     `json:"firstAdmittedLedgerPosition"`
	LastAdmittedLedgerPosition  int64     `json:"lastAdmittedLedgerPosition"`
}

// FrozenGateAdmissionSummary is an immutable boundary snapshot retained on
// exit. It is deliberately a value object: callers cannot mutate the active
// counters through a returned result.
type FrozenGateAdmissionSummary struct {
	ControlOwnerType            string    `json:"controlOwnerType"`
	ControlOwnerID              uuid.UUID `json:"controlOwnerId"`
	ControlOwnerEpoch           *int64    `json:"controlOwnerEpoch,omitempty"`
	ClosePostingAdmitted        bool      `json:"closePostingAdmitted"`
	ReopenPostingAdmitted       bool      `json:"reopenPostingAdmitted"`
	AdmittedPostingCount        int64     `json:"admittedPostingCount"`
	FirstAdmittedLedgerPosition int64     `json:"firstAdmittedLedgerPosition"`
	LastAdmittedLedgerPosition  int64     `json:"lastAdmittedLedgerPosition"`
	FrozenAtGateVersion         int64     `json:"frozenAtGateVersion"`
	FrozenAt                    time.Time `json:"frozenAt"`
	FrozenByCommandFingerprint  string    `json:"frozenByCommandFingerprint"`
	FrozenByCommandType         string    `json:"frozenByCommandType"`
}

type SoftCloseGatePolicy struct {
	Version      string          `json:"version"`
	Restrictions json.RawMessage `json:"restrictions"`
}

// PeriodPostingGate is owned by GL. Fiscal-period process ownership is
// represented only by the active soft-close run and control epoch; GL never
// writes the FPM schema.
type PeriodPostingGate struct {
	ID                             uuid.UUID                    `json:"id"`
	AccountingScopeID              uuid.UUID                    `json:"accountingScopeId"`
	FiscalPeriodID                 uuid.UUID                    `json:"fiscalPeriodId"`
	GateMode                       string                       `json:"gateMode"`
	PeriodStateVersion             int64                        `json:"periodStateVersion"`
	GateVersion                    int64                        `json:"gateVersion"`
	AggregateVersion               int64                        `json:"aggregateVersion"`
	NextLedgerPosition             int64                        `json:"nextLedgerPosition"`
	ActiveSoftCloseRunID           *uuid.UUID                   `json:"activeSoftCloseRunId,omitempty"`
	ActiveSoftCloseControlEpoch    int64                        `json:"activeSoftCloseControlEpoch,omitempty"`
	SoftClosePolicyVersion         string                       `json:"softClosePolicyVersion,omitempty"`
	SoftCloseRestrictions          json.RawMessage              `json:"softCloseRestrictions,omitempty"`
	ActiveReopenRequestID          *uuid.UUID                   `json:"activeReopenRequestId,omitempty"`
	ReopenAuthorizationScope       json.RawMessage              `json:"reopenAuthorizationScope,omitempty"`
	ReopenAuthorizationExpiry      *time.Time                   `json:"reopenAuthorizationExpiry,omitempty"`
	ReopenControlAuthorityEpoch    *int64                       `json:"reopenControlAuthorityEpoch,omitempty"`
	ReopenApproval                 *ApprovalDecisionReference   `json:"reopenApproval,omitempty"`
	ReopenNoChangeClosePermitted   bool                         `json:"reopenNoChangeClosePermitted,omitempty"`
	ActiveCloseRunID               *uuid.UUID                   `json:"activeCloseRunId,omitempty"`
	ActiveCloseType                string                       `json:"activeCloseType,omitempty"`
	PriorControlProcessID          *uuid.UUID                   `json:"priorControlProcessId,omitempty"`
	PriorControlProcessType        string                       `json:"priorControlProcessType,omitempty"`
	PriorControlEpoch              *int64                       `json:"priorControlEpoch,omitempty"`
	PriorGateMode                  string                       `json:"priorGateMode,omitempty"`
	BarrierLedgerPosition          int64                        `json:"barrierLedgerPosition"`
	FinalLedgerWatermark           *int64                       `json:"finalLedgerWatermark,omitempty"`
	FinalizedCloseRunID            *uuid.UUID                   `json:"finalizedCloseRunId,omitempty"`
	FinalizationCommandFingerprint string                       `json:"finalizationCommandFingerprint,omitempty"`
	ActiveAdmissionCounters        ActiveGateAdmissionCounters  `json:"activeAdmissionCounters"`
	LastFrozenAdmissionSummary     *FrozenGateAdmissionSummary  `json:"lastFrozenAdmissionSummary,omitempty"`
	FrozenAdmissionSummaryHistory  []FrozenGateAdmissionSummary `json:"frozenAdmissionSummaryHistory,omitempty"`
	CreatedAt                      time.Time                    `json:"createdAt"`
	UpdatedAt                      time.Time                    `json:"updatedAt"`
}

func (gate PeriodPostingGate) Validate() error {
	if gate.ID == uuid.Nil || gate.AccountingScopeID == uuid.Nil || gate.FiscalPeriodID == uuid.Nil {
		return ErrSoftCloseGateReferenceUnavailable
	}
	if gate.GateMode != PostingGateOpen && gate.GateMode != PostingGateSoftClosePolicy && gate.GateMode != PostingGateCloseOnly && gate.GateMode != PostingGateHardClosed && gate.GateMode != PostingGateScopedReopen && gate.GateMode != PostingGateOperationalReopen {
		return fmt.Errorf("%w: unsupported mode %q", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if gate.PeriodStateVersion < 1 || gate.GateVersion < 1 || gate.AggregateVersion < 1 || gate.NextLedgerPosition < 1 {
		return ErrSoftCloseGateVersionConflict
	}
	return nil
}

type EnterSoftCloseGateCommand struct {
	CommandID           uuid.UUID
	AccountingScopeID   uuid.UUID
	FiscalPeriodID      uuid.UUID
	ExpectedGateVersion int64
	SoftCloseRunID      uuid.UUID
	ControlEpoch        int64
	PolicyVersion       string
	Restrictions        json.RawMessage
	IdempotencyKey      string
	CorrelationID       string
	CausationID         string
}

func (command EnterSoftCloseGateCommand) Canonical() EnterSoftCloseGateCommand {
	command.PolicyVersion = strings.TrimSpace(command.PolicyVersion)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	if command.ControlEpoch == 0 {
		command.ControlEpoch = 1
	}
	if canonical, err := canonicalSoftCloseRestrictions(command.Restrictions); err == nil {
		command.Restrictions = canonical
	}
	return command
}

func (command EnterSoftCloseGateCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "accountingScopeId": command.AccountingScopeID,
		"fiscalPeriodId": command.FiscalPeriodID, "softCloseRunId": command.SoftCloseRunID,
	} {
		if value == uuid.Nil {
			issue("required", field, field+" is required")
		}
	}
	if command.ExpectedGateVersion < 1 {
		issue("invalid_version", "expectedVersion", "expectedVersion must be positive")
	}
	if command.ControlEpoch < 1 {
		issue("invalid_epoch", "controlEpoch", "controlEpoch must be positive")
	}
	if command.PolicyVersion == "" {
		issue("required", "policyVersion", "policyVersion is required")
	}
	if _, err := canonicalSoftCloseRestrictions(command.Restrictions); err != nil {
		issue("invalid_policy", "restrictions", "restrictions must be valid JSON")
	}
	if command.IdempotencyKey == "" {
		issue("required", "Idempotency-Key", "Idempotency-Key is required")
	}
	if command.CorrelationID == "" || command.CausationID == "" {
		issue("required", "correlation", "correlation and causation references are required")
	}
	if len(issues) > 0 {
		return &SoftCloseGateValidationError{Issues: issues}
	}
	return nil
}

type ExitSoftCloseGateCommand struct {
	CommandID           uuid.UUID
	AccountingScopeID   uuid.UUID
	FiscalPeriodID      uuid.UUID
	ExpectedGateVersion int64
	SoftCloseRunID      uuid.UUID
	ControlEpoch        int64
	IdempotencyKey      string
	CorrelationID       string
	CausationID         string
}

func (command ExitSoftCloseGateCommand) Canonical() ExitSoftCloseGateCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command ExitSoftCloseGateCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "accountingScopeId": command.AccountingScopeID,
		"fiscalPeriodId": command.FiscalPeriodID, "softCloseRunId": command.SoftCloseRunID,
	} {
		if value == uuid.Nil {
			issue("required", field, field+" is required")
		}
	}
	if command.ExpectedGateVersion < 1 {
		issue("invalid_version", "expectedVersion", "expectedVersion must be positive")
	}
	if command.ControlEpoch < 1 {
		issue("invalid_epoch", "controlEpoch", "controlEpoch must be positive")
	}
	if command.IdempotencyKey == "" {
		issue("required", "Idempotency-Key", "Idempotency-Key is required")
	}
	if command.CorrelationID == "" || command.CausationID == "" {
		issue("required", "correlation", "correlation and causation references are required")
	}
	if len(issues) > 0 {
		return &SoftCloseGateValidationError{Issues: issues}
	}
	return nil
}

type SoftCloseGateAuthorizer interface {
	AuthorizeSoftCloseGate(context.Context, Actor, string, uuid.UUID) (AuthorizationDecision, error)
}

type SoftClosePolicyEvaluator interface {
	ValidateSoftClosePolicy(context.Context, Actor, SoftCloseGatePolicy) error
}

type AllowAllSoftClosePolicyEvaluator struct{}

func (AllowAllSoftClosePolicyEvaluator) ValidateSoftClosePolicy(_ context.Context, _ Actor, policy SoftCloseGatePolicy) error {
	if strings.TrimSpace(policy.Version) == "" {
		return ErrSoftCloseGatePolicyRejected
	}
	if _, err := canonicalSoftCloseRestrictions(policy.Restrictions); err != nil {
		return fmt.Errorf("%w: %v", ErrSoftCloseGatePolicyRejected, err)
	}
	return nil
}

type UnavailableSoftClosePolicyEvaluator struct{}

func (UnavailableSoftClosePolicyEvaluator) ValidateSoftClosePolicy(context.Context, Actor, SoftCloseGatePolicy) error {
	return ErrSoftCloseGatePolicyUnavailable
}

type MemorySoftCloseGateAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemorySoftCloseGateAuthorizer) AuthorizeSoftCloseGate(_ context.Context, _ Actor, operation string, _ uuid.UUID) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	decision := authorizer.Decision
	if decision.Permission == "" {
		switch operation {
		case SoftCloseGateEnterOperation:
			decision.Permission = SoftCloseGateEnterPermission
		case SoftCloseGateExitOperation:
			decision.Permission = SoftCloseGateExitPermission
		case ScopedReopenGateOpenOperation:
			decision.Permission = ScopedReopenGateOpenPermission
		case ScopedReopenGateCloseOperation:
			decision.Permission = ScopedReopenGateClosePermission
		case OperationalReopenGateOpenOperation:
			decision.Permission = OperationalReopenGateOpenPermission
		case OperationalReopenGateCloseOperation:
			decision.Permission = OperationalReopenGateClosePermission
		case BeginRecloseGateOperation:
			decision.Permission = BeginRecloseGatePermission
		case PostingBarrierAcquireOperation:
			decision.Permission = PostingBarrierAcquirePermission
		case PostingBarrierReleaseOperation:
			decision.Permission = PostingBarrierReleasePermission
		case PostingBarrierFinalizeOperation:
			decision.Permission = PostingBarrierFinalizePermission
		}
	}
	return decision, nil
}

type SoftCloseGateAuditRecord struct {
	GateID                uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Action                string
	AccountingScopeID     uuid.UUID
	FiscalPeriodID        uuid.UUID
	Permission            string
	PolicyReference       string
	PolicyVersion         string
	DecisionReference     uuid.UUID
	ApprovalRequestID     uuid.UUID
	ApprovalDecisionID    uuid.UUID
	ApproverUserID        uuid.UUID
	RevisionNumber        int64
	BeforeFingerprint     string
	AfterFingerprint      string
	CommandFingerprint    string
	CorrelationID         string
	CausationID           string
}

type SoftCloseGateAuditRecorder interface {
	RecordSoftCloseGateMutation(context.Context, SoftCloseGateAuditRecord) (uuid.UUID, error)
}

type SoftCloseGateMutation struct {
	Before               PeriodPostingGate
	After                PeriodPostingGate
	Operation            string
	AdditionalEventTypes []string
	CommandID            uuid.UUID
	CommandFingerprint   string
	Actor                Actor
	Authorization        AuthorizationDecision
	Audit                SoftCloseGateAuditRecord
	Now                  time.Time
}

type SoftCloseGateRepository interface {
	GetSoftCloseGate(context.Context, uuid.UUID, uuid.UUID) (PeriodPostingGate, error)
	CommitSoftCloseGateMutation(context.Context, SoftCloseGateMutation) (SoftCloseGateResult, error)
}

type PostingGateStatusReader interface {
	GetPostingGateStatus(context.Context, uuid.UUID, uuid.UUID) (PeriodPostingGate, error)
}

type DurableSoftCloseGateCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
}

type DurableSoftCloseGateRepository interface {
	CommitSoftCloseGateMutationWithIdempotency(context.Context, SoftCloseGateMutation, DurableSoftCloseGateCommit) (SoftCloseGateResult, error)
}

type DurableSoftCloseGateServiceConfig struct {
	Database                          platformidempotency.TxBeginner
	Coordinator                       platformidempotency.DurableCoordinator
	Policy                            platformidempotency.IdempotencyPolicy
	EnterOperationID                  string
	ExitOperationID                   string
	AcquireOperationID                string
	ReleaseOperationID                string
	FinalizeOperationID               string
	OpenScopedReopenOperationID       string
	CloseScopedReopenOperationID      string
	OpenOperationalReopenOperationID  string
	CloseOperationalReopenOperationID string
	BeginRecloseOperationID           string
}

type SoftCloseGateResult struct {
	Operation                  string            `json:"operation"`
	Gate                       PeriodPostingGate `json:"gate"`
	DecisionReference          uuid.UUID         `json:"decisionReference"`
	PolicyReference            string            `json:"policyReference"`
	AuthorizationPolicyVersion string            `json:"authorizationPolicyVersion"`
	ValidationOutcome          string            `json:"validationOutcome"`
	AuditReference             uuid.UUID         `json:"auditReference"`
	NextAction                 string            `json:"nextAction"`
	Replayed                   bool              `json:"replayed,omitempty"`
}

func (result SoftCloseGateResult) AggregateID() uuid.UUID { return result.Gate.ID }

func (result SoftCloseGateResult) AggregateVersion() int64 {
	if result.Gate.AggregateVersion > 0 {
		return result.Gate.AggregateVersion
	}
	return result.Gate.GateVersion
}

type storedSoftCloseGateResult struct {
	fingerprint string
	result      SoftCloseGateResult
}

type SoftCloseGateService struct {
	repository  SoftCloseGateRepository
	authorizer  SoftCloseGateAuthorizer
	policy      SoftClosePolicyEvaluator
	clock       func() time.Time
	durable     *DurableSoftCloseGateServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedSoftCloseGateResult
}

func NewSoftCloseGateService(repository SoftCloseGateRepository, authorizer SoftCloseGateAuthorizer, policy SoftClosePolicyEvaluator, clock func() time.Time) (*SoftCloseGateService, error) {
	return newSoftCloseGateService(repository, authorizer, policy, clock, nil)
}

func NewSoftCloseGateServiceWithDurableIdempotency(repository SoftCloseGateRepository, authorizer SoftCloseGateAuthorizer, policy SoftClosePolicyEvaluator, clock func() time.Time, durable DurableSoftCloseGateServiceConfig) (*SoftCloseGateService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.EnterOperationID) == "" || strings.TrimSpace(durable.ExitOperationID) == "" {
		return nil, ErrInvalidSoftCloseGateService
	}
	return newSoftCloseGateService(repository, authorizer, policy, clock, &durable)
}

func newSoftCloseGateService(repository SoftCloseGateRepository, authorizer SoftCloseGateAuthorizer, policy SoftClosePolicyEvaluator, clock func() time.Time, durable *DurableSoftCloseGateServiceConfig) (*SoftCloseGateService, error) {
	if repository == nil || authorizer == nil || policy == nil || clock == nil {
		return nil, ErrInvalidSoftCloseGateService
	}
	return &SoftCloseGateService{repository: repository, authorizer: authorizer, policy: policy, clock: clock, durable: durable, idempotency: make(map[string]storedSoftCloseGateResult)}, nil
}

func (service *SoftCloseGateService) EnterSoftCloseGate(ctx context.Context, actor Actor, command EnterSoftCloseGateCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := enterSoftCloseGateFingerprint(command)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, SoftCloseGateEnterOperation, SoftCloseGateEnterCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, &SoftCloseGatePolicy{Version: command.PolicyVersion, Restrictions: append(json.RawMessage(nil), command.Restrictions...)}, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.enter(command, service.clock().UTC(), fingerprint)
	})
}

func (service *SoftCloseGateService) ExitSoftCloseGate(ctx context.Context, actor Actor, command ExitSoftCloseGateCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := exitSoftCloseGateFingerprint(command)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, SoftCloseGateExitOperation, SoftCloseGateExitCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.exit(command, service.clock().UTC(), fingerprint)
	})
}

// GetPostingGateStatus is the internal authoritative recovery seam. The
// public reference endpoint remains owned by the later status-read story.
func (service *SoftCloseGateService) GetPostingGateStatus(ctx context.Context, scopeID, periodID uuid.UUID) (PeriodPostingGate, error) {
	if service == nil || service.repository == nil {
		return PeriodPostingGate{}, ErrInvalidSoftCloseGateService
	}
	if reader, ok := service.repository.(PostingGateStatusReader); ok {
		return reader.GetPostingGateStatus(ctx, scopeID, periodID)
	}
	return service.repository.GetSoftCloseGate(ctx, scopeID, periodID)
}

func (service *SoftCloseGateService) execute(ctx context.Context, actor Actor, operation, commandIdentity string, commandID, scopeID, periodID uuid.UUID, idempotencyKey, correlationID, causationID, fingerprint string, policy *SoftCloseGatePolicy, transition func(PeriodPostingGate) (PeriodPostingGate, error)) (SoftCloseGateResult, error) {
	if service == nil {
		return SoftCloseGateResult{}, ErrInvalidSoftCloseGateService
	}
	if err := actor.Validate(); err != nil {
		return SoftCloseGateResult{}, ErrSoftCloseGateAuthorizationDenied
	}

	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	operationID := commandIdentity
	if service.durable != nil {
		operationID = service.durableOperationID(operation, commandIdentity)
		var err error
		durableIdentity, err = newDurableIdentity("soft-close-gate:"+operation, actor, scopeID, idempotencyKey)
		if err != nil {
			return SoftCloseGateResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), operationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return SoftCloseGateResult{}, ErrSoftCloseGateIdempotencyConflict
		}
		if err != nil {
			return SoftCloseGateResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return SoftCloseGateResult{}, ErrSoftCloseGateCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return SoftCloseGateResult{}, ErrSoftCloseGateDurableCommandFailed
			}
			result, err := decodeSoftCloseGateResult(acquisition.Result().ResultBody())
			if err != nil {
				return SoftCloseGateResult{}, err
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		key := operation + ":" + actor.UserID.String() + ":" + scopeID.String() + ":" + idempotencyKey
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return SoftCloseGateResult{}, ErrSoftCloseGateIdempotencyConflict
			}
			result := cloneSoftCloseGateResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	before, err := service.repository.GetSoftCloseGate(ctx, scopeID, periodID)
	if err != nil {
		service.finalizeSoftCloseFailure(ctx, acquisition, err)
		return SoftCloseGateResult{}, err
	}
	if before.AccountingScopeID != scopeID || before.FiscalPeriodID != periodID {
		service.finalizeSoftCloseFailure(ctx, acquisition, ErrSoftCloseGateScopeConflict)
		return SoftCloseGateResult{}, ErrSoftCloseGateScopeConflict
	}
	authorization, err := service.authorizer.AuthorizeSoftCloseGate(ctx, actor, operation, scopeID)
	if err != nil {
		service.finalizeSoftCloseFailure(ctx, acquisition, err)
		return SoftCloseGateResult{}, err
	}
	if err := authorizeSoftCloseGateDecision(authorization, operation, scopeID); err != nil {
		service.finalizeSoftCloseFailure(ctx, acquisition, err)
		return SoftCloseGateResult{}, err
	}
	if policy != nil {
		if err := service.policy.ValidateSoftClosePolicy(ctx, actor, *policy); err != nil {
			service.finalizeSoftCloseFailure(ctx, acquisition, err)
			return SoftCloseGateResult{}, err
		}
	}
	after, err := transition(clonePeriodPostingGate(before))
	if err != nil {
		service.finalizeSoftCloseFailure(ctx, acquisition, err)
		return SoftCloseGateResult{}, err
	}
	now := service.clock().UTC()
	var additionalEventTypes []string
	if operation == OperationalReopenGateCloseOperation && before.GateMode == PostingGateOperationalReopen && before.ReopenAuthorizationExpiry != nil && !now.Before(before.ReopenAuthorizationExpiry.UTC()) {
		additionalEventTypes = []string{OperationalReopenGateExpiredEventType}
	}
	approval := after.ReopenApproval
	if approval == nil {
		approval = before.ReopenApproval
	}
	audit := SoftCloseGateAuditRecord{GateID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: operation, AccountingScopeID: scopeID, FiscalPeriodID: periodID, Permission: authorization.Permission, PolicyReference: authorization.PolicyReference, PolicyVersion: authorization.PolicyVersion, DecisionReference: authorization.DecisionReference, RevisionNumber: after.GateVersion, BeforeFingerprint: fingerprintPeriodPostingGate(before), AfterFingerprint: fingerprintPeriodPostingGate(after), CommandFingerprint: fingerprint, CorrelationID: correlationID, CausationID: causationID}
	if approval != nil {
		audit.ApprovalRequestID = approval.ApprovalRequestID
		audit.ApprovalDecisionID = approval.DecisionID
		audit.ApproverUserID = approval.ApproverUserID
	}
	mutation := SoftCloseGateMutation{Before: before, After: after, Operation: operation, AdditionalEventTypes: additionalEventTypes, CommandID: commandID, CommandFingerprint: fingerprint, Actor: actor, Authorization: authorization, Audit: audit, Now: now}

	var result SoftCloseGateResult
	if service.durable != nil {
		committer, ok := service.repository.(DurableSoftCloseGateRepository)
		if !ok {
			service.finalizeSoftCloseFailure(ctx, acquisition, ErrInvalidSoftCloseGateService)
			return SoftCloseGateResult{}, ErrInvalidSoftCloseGateService
		}
		result, err = committer.CommitSoftCloseGateMutationWithIdempotency(ctx, mutation, DurableSoftCloseGateCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition})
	} else {
		result, err = service.repository.CommitSoftCloseGateMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeSoftCloseFailure(ctx, acquisition, err)
		return SoftCloseGateResult{}, err
	}
	if service.durable == nil {
		key := operation + ":" + actor.UserID.String() + ":" + scopeID.String() + ":" + idempotencyKey
		service.idempotency[key] = storedSoftCloseGateResult{fingerprint: fingerprint, result: cloneSoftCloseGateResult(result)}
	}
	return result, nil
}

func (service *SoftCloseGateService) durableOperationID(operation, fallback string) string {
	if service == nil || service.durable == nil {
		return fallback
	}
	switch operation {
	case SoftCloseGateEnterOperation:
		if strings.TrimSpace(service.durable.EnterOperationID) != "" {
			return service.durable.EnterOperationID
		}
	case SoftCloseGateExitOperation:
		if strings.TrimSpace(service.durable.ExitOperationID) != "" {
			return service.durable.ExitOperationID
		}
	case ScopedReopenGateOpenOperation:
		if strings.TrimSpace(service.durable.OpenScopedReopenOperationID) != "" {
			return service.durable.OpenScopedReopenOperationID
		}
	case ScopedReopenGateCloseOperation:
		if strings.TrimSpace(service.durable.CloseScopedReopenOperationID) != "" {
			return service.durable.CloseScopedReopenOperationID
		}
	case OperationalReopenGateOpenOperation:
		if strings.TrimSpace(service.durable.OpenOperationalReopenOperationID) != "" {
			return service.durable.OpenOperationalReopenOperationID
		}
	case OperationalReopenGateCloseOperation:
		if strings.TrimSpace(service.durable.CloseOperationalReopenOperationID) != "" {
			return service.durable.CloseOperationalReopenOperationID
		}
	case BeginRecloseGateOperation:
		if strings.TrimSpace(service.durable.BeginRecloseOperationID) != "" {
			return service.durable.BeginRecloseOperationID
		}
	case PostingBarrierAcquireOperation:
		if strings.TrimSpace(service.durable.AcquireOperationID) != "" {
			return service.durable.AcquireOperationID
		}
	case PostingBarrierReleaseOperation:
		if strings.TrimSpace(service.durable.ReleaseOperationID) != "" {
			return service.durable.ReleaseOperationID
		}
	case PostingBarrierFinalizeOperation:
		if strings.TrimSpace(service.durable.FinalizeOperationID) != "" {
			return service.durable.FinalizeOperationID
		}
	}
	return fallback
}

func authorizeSoftCloseGateDecision(decision AuthorizationDecision, operation string, scopeID uuid.UUID) error {
	expectedPermission := ""
	switch operation {
	case SoftCloseGateEnterOperation:
		expectedPermission = SoftCloseGateEnterPermission
	case SoftCloseGateExitOperation:
		expectedPermission = SoftCloseGateExitPermission
	case ScopedReopenGateOpenOperation:
		expectedPermission = ScopedReopenGateOpenPermission
	case ScopedReopenGateCloseOperation:
		expectedPermission = ScopedReopenGateClosePermission
	case OperationalReopenGateOpenOperation:
		expectedPermission = OperationalReopenGateOpenPermission
	case OperationalReopenGateCloseOperation:
		expectedPermission = OperationalReopenGateClosePermission
	case BeginRecloseGateOperation:
		expectedPermission = BeginRecloseGatePermission
	case PostingBarrierAcquireOperation:
		expectedPermission = PostingBarrierAcquirePermission
	case PostingBarrierReleaseOperation:
		expectedPermission = PostingBarrierReleasePermission
	case PostingBarrierFinalizeOperation:
		expectedPermission = PostingBarrierFinalizePermission
	default:
		return ErrSoftCloseGateAuthorizationDenied
	}
	if !decision.Allowed || decision.Permission != expectedPermission || decision.DecisionReference == uuid.Nil {
		return ErrSoftCloseGateAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scopeID {
			return nil
		}
	}
	return ErrSoftCloseGateAuthorizationDenied
}

func (gate PeriodPostingGate) enter(command EnterSoftCloseGateCommand, now time.Time, fingerprint string) (PeriodPostingGate, error) {
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	if gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID {
		return PeriodPostingGate{}, ErrSoftCloseGateScopeConflict
	}
	if gate.GateVersion != command.ExpectedGateVersion {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateVersionConflict, command.ExpectedGateVersion, gate.GateVersion)
	}
	if gate.GateMode != PostingGateOpen {
		if gate.GateMode == PostingGateSoftClosePolicy {
			if gate.ActiveSoftCloseRunID != nil && *gate.ActiveSoftCloseRunID != command.SoftCloseRunID {
				return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
			}
			if gate.ActiveSoftCloseControlEpoch != command.ControlEpoch {
				return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateEpochConflict, command.ControlEpoch, gate.ActiveSoftCloseControlEpoch)
			}
			canonicalRestrictions, err := canonicalSoftCloseRestrictions(command.Restrictions)
			if err != nil {
				return PeriodPostingGate{}, fmt.Errorf("%w: restrictions: %v", ErrSoftCloseGatePolicyRejected, err)
			}
			currentRestrictions, err := canonicalSoftCloseRestrictions(gate.SoftCloseRestrictions)
			if err != nil || gate.SoftClosePolicyVersion != command.PolicyVersion || string(currentRestrictions) != string(canonicalRestrictions) {
				return PeriodPostingGate{}, ErrSoftCloseGatePolicyConflict
			}
		}
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not eligible for entry", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	canonicalRestrictions, err := canonicalSoftCloseRestrictions(command.Restrictions)
	if err != nil {
		return PeriodPostingGate{}, fmt.Errorf("%w: restrictions: %v", ErrSoftCloseGatePolicyRejected, err)
	}
	gate.GateMode = PostingGateSoftClosePolicy
	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	runID := command.SoftCloseRunID
	gate.ActiveSoftCloseRunID = &runID
	gate.ActiveSoftCloseControlEpoch = command.ControlEpoch
	gate.SoftClosePolicyVersion = command.PolicyVersion
	gate.SoftCloseRestrictions = canonicalRestrictions
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{ControlOwnerType: SoftCloseGateOwnerType, ControlOwnerID: runID, ControlOwnerEpoch: int64Pointer(command.ControlEpoch)}
	gate.UpdatedAt = now
	_ = fingerprint // retained in the command event/audit identity
	return gate, nil
}

func (gate PeriodPostingGate) exit(command ExitSoftCloseGateCommand, now time.Time, fingerprint string) (PeriodPostingGate, error) {
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	if gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID {
		return PeriodPostingGate{}, ErrSoftCloseGateScopeConflict
	}
	if gate.GateVersion != command.ExpectedGateVersion {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateVersionConflict, command.ExpectedGateVersion, gate.GateVersion)
	}
	if gate.GateMode != PostingGateSoftClosePolicy {
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not soft-close", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if gate.ActiveSoftCloseRunID == nil || *gate.ActiveSoftCloseRunID != command.SoftCloseRunID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if gate.ActiveSoftCloseControlEpoch != command.ControlEpoch {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateEpochConflict, command.ControlEpoch, gate.ActiveSoftCloseControlEpoch)
	}

	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	summary := FrozenGateAdmissionSummary{
		ControlOwnerType:            gate.ActiveAdmissionCounters.ControlOwnerType,
		ControlOwnerID:              gate.ActiveAdmissionCounters.ControlOwnerID,
		ControlOwnerEpoch:           gate.ActiveAdmissionCounters.ControlOwnerEpoch,
		ClosePostingAdmitted:        gate.ActiveAdmissionCounters.ClosePostingAdmitted,
		ReopenPostingAdmitted:       gate.ActiveAdmissionCounters.ReopenPostingAdmitted,
		AdmittedPostingCount:        gate.ActiveAdmissionCounters.AdmittedPostingCount,
		FirstAdmittedLedgerPosition: gate.ActiveAdmissionCounters.FirstAdmittedLedgerPosition,
		LastAdmittedLedgerPosition:  gate.ActiveAdmissionCounters.LastAdmittedLedgerPosition,
		FrozenAtGateVersion:         gate.GateVersion,
		FrozenAt:                    now,
		FrozenByCommandFingerprint:  fingerprint,
		FrozenByCommandType:         SoftCloseGateExitCommandIdentity,
	}
	gate.LastFrozenAdmissionSummary = &summary
	gate.FrozenAdmissionSummaryHistory = append(gate.FrozenAdmissionSummaryHistory, summary)
	gate.GateMode = PostingGateOpen
	gate.ActiveSoftCloseRunID = nil
	gate.ActiveSoftCloseControlEpoch = 0
	gate.SoftClosePolicyVersion = ""
	gate.SoftCloseRestrictions = json.RawMessage(`{}`)
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{}
	gate.UpdatedAt = now
	return gate, nil
}

func enterSoftCloseGateFingerprint(command EnterSoftCloseGateCommand) (string, error) {
	command = command.Canonical()
	restrictions, err := canonicalSoftCloseRestrictions(command.Restrictions)
	if err != nil {
		return "", err
	}
	return softCloseGateFingerprint(struct {
		Operation           string          `json:"operation"`
		AccountingScopeID   uuid.UUID       `json:"accountingScopeId"`
		FiscalPeriodID      uuid.UUID       `json:"fiscalPeriodId"`
		ExpectedGateVersion int64           `json:"expectedGateVersion"`
		SoftCloseRunID      uuid.UUID       `json:"softCloseRunId"`
		ControlEpoch        int64           `json:"controlEpoch"`
		PolicyVersion       string          `json:"policyVersion"`
		Restrictions        json.RawMessage `json:"restrictions"`
	}{SoftCloseGateEnterOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.SoftCloseRunID, command.ControlEpoch, command.PolicyVersion, restrictions})
}

func exitSoftCloseGateFingerprint(command ExitSoftCloseGateCommand) (string, error) {
	return softCloseGateFingerprint(struct {
		Operation           string    `json:"operation"`
		AccountingScopeID   uuid.UUID `json:"accountingScopeId"`
		FiscalPeriodID      uuid.UUID `json:"fiscalPeriodId"`
		ExpectedGateVersion int64     `json:"expectedGateVersion"`
		SoftCloseRunID      uuid.UUID `json:"softCloseRunId"`
		ControlEpoch        int64     `json:"controlEpoch"`
	}{SoftCloseGateExitOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.SoftCloseRunID, command.ControlEpoch})
}

func softCloseGateFingerprint(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := platformevents.CanonicalizePayload(data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func canonicalSoftCloseRestrictions(value []byte) (json.RawMessage, error) {
	if len(value) == 0 {
		value = []byte(`{}`)
	}
	canonical, err := platformevents.CanonicalizePayload(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func fingerprintPeriodPostingGate(gate PeriodPostingGate) string {
	data, err := json.Marshal(gate)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func decodeSoftCloseGateResult(body []byte) (SoftCloseGateResult, error) {
	var result SoftCloseGateResult
	if err := json.Unmarshal(body, &result); err != nil {
		return SoftCloseGateResult{}, ErrSoftCloseGateCommandInProgress
	}
	return result, nil
}

func cloneSoftCloseGateResult(result SoftCloseGateResult) SoftCloseGateResult {
	result.Gate = clonePeriodPostingGate(result.Gate)
	return result
}

func clonePeriodPostingGate(gate PeriodPostingGate) PeriodPostingGate {
	if gate.ActiveSoftCloseRunID != nil {
		value := *gate.ActiveSoftCloseRunID
		gate.ActiveSoftCloseRunID = &value
	}
	if gate.ActiveCloseRunID != nil {
		value := *gate.ActiveCloseRunID
		gate.ActiveCloseRunID = &value
	}
	if gate.ActiveReopenRequestID != nil {
		value := *gate.ActiveReopenRequestID
		gate.ActiveReopenRequestID = &value
	}
	if gate.PriorControlProcessID != nil {
		value := *gate.PriorControlProcessID
		gate.PriorControlProcessID = &value
	}
	gate.PriorControlEpoch = cloneInt64Pointer(gate.PriorControlEpoch)
	if gate.ReopenAuthorizationExpiry != nil {
		value := gate.ReopenAuthorizationExpiry.UTC()
		gate.ReopenAuthorizationExpiry = &value
	}
	gate.ReopenControlAuthorityEpoch = cloneInt64Pointer(gate.ReopenControlAuthorityEpoch)
	gate.ReopenApproval = cloneApproval(gate.ReopenApproval)
	gate.FinalLedgerWatermark = cloneInt64Pointer(gate.FinalLedgerWatermark)
	if gate.FinalizedCloseRunID != nil {
		value := *gate.FinalizedCloseRunID
		gate.FinalizedCloseRunID = &value
	}
	gate.SoftCloseRestrictions = append(json.RawMessage(nil), gate.SoftCloseRestrictions...)
	gate.ReopenAuthorizationScope = append(json.RawMessage(nil), gate.ReopenAuthorizationScope...)
	gate.ActiveAdmissionCounters.ControlOwnerEpoch = cloneInt64Pointer(gate.ActiveAdmissionCounters.ControlOwnerEpoch)
	if gate.LastFrozenAdmissionSummary != nil {
		value := *gate.LastFrozenAdmissionSummary
		value.ControlOwnerEpoch = cloneInt64Pointer(value.ControlOwnerEpoch)
		gate.LastFrozenAdmissionSummary = &value
	}
	gate.FrozenAdmissionSummaryHistory = append([]FrozenGateAdmissionSummary(nil), gate.FrozenAdmissionSummaryHistory...)
	for index := range gate.FrozenAdmissionSummaryHistory {
		gate.FrozenAdmissionSummaryHistory[index].ControlOwnerEpoch = cloneInt64Pointer(gate.FrozenAdmissionSummaryHistory[index].ControlOwnerEpoch)
	}
	return gate
}

func int64Pointer(value int64) *int64 {
	copy := value
	return &copy
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

type MemorySoftCloseGateAuditRecorder struct {
	mu      sync.Mutex
	Records []SoftCloseGateAuditRecord
	Err     error
}

func (recorder *MemorySoftCloseGateAuditRecorder) RecordSoftCloseGateMutation(_ context.Context, record SoftCloseGateAuditRecord) (uuid.UUID, error) {
	if recorder == nil {
		return uuid.Nil, ErrSoftCloseGateAuditUnavailable
	}
	if recorder.Err != nil {
		return uuid.Nil, recorder.Err
	}
	reference := uuid.New()
	recorder.mu.Lock()
	recorder.Records = append(recorder.Records, record)
	recorder.mu.Unlock()
	return reference, nil
}

type MemorySoftCloseGateRepository struct {
	mu        sync.RWMutex
	gates     map[string]PeriodPostingGate
	audit     SoftCloseGateAuditRecorder
	events    []SoftCloseGateEvent
	outboxErr error
}

type SoftCloseGateEvent struct {
	EventType          string
	Version            int
	Gate               PeriodPostingGate
	AuditReference     uuid.UUID
	CommandFingerprint string
	CorrelationID      string
	CausationID        string
}

func NewMemorySoftCloseGateRepository(audit ...SoftCloseGateAuditRecorder) *MemorySoftCloseGateRepository {
	var recorder SoftCloseGateAuditRecorder
	if len(audit) > 0 {
		recorder = audit[0]
	}
	if recorder == nil {
		recorder = &MemorySoftCloseGateAuditRecorder{}
	}
	return &MemorySoftCloseGateRepository{gates: make(map[string]PeriodPostingGate), audit: recorder}
}

func (repository *MemorySoftCloseGateRepository) Seed(gate PeriodPostingGate) error {
	if repository == nil {
		return ErrSoftCloseGateReferenceUnavailable
	}
	if gate.AggregateVersion == 0 {
		gate.AggregateVersion = gate.GateVersion
	}
	if err := gate.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	repository.gates[softCloseGateKey(gate.AccountingScopeID, gate.FiscalPeriodID)] = clonePeriodPostingGate(gate)
	repository.mu.Unlock()
	return nil
}

func NewOpenMemorySoftCloseGate(scopeID, periodID uuid.UUID, periodStateVersion int64) PeriodPostingGate {
	now := time.Now().UTC()
	return PeriodPostingGate{ID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, GateMode: PostingGateOpen, PeriodStateVersion: periodStateVersion, GateVersion: 1, AggregateVersion: 1, NextLedgerPosition: 1, CreatedAt: now, UpdatedAt: now}
}

func (repository *MemorySoftCloseGateRepository) GetSoftCloseGate(_ context.Context, scopeID, periodID uuid.UUID) (PeriodPostingGate, error) {
	if repository == nil {
		return PeriodPostingGate{}, ErrSoftCloseGateReferenceUnavailable
	}
	repository.mu.RLock()
	gate, ok := repository.gates[softCloseGateKey(scopeID, periodID)]
	repository.mu.RUnlock()
	if !ok {
		return PeriodPostingGate{}, ErrSoftCloseGateNotFound
	}
	return clonePeriodPostingGate(gate), nil
}

func (repository *MemorySoftCloseGateRepository) GetPostingGateStatus(ctx context.Context, scopeID, periodID uuid.UUID) (PeriodPostingGate, error) {
	return repository.GetSoftCloseGate(ctx, scopeID, periodID)
}

func (repository *MemorySoftCloseGateRepository) CommitSoftCloseGateMutation(ctx context.Context, mutation SoftCloseGateMutation) (SoftCloseGateResult, error) {
	if repository == nil || repository.audit == nil {
		return SoftCloseGateResult{}, ErrSoftCloseGateReferenceUnavailable
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := softCloseGateKey(mutation.Before.AccountingScopeID, mutation.Before.FiscalPeriodID)
	current, ok := repository.gates[key]
	if !ok {
		return SoftCloseGateResult{}, ErrSoftCloseGateNotFound
	}
	if current.GateVersion != mutation.Before.GateVersion {
		return SoftCloseGateResult{}, ErrSoftCloseGateVersionConflict
	}
	if repository.outboxErr != nil {
		return SoftCloseGateResult{}, repository.outboxErr
	}
	auditReference, err := repository.audit.RecordSoftCloseGateMutation(ctx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return SoftCloseGateResult{}, err
		}
		return SoftCloseGateResult{}, ErrSoftCloseGateAuditUnavailable
	}
	repository.gates[key] = clonePeriodPostingGate(mutation.After)
	eventTypes := append(append([]string(nil), mutation.AdditionalEventTypes...), softCloseGateEventType(mutation.Operation))
	for _, eventType := range eventTypes {
		repository.events = append(repository.events, SoftCloseGateEvent{EventType: eventType, Version: 1, Gate: clonePeriodPostingGate(mutation.After), AuditReference: auditReference, CommandFingerprint: mutation.CommandFingerprint, CorrelationID: mutation.Audit.CorrelationID, CausationID: mutation.Audit.CausationID})
	}
	return SoftCloseGateResult{Operation: mutation.Operation, Gate: clonePeriodPostingGate(mutation.After), DecisionReference: mutation.Authorization.DecisionReference, PolicyReference: mutation.Authorization.PolicyReference, AuthorizationPolicyVersion: mutation.Authorization.PolicyVersion, ValidationOutcome: "valid", AuditReference: auditReference, NextAction: softCloseGateNextAction(mutation.Operation)}, nil
}

func (repository *MemorySoftCloseGateRepository) Events() []SoftCloseGateEvent {
	if repository == nil {
		return nil
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]SoftCloseGateEvent, len(repository.events))
	copy(result, repository.events)
	for index := range result {
		result[index].Gate = clonePeriodPostingGate(result[index].Gate)
	}
	return result
}

func (repository *MemorySoftCloseGateRepository) SetOutboxError(err error) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	repository.outboxErr = err
	repository.mu.Unlock()
}

func softCloseGateKey(scopeID, periodID uuid.UUID) string {
	return scopeID.String() + ":" + periodID.String()
}

func softCloseGateNextAction(operation string) string {
	switch operation {
	case SoftCloseGateEnterOperation:
		return "Soft-close policy is active. Review blocked ordinary actions and exit with the matching run and control epoch when ready."
	case SoftCloseGateExitOperation:
		return "The gate is open. Review the frozen soft-close admission summary before continuing the period workflow."
	case ScopedReopenGateOpenOperation:
		return "The scoped reopen gate is active. Admit only authorized ReopenCorrection postings within the recorded scope and expiry."
	case ScopedReopenGateCloseOperation:
		return "The scoped reopen gate is closed. Review the frozen correction summary; hard-close or mandatory reclose follows the authoritative result."
	case OperationalReopenGateOpenOperation:
		return "The operational reopen gate is active. Admit only the approved actor and transaction-class combination before closing into the mandatory reclose decision."
	case OperationalReopenGateCloseOperation:
		return "The operational reopen gate is closed. Review the frozen operational summary; hard-close or mandatory reclose follows the authoritative result."
	case BeginRecloseGateOperation:
		return "The reclose barrier is active. The reclose owner must complete close-only work and finalize; release is unavailable."
	case PostingBarrierAcquireOperation:
		return "The posting barrier is acquired. Complete close-only work and finalize or release only under the authoritative barrier rules."
	case PostingBarrierReleaseOperation:
		return "The soft-close epoch was restored. Review the frozen close summary before resuming the close workflow."
	case PostingBarrierFinalizeOperation:
		return "The posting gate is hard closed. Use the authoritative final watermark for downstream close and sealing work."
	}
	return "Review the authoritative posting-gate result before continuing the period workflow."
}

func softCloseGateEventType(operation string) string {
	switch operation {
	case SoftCloseGateEnterOperation:
		return SoftCloseGateEnteredEventType
	case SoftCloseGateExitOperation:
		return SoftCloseGateExitedEventType
	case ScopedReopenGateOpenOperation:
		return ScopedReopenGateOpenedEventType
	case ScopedReopenGateCloseOperation:
		return ScopedReopenGateClosedEventType
	case OperationalReopenGateOpenOperation:
		return OperationalReopenGateOpenedEventType
	case OperationalReopenGateCloseOperation:
		return OperationalReopenGateClosedEventType
	case BeginRecloseGateOperation:
		return RecloseGateBegunEventType
	case PostingBarrierAcquireOperation:
		return PostingBarrierAcquiredEventType
	case PostingBarrierReleaseOperation:
		return PostingBarrierReleasedEventType
	case PostingBarrierFinalizeOperation:
		return PostingGateFinalizedEventType
	default:
		return operation
	}
}

func (service *SoftCloseGateService) finalizeSoftCloseFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{softCloseGateFailureCode(commandErr)})
	if err != nil {
		return
	}
	initial := acquisition.Result()
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
	if err := service.durable.Coordinator.Finalize(ctx, tx, acquisition, metadata); err != nil {
		return
	}
	if err := tx.Commit(ctx); err != nil {
		return
	}
	committed = true
}

func softCloseGateFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrSoftCloseGateAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrSoftCloseGateAuthorizationUnavailable), errors.Is(err, ErrSoftCloseGateAuthorizationStale), errors.Is(err, ErrSoftCloseGatePolicyUnavailable), errors.Is(err, ErrSoftCloseGateAuditUnavailable), errors.Is(err, ErrSoftCloseGateOutboxUnavailable), errors.Is(err, ErrSoftCloseGateReferenceUnavailable):
		return "SERVICE_UNAVAILABLE"
	case errors.Is(err, ErrSoftCloseGateVersionConflict), errors.Is(err, ErrSoftCloseGateScopeConflict), errors.Is(err, ErrSoftCloseGateOwnerConflict), errors.Is(err, ErrSoftCloseGateEpochConflict), errors.Is(err, ErrSoftCloseGateModeConflict), errors.Is(err, ErrSoftCloseGatePolicyConflict), errors.Is(err, ErrSoftCloseGateAuthorizationExpired), errors.Is(err, ErrSoftCloseGateRecloseSummaryConflict), errors.Is(err, ErrSoftCloseGateIdempotencyConflict):
		return "VERSION_CONFLICT"
	case errors.Is(err, ErrPostingBarrierNotReleasable), errors.Is(err, ErrPostingBarrierAdmissionConflict), errors.Is(err, ErrPostingBarrierWatermarkConflict), errors.Is(err, ErrPostingBarrierFinalized):
		return "VERSION_CONFLICT"
	default:
		return "VALIDATION_FAILED"
	}
}
