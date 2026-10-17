package fiscalperiod

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
)

const (
	ControlProcessSoftClose = "softClose"
	ControlProcessCloseRun  = "closeRun"
)

// ControlAuthority is FPM's process-owner projection. Its epoch is separate
// from the GL control epoch: taking over a process never changes GL admission
// authority. A nil value is intentionally preserved for legacy rows and is
// not eligible for takeover.
type ControlAuthority struct {
	CurrentOwner     Actor                    `json:"currentOwner"`
	AuthorityEpoch   int64                    `json:"authorityEpoch"`
	Version          int64                    `json:"version"`
	ExpiresAt        time.Time                `json:"expiresAt"`
	CutoffAt         time.Time                `json:"cutoffAt"`
	OwnershipHistory []ControlOwnershipChange `json:"ownershipHistory,omitempty"`
}

func (authority ControlAuthority) Validate() error {
	if err := authority.CurrentOwner.Validate(); err != nil {
		return err
	}
	if authority.AuthorityEpoch < 1 || authority.Version < 1 || authority.ExpiresAt.IsZero() || authority.CutoffAt.IsZero() {
		return ErrFiscalPeriodControlAuthorityMissing
	}
	for _, change := range authority.OwnershipHistory {
		if err := change.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ControlOwnershipChange is immutable lineage for one FPM ownership change.
// It deliberately contains references to authorization and Workflow
// evidence, never a mutable copy of either owning context's record.
type ControlOwnershipChange struct {
	PriorOwner                   Actor     `json:"priorOwner"`
	NewOwner                     Actor     `json:"newOwner"`
	ChangedAt                    time.Time `json:"changedAt"`
	ExpectedGateVersion          int64     `json:"expectedGateVersion"`
	Reason                       string    `json:"reason"`
	EvidenceReference            string    `json:"evidenceReference"`
	AuthorizationReference       uuid.UUID `json:"authorizationReference"`
	AuthorizationPolicyReference string    `json:"authorizationPolicyReference"`
	AuthorizationPolicyVersion   string    `json:"authorizationPolicyVersion"`
	ApprovalRequestID            uuid.UUID `json:"approvalRequestId"`
	ApprovalDecisionID           uuid.UUID `json:"approvalDecisionId"`
	ApprovalDecisionVersion      int64     `json:"approvalDecisionVersion"`
	ApprovalDecisionReference    uuid.UUID `json:"approvalDecisionReference"`
}

func (change ControlOwnershipChange) Validate() error {
	if err := change.PriorOwner.Validate(); err != nil {
		return err
	}
	if err := change.NewOwner.Validate(); err != nil {
		return err
	}
	if change.ChangedAt.IsZero() || change.ExpectedGateVersion < 1 || strings.TrimSpace(change.Reason) == "" || strings.TrimSpace(change.EvidenceReference) == "" || change.AuthorizationReference == uuid.Nil || strings.TrimSpace(change.AuthorizationPolicyReference) == "" || strings.TrimSpace(change.AuthorizationPolicyVersion) == "" || change.ApprovalRequestID == uuid.Nil || change.ApprovalDecisionID == uuid.Nil || change.ApprovalDecisionVersion < 1 || change.ApprovalDecisionReference == uuid.Nil {
		return ErrFiscalPeriodPolicyConflict
	}
	return nil
}

// ControlRecoveryApprovalDecision is the dedicated US6 Workflow evidence
// contract. It is separate from close approval evidence because a takeover
// or exception extension changes the control-recovery candidate itself.
type ControlRecoveryApprovalDecision struct {
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
	SeparationOfDutiesReference string    `json:"separationOfDutiesReference"`
	RecordedAt                  time.Time `json:"recordedAt"`
}

func (decision ControlRecoveryApprovalDecision) Validate() error {
	if decision.RequestID == uuid.Nil || decision.DecisionID == uuid.Nil || decision.DecisionVersion < 1 || decision.DecisionReference == uuid.Nil || decision.ApproverUserID == uuid.Nil || decision.SubjectVersion < 1 || strings.TrimSpace(decision.PolicyVersion) == "" || strings.TrimSpace(decision.CandidateFingerprint) == "" || strings.TrimSpace(decision.Outcome) == "" || strings.TrimSpace(decision.SeparationOfDutiesReference) == "" || decision.RecordedAt.IsZero() {
		return ErrControlRecoveryDecisionRejected
	}
	if decision.Outcome != CloseApprovalApproved && decision.Outcome != CloseApprovalRejected {
		return ErrControlRecoveryDecisionRejected
	}
	return nil
}

// EvidenceControlRecoveryDecisionPort is useful for local composition and
// focused domain tests. Production composition binds the Workflow-backed
// validator in cmd/api.
type EvidenceControlRecoveryDecisionPort struct{}

func (EvidenceControlRecoveryDecisionPort) Validate(_ context.Context, _ Actor, decision ControlRecoveryApprovalDecision) error {
	return decision.Validate()
}

type TakeOverPeriodControlCommand struct {
	CommandID                       uuid.UUID
	AccountingScopeID               uuid.UUID
	FiscalPeriodID                  uuid.UUID
	Target                          string
	ProcessID                       uuid.UUID
	ExpectedPeriodVersion           int64
	ExpectedProcessVersion          int64
	ExpectedGateVersion             int64
	ExpectedControlAuthorityVersion int64
	ExpectedControlAuthorityEpoch   int64
	Reason                          string
	EvidenceReference               string
	NewAuthorityExpiresAt           time.Time
	Approval                        ControlRecoveryApprovalDecision
	IdempotencyKey                  string
	CorrelationID                   string
	CausationID                     string
}

func (command TakeOverPeriodControlCommand) canonical() TakeOverPeriodControlCommand {
	command.Target = strings.TrimSpace(command.Target)
	command.Reason = strings.TrimSpace(command.Reason)
	command.EvidenceReference = strings.TrimSpace(command.EvidenceReference)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Approval.Reason = strings.TrimSpace(command.Approval.Reason)
	command.Approval.PolicyVersion = strings.TrimSpace(command.Approval.PolicyVersion)
	command.Approval.CandidateFingerprint = strings.TrimSpace(command.Approval.CandidateFingerprint)
	command.Approval.SeparationOfDutiesReference = strings.TrimSpace(command.Approval.SeparationOfDutiesReference)
	return command
}

func (command TakeOverPeriodControlCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil || command.ProcessID == uuid.Nil {
		return ErrInvalidFiscalPeriodCommand
	}
	if command.Target != ControlProcessSoftClose && command.Target != ControlProcessCloseRun {
		return ErrInvalidFiscalPeriodCommand
	}
	if command.ExpectedPeriodVersion < 1 || command.ExpectedProcessVersion < 1 || command.ExpectedGateVersion < 1 || command.ExpectedControlAuthorityVersion < 1 || command.ExpectedControlAuthorityEpoch < 1 || command.NewAuthorityExpiresAt.IsZero() || strings.TrimSpace(command.Reason) == "" || strings.TrimSpace(command.EvidenceReference) == "" || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	if err := command.Approval.Validate(); err != nil {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

type ExtendCloseExceptionCommand struct {
	CommandID                       uuid.UUID
	AccountingScopeID               uuid.UUID
	FiscalPeriodID                  uuid.UUID
	CloseRunID                      uuid.UUID
	ExceptionID                     uuid.UUID
	ExpectedPeriodVersion           int64
	ExpectedCloseRunVersion         int64
	ExpectedGateVersion             int64
	ExpectedControlAuthorityVersion int64
	ExpectedControlAuthorityEpoch   int64
	Justification                   string
	NewDueAt                        time.Time
	NewExpiresAt                    time.Time
	EvidenceReference               string
	Approval                        ControlRecoveryApprovalDecision
	IdempotencyKey                  string
	CorrelationID                   string
	CausationID                     string
}

func (command ExtendCloseExceptionCommand) canonical() ExtendCloseExceptionCommand {
	command.Justification = strings.TrimSpace(command.Justification)
	command.EvidenceReference = strings.TrimSpace(command.EvidenceReference)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Approval.Reason = strings.TrimSpace(command.Approval.Reason)
	command.Approval.PolicyVersion = strings.TrimSpace(command.Approval.PolicyVersion)
	command.Approval.CandidateFingerprint = strings.TrimSpace(command.Approval.CandidateFingerprint)
	command.Approval.SeparationOfDutiesReference = strings.TrimSpace(command.Approval.SeparationOfDutiesReference)
	return command
}

func (command ExtendCloseExceptionCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil || command.CloseRunID == uuid.Nil || command.ExceptionID == uuid.Nil {
		return ErrInvalidFiscalPeriodCommand
	}
	if command.ExpectedPeriodVersion < 1 || command.ExpectedCloseRunVersion < 1 || command.ExpectedGateVersion < 1 || command.ExpectedControlAuthorityVersion < 1 || command.ExpectedControlAuthorityEpoch < 1 || command.NewDueAt.IsZero() || command.NewExpiresAt.IsZero() || command.NewDueAt.After(command.NewExpiresAt) || strings.TrimSpace(command.Justification) == "" || strings.TrimSpace(command.EvidenceReference) == "" || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	if err := command.Approval.Validate(); err != nil {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

// CloseExceptionExtension is append-only. The current exception projection
// contains the newest due/expiry values while this record retains each prior
// value and the independent extension decision that authorized it.
type CloseExceptionExtension struct {
	ID                           uuid.UUID                       `json:"id"`
	CloseRunID                   uuid.UUID                       `json:"closeRunId"`
	ExceptionID                  uuid.UUID                       `json:"exceptionId"`
	PriorDueAt                   *time.Time                      `json:"priorDueAt,omitempty"`
	PriorExpiresAt               *time.Time                      `json:"priorExpiresAt,omitempty"`
	NewDueAt                     time.Time                       `json:"newDueAt"`
	NewExpiresAt                 time.Time                       `json:"newExpiresAt"`
	Justification                string                          `json:"justification"`
	EvidenceReference            string                          `json:"evidenceReference"`
	RequestedBy                  Actor                           `json:"requestedBy"`
	RequestedAt                  time.Time                       `json:"requestedAt"`
	AuthorizationReference       uuid.UUID                       `json:"authorizationReference"`
	AuthorizationPolicyReference string                          `json:"authorizationPolicyReference"`
	AuthorizationPolicyVersion   string                          `json:"authorizationPolicyVersion"`
	Approval                     ControlRecoveryApprovalDecision `json:"approval"`
	ApprovedAt                   time.Time                       `json:"approvedAt"`
}

func (extension CloseExceptionExtension) Validate() error {
	if extension.ID == uuid.Nil || extension.CloseRunID == uuid.Nil || extension.ExceptionID == uuid.Nil || extension.NewDueAt.IsZero() || extension.NewExpiresAt.IsZero() || extension.NewDueAt.After(extension.NewExpiresAt) || strings.TrimSpace(extension.Justification) == "" || strings.TrimSpace(extension.EvidenceReference) == "" || extension.RequestedAt.IsZero() || extension.AuthorizationReference == uuid.Nil || strings.TrimSpace(extension.AuthorizationPolicyReference) == "" || strings.TrimSpace(extension.AuthorizationPolicyVersion) == "" || extension.ApprovedAt.IsZero() {
		return ErrFiscalPeriodPolicyConflict
	}
	if extension.PriorDueAt != nil && extension.PriorExpiresAt != nil && extension.PriorDueAt.After(*extension.PriorExpiresAt) {
		return ErrFiscalPeriodPolicyConflict
	}
	if err := extension.RequestedBy.Validate(); err != nil {
		return err
	}
	if err := extension.Approval.Validate(); err != nil {
		return err
	}
	if extension.Approval.Outcome != CloseApprovalApproved {
		return ErrCloseExceptionExtensionRejected
	}
	return nil
}

func takeoverCandidateFingerprint(command TakeOverPeriodControlCommand) string {
	return fingerprint(struct {
		Operation                string
		AccountingScopeID        uuid.UUID
		FiscalPeriodID           uuid.UUID
		Target                   string
		ProcessID                uuid.UUID
		ExpectedPeriodVersion    int64
		ExpectedProcessVersion   int64
		ExpectedGateVersion      int64
		ExpectedAuthorityVersion int64
		ExpectedAuthorityEpoch   int64
		Reason                   string
		EvidenceReference        string
		NewAuthorityExpiresAt    time.Time
	}{TakeOverPeriodControlCommandIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.Target, command.ProcessID, command.ExpectedPeriodVersion, command.ExpectedProcessVersion, command.ExpectedGateVersion, command.ExpectedControlAuthorityVersion, command.ExpectedControlAuthorityEpoch, command.Reason, command.EvidenceReference, command.NewAuthorityExpiresAt.UTC()})
}

func takeoverCommandFingerprint(command TakeOverPeriodControlCommand) string {
	return fingerprint(struct {
		Operation            string
		CandidateFingerprint string
		Approval             ControlRecoveryApprovalDecision
	}{TakeOverPeriodControlCommandIdentity, takeoverCandidateFingerprint(command), command.Approval})
}

func extensionCandidateFingerprint(command ExtendCloseExceptionCommand) string {
	return fingerprint(struct {
		Operation                string
		AccountingScopeID        uuid.UUID
		FiscalPeriodID           uuid.UUID
		CloseRunID               uuid.UUID
		ExceptionID              uuid.UUID
		ExpectedPeriodVersion    int64
		ExpectedCloseRunVersion  int64
		ExpectedGateVersion      int64
		ExpectedAuthorityVersion int64
		ExpectedAuthorityEpoch   int64
		Justification            string
		NewDueAt                 time.Time
		NewExpiresAt             time.Time
		EvidenceReference        string
	}{ExtendCloseExceptionCommandIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.CloseRunID, command.ExceptionID, command.ExpectedPeriodVersion, command.ExpectedCloseRunVersion, command.ExpectedGateVersion, command.ExpectedControlAuthorityVersion, command.ExpectedControlAuthorityEpoch, command.Justification, command.NewDueAt.UTC(), command.NewExpiresAt.UTC(), command.EvidenceReference})
}

func extensionCommandFingerprint(command ExtendCloseExceptionCommand) string {
	return fingerprint(struct {
		Operation            string
		CandidateFingerprint string
		Approval             ControlRecoveryApprovalDecision
	}{ExtendCloseExceptionCommandIdentity, extensionCandidateFingerprint(command), command.Approval})
}

func cloneControlAuthority(authority *ControlAuthority) *ControlAuthority {
	if authority == nil {
		return nil
	}
	data, err := json.Marshal(authority)
	if err != nil {
		return nil
	}
	var clone ControlAuthority
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil
	}
	return &clone
}

func sameControlAuthorityLineage(left, right *ControlAuthority) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Version == right.Version && left.AuthorityEpoch == right.AuthorityEpoch && left.CutoffAt.Equal(right.CutoffAt)
}

func validateImmutableControlAuthority(before, after *ControlAuthority) error {
	if before != nil && after == nil {
		return ErrFiscalPeriodPolicyConflict
	}
	if before == nil || after == nil {
		return nil
	}
	if !before.CutoffAt.Equal(after.CutoffAt) {
		return ErrFiscalPeriodPolicyConflict
	}
	if len(after.OwnershipHistory) < len(before.OwnershipHistory) {
		return ErrFiscalPeriodPolicyConflict
	}
	for index, prior := range before.OwnershipHistory {
		if !sameControlOwnershipChange(prior, after.OwnershipHistory[index]) {
			return ErrFiscalPeriodPolicyConflict
		}
	}
	if len(after.OwnershipHistory) == len(before.OwnershipHistory) {
		if after.CurrentOwner != before.CurrentOwner || after.AuthorityEpoch != before.AuthorityEpoch || after.Version != before.Version || !after.ExpiresAt.Equal(before.ExpiresAt) {
			return ErrFiscalPeriodPolicyConflict
		}
		return nil
	}
	if len(after.OwnershipHistory) != len(before.OwnershipHistory)+1 || after.AuthorityEpoch != before.AuthorityEpoch+1 || after.Version != before.Version+1 {
		return ErrFiscalPeriodPolicyConflict
	}
	change := after.OwnershipHistory[len(after.OwnershipHistory)-1]
	if change.PriorOwner != before.CurrentOwner || change.NewOwner != after.CurrentOwner {
		return ErrFiscalPeriodPolicyConflict
	}
	return nil
}

func sameControlOwnershipChange(left, right ControlOwnershipChange) bool {
	return left.PriorOwner == right.PriorOwner && left.NewOwner == right.NewOwner && left.ChangedAt.Equal(right.ChangedAt) && left.ExpectedGateVersion == right.ExpectedGateVersion && left.Reason == right.Reason && left.EvidenceReference == right.EvidenceReference && left.AuthorizationReference == right.AuthorizationReference && left.AuthorizationPolicyReference == right.AuthorizationPolicyReference && left.AuthorizationPolicyVersion == right.AuthorizationPolicyVersion && left.ApprovalRequestID == right.ApprovalRequestID && left.ApprovalDecisionID == right.ApprovalDecisionID && left.ApprovalDecisionVersion == right.ApprovalDecisionVersion && left.ApprovalDecisionReference == right.ApprovalDecisionReference
}

func validateImmutableExceptionHistory(before, after ControlState) error {
	if before.CloseRun == nil {
		return nil
	}
	if after.CloseRun == nil {
		if len(before.CloseRun.Exceptions) == 0 {
			return nil
		}
		return ErrFiscalPeriodPolicyConflict
	}
	for _, prior := range before.CloseRun.Exceptions {
		var current *CloseException
		for index := range after.CloseRun.Exceptions {
			if after.CloseRun.Exceptions[index].ID == prior.ID {
				current = &after.CloseRun.Exceptions[index]
				break
			}
		}
		if current == nil || len(current.ExtensionHistory) < len(prior.ExtensionHistory) {
			return ErrFiscalPeriodPolicyConflict
		}
		for index, extension := range prior.ExtensionHistory {
			if !sameCloseExceptionExtension(extension, current.ExtensionHistory[index]) {
				return ErrFiscalPeriodPolicyConflict
			}
		}
	}
	return nil
}

func sameCloseExceptionExtension(left, right CloseExceptionExtension) bool {
	leftData, leftErr := json.Marshal(left)
	rightData, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftData) == string(rightData)
}

func validateControlAuthorityMutation(before, after ControlState) error {
	if err := validateImmutableControlAuthority(before.SoftClose.ControlAuthority, after.SoftClose.ControlAuthority); err != nil {
		return err
	}
	var beforeCloseAuthority, afterCloseAuthority *ControlAuthority
	if before.CloseRun != nil {
		beforeCloseAuthority = before.CloseRun.ControlAuthority
	}
	if after.CloseRun != nil {
		afterCloseAuthority = after.CloseRun.ControlAuthority
	}
	if err := validateImmutableControlAuthority(beforeCloseAuthority, afterCloseAuthority); err != nil {
		return err
	}
	return validateImmutableExceptionHistory(before, after)
}

func (service *Service) TakeOverPeriodControl(ctx context.Context, actor Actor, command TakeOverPeriodControlCommand) (CommandResult, error) {
	command = command.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.postingGate == nil || service.recoveryWorkflow == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	candidateFingerprint := takeoverCandidateFingerprint(command)
	fingerprint := takeoverCommandFingerprint(command)
	durable, replay, unlock, err := service.beginCloseCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, TakeOverPeriodControlOperation)
	if err != nil || replay != nil {
		return closeCommandReturn(replay, err)
	}
	if unlock != nil {
		defer unlock()
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionTakeOverPeriodControl, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionTakeOverPeriodControl); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
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
	if err := validateTakeoverTarget(state, command, service.clock().UTC()); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
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
	expectedGateVersion := command.ExpectedGateVersion
	if takeoverReconciliationAlreadyApplied(state, command) {
		expectedGateVersion = command.ExpectedGateVersion + 1
	}
	if err := validateRecoveryGateStatus(state, command.Target, command.ProcessID, expectedGateVersion, status); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	initialPeriodVersion := state.Period.AggregateVersion
	initialProcessVersion := recoveryProcessVersion(state, command.Target)
	if command.Target == ControlProcessCloseRun && state.CloseRun != nil && state.CloseRun.BarrierEvidence == nil && status.GateMode == gl.PostingGateCloseOnly {
		applyCommand := ApplyPostingGateResultCommand{
			CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
			SoftCloseRunID: state.SoftClose.ID, CloseRunID: command.ProcessID,
			ExpectedPeriodVersion: command.ExpectedPeriodVersion, ExpectedGateVersion: command.ExpectedGateVersion,
			SoftCloseControlEpoch: state.ControlEpoch.Number, SoftClosePolicyVersion: state.SoftClose.PolicyVersion,
			Result:         gl.SoftCloseGateResult{Operation: gl.PostingBarrierAcquireOperation, Gate: status, PolicyReference: "reconciled-from-gl-status", AuthorizationPolicyVersion: state.SoftClose.PolicyVersion, ValidationOutcome: "reconciled", NextAction: "continue-close"},
			IdempotencyKey: command.IdempotencyKey + ":reconcile-barrier", CorrelationID: command.CorrelationID, CausationID: command.CausationID,
		}
		if _, err := service.applyPostingGateResult(ctx, actor, decision, applyCommand, false, nil); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		state, err = service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
		if err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		status, err = service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
		if err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrPostingBarrierUnavailable)
			return CommandResult{}, ErrPostingBarrierUnavailable
		}
		if err := validateRecoveryGateStatus(state, command.Target, command.ProcessID, status.GateVersion, status); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
	}
	if err := validateTakeoverVersions(state, command, initialPeriodVersion, initialProcessVersion); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := service.validateRecoveryApproval(ctx, actor, command.Approval, candidateFingerprint, command.ExpectedProcessVersion); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	working := cloneControlState(state)
	authority := recoveryAuthority(&working, command.Target)
	if authority == nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodControlAuthorityMissing)
		return CommandResult{}, ErrFiscalPeriodControlAuthorityMissing
	}
	now := service.clock().UTC()
	priorOwner := authority.CurrentOwner
	change := ControlOwnershipChange{
		PriorOwner: priorOwner, NewOwner: actor, ChangedAt: now, ExpectedGateVersion: command.ExpectedGateVersion,
		Reason: command.Reason, EvidenceReference: command.EvidenceReference,
		AuthorizationReference: decision.DecisionReference, AuthorizationPolicyReference: decision.PolicyReference, AuthorizationPolicyVersion: decision.PolicyVersion,
		ApprovalRequestID: command.Approval.RequestID, ApprovalDecisionID: command.Approval.DecisionID, ApprovalDecisionVersion: command.Approval.DecisionVersion, ApprovalDecisionReference: command.Approval.DecisionReference,
	}
	authority.CurrentOwner = actor
	authority.AuthorityEpoch++
	authority.Version++
	authority.ExpiresAt = command.NewAuthorityExpiresAt.UTC()
	authority.OwnershipHistory = append(authority.OwnershipHistory, change)
	if err := authority.Validate(); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	processID := command.ProcessID
	processVersion := recoveryProcessVersion(working, command.Target)
	if command.Target == ControlProcessSoftClose {
		working.SoftClose.AggregateVersion++
		working.SoftClose.UpdatedAt = now
	} else {
		working.CloseRun.AggregateVersion++
		working.CloseRun.UpdatedAt = now
	}
	payload := recoveryStatePayload(working, status, command.Target, change)
	eventType := "SoftCloseResumed"
	if command.Target == ControlProcessCloseRun {
		eventType = "CloseRunResumed"
	}
	events := []Event{{EventType: eventType, AggregateID: processID, AggregateVersion: processVersion + 1, AccountingScopeID: working.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}}
	result := CommandResult{State: working, GateResult: recoveryGateResult(status, command.Target), Decision: decision}
	mutation := Mutation{Before: state, After: working, Operation: TakeOverPeriodControlOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, processID, TakeOverPeriodControlOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID), Events: events}
	if err := service.commitCloseMutation(ctx, mutation, durable, result); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	service.rememberCommand(actor, command.AccountingScopeID, TakeOverPeriodControlOperation, command.IdempotencyKey, fingerprint, result)
	return result, nil
}

