package gl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PostingGateStatus is the GL-owned authoritative read model used by command
// recovery. The public status operation is intentionally implemented later.
type PostingGateStatus = PeriodPostingGate

// PostingGateApplicationPort is the GL-owned command/result boundary for a
// future FPM orchestrator. FPM does not receive access to the repository or
// aggregate and does not write GL state directly.
type PostingGateApplicationPort interface {
	AcquirePostingBarrier(context.Context, Actor, AcquirePostingBarrierCommand) (SoftCloseGateResult, error)
	ReleasePostingBarrier(context.Context, Actor, ReleasePostingBarrierCommand) (SoftCloseGateResult, error)
	FinalizePostingGate(context.Context, Actor, FinalizePostingGateCommand) (SoftCloseGateResult, error)
	OpenScopedReopenGate(context.Context, Actor, OpenScopedReopenGateCommand) (SoftCloseGateResult, error)
	CloseScopedReopenGate(context.Context, Actor, CloseScopedReopenGateCommand) (SoftCloseGateResult, error)
	GetPostingGateStatus(context.Context, uuid.UUID, uuid.UUID) (PostingGateStatus, error)
}

type AcquirePostingBarrierCommand struct {
	CommandID              uuid.UUID
	AccountingScopeID      uuid.UUID
	FiscalPeriodID         uuid.UUID
	ExpectedGateVersion    int64
	SoftCloseRunID         uuid.UUID
	SoftCloseControlEpoch  int64
	SoftClosePolicyVersion string
	CloseRunID             uuid.UUID
	CloseType              string
	IdempotencyKey         string
	CorrelationID          string
	CausationID            string
}

func (command AcquirePostingBarrierCommand) Canonical() AcquirePostingBarrierCommand {
	command.SoftClosePolicyVersion = strings.TrimSpace(command.SoftClosePolicyVersion)
	command.CloseType = strings.TrimSpace(command.CloseType)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command AcquirePostingBarrierCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "accountingScopeId": command.AccountingScopeID,
		"fiscalPeriodId": command.FiscalPeriodID, "softCloseRunId": command.SoftCloseRunID,
		"closeRunId": command.CloseRunID,
	} {
		if value == uuid.Nil {
			issue("required", field, field+" is required")
		}
	}
	if command.ExpectedGateVersion < 1 {
		issue("invalid_version", "expectedVersion", "expectedVersion must be positive")
	}
	if command.SoftCloseControlEpoch < 1 {
		issue("invalid_epoch", "softCloseControlEpoch", "softCloseControlEpoch must be positive")
	}
	if command.SoftClosePolicyVersion == "" {
		issue("required", "policyVersion", "policyVersion is required")
	}
	if command.CloseType != CloseTypeInitialHardClose {
		issue("invalid_close_type", "closeType", "closeType must be InitialHardClose")
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

type ReleasePostingBarrierCommand struct {
	CommandID           uuid.UUID
	AccountingScopeID   uuid.UUID
	FiscalPeriodID      uuid.UUID
	ExpectedGateVersion int64
	CloseRunID          uuid.UUID
	IdempotencyKey      string
	CorrelationID       string
	CausationID         string
}

func (command ReleasePostingBarrierCommand) Canonical() ReleasePostingBarrierCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command ReleasePostingBarrierCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "accountingScopeId": command.AccountingScopeID,
		"fiscalPeriodId": command.FiscalPeriodID, "closeRunId": command.CloseRunID,
	} {
		if value == uuid.Nil {
			issue("required", field, field+" is required")
		}
	}
	if command.ExpectedGateVersion < 1 {
		issue("invalid_version", "expectedVersion", "expectedVersion must be positive")
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

type FinalizePostingGateCommand struct {
	CommandID            uuid.UUID
	AccountingScopeID    uuid.UUID
	FiscalPeriodID       uuid.UUID
	ExpectedGateVersion  int64
	CloseRunID           uuid.UUID
	FinalLedgerWatermark int64
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

func (command FinalizePostingGateCommand) Canonical() FinalizePostingGateCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command FinalizePostingGateCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "accountingScopeId": command.AccountingScopeID,
		"fiscalPeriodId": command.FiscalPeriodID, "closeRunId": command.CloseRunID,
	} {
		if value == uuid.Nil {
			issue("required", field, field+" is required")
		}
	}
	if command.ExpectedGateVersion < 1 {
		issue("invalid_version", "expectedVersion", "expectedVersion must be positive")
	}
	if command.FinalLedgerWatermark < 0 {
		issue("invalid_watermark", "finalLedgerWatermark", "finalLedgerWatermark must not be negative")
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

func (service *SoftCloseGateService) AcquirePostingBarrier(ctx context.Context, actor Actor, command AcquirePostingBarrierCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := postingBarrierFingerprint(struct {
		Operation              string    `json:"operation"`
		AccountingScopeID      uuid.UUID `json:"accountingScopeId"`
		FiscalPeriodID         uuid.UUID `json:"fiscalPeriodId"`
		ExpectedGateVersion    int64     `json:"expectedGateVersion"`
		SoftCloseRunID         uuid.UUID `json:"softCloseRunId"`
		SoftCloseControlEpoch  int64     `json:"softCloseControlEpoch"`
		SoftClosePolicyVersion string    `json:"softClosePolicyVersion"`
		CloseRunID             uuid.UUID `json:"closeRunId"`
		CloseType              string    `json:"closeType"`
	}{PostingBarrierAcquireOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.SoftCloseRunID, command.SoftCloseControlEpoch, command.SoftClosePolicyVersion, command.CloseRunID, command.CloseType})
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, PostingBarrierAcquireOperation, PostingBarrierAcquireCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.acquirePostingBarrier(command, service.clock().UTC(), fingerprint)
	})
}

