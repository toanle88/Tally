package gl

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	platformevents "github.com/toanle88/Tally/internal/platform/events"
)

const (
	ScopedReopenGateOpenPermission  = "finance.gl.open.scoped.reopen.gate"
	ScopedReopenGateClosePermission = "finance.gl.close.scoped.reopen.gate"

	ScopedReopenGateOpenOperation  = "open-scoped-reopen-gate"
	ScopedReopenGateCloseOperation = "close-scoped-reopen-gate"

	ScopedReopenGateOpenCommandIdentity  = "open-scoped-reopen-gate.v1"
	ScopedReopenGateCloseCommandIdentity = "close-scoped-reopen-gate.v1"

	ScopedReopenGateOpenedEventType = "scoped_reopen_gate_opened"
	ScopedReopenGateClosedEventType = "scoped_reopen_gate_closed"
	ScopedReopenGateOwnerType       = "ReopenRequest"
)

// ScopedReopenScope is the bounded correction envelope recorded on the GL
// gate. Account and transaction-class restrictions are conjunctive when both
// dimensions are populated.
type ScopedReopenScope struct {
	AccountIDs         []uuid.UUID `json:"accountIds,omitempty"`
	TransactionClasses []string    `json:"transactionClasses,omitempty"`
}

func (scope ScopedReopenScope) Canonical() (ScopedReopenScope, error) {
	accounts := append([]uuid.UUID(nil), scope.AccountIDs...)
	for _, accountID := range accounts {
		if accountID == uuid.Nil {
			return ScopedReopenScope{}, fmt.Errorf("%w: accountIds cannot contain the nil UUID", ErrInvalidSoftCloseGateCommand)
		}
	}
	sort.Slice(accounts, func(left, right int) bool { return accounts[left].String() < accounts[right].String() })
	accounts = dedupeUUIDs(accounts)

	classes := make([]string, 0, len(scope.TransactionClasses))
	seenClasses := make(map[string]struct{}, len(scope.TransactionClasses))
	for _, transactionClass := range scope.TransactionClasses {
		transactionClass = strings.TrimSpace(transactionClass)
		if transactionClass == "" {
			return ScopedReopenScope{}, fmt.Errorf("%w: transactionClasses cannot contain an empty value", ErrInvalidSoftCloseGateCommand)
		}
		if _, exists := seenClasses[transactionClass]; exists {
			continue
		}
		seenClasses[transactionClass] = struct{}{}
		classes = append(classes, transactionClass)
	}
	sort.Strings(classes)
	if len(accounts) == 0 && len(classes) == 0 {
		return ScopedReopenScope{}, fmt.Errorf("%w: at least one account or transaction class is required", ErrInvalidSoftCloseGateCommand)
	}
	return ScopedReopenScope{AccountIDs: accounts, TransactionClasses: classes}, nil
}

func dedupeUUIDs(values []uuid.UUID) []uuid.UUID {
	if len(values) < 2 {
		return values
	}
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func canonicalScopedReopenScope(scope ScopedReopenScope) (ScopedReopenScope, json.RawMessage, error) {
	canonical, err := scope.Canonical()
	if err != nil {
		return ScopedReopenScope{}, nil, err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return ScopedReopenScope{}, nil, err
	}
	canonicalPayload, err := platformevents.CanonicalizePayload(payload)
	if err != nil {
		return ScopedReopenScope{}, nil, err
	}
	return canonical, json.RawMessage(canonicalPayload), nil
}

func decodeScopedReopenScope(value json.RawMessage) (ScopedReopenScope, error) {
	var scope ScopedReopenScope
	if len(value) == 0 {
		return scope, fmt.Errorf("%w: authorization scope is required", ErrInvalidSoftCloseGateCommand)
	}
	if err := json.Unmarshal(value, &scope); err != nil {
		return scope, fmt.Errorf("%w: authorization scope is invalid", ErrInvalidSoftCloseGateCommand)
	}
	canonical, _, err := canonicalScopedReopenScope(scope)
	return canonical, err
}

type OpenScopedReopenGateCommand struct {
	CommandID              uuid.UUID
	AccountingScopeID      uuid.UUID
	FiscalPeriodID         uuid.UUID
	ExpectedGateVersion    int64
	ReopenRequestID        uuid.UUID
	AuthorizationScope     ScopedReopenScope
	AuthorizationExpiry    time.Time
	ControlAuthorityEpoch  int64
	Approval               ApprovalDecisionReference
	NoChangeClosePermitted bool
	IdempotencyKey         string
	CorrelationID          string
	CausationID            string
}

func (command OpenScopedReopenGateCommand) Canonical() OpenScopedReopenGateCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.AuthorizationExpiry = command.AuthorizationExpiry.UTC()
	if canonical, err := command.AuthorizationScope.Canonical(); err == nil {
		command.AuthorizationScope = canonical
	}
	return command
}