func (service *Service) ExtendCloseException(ctx context.Context, actor Actor, command ExtendCloseExceptionCommand) (CommandResult, error) {
	command = command.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.recoveryWorkflow == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	now := service.clock().UTC()
	if !command.NewDueAt.After(now) || !command.NewExpiresAt.After(now) {
		return CommandResult{}, ErrInvalidFiscalPeriodCommand
	}
	candidateFingerprint := extensionCandidateFingerprint(command)
	fingerprint := extensionCommandFingerprint(command)
	durable, replay, unlock, err := service.beginCloseCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, ExtendCloseExceptionOperation)
	if err != nil || replay != nil {
		return closeCommandReturn(replay, err)
	}
	if unlock != nil {
		defer unlock()
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionExtendCloseException, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionExtendCloseException); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
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
	if err := validateExtensionTarget(state, command); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := service.validateRecoveryApproval(ctx, actor, command.Approval, candidateFingerprint, command.ExpectedCloseRunVersion); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	working := cloneControlState(state)
	var exception *CloseException
	for index := range working.CloseRun.Exceptions {
		if working.CloseRun.Exceptions[index].ID == command.ExceptionID {
			exception = &working.CloseRun.Exceptions[index]
			break
		}
	}
	if exception == nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}
	if exception.Status == CloseExceptionRejected || exception.Status == CloseExceptionExpired && (exception.ApprovalDecision == nil || exception.ApprovalDecision.Outcome != CloseApprovalApproved) {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrCloseExceptionExtensionRejected)
		return CommandResult{}, ErrCloseExceptionExtensionRejected
	}
	if exception.Owner.UserID == uuid.Nil {
		exception.Owner = actor
	}
	if exception.CreatedAt.IsZero() {
		exception.CreatedAt = now
	}
	extension := CloseExceptionExtension{
		ID: uuid.New(), CloseRunID: command.CloseRunID, ExceptionID: command.ExceptionID,
		PriorDueAt: cloneRecoveryTime(exception.DueAt), PriorExpiresAt: cloneRecoveryTime(exception.ExpiresAt), NewDueAt: command.NewDueAt.UTC(), NewExpiresAt: command.NewExpiresAt.UTC(),
		Justification: command.Justification, EvidenceReference: command.EvidenceReference, RequestedBy: actor, RequestedAt: now,
		AuthorizationReference: decision.DecisionReference, AuthorizationPolicyReference: decision.PolicyReference, AuthorizationPolicyVersion: decision.PolicyVersion,
		Approval: command.Approval, ApprovedAt: command.Approval.RecordedAt,
	}
	if err := extension.Validate(); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	exception.DueAt = cloneRecoveryTime(&extension.NewDueAt)
	exception.ExpiresAt = cloneRecoveryTime(&extension.NewExpiresAt)
	exception.Justification = command.Justification
	exception.UpdatedAt = now
	exception.ExtensionHistory = append(exception.ExtensionHistory, extension)
	if exception.Status == CloseExceptionApproved || exception.Status == CloseExceptionExpired {
		exception.Status = CloseExceptionApproved
		exception.Blocker = "Approved exception extension retained as close evidence"
	} else {
		// An extension changes only the window. It cannot resolve an unresolved
		// exception or retroactively alter its original approval decision.
		exception.Blocker = "Exception extension approved; resolution remains required"
	}
	working.CloseRun.AggregateVersion++
	working.CloseRun.UpdatedAt = now
	working.CloseRun.LastFailureCode = ""
	working.CloseRun.LastFailureDetail = ""
	gate := extensionGateResult(*working.CloseRun)
	result := CommandResult{State: working, GateResult: gate, Decision: decision}
	if err := service.commitCloseMutation(ctx, Mutation{Before: state, After: working, Operation: ExtendCloseExceptionOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, command.CloseRunID, ExtendCloseExceptionOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID)}, durable, result); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	service.rememberCommand(actor, command.AccountingScopeID, ExtendCloseExceptionOperation, command.IdempotencyKey, fingerprint, result)
	return result, nil
}