func (service *SoftCloseGateService) ReleasePostingBarrier(ctx context.Context, actor Actor, command ReleasePostingBarrierCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := postingBarrierFingerprint(struct {
		Operation           string    `json:"operation"`
		AccountingScopeID   uuid.UUID `json:"accountingScopeId"`
		FiscalPeriodID      uuid.UUID `json:"fiscalPeriodId"`
		ExpectedGateVersion int64     `json:"expectedGateVersion"`
		CloseRunID          uuid.UUID `json:"closeRunId"`
	}{PostingBarrierReleaseOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.CloseRunID})
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, PostingBarrierReleaseOperation, PostingBarrierReleaseCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.releasePostingBarrier(command, service.clock().UTC(), fingerprint)
	})
}

func (service *SoftCloseGateService) FinalizePostingGate(ctx context.Context, actor Actor, command FinalizePostingGateCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := postingBarrierFingerprint(struct {
		Operation            string    `json:"operation"`
		AccountingScopeID    uuid.UUID `json:"accountingScopeId"`
		FiscalPeriodID       uuid.UUID `json:"fiscalPeriodId"`
		ExpectedGateVersion  int64     `json:"expectedGateVersion"`
		CloseRunID           uuid.UUID `json:"closeRunId"`
		FinalLedgerWatermark int64     `json:"finalLedgerWatermark"`
	}{PostingBarrierFinalizeOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.CloseRunID, command.FinalLedgerWatermark})
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, PostingBarrierFinalizeOperation, PostingBarrierFinalizeCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.finalizePostingGate(command, service.clock().UTC(), fingerprint)
	})
}

