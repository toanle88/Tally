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
	OperationalReopenGateOpenPermission  = "finance.gl.open.operational.reopen.gate"
	OperationalReopenGateClosePermission = "finance.gl.close.operational.reopen.gate"
	BeginRecloseGatePermission           = "finance.gl.begin.reclose.gate"

	OperationalReopenGateOpenOperation  = "open-operational-reopen-gate"
	OperationalReopenGateCloseOperation = "close-operational-reopen-gate"
	BeginRecloseGateOperation           = "begin-reclose-gate"

	OperationalReopenGateOpenCommandIdentity  = "open-operational-reopen-gate.v1"
	OperationalReopenGateCloseCommandIdentity = "close-operational-reopen-gate.v1"
	BeginRecloseGateCommandIdentity           = "begin-reclose-gate.v1"

	OperationalReopenGateOpenedEventType  = "operational_reopen_gate_opened"
	OperationalReopenGateClosedEventType  = "operational_reopen_gate_closed"
	OperationalReopenGateExpiredEventType = "operational_reopen_gate_expired"
	RecloseGateBegunEventType             = "reclose_gate_begun"

	OperationalReopenGateOwnerType = "ReopenRequest"
)

// OperationalReopenScope is the bounded operational-recovery envelope
// recorded on the GL gate. Both dimensions are required and are evaluated
// conjunctively at posting admission.
type OperationalReopenScope struct {
	ActorIDs           []uuid.UUID `json:"actorIds"`
	TransactionClasses []string    `json:"transactionClasses"`
}

func (scope OperationalReopenScope) Canonical() (OperationalReopenScope, error) {
	actors := append([]uuid.UUID(nil), scope.ActorIDs...)
	for _, actorID := range actors {
		if actorID == uuid.Nil {
			return OperationalReopenScope{}, fmt.Errorf("%w: actorIds cannot contain the nil UUID", ErrInvalidSoftCloseGateCommand)
		}
	}
	sort.Slice(actors, func(left, right int) bool { return actors[left].String() < actors[right].String() })
	actors = dedupeUUIDs(actors)

	classes := make([]string, 0, len(scope.TransactionClasses))
	seenClasses := make(map[string]struct{}, len(scope.TransactionClasses))
	for _, transactionClass := range scope.TransactionClasses {
		transactionClass = strings.TrimSpace(transactionClass)
		if transactionClass == "" {
			return OperationalReopenScope{}, fmt.Errorf("%w: transactionClasses cannot contain an empty value", ErrInvalidSoftCloseGateCommand)
		}
		if _, exists := seenClasses[transactionClass]; exists {
			continue
		}
		seenClasses[transactionClass] = struct{}{}
		classes = append(classes, transactionClass)
	}
	sort.Strings(classes)
	if len(actors) == 0 || len(classes) == 0 {
		return OperationalReopenScope{}, fmt.Errorf("%w: actorIds and transactionClasses are both required", ErrInvalidSoftCloseGateCommand)
	}
	return OperationalReopenScope{ActorIDs: actors, TransactionClasses: classes}, nil
}

func canonicalOperationalReopenScope(scope OperationalReopenScope) (OperationalReopenScope, json.RawMessage, error) {
	canonical, err := scope.Canonical()
	if err != nil {
		return OperationalReopenScope{}, nil, err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return OperationalReopenScope{}, nil, err
	}
	canonicalPayload, err := platformevents.CanonicalizePayload(payload)
	if err != nil {
		return OperationalReopenScope{}, nil, err
	}
	return canonical, json.RawMessage(canonicalPayload), nil
}

func decodeOperationalReopenScope(value json.RawMessage) (OperationalReopenScope, error) {
	var scope OperationalReopenScope
	if len(value) == 0 {
		return scope, fmt.Errorf("%w: authorization scope is required", ErrInvalidSoftCloseGateCommand)
	}
	if err := json.Unmarshal(value, &scope); err != nil {
		return scope, fmt.Errorf("%w: authorization scope is invalid", ErrInvalidSoftCloseGateCommand)
	}
	canonical, _, err := canonicalOperationalReopenScope(scope)
	return canonical, err
}