func (command OpenScopedReopenGateCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "accountingScopeId": command.AccountingScopeID,
		"fiscalPeriodId": command.FiscalPeriodID, "reopenRequestId": command.ReopenRequestID,
	} {
		if value == uuid.Nil {
			issue("required", field, field+" is required")
		}
	}
	if command.ExpectedGateVersion < 1 {
		issue("invalid_version", "expectedVersion", "expectedVersion must be positive")
	}
	if command.AuthorizationExpiry.IsZero() {
		issue("required", "authorizationExpiry", "authorizationExpiry is required")
	}
	if command.ControlAuthorityEpoch < 1 {
		issue("invalid_epoch", "controlAuthorityEpoch", "controlAuthorityEpoch must be positive")
	}
	if _, err := command.AuthorizationScope.Canonical(); err != nil {
		issue("invalid_scope", "authorizationScope", "authorizationScope must contain at least one valid account or transaction class")
	}
	if err := command.Approval.Validate(); err != nil {
		issue("invalid_approval", "approval", "approval evidence is incomplete")
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

type CloseScopedReopenGateCommand struct {
	CommandID           uuid.UUID
	AccountingScopeID   uuid.UUID
	FiscalPeriodID      uuid.UUID
	ExpectedGateVersion int64
	ReopenRequestID     uuid.UUID
	IdempotencyKey      string
	CorrelationID       string
	CausationID         string
}

func (command CloseScopedReopenGateCommand) Canonical() CloseScopedReopenGateCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command CloseScopedReopenGateCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "accountingScopeId": command.AccountingScopeID,
		"fiscalPeriodId": command.FiscalPeriodID, "reopenRequestId": command.ReopenRequestID,
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

func (service *SoftCloseGateService) OpenScopedReopenGate(ctx context.Context, actor Actor, command OpenScopedReopenGateCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := openScopedReopenGateFingerprint(command)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, ScopedReopenGateOpenOperation, ScopedReopenGateOpenCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.openScopedReopenGate(command, service.clock().UTC())
	})
}

func (service *SoftCloseGateService) CloseScopedReopenGate(ctx context.Context, actor Actor, command CloseScopedReopenGateCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := closeScopedReopenGateFingerprint(command)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, ScopedReopenGateCloseOperation, ScopedReopenGateCloseCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.closeScopedReopenGate(command, service.clock().UTC(), fingerprint)
	})
}

func (gate PeriodPostingGate) openScopedReopenGate(command OpenScopedReopenGateCommand, now time.Time) (PeriodPostingGate, error) {
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	if gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID {
		return PeriodPostingGate{}, ErrSoftCloseGateScopeConflict
	}
	if gate.GateVersion != command.ExpectedGateVersion {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateVersionConflict, command.ExpectedGateVersion, gate.GateVersion)
	}
	if gate.GateMode != PostingGateHardClosed {
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not hard-closed", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if !command.AuthorizationExpiry.After(now) {
		return PeriodPostingGate{}, ErrSoftCloseGateAuthorizationExpired
	}
	_, scopeJSON, err := canonicalScopedReopenScope(command.AuthorizationScope)
	if err != nil {
		return PeriodPostingGate{}, err
	}
	requestID := command.ReopenRequestID
	authorityEpoch := command.ControlAuthorityEpoch
	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	gate.GateMode = PostingGateScopedReopen
	gate.ActiveReopenRequestID = &requestID
	gate.ReopenAuthorizationScope = scopeJSON
	expiry := command.AuthorizationExpiry
	gate.ReopenAuthorizationExpiry = &expiry
	gate.ReopenControlAuthorityEpoch = &authorityEpoch
	gate.ReopenApproval = cloneApproval(&command.Approval)
	gate.ReopenNoChangeClosePermitted = command.NoChangeClosePermitted
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{ControlOwnerType: ScopedReopenGateOwnerType, ControlOwnerID: requestID, ControlOwnerEpoch: int64Pointer(authorityEpoch)}
	gate.UpdatedAt = now
	return gate, nil
}