func (gate PeriodPostingGate) acquirePostingBarrier(command AcquirePostingBarrierCommand, now time.Time, fingerprint string) (PeriodPostingGate, error) {
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	if gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID {
		return PeriodPostingGate{}, ErrSoftCloseGateScopeConflict
	}
	if gate.GateVersion != command.ExpectedGateVersion {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateVersionConflict, command.ExpectedGateVersion, gate.GateVersion)
	}
	if command.CloseType != CloseTypeInitialHardClose {
		return PeriodPostingGate{}, ErrSoftCloseGatePolicyConflict
	}
	if gate.GateMode != PostingGateSoftClosePolicy {
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not eligible for barrier acquisition", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if gate.ActiveSoftCloseRunID == nil || *gate.ActiveSoftCloseRunID != command.SoftCloseRunID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if gate.ActiveSoftCloseControlEpoch != command.SoftCloseControlEpoch {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateEpochConflict, command.SoftCloseControlEpoch, gate.ActiveSoftCloseControlEpoch)
	}
	if gate.SoftClosePolicyVersion != command.SoftClosePolicyVersion {
		return PeriodPostingGate{}, ErrSoftCloseGatePolicyConflict
	}
	if gate.ActiveCloseRunID != nil {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}

	priorOwner := *gate.ActiveSoftCloseRunID
	priorEpoch := int64Pointer(gate.ActiveSoftCloseControlEpoch)
	closeOwner := command.CloseRunID
	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	summary := freezeGateAdmissionSummary(gate.ActiveAdmissionCounters, gate.GateVersion, now, fingerprint, PostingBarrierAcquireCommandIdentity)
	gate.LastFrozenAdmissionSummary = &summary
	gate.FrozenAdmissionSummaryHistory = append(gate.FrozenAdmissionSummaryHistory, summary)
	gate.PriorControlProcessID = &priorOwner
	gate.PriorControlProcessType = SoftCloseGateOwnerType
	gate.PriorControlEpoch = priorEpoch
	gate.PriorGateMode = gate.GateMode
	gate.ActiveSoftCloseRunID = nil
	gate.ActiveSoftCloseControlEpoch = 0
	gate.ActiveCloseRunID = &closeOwner
	gate.ActiveCloseType = command.CloseType
	gate.GateMode = PostingGateCloseOnly
	gate.BarrierLedgerPosition = gate.NextLedgerPosition - 1
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{ControlOwnerType: PostingBarrierOwnerType, ControlOwnerID: closeOwner}
	gate.UpdatedAt = now
	_ = fingerprint
	return gate, nil
}

func (gate PeriodPostingGate) releasePostingBarrier(command ReleasePostingBarrierCommand, now time.Time, fingerprint string) (PeriodPostingGate, error) {
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	if gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID {
		return PeriodPostingGate{}, ErrSoftCloseGateScopeConflict
	}
	if gate.GateVersion != command.ExpectedGateVersion {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateVersionConflict, command.ExpectedGateVersion, gate.GateVersion)
	}
	if gate.GateMode != PostingGateCloseOnly {
		if gate.GateMode == PostingGateHardClosed {
			return PeriodPostingGate{}, ErrPostingBarrierNotReleasable
		}
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not close-only", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if gate.ActiveCloseRunID == nil || *gate.ActiveCloseRunID != command.CloseRunID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if gate.ActiveCloseType != CloseTypeInitialHardClose {
		return PeriodPostingGate{}, ErrPostingBarrierNotReleasable
	}
	if gate.ActiveAdmissionCounters.ClosePostingAdmitted || gate.ActiveAdmissionCounters.AdmittedPostingCount > 0 {
		return PeriodPostingGate{}, ErrPostingBarrierAdmissionConflict
	}
	if gate.PriorControlProcessID == nil || gate.PriorControlProcessType != SoftCloseGateOwnerType || gate.PriorControlEpoch == nil || gate.PriorGateMode == "" {
		return PeriodPostingGate{}, ErrPostingBarrierNotReleasable
	}

	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	summary := freezeGateAdmissionSummary(gate.ActiveAdmissionCounters, gate.GateVersion, now, fingerprint, PostingBarrierReleaseCommandIdentity)
	gate.LastFrozenAdmissionSummary = &summary
	gate.FrozenAdmissionSummaryHistory = append(gate.FrozenAdmissionSummaryHistory, summary)
	newEpoch := *gate.PriorControlEpoch + 1
	softOwner := *gate.PriorControlProcessID
	gate.GateMode = gate.PriorGateMode
	gate.ActiveSoftCloseRunID = &softOwner
	gate.ActiveSoftCloseControlEpoch = newEpoch
	gate.ActiveCloseRunID = nil
	gate.ActiveCloseType = ""
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{ControlOwnerType: SoftCloseGateOwnerType, ControlOwnerID: softOwner, ControlOwnerEpoch: int64Pointer(newEpoch)}
	gate.UpdatedAt = now
	return gate, nil
}