type OpenOperationalReopenGateCommand struct {
	CommandID              uuid.UUID
	AccountingScopeID      uuid.UUID
	FiscalPeriodID         uuid.UUID
	ExpectedGateVersion    int64
	ReopenRequestID        uuid.UUID
	AuthorizationScope     OperationalReopenScope
	AuthorizationExpiry    time.Time
	ControlAuthorityEpoch  int64
	Approval               ApprovalDecisionReference
	NoChangeClosePermitted bool
	IdempotencyKey         string
	CorrelationID          string
	CausationID            string
}

func (command OpenOperationalReopenGateCommand) Canonical() OpenOperationalReopenGateCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.AuthorizationExpiry = command.AuthorizationExpiry.UTC()
	if canonical, err := command.AuthorizationScope.Canonical(); err == nil {
		command.AuthorizationScope = canonical
	}
	return command
}

func (command OpenOperationalReopenGateCommand) Validate() error {
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
		issue("invalid_scope", "authorizationScope", "authorizationScope must contain actorIds and transactionClasses")
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

type CloseOperationalReopenGateCommand struct {
	CommandID           uuid.UUID
	AccountingScopeID   uuid.UUID
	FiscalPeriodID      uuid.UUID
	ExpectedGateVersion int64
	ReopenRequestID     uuid.UUID
	IdempotencyKey      string
	CorrelationID       string
	CausationID         string
}

func (command CloseOperationalReopenGateCommand) Canonical() CloseOperationalReopenGateCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command CloseOperationalReopenGateCommand) Validate() error {
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

type BeginRecloseGateCommand struct {
	CommandID           uuid.UUID
	AccountingScopeID   uuid.UUID
	FiscalPeriodID      uuid.UUID
	ExpectedGateVersion int64
	ReopenRequestID     uuid.UUID
	RecloseRunID        uuid.UUID
	IdempotencyKey      string
	CorrelationID       string
	CausationID         string
}

func (command BeginRecloseGateCommand) Canonical() BeginRecloseGateCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command BeginRecloseGateCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "accountingScopeId": command.AccountingScopeID,
		"fiscalPeriodId": command.FiscalPeriodID, "reopenRequestId": command.ReopenRequestID,
		"recloseRunId": command.RecloseRunID,
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

func (service *SoftCloseGateService) OpenOperationalReopenGate(ctx context.Context, actor Actor, command OpenOperationalReopenGateCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := openOperationalReopenGateFingerprint(command)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, OperationalReopenGateOpenOperation, OperationalReopenGateOpenCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.openOperationalReopenGate(command, service.clock().UTC())
	})
}

func (service *SoftCloseGateService) CloseOperationalReopenGate(ctx context.Context, actor Actor, command CloseOperationalReopenGateCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := closeOperationalReopenGateFingerprint(command)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, OperationalReopenGateCloseOperation, OperationalReopenGateCloseCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.closeOperationalReopenGate(command, service.clock().UTC(), fingerprint)
	})
}

func (service *SoftCloseGateService) BeginRecloseGate(ctx context.Context, actor Actor, command BeginRecloseGateCommand) (SoftCloseGateResult, error) {
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SoftCloseGateResult{}, err
	}
	fingerprint, err := beginRecloseGateFingerprint(command)
	if err != nil {
		return SoftCloseGateResult{}, err
	}
	return service.execute(ctx, actor, BeginRecloseGateOperation, BeginRecloseGateCommandIdentity, command.CommandID, command.AccountingScopeID, command.FiscalPeriodID, command.IdempotencyKey, command.CorrelationID, command.CausationID, fingerprint, nil, func(before PeriodPostingGate) (PeriodPostingGate, error) {
		return before.beginRecloseGate(command, service.clock().UTC(), fingerprint)
	})
}

func (gate PeriodPostingGate) openOperationalReopenGate(command OpenOperationalReopenGateCommand, now time.Time) (PeriodPostingGate, error) {
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
	_, scopeJSON, err := canonicalOperationalReopenScope(command.AuthorizationScope)
	if err != nil {
		return PeriodPostingGate{}, err
	}
	requestID := command.ReopenRequestID
	authorityEpoch := command.ControlAuthorityEpoch
	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	gate.GateMode = PostingGateOperationalReopen
	gate.ActiveReopenRequestID = &requestID
	gate.ReopenAuthorizationScope = scopeJSON
	expiry := command.AuthorizationExpiry
	gate.ReopenAuthorizationExpiry = &expiry
	gate.ReopenControlAuthorityEpoch = &authorityEpoch
	gate.ReopenApproval = cloneApproval(&command.Approval)
	gate.ReopenNoChangeClosePermitted = command.NoChangeClosePermitted
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{ControlOwnerType: OperationalReopenGateOwnerType, ControlOwnerID: requestID, ControlOwnerEpoch: int64Pointer(authorityEpoch)}
	gate.UpdatedAt = now
	return gate, nil
}