func (gate PeriodPostingGate) closeScopedReopenGate(command CloseScopedReopenGateCommand, now time.Time, fingerprint string) (PeriodPostingGate, error) {
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	if gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID {
		return PeriodPostingGate{}, ErrSoftCloseGateScopeConflict
	}
	if gate.GateVersion != command.ExpectedGateVersion {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateVersionConflict, command.ExpectedGateVersion, gate.GateVersion)
	}
	if gate.GateMode != PostingGateScopedReopen {
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not scoped reopen", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if gate.ActiveReopenRequestID == nil || *gate.ActiveReopenRequestID != command.ReopenRequestID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if gate.ActiveAdmissionCounters.ControlOwnerType != ScopedReopenGateOwnerType || gate.ActiveAdmissionCounters.ControlOwnerID != command.ReopenRequestID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}

	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	summary := freezeGateAdmissionSummary(gate.ActiveAdmissionCounters, gate.GateVersion, now, fingerprint, ScopedReopenGateCloseCommandIdentity)
	gate.LastFrozenAdmissionSummary = &summary
	gate.FrozenAdmissionSummaryHistory = append(gate.FrozenAdmissionSummaryHistory, summary)
	if gate.ActiveAdmissionCounters.AdmittedPostingCount == 0 && !gate.ActiveAdmissionCounters.ReopenPostingAdmitted && gate.ReopenNoChangeClosePermitted {
		gate.GateMode = PostingGateHardClosed
		gate.ActiveReopenRequestID = nil
		gate.ReopenAuthorizationScope = nil
		gate.ReopenAuthorizationExpiry = nil
		gate.ReopenControlAuthorityEpoch = nil
		gate.ReopenApproval = nil
		gate.ReopenNoChangeClosePermitted = false
		gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{}
	} else {
		gate.GateMode = PostingGateCloseOnly
	}
	gate.UpdatedAt = now
	return gate, nil
}

func openScopedReopenGateFingerprint(command OpenScopedReopenGateCommand) (string, error) {
	_, scopeJSON, err := canonicalScopedReopenScope(command.AuthorizationScope)
	if err != nil {
		return "", err
	}
	return softCloseGateFingerprint(struct {
		Operation              string                    `json:"operation"`
		AccountingScopeID      uuid.UUID                 `json:"accountingScopeId"`
		FiscalPeriodID         uuid.UUID                 `json:"fiscalPeriodId"`
		ExpectedGateVersion    int64                     `json:"expectedGateVersion"`
		ReopenRequestID        uuid.UUID                 `json:"reopenRequestId"`
		AuthorizationScope     json.RawMessage           `json:"authorizationScope"`
		AuthorizationExpiry    time.Time                 `json:"authorizationExpiry"`
		ControlAuthorityEpoch  int64                     `json:"controlAuthorityEpoch"`
		Approval               ApprovalDecisionReference `json:"approval"`
		NoChangeClosePermitted bool                      `json:"noChangeClosePermitted"`
	}{ScopedReopenGateOpenOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.ReopenRequestID, scopeJSON, command.AuthorizationExpiry.UTC(), command.ControlAuthorityEpoch, command.Approval, command.NoChangeClosePermitted})
}

func closeScopedReopenGateFingerprint(command CloseScopedReopenGateCommand) (string, error) {
	return softCloseGateFingerprint(struct {
		Operation           string    `json:"operation"`
		AccountingScopeID   uuid.UUID `json:"accountingScopeId"`
		FiscalPeriodID      uuid.UUID `json:"fiscalPeriodId"`
		ExpectedGateVersion int64     `json:"expectedGateVersion"`
		ReopenRequestID     uuid.UUID `json:"reopenRequestId"`
	}{ScopedReopenGateCloseOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.ReopenRequestID})
}

func validateScopedReopenScope(request PostingRequest, scopeJSON json.RawMessage) error {
	scope, err := decodeScopedReopenScope(scopeJSON)
	if err != nil {
		return fmt.Errorf("%w: authorization scope is invalid", ErrPostingGateClosed)
	}
	if len(scope.AccountIDs) > 0 {
		allowed := make(map[uuid.UUID]struct{}, len(scope.AccountIDs))
		for _, accountID := range scope.AccountIDs {
			allowed[accountID] = struct{}{}
		}
		for _, line := range request.Lines {
			if _, ok := allowed[line.AccountID]; !ok {
				return fmt.Errorf("%w: account %s is outside the approved reopen scope", ErrPostingGateClosed, line.AccountID)
			}
		}
	}
	if len(scope.TransactionClasses) > 0 {
		allowed := make(map[string]struct{}, len(scope.TransactionClasses))
		for _, transactionClass := range scope.TransactionClasses {
			allowed[transactionClass] = struct{}{}
		}
		if _, ok := allowed[request.SourceAggregateType]; !ok {
			return fmt.Errorf("%w: transaction class %q is outside the approved reopen scope", ErrPostingGateClosed, request.SourceAggregateType)
		}
	}
	return nil
}