func (gate PeriodPostingGate) finalizePostingGate(command FinalizePostingGateCommand, now time.Time, fingerprint string) (PeriodPostingGate, error) {
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	if gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID {
		return PeriodPostingGate{}, ErrSoftCloseGateScopeConflict
	}
	if gate.GateVersion != command.ExpectedGateVersion {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateVersionConflict, command.ExpectedGateVersion, gate.GateVersion)
	}
	if gate.GateMode == PostingGateHardClosed {
		return PeriodPostingGate{}, ErrPostingBarrierFinalized
	}
	if gate.GateMode != PostingGateCloseOnly {
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not close-only", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if gate.ActiveCloseRunID == nil || *gate.ActiveCloseRunID != command.CloseRunID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if command.FinalLedgerWatermark < gate.BarrierLedgerPosition {
		return PeriodPostingGate{}, ErrPostingBarrierWatermarkConflict
	}

	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	summary := freezeGateAdmissionSummary(gate.ActiveAdmissionCounters, gate.GateVersion, now, fingerprint, PostingBarrierFinalizeCommandIdentity)
	gate.LastFrozenAdmissionSummary = &summary
	gate.FrozenAdmissionSummaryHistory = append(gate.FrozenAdmissionSummaryHistory, summary)
	finalizedOwner := *gate.ActiveCloseRunID
	watermark := command.FinalLedgerWatermark
	gate.FinalLedgerWatermark = &watermark
	gate.FinalizedCloseRunID = &finalizedOwner
	gate.FinalizationCommandFingerprint = fingerprint
	gate.GateMode = PostingGateHardClosed
	gate.ActiveSoftCloseRunID = nil
	gate.ActiveSoftCloseControlEpoch = 0
	gate.ActiveCloseRunID = nil
	gate.ActiveCloseType = ""
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{}
	gate.UpdatedAt = now
	return gate, nil
}

func freezeGateAdmissionSummary(counters ActiveGateAdmissionCounters, gateVersion int64, now time.Time, fingerprint, commandType string) FrozenGateAdmissionSummary {
	return FrozenGateAdmissionSummary{
		ControlOwnerType:            counters.ControlOwnerType,
		ControlOwnerID:              counters.ControlOwnerID,
		ControlOwnerEpoch:           cloneInt64Pointer(counters.ControlOwnerEpoch),
		ClosePostingAdmitted:        counters.ClosePostingAdmitted,
		ReopenPostingAdmitted:       counters.ReopenPostingAdmitted,
		AdmittedPostingCount:        counters.AdmittedPostingCount,
		FirstAdmittedLedgerPosition: counters.FirstAdmittedLedgerPosition,
		LastAdmittedLedgerPosition:  counters.LastAdmittedLedgerPosition,
		FrozenAtGateVersion:         gateVersion,
		FrozenAt:                    now,
		FrozenByCommandFingerprint:  fingerprint,
		FrozenByCommandType:         commandType,
	}
}

func postingBarrierFingerprint(value any) (string, error) {
	return softCloseGateFingerprint(value)
}

func postingGateProcessID(gate PeriodPostingGate) *uuid.UUID {
	for _, value := range []*uuid.UUID{gate.ActiveCloseRunID, gate.ActiveReopenRequestID, gate.ActiveSoftCloseRunID, gate.FinalizedCloseRunID} {
		if value != nil {
			copy := *value
			return &copy
		}
	}
	return nil
}