func (gate PeriodPostingGate) closeOperationalReopenGate(command CloseOperationalReopenGateCommand, now time.Time, fingerprint string) (PeriodPostingGate, error) {
	if err := gate.Validate(); err != nil {
		return PeriodPostingGate{}, err
	}
	if gate.AccountingScopeID != command.AccountingScopeID || gate.FiscalPeriodID != command.FiscalPeriodID {
		return PeriodPostingGate{}, ErrSoftCloseGateScopeConflict
	}
	if gate.GateVersion != command.ExpectedGateVersion {
		return PeriodPostingGate{}, fmt.Errorf("%w: expected %d, current %d", ErrSoftCloseGateVersionConflict, command.ExpectedGateVersion, gate.GateVersion)
	}
	if gate.GateMode != PostingGateOperationalReopen {
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not operational reopen", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if gate.ActiveReopenRequestID == nil || *gate.ActiveReopenRequestID != command.ReopenRequestID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if gate.ActiveAdmissionCounters.ControlOwnerType != OperationalReopenGateOwnerType || gate.ActiveAdmissionCounters.ControlOwnerID != command.ReopenRequestID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}

	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	summary := freezeGateAdmissionSummary(gate.ActiveAdmissionCounters, gate.GateVersion, now, fingerprint, OperationalReopenGateCloseCommandIdentity)
	gate.LastFrozenAdmissionSummary = &summary
	gate.FrozenAdmissionSummaryHistory = append(gate.FrozenAdmissionSummaryHistory, summary)
	if gate.ActiveAdmissionCounters.AdmittedPostingCount == 0 && !gate.ActiveAdmissionCounters.ReopenPostingAdmitted && gate.ReopenNoChangeClosePermitted {
		gate.GateMode = PostingGateHardClosed
		clearReopenAuthority(&gate)
		gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{}
	} else {
		gate.GateMode = PostingGateCloseOnly
	}
	gate.UpdatedAt = now
	return gate, nil
}

func (gate PeriodPostingGate) beginRecloseGate(command BeginRecloseGateCommand, now time.Time, fingerprint string) (PeriodPostingGate, error) {
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
		return PeriodPostingGate{}, fmt.Errorf("%w: gate mode %s is not close-only", ErrSoftCloseGateModeConflict, gate.GateMode)
	}
	if gate.ActiveCloseRunID != nil {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if gate.ActiveReopenRequestID == nil || *gate.ActiveReopenRequestID != command.ReopenRequestID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if (gate.ActiveAdmissionCounters.ControlOwnerType != ScopedReopenGateOwnerType && gate.ActiveAdmissionCounters.ControlOwnerType != OperationalReopenGateOwnerType) || gate.ActiveAdmissionCounters.ControlOwnerID != command.ReopenRequestID {
		return PeriodPostingGate{}, ErrSoftCloseGateOwnerConflict
	}
	if gate.LastFrozenAdmissionSummary == nil || gate.LastFrozenAdmissionSummary.ControlOwnerType != gate.ActiveAdmissionCounters.ControlOwnerType || gate.LastFrozenAdmissionSummary.ControlOwnerID != command.ReopenRequestID || (gate.LastFrozenAdmissionSummary.AdmittedPostingCount == 0 && !gate.LastFrozenAdmissionSummary.ReopenPostingAdmitted) {
		return PeriodPostingGate{}, ErrSoftCloseGateRecloseSummaryConflict
	}
	priorGateMode := ""
	switch gate.LastFrozenAdmissionSummary.FrozenByCommandType {
	case ScopedReopenGateCloseCommandIdentity:
		priorGateMode = PostingGateScopedReopen
	case OperationalReopenGateCloseCommandIdentity:
		priorGateMode = PostingGateOperationalReopen
	default:
		return PeriodPostingGate{}, ErrSoftCloseGateRecloseSummaryConflict
	}

	priorOwner := command.ReopenRequestID
	priorEpoch := cloneInt64Pointer(gate.ReopenControlAuthorityEpoch)
	closeOwner := command.RecloseRunID
	gate.GateVersion++
	gate.AggregateVersion = gate.GateVersion
	gate.PriorControlProcessID = &priorOwner
	gate.PriorControlProcessType = gate.LastFrozenAdmissionSummary.ControlOwnerType
	gate.PriorControlEpoch = priorEpoch
	gate.PriorGateMode = priorGateMode
	clearReopenAuthority(&gate)
	gate.ActiveCloseRunID = &closeOwner
	gate.ActiveCloseType = CloseTypeReclose
	gate.GateMode = PostingGateCloseOnly
	gate.BarrierLedgerPosition = gate.NextLedgerPosition - 1
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{ControlOwnerType: PostingBarrierOwnerType, ControlOwnerID: closeOwner}
	gate.UpdatedAt = now
	_ = fingerprint
	return gate, nil
}

func clearReopenAuthority(gate *PeriodPostingGate) {
	if gate == nil {
		return
	}
	gate.ActiveReopenRequestID = nil
	gate.ReopenAuthorizationScope = nil
	gate.ReopenAuthorizationExpiry = nil
	gate.ReopenControlAuthorityEpoch = nil
	gate.ReopenApproval = nil
	gate.ReopenNoChangeClosePermitted = false
}

func openOperationalReopenGateFingerprint(command OpenOperationalReopenGateCommand) (string, error) {
	_, scopeJSON, err := canonicalOperationalReopenScope(command.AuthorizationScope)
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
	}{OperationalReopenGateOpenOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.ReopenRequestID, scopeJSON, command.AuthorizationExpiry.UTC(), command.ControlAuthorityEpoch, command.Approval, command.NoChangeClosePermitted})
}