func (service *Service) validateRecoveryApproval(ctx context.Context, actor Actor, approval ControlRecoveryApprovalDecision, candidateFingerprint string, subjectVersion int64) error {
	if err := service.recoveryWorkflow.Validate(ctx, actor, approval); err != nil {
		if errors.Is(err, ErrControlRecoveryDecisionUnavailable) {
			return ErrControlRecoveryDecisionUnavailable
		}
		return ErrControlRecoveryDecisionRejected
	}
	if approval.CandidateFingerprint != candidateFingerprint || approval.SubjectVersion != subjectVersion || approval.ApproverUserID == actor.UserID || approval.Outcome != CloseApprovalApproved || strings.TrimSpace(approval.SeparationOfDutiesReference) == "" {
		return ErrControlRecoveryDecisionRejected
	}
	return nil
}

func validateTakeoverTarget(state ControlState, command TakeOverPeriodControlCommand, now time.Time) error {
	authority := recoveryAuthority(&state, command.Target)
	if authority == nil {
		return ErrFiscalPeriodControlAuthorityMissing
	}
	if err := authority.Validate(); err != nil {
		return ErrFiscalPeriodControlAuthorityMissing
	}
	if authority.Version != command.ExpectedControlAuthorityVersion || authority.AuthorityEpoch != command.ExpectedControlAuthorityEpoch {
		return ErrFiscalPeriodVersionConflict
	}
	if !authority.ExpiresAt.UTC().Before(now) && !authority.ExpiresAt.UTC().Equal(now) {
		return ErrFiscalPeriodRecoveryNotEligible
	}
	if !command.NewAuthorityExpiresAt.UTC().After(now) {
		return ErrInvalidFiscalPeriodCommand
	}
	if state.Period.ID != command.FiscalPeriodID || state.Period.AccountingScopeID != command.AccountingScopeID {
		return ErrFiscalPeriodScopeConflict
	}
	switch command.Target {
	case ControlProcessSoftClose:
		if state.SoftClose.ID != command.ProcessID || state.SoftClose.Status != SoftCloseRunActive || state.Period.State != PeriodStateSoftClosed {
			return ErrFiscalPeriodRecoveryNotEligible
		}
	case ControlProcessCloseRun:
		if state.CloseRun == nil || state.CloseRun.ID != command.ProcessID || state.Period.State == PeriodStateHardClosed {
			return ErrFiscalPeriodRecoveryNotEligible
		}
		switch state.CloseRun.Status {
		case CloseRunInitiating, CloseRunBarrierAcquired, CloseRunInProgress, CloseRunAwaitingApproval, CloseRunFinalizing, CloseRunFailed:
		default:
			return ErrFiscalPeriodRecoveryNotEligible
		}
	default:
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

func validateTakeoverVersions(state ControlState, command TakeOverPeriodControlCommand, initialPeriodVersion, initialProcessVersion int64) error {
	periodVersionOK := state.Period.AggregateVersion == command.ExpectedPeriodVersion
	processVersionOK := recoveryProcessVersion(state, command.Target) == command.ExpectedProcessVersion
	// A close-run takeover can first reconcile an already-acquired GL barrier
	// through ApplyPostingGateResult. That existing path commits the handoff
	// before the ownership mutation. If the second commit is interrupted, the
	// same idempotent command must be able to resume against the one-step-ahead
	// handoff rather than treating the valid partial progress as a conflict.
	if command.Target == ControlProcessCloseRun && takeoverReconciliationAlreadyApplied(state, command) {
		return nil
	}
	if !periodVersionOK && state.Period.AggregateVersion == initialPeriodVersion+1 && command.ExpectedPeriodVersion == initialPeriodVersion {
		periodVersionOK = true
	}
	if !processVersionOK && recoveryProcessVersion(state, command.Target) == initialProcessVersion+1 && command.ExpectedProcessVersion == initialProcessVersion {
		processVersionOK = true
	}
	if !periodVersionOK || !processVersionOK {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func takeoverReconciliationAlreadyApplied(state ControlState, command TakeOverPeriodControlCommand) bool {
	if command.Target != ControlProcessCloseRun || state.CloseRun == nil || state.CloseRun.ID != command.ProcessID || state.CloseRun.BarrierEvidence == nil {
		return false
	}
	if state.Period.AggregateVersion != command.ExpectedPeriodVersion+1 || state.CloseRun.AggregateVersion != command.ExpectedProcessVersion+1 {
		return false
	}
	if state.Period.State != PeriodStateClosing || state.Period.ActiveCloseRunID == nil || *state.Period.ActiveCloseRunID != command.ProcessID || state.Period.ActiveSoftCloseRunID != nil {
		return false
	}
	if state.SoftClose.Status != SoftCloseRunHandoffPending || state.ControlEpoch.Status != ControlEpochHandoffPending {
		return false
	}
	return state.CloseRun.BarrierEvidence.GateVersion == command.ExpectedGateVersion+1
}

func validateExtensionTarget(state ControlState, command ExtendCloseExceptionCommand) error {
	if state.Period.ID != command.FiscalPeriodID || state.Period.AccountingScopeID != command.AccountingScopeID || state.CloseRun == nil || state.CloseRun.ID != command.CloseRunID {
		return ErrFiscalPeriodScopeConflict
	}
	if state.Period.AggregateVersion != command.ExpectedPeriodVersion || state.CloseRun.AggregateVersion != command.ExpectedCloseRunVersion {
		return ErrFiscalPeriodVersionConflict
	}
	if state.Period.State == PeriodStateHardClosed || state.CloseRun.Status == CloseRunCompleted || state.CloseRun.Status == CloseRunFinalizing || state.CloseRun.Status == CloseRunAborted || state.CloseRun.Status == CloseRunAborting {
		return ErrCloseExceptionExtensionRejected
	}
	if state.CloseRun.BarrierEvidence != nil && state.CloseRun.BarrierEvidence.GateVersion != command.ExpectedGateVersion {
		return ErrFiscalPeriodVersionConflict
	}
	authority := recoveryAuthority(&state, ControlProcessCloseRun)
	if authority == nil {
		return ErrFiscalPeriodControlAuthorityMissing
	}
	if err := authority.Validate(); err != nil || authority.Version != command.ExpectedControlAuthorityVersion || authority.AuthorityEpoch != command.ExpectedControlAuthorityEpoch {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func recoveryAuthority(state *ControlState, target string) *ControlAuthority {
	if state == nil {
		return nil
	}
	if target == ControlProcessCloseRun {
		if state.CloseRun == nil {
			return nil
		}
		return state.CloseRun.ControlAuthority
	}
	return state.SoftClose.ControlAuthority
}

func recoveryProcessVersion(state ControlState, target string) int64 {
	if target == ControlProcessCloseRun && state.CloseRun != nil {
		return state.CloseRun.AggregateVersion
	}
	return state.SoftClose.AggregateVersion
}

func validateRecoveryGateStatus(state ControlState, target string, processID uuid.UUID, expectedGateVersion int64, status gl.PostingGateStatus) error {
	if status.AccountingScopeID != state.Period.AccountingScopeID || status.FiscalPeriodID != state.Period.ID {
		return ErrFiscalPeriodScopeConflict
	}
	if target == ControlProcessSoftClose {
		if status.GateVersion != expectedGateVersion || status.GateMode != gl.PostingGateSoftClosePolicy || status.ActiveSoftCloseRunID == nil || *status.ActiveSoftCloseRunID != processID || status.ActiveSoftCloseControlEpoch != state.ControlEpoch.Number || status.ActiveCloseRunID != nil || status.ActiveAdmissionCounters.ControlOwnerType != gl.SoftCloseGateOwnerType || status.ActiveAdmissionCounters.ControlOwnerID != processID || status.ActiveAdmissionCounters.ControlOwnerEpoch == nil || *status.ActiveAdmissionCounters.ControlOwnerEpoch != state.ControlEpoch.Number {
			return ErrPostingBarrierResultConflict
		}
		if status.SoftClosePolicyVersion != "" && status.SoftClosePolicyVersion != state.SoftClose.PolicyVersion {
			return ErrFiscalPeriodPolicyConflict
		}
		return nil
	}
	if status.GateMode == gl.PostingGateSoftClosePolicy {
		if status.GateVersion != expectedGateVersion || status.ActiveSoftCloseRunID == nil || *status.ActiveSoftCloseRunID != state.SoftClose.ID || status.ActiveSoftCloseControlEpoch != state.ControlEpoch.Number || status.ActiveCloseRunID != nil || status.ActiveAdmissionCounters.ControlOwnerType != gl.SoftCloseGateOwnerType || status.ActiveAdmissionCounters.ControlOwnerID != state.SoftClose.ID || status.ActiveAdmissionCounters.ControlOwnerEpoch == nil || *status.ActiveAdmissionCounters.ControlOwnerEpoch != state.ControlEpoch.Number {
			return ErrPostingBarrierResultConflict
		}
		if status.SoftClosePolicyVersion != "" && status.SoftClosePolicyVersion != state.SoftClose.PolicyVersion {
			return ErrFiscalPeriodPolicyConflict
		}
		return nil
	}
	if state.CloseRun == nil || status.ActiveCloseRunID == nil || *status.ActiveCloseRunID != processID || status.ActiveCloseType != state.CloseRun.CloseType || status.GateMode != gl.PostingGateCloseOnly || status.ActiveAdmissionCounters.ControlOwnerType != gl.PostingBarrierOwnerType || status.ActiveAdmissionCounters.ControlOwnerID != processID {
		return ErrPostingBarrierResultConflict
	}
	if status.PriorControlProcessID == nil || *status.PriorControlProcessID != state.SoftClose.ID || status.PriorControlProcessType != gl.SoftCloseGateOwnerType || status.PriorControlEpoch == nil || *status.PriorControlEpoch != state.ControlEpoch.Number || status.PriorGateMode != gl.PostingGateSoftClosePolicy || status.LastFrozenAdmissionSummary == nil || status.LastFrozenAdmissionSummary.ControlOwnerType != gl.SoftCloseGateOwnerType || status.LastFrozenAdmissionSummary.ControlOwnerID != state.SoftClose.ID || status.LastFrozenAdmissionSummary.ControlOwnerEpoch == nil || *status.LastFrozenAdmissionSummary.ControlOwnerEpoch != state.ControlEpoch.Number || status.LastFrozenAdmissionSummary.FrozenAtGateVersion != status.GateVersion || status.LastFrozenAdmissionSummary.FrozenByCommandType != gl.PostingBarrierAcquireCommandIdentity || strings.TrimSpace(status.LastFrozenAdmissionSummary.FrozenByCommandFingerprint) == "" {
		return ErrPostingBarrierResultConflict
	}
	if state.CloseRun.BarrierEvidence == nil {
		if status.GateVersion != expectedGateVersion+1 {
			return ErrPostingBarrierResultConflict
		}
	} else if status.GateVersion != expectedGateVersion {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func recoveryGateResult(status gl.PostingGateStatus, target string) gl.SoftCloseGateResult {
	nextAction := "resume-soft-close"
	if target == ControlProcessCloseRun {
		nextAction = "resume-close-run"
	}
	return gl.SoftCloseGateResult{Operation: "get-posting-gate-status", Gate: status, PolicyReference: "authoritative-gl-status", ValidationOutcome: "authoritative", NextAction: nextAction}
}

func extensionGateResult(run CloseRun) gl.SoftCloseGateResult {
	if run.FinalizationGateResult != nil {
		return *run.FinalizationGateResult
	}
	if run.ReleaseGateResult != nil {
		return *run.ReleaseGateResult
	}
	if run.BarrierEvidence != nil {
		return evidenceResult(run)
	}
	return gl.SoftCloseGateResult{}
}

func recoveryStatePayload(state ControlState, status gl.PostingGateStatus, target string, change ControlOwnershipChange) []byte {
	payload, err := json.Marshal(struct {
		Period       FiscalPeriod           `json:"period"`
		SoftClose    SoftCloseRun           `json:"softClose"`
		ControlEpoch SoftCloseControlEpoch  `json:"controlEpoch"`
		CloseRun     *CloseRun              `json:"closeRun,omitempty"`
		Target       string                 `json:"target"`
		Ownership    ControlOwnershipChange `json:"ownershipChange"`
		Gate         gl.PostingGateStatus   `json:"gate"`
	}{state.Period, state.SoftClose, state.ControlEpoch, state.CloseRun, target, change, status})
	if err != nil {
		return []byte(`{"recovery":"unserializable"}`)
	}
	return payload
}

func cloneRecoveryTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

var _ ControlRecoveryDecisionPort = EvidenceControlRecoveryDecisionPort{}