func closeOperationalReopenGateFingerprint(command CloseOperationalReopenGateCommand) (string, error) {
	return softCloseGateFingerprint(struct {
		Operation           string    `json:"operation"`
		AccountingScopeID   uuid.UUID `json:"accountingScopeId"`
		FiscalPeriodID      uuid.UUID `json:"fiscalPeriodId"`
		ExpectedGateVersion int64     `json:"expectedGateVersion"`
		ReopenRequestID     uuid.UUID `json:"reopenRequestId"`
	}{OperationalReopenGateCloseOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.ReopenRequestID})
}

func beginRecloseGateFingerprint(command BeginRecloseGateCommand) (string, error) {
	return softCloseGateFingerprint(struct {
		Operation           string    `json:"operation"`
		AccountingScopeID   uuid.UUID `json:"accountingScopeId"`
		FiscalPeriodID      uuid.UUID `json:"fiscalPeriodId"`
		ExpectedGateVersion int64     `json:"expectedGateVersion"`
		ReopenRequestID     uuid.UUID `json:"reopenRequestId"`
		RecloseRunID        uuid.UUID `json:"recloseRunId"`
	}{BeginRecloseGateOperation, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion, command.ReopenRequestID, command.RecloseRunID})
}

func validateOperationalReopenScope(request PostingRequest, actor Actor, scopeJSON json.RawMessage) error {
	scope, err := decodeOperationalReopenScope(scopeJSON)
	if err != nil {
		return fmt.Errorf("%w: authorization scope is invalid", ErrPostingGateClosed)
	}
	allowedActors := make(map[uuid.UUID]struct{}, len(scope.ActorIDs))
	for _, actorID := range scope.ActorIDs {
		allowedActors[actorID] = struct{}{}
	}
	if _, ok := allowedActors[actor.UserID]; !ok {
		return fmt.Errorf("%w: actor %s is outside the approved operational reopen scope", ErrPostingGateClosed, actor.UserID)
	}
	allowedClasses := make(map[string]struct{}, len(scope.TransactionClasses))
	for _, transactionClass := range scope.TransactionClasses {
		allowedClasses[transactionClass] = struct{}{}
	}
	if _, ok := allowedClasses[request.SourceAggregateType]; !ok {
		return fmt.Errorf("%w: transaction class %q is outside the approved operational reopen scope", ErrPostingGateClosed, request.SourceAggregateType)
	}
	return nil
}
