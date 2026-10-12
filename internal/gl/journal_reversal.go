package gl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/idempotency"
	"github.com/toanle88/Tally/internal/platform/money"
)

const (
	JournalReversalPermission = "finance.gl.reverse.journal.entry"
	JournalReversalOperation  = "reverse-journal-entry"
)

var (
	ErrInvalidJournalReversalCommand  = errors.New("invalid journal reversal command")
	ErrJournalEntryNotFound           = errors.New("journal entry not found")
	ErrJournalEntryNotPosted          = errors.New("journal entry is not eligible for reversal")
	ErrJournalEntryAlreadyReversed    = errors.New("journal entry has already been reversed")
	ErrJournalReversalVersionConflict = errors.New("journal reversal journal version conflict")
)

// ReverseJournalEntryCommand is the transport-independent reversal command.
// The original journal is never changed; the supplied source identity belongs
// to the correction owner that requested the new accounting effect.
type ReverseJournalEntryCommand struct {
	CommandID              uuid.UUID
	OriginalJournalID      uuid.UUID
	AccountingScopeID      uuid.UUID
	ExpectedJournalVersion int64
	CorrectionReason       string
	EffectivePostingDate   time.Time
	FiscalPeriodID         uuid.UUID
	PeriodStateVersion     int64
	PostingGateVersion     int64
	PostingAuthorizationID uuid.UUID
	ReopenRequestID        *uuid.UUID
	ControlAuthorityEpoch  *int64

	SourceContext       string
	SourceAggregateType string
	SourceAggregateID   uuid.UUID
	SourceVersion       int64

	IdempotencyKey string
	CorrelationID  string
	CausationID    string
}

func (command ReverseJournalEntryCommand) Canonical() ReverseJournalEntryCommand {
	command.CorrectionReason = strings.TrimSpace(command.CorrectionReason)
	command.EffectivePostingDate = dateOnly(command.EffectivePostingDate)
	command.SourceContext = strings.TrimSpace(command.SourceContext)
	command.SourceAggregateType = strings.TrimSpace(command.SourceAggregateType)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command ReverseJournalEntryCommand) Validate() error {
	command = command.Canonical()
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	for field, value := range map[string]uuid.UUID{
		"commandId": command.CommandID, "journalId": command.OriginalJournalID,
		"accountingScopeId": command.AccountingScopeID, "fiscalPeriodId": command.FiscalPeriodID,
		"postingAuthorizationId": command.PostingAuthorizationID, "sourceAggregateId": command.SourceAggregateID,
	} {
		if value == uuid.Nil {
			issue("required", field, field+" is required")
		}
	}
	if command.ExpectedJournalVersion < 1 {
		issue("invalid_version", "expectedVersion", "expectedVersion must be positive")
	}
	if command.CorrectionReason == "" {
		issue("required", "reason", "reason is required")
	}
	if command.EffectivePostingDate.IsZero() {
		issue("required", "effectivePostingDate", "effectivePostingDate is required")
	}
	if command.PeriodStateVersion < 1 {
		issue("invalid_version", "periodStateVersion", "periodStateVersion must be positive")
	}
	if command.PostingGateVersion < 1 {
		issue("invalid_version", "postingGateVersion", "postingGateVersion must be positive")
	}
	if command.SourceContext == "" {
		issue("required", "sourceContext", "sourceContext is required")
	}
	if command.SourceAggregateType == "" {
		issue("required", "sourceAggregateType", "sourceAggregateType is required")
	}
	if command.SourceVersion < 1 {
		issue("invalid_version", "sourceVersion", "sourceVersion must be positive")
	}
	if command.IdempotencyKey == "" {
		issue("required", "Idempotency-Key", "Idempotency-Key is required")
	}
	if command.CorrelationID == "" || command.CausationID == "" {
		issue("required", "correlation", "correlation and causation references are required")
	}
	if (command.ReopenRequestID == nil) != (command.ControlAuthorityEpoch == nil) {
		issue("required", "reopenEvidence", "reopen request and control authority epoch must be supplied together")
	}
	if command.ReopenRequestID != nil && *command.ReopenRequestID == uuid.Nil {
		issue("required", "reopenRequestId", "reopenRequestId must be a non-zero UUID")
	}
	if command.ControlAuthorityEpoch != nil && *command.ControlAuthorityEpoch < 1 {
		issue("invalid_version", "controlAuthorityEpoch", "controlAuthorityEpoch must be positive")
	}
	if len(issues) > 0 {
		return &PostingValidationError{Issues: issues}
	}
	return nil
}

type JournalEntrySnapshot struct {
	Journal PostingJournal
	Request PostingRequest
	Lines   []PostingLine
}

type JournalReversalResult struct {
	Outcome                string
	Original               JournalEntrySnapshot
	Reversal               JournalEntrySnapshot
	CorrectionReason       string
	AuthorizationReference uuid.UUID
	GateEvidence           PostingGateEvidence
	AuditReference         uuid.UUID
	NextAction             string
	RequestFingerprint     string
	Replayed               bool
}

func (result JournalReversalResult) AggregateID() uuid.UUID {
	return result.Reversal.Journal.ID
}

func (result JournalReversalResult) AggregateVersion() int64 {
	return result.Reversal.Journal.Version
}

func (result JournalReversalResult) LifecycleStatus() string {
	if result.Reversal.Journal.ID == uuid.Nil {
		return "Rejected"
	}
	return result.Reversal.Journal.Status
}

type JournalReversalAuthorizer interface {
	AuthorizeJournalReversal(context.Context, Actor, ReverseJournalEntryCommand, PostingRequest) (PostingAuthorizationDecision, error)
}

type JournalReversalCommit struct {
	Command            ReverseJournalEntryCommand
	Original           JournalEntrySnapshot
	Request            PostingRequest
	Actor              Actor
	Authorization      PostingAuthorizationDecision
	GateEvidence       PostingGateEvidence
	RequestFingerprint string
	Now                time.Time
}

type JournalReversalRepository interface {
	ResolveExistingJournalReversal(context.Context, ReverseJournalEntryCommand, string) (JournalReversalResult, bool, error)
	LoadJournalForReversal(context.Context, uuid.UUID, uuid.UUID) (JournalEntrySnapshot, error)
	CommitJournalReversal(context.Context, JournalReversalCommit) (JournalReversalResult, error)
	RecordJournalReversalAttempt(context.Context, ReverseJournalEntryCommand, Actor, string, string, []PostingIssue) error
}

type JournalReversalService struct {
	repository JournalReversalRepository
	authorizer JournalReversalAuthorizer
	references PostingReferenceValidator
	gate       PostingGateValidator
	currency   money.CurrencyRegistry
	clock      func() time.Time
}

func NewJournalReversalService(
	repository JournalReversalRepository,
	authorizer JournalReversalAuthorizer,
	references PostingReferenceValidator,
	gate PostingGateValidator,
	currency money.CurrencyRegistry,
	clock func() time.Time,
) (*JournalReversalService, error) {
	if repository == nil || authorizer == nil || references == nil || gate == nil || clock == nil {
		return nil, ErrInvalidPostingService
	}
	return &JournalReversalService{repository: repository, authorizer: authorizer, references: references, gate: gate, currency: currency, clock: clock}, nil
}

func (service *JournalReversalService) Execute(ctx context.Context, actor Actor, command ReverseJournalEntryCommand) (JournalReversalResult, error) {
	if service == nil {
		return JournalReversalResult{}, ErrInvalidPostingService
	}
	command = command.Canonical()
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return JournalReversalResult{}, ErrPostingAuthorizationDenied
	}
	if err := command.Validate(); err != nil {
		return service.reject(ctx, command, actor, "invalid", err)
	}
	fingerprint, err := journalReversalFingerprint(command)
	if err != nil {
		return JournalReversalResult{}, fmt.Errorf("%w: fingerprint: %v", ErrInvalidJournalReversalCommand, err)
	}
	if result, found, err := service.repository.ResolveExistingJournalReversal(ctx, command, fingerprint); err != nil {
		return service.reject(ctx, command, actor, "conflict", err)
	} else if found {
		authorizationRequest := result.Reversal.Request
		authorizationRequest.ReversalOfJournalEntryID = nil
		authorization, authorizationErr := service.authorizer.AuthorizeJournalReversal(ctx, actor, command, authorizationRequest)
		if authorizationErr != nil {
			return service.reject(ctx, command, actor, "rejected", authorizationErr)
		}
		if !authorization.Allowed {
			return service.reject(ctx, command, actor, "rejected", ErrPostingAuthorizationDenied)
		}
		result.Replayed = true
		return result, nil
	}

	original, err := service.repository.LoadJournalForReversal(ctx, command.AccountingScopeID, command.OriginalJournalID)
	if err != nil {
		return service.reject(ctx, command, actor, "rejected", err)
	}
	if err := validateReversalOriginal(command, original); err != nil {
		return service.reject(ctx, command, actor, "rejected", err)
	}
	reversalRequest := buildReversalRequest(command, original)
	validationRequest := reversalRequest
	validationRequest.ReversalOfJournalEntryID = nil
	if err := validationRequest.Validate(service.currency); err != nil {
		return service.reject(ctx, command, actor, "rejected", err)
	}
	authorization, err := service.authorizer.AuthorizeJournalReversal(ctx, actor, command, validationRequest)
	if err != nil {
		return service.reject(ctx, command, actor, "rejected", err)
	}
	if !authorization.Allowed {
		return service.reject(ctx, command, actor, "rejected", ErrPostingAuthorizationDenied)
	}
	if err := service.references.ValidatePostingReferences(ctx, actor, validationRequest); err != nil {
		return service.reject(ctx, command, actor, "rejected", err)
	}
	gateEvidence, err := service.gate.ValidatePostingGate(ctx, validationRequest)
	if err != nil {
		return service.reject(ctx, command, actor, "rejected", err)
	}
	if err := validatePostingGateEvidenceForActorAt(validationRequest, gateEvidence, actor, service.clock().UTC()); err != nil {
		return service.reject(ctx, command, actor, "rejected", err)
	}
	reversalID := command.OriginalJournalID
	reversalRequest.ReversalOfJournalEntryID = &reversalID
	reversalRequest.RequestFingerprint = fingerprint
	result, err := service.repository.CommitJournalReversal(ctx, JournalReversalCommit{
		Command: command, Original: original, Request: reversalRequest, Actor: actor,
		Authorization: authorization, GateEvidence: gateEvidence, RequestFingerprint: fingerprint, Now: service.clock().UTC(),
	})
	if err != nil {
		return service.reject(ctx, command, actor, "conflict", err)
	}
	return result, nil
}

func (service *JournalReversalService) reject(ctx context.Context, command ReverseJournalEntryCommand, actor Actor, outcome string, operationErr error) (JournalReversalResult, error) {
	issues := reversalIssues(operationErr)
	fingerprint, _ := journalReversalFingerprint(command)
	if auditErr := service.repository.RecordJournalReversalAttempt(ctx, command, actor, fingerprint, outcome, issues); auditErr != nil {
		if errors.Is(operationErr, ErrInvalidJournalReversalCommand) && errors.Is(auditErr, ErrInvalidJournalReversalCommand) {
			return JournalReversalResult{}, operationErr
		}
		return JournalReversalResult{}, auditErr
	}
	return JournalReversalResult{}, operationErr
}

func validateReversalOriginal(command ReverseJournalEntryCommand, original JournalEntrySnapshot) error {
	if original.Journal.ID == uuid.Nil || original.Journal.ID != command.OriginalJournalID {
		return ErrJournalEntryNotFound
	}
	if original.Request.AccountingScopeID != command.AccountingScopeID {
		return ErrPostingReferenceInvalid
	}
	if original.Journal.Status != PostingStatusPosted {
		return ErrJournalEntryNotPosted
	}
	if original.Journal.ReversalOfJournalEntryID != nil || original.Request.ReversalOfJournalEntryID != nil {
		return ErrJournalEntryAlreadyReversed
	}
	if original.Journal.Version != command.ExpectedJournalVersion {
		return ErrJournalReversalVersionConflict
	}
	if original.Request.SourceContext == command.SourceContext && original.Request.SourceAggregateType == command.SourceAggregateType && original.Request.SourceAggregateID == command.SourceAggregateID && original.Request.SourceVersion == command.SourceVersion {
		return ErrPostingSourceDuplicate
	}
	return nil
}

func buildReversalRequest(command ReverseJournalEntryCommand, original JournalEntrySnapshot) PostingRequest {
	authorizationID := command.PostingAuthorizationID
	postingPurpose := PostingPurposePolicyAdjustment
	var reopenRequestID *uuid.UUID
	var controlAuthorityEpoch *int64
	if command.ReopenRequestID != nil {
		postingPurpose = PostingPurposeReopenCorrection
		reopenRequestID = cloneUUIDValue(command.ReopenRequestID)
		controlAuthorityEpoch = cloneInt64Pointer(command.ControlAuthorityEpoch)
	}
	return PostingRequest{
		ContractVersion: PostingContractVersion,
		RequestID:       command.CommandID,
		SourceContext:   command.SourceContext, SourceAggregateType: command.SourceAggregateType, SourceAggregateID: command.SourceAggregateID, SourceVersion: command.SourceVersion,
		AccountingScope: original.Request.AccountingScope, AccountingScopeID: command.AccountingScopeID,
		PostingDate: command.EffectivePostingDate, FiscalPeriodID: command.FiscalPeriodID,
		PeriodStateVersion: command.PeriodStateVersion, PostingGateVersion: command.PostingGateVersion,
		PostingPurpose: postingPurpose, PostingAuthorizationID: &authorizationID, ReopenRequestID: reopenRequestID, ControlAuthorityEpoch: controlAuthorityEpoch,
		TransactionCurrency: original.Request.TransactionCurrency, ConversionEvidence: original.Request.ConversionEvidence,
		Description: "Reversal of " + original.Journal.Number + ": " + command.CorrectionReason,
		Lines:       invertPostingLines(original.Lines), IdempotencyKey: command.IdempotencyKey,
		CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
}

func invertPostingLines(lines []PostingLine) []PostingLine {
	result := make([]PostingLine, len(lines))
	for index, line := range lines {
		result[index] = line
		switch line.DebitOrCredit {
		case PostingDebit:
			result[index].DebitOrCredit = PostingCredit
		case PostingCredit:
			result[index].DebitOrCredit = PostingDebit
		}
	}
	return result
}

func journalReversalFingerprint(command ReverseJournalEntryCommand) (string, error) {
	payload := struct {
		OriginalJournalID      uuid.UUID  `json:"journalId"`
		AccountingScopeID      uuid.UUID  `json:"accountingScopeId"`
		ExpectedJournalVersion int64      `json:"expectedVersion"`
		CorrectionReason       string     `json:"reason"`
		EffectivePostingDate   time.Time  `json:"effectivePostingDate"`
		FiscalPeriodID         uuid.UUID  `json:"fiscalPeriodId"`
		PeriodStateVersion     int64      `json:"periodStateVersion"`
		PostingGateVersion     int64      `json:"postingGateVersion"`
		PostingAuthorizationID uuid.UUID  `json:"postingAuthorizationId"`
		ReopenRequestID        *uuid.UUID `json:"reopenRequestId,omitempty"`
		ControlAuthorityEpoch  *int64     `json:"controlAuthorityEpoch,omitempty"`
		SourceContext          string     `json:"sourceContext"`
		SourceAggregateType    string     `json:"sourceAggregateType"`
		SourceAggregateID      uuid.UUID  `json:"sourceAggregateId"`
		SourceVersion          int64      `json:"sourceVersion"`
	}{
		OriginalJournalID: command.OriginalJournalID, AccountingScopeID: command.AccountingScopeID, ExpectedJournalVersion: command.ExpectedJournalVersion,
		CorrectionReason: command.CorrectionReason, EffectivePostingDate: command.EffectivePostingDate, FiscalPeriodID: command.FiscalPeriodID,
		PeriodStateVersion: command.PeriodStateVersion, PostingGateVersion: command.PostingGateVersion, PostingAuthorizationID: command.PostingAuthorizationID,
		SourceContext: command.SourceContext, SourceAggregateType: command.SourceAggregateType, SourceAggregateID: command.SourceAggregateID, SourceVersion: command.SourceVersion,
		ReopenRequestID: command.ReopenRequestID, ControlAuthorityEpoch: command.ControlAuthorityEpoch,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	fingerprint, err := idempotency.ComputeFingerprint(data)
	return string(fingerprint), err
}

func reversalIssues(err error) []PostingIssue {
	if err == nil {
		return nil
	}
	var validation *PostingValidationError
	if errors.As(err, &validation) {
		return append([]PostingIssue(nil), validation.Issues...)
	}
	code, field := "reversal_failed", "journalId"
	switch {
	case errors.Is(err, ErrPostingIdempotencyConflict):
		code, field = "idempotency_conflict", "Idempotency-Key"
	case errors.Is(err, ErrJournalEntryAlreadyReversed):
		code = "already_reversed"
	case errors.Is(err, ErrJournalReversalVersionConflict):
		code, field = "version_conflict", "expectedVersion"
	case errors.Is(err, ErrJournalEntryNotPosted):
		code = "not_posted"
	case errors.Is(err, ErrJournalEntryNotFound):
		code = "not_found"
	case errors.Is(err, ErrPostingSourceDuplicate):
		code, field = "source_duplicate", "sourceAggregateId"
	case errors.Is(err, ErrPostingReferenceInvalid):
		code, field = "reference_invalid", "reference"
	case errors.Is(err, ErrPostingPeriodConflict):
		code, field = "period_conflict", "fiscalPeriodId"
	case errors.Is(err, ErrPostingGateConflict):
		code, field = "gate_conflict", "postingGateVersion"
	case errors.Is(err, ErrPostingGateClosed):
		code = "gate_closed"
	case errors.Is(err, ErrPostingAuthorizationDenied):
		code, field = "authorization_denied", "authorization"
	}
	return []PostingIssue{{Code: code, Field: field, Message: err.Error()}}
}

type MemoryJournalReversalAuthorizer struct {
	Decision PostingAuthorizationDecision
	Err      error
}

func (authorizer MemoryJournalReversalAuthorizer) AuthorizeJournalReversal(context.Context, Actor, ReverseJournalEntryCommand, PostingRequest) (PostingAuthorizationDecision, error) {
	if authorizer.Err != nil {
		return PostingAuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

func cloneJournalEntrySnapshot(value JournalEntrySnapshot) JournalEntrySnapshot {
	value.Journal = clonePostingJournal(value.Journal)
	value.Request = clonePostingRequest(value.Request)
	value.Lines = append([]PostingLine(nil), value.Lines...)
	value.Request.Lines = append([]PostingLine(nil), value.Request.Lines...)
	return value
}

func cloneJournalReversalResult(value JournalReversalResult) JournalReversalResult {
	value.Original = cloneJournalEntrySnapshot(value.Original)
	value.Reversal = cloneJournalEntrySnapshot(value.Reversal)
	return value
}

func cloneUUIDValue(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func uuidValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

func memoryJournalReversalIdempotencyKey(command ReverseJournalEntryCommand) string {
	return command.AccountingScopeID.String() + "|" + command.IdempotencyKey
}

func memoryJournalReversalSourceKey(command ReverseJournalEntryCommand) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d", command.AccountingScopeID, command.SourceContext, command.SourceAggregateType, command.SourceAggregateID, command.SourceVersion)
}

func (repository *MemoryPostingRepository) ResolveExistingJournalReversal(_ context.Context, command ReverseJournalEntryCommand, fingerprint string) (JournalReversalResult, bool, error) {
	if repository == nil {
		return JournalReversalResult{}, false, ErrInvalidPostingService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	if existing, ok := repository.reversalResults[memoryJournalReversalIdempotencyKey(command)]; ok {
		if existing.RequestFingerprint != fingerprint {
			return JournalReversalResult{}, false, ErrPostingIdempotencyConflict
		}
		return cloneJournalReversalResult(existing), true, nil
	}
	if existing, ok := repository.reversalSources[memoryJournalReversalSourceKey(command)]; ok {
		if existing.RequestFingerprint != fingerprint {
			return JournalReversalResult{}, false, ErrPostingSourceDuplicate
		}
		return cloneJournalReversalResult(existing), true, nil
	}
	if _, ok := repository.reversalsByOriginal[command.OriginalJournalID]; ok {
		return JournalReversalResult{}, false, ErrJournalEntryAlreadyReversed
	}
	return JournalReversalResult{}, false, nil
}

func (repository *MemoryPostingRepository) LoadJournalForReversal(_ context.Context, scopeID, journalID uuid.UUID) (JournalEntrySnapshot, error) {
	if repository == nil {
		return JournalEntrySnapshot{}, ErrInvalidPostingService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	for _, journal := range repository.journals {
		if journal.ID != journalID {
			continue
		}
		request, ok := repository.requests[journalID]
		if !ok {
			return JournalEntrySnapshot{}, ErrJournalEntryNotFound
		}
		if request.AccountingScopeID != scopeID {
			return JournalEntrySnapshot{}, ErrPostingReferenceInvalid
		}
		lines := append([]PostingLine(nil), request.Lines...)
		journal.Lines = append([]PostingLine(nil), lines...)
		return JournalEntrySnapshot{Journal: clonePostingJournal(journal), Request: clonePostingRequest(request), Lines: lines}, nil
	}
	return JournalEntrySnapshot{}, ErrJournalEntryNotFound
}

func (repository *MemoryPostingRepository) CommitJournalReversal(_ context.Context, commit JournalReversalCommit) (JournalReversalResult, error) {
	if repository == nil {
		return JournalReversalResult{}, ErrInvalidPostingService
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	command := commit.Command
	idempotencyKey := memoryJournalReversalIdempotencyKey(command)
	sourceKey := memoryJournalReversalSourceKey(command)
	if existing, ok := repository.reversalResults[idempotencyKey]; ok {
		if existing.RequestFingerprint != commit.RequestFingerprint {
			return JournalReversalResult{}, ErrPostingIdempotencyConflict
		}
		existing.Replayed = true
		return cloneJournalReversalResult(existing), nil
	}
	if existing, ok := repository.reversalSources[sourceKey]; ok {
		if existing.RequestFingerprint != commit.RequestFingerprint {
			return JournalReversalResult{}, ErrPostingSourceDuplicate
		}
		existing.Replayed = true
		return cloneJournalReversalResult(existing), nil
	}
	if _, ok := repository.reversalsByOriginal[command.OriginalJournalID]; ok {
		return JournalReversalResult{}, ErrJournalEntryAlreadyReversed
	}

	originalIndex := -1
	for index := range repository.journals {
		if repository.journals[index].ID == command.OriginalJournalID {
			originalIndex = index
			break
		}
	}
	if originalIndex < 0 {
		return JournalReversalResult{}, ErrJournalEntryNotFound
	}
	originalJournal := clonePostingJournal(repository.journals[originalIndex])
	if originalJournal.Status != PostingStatusPosted {
		return JournalReversalResult{}, ErrJournalEntryNotPosted
	}
	originalRequest, ok := repository.requests[command.OriginalJournalID]
	if !ok {
		return JournalReversalResult{}, ErrJournalEntryNotFound
	}

	scopeKey, err := postingScopeKey(commit.Request)
	if err != nil {
		return JournalReversalResult{}, err
	}
	reversalID := uuid.New()
	auditReference, err := repository.recordReversalAuditLocked(commit, uuid.New(), reversalID, PostingOutcomeJournalEntryReversed, nil)
	if err != nil {
		return JournalReversalResult{}, err
	}
	repository.nextNumber++
	reversalJournal := PostingJournal{
		ID: reversalID, Number: fmt.Sprintf("JE-%08d", repository.nextNumber), Version: 1, LedgerPosition: repository.nextNumber,
		Status: PostingStatusPosted, SourceReference: sourceReference(commit.Request), RequestFingerprint: commit.RequestFingerprint,
		GateEvidence: commit.GateEvidence, SubmittedByUserID: commit.Actor.UserID, PostingPurpose: commit.Request.PostingPurpose,
		PostingAuthorizationID: cloneUUIDValue(commit.Request.PostingAuthorizationID), ReversalOfJournalEntryID: cloneUUIDValue(&command.OriginalJournalID),
		CorrectionReason: command.CorrectionReason, Lines: append([]PostingLine(nil), commit.Request.Lines...),
	}
	reversalJournal.AuditReference = auditReference

	originalSnapshot := JournalEntrySnapshot{Journal: originalJournal, Request: clonePostingRequest(originalRequest), Lines: append([]PostingLine(nil), originalRequest.Lines...)}
	reversalRequest := clonePostingRequest(commit.Request)
	reversalSnapshot := JournalEntrySnapshot{Journal: reversalJournal, Request: reversalRequest, Lines: append([]PostingLine(nil), reversalRequest.Lines...)}
	result := JournalReversalResult{
		Outcome: PostingOutcomeJournalEntryReversed, Original: originalSnapshot, Reversal: reversalSnapshot,
		CorrectionReason: command.CorrectionReason, AuthorizationReference: command.PostingAuthorizationID,
		GateEvidence: commit.GateEvidence, AuditReference: auditReference, NextAction: "No action required",
		RequestFingerprint: commit.RequestFingerprint,
	}
	repository.reversalResults[idempotencyKey] = cloneJournalReversalResult(result)
	repository.reversalSources[sourceKey] = cloneJournalReversalResult(result)
	repository.reversalsByOriginal[command.OriginalJournalID] = cloneJournalReversalResult(result)
	repository.requests[reversalID] = clonePostingRequest(reversalRequest)
	repository.journals = append(repository.journals, reversalJournal)
	repository.attempts = append(repository.attempts, PostingAttempt{
		ID: uuid.New(), RequestID: command.CommandID, JournalID: reversalID, Outcome: PostingOutcomeJournalEntryReversed,
		AccountingScopeKey: scopeKey, SourceContext: command.SourceContext, SourceAggregateID: command.SourceAggregateID,
		SourceVersion: command.SourceVersion, IdempotencyKey: command.IdempotencyKey, RequestFingerprint: commit.RequestFingerprint,
		Operation: JournalReversalOperation, CreatedAt: commit.Now,
	})
	repository.events = append(repository.events, PostingEvent{ID: uuid.New(), Type: PostingOutcomeJournalEntryReversed, OccurredAt: commit.Now, Result: PostingResult{
		Outcome: PostingOutcomeJournalEntryReversed, Journal: &reversalJournal, SourceReference: reversalJournal.SourceReference,
		GateEvidence: commit.GateEvidence, AuditReference: auditReference,
	}})
	return result, nil
}

func (repository *MemoryPostingRepository) RecordJournalReversalAttempt(_ context.Context, command ReverseJournalEntryCommand, actor Actor, fingerprint, outcome string, issues []PostingIssue) error {
	if repository == nil {
		return ErrInvalidPostingService
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if fingerprint == "" {
		fingerprint, _ = journalReversalFingerprint(command)
	}
	storedOutcome := PostingOutcomePostingRejected
	if outcome == "conflict" && hasReversalIssue(issues, "idempotency_conflict") {
		storedOutcome = PostingOutcomeIdempotencyConflict
	}
	attemptID := uuid.New()
	if err := repository.recordReversalAuditOnlyLocked(command, actor, attemptID, storedOutcome, fingerprint, issues); err != nil {
		return err
	}
	repository.attempts = append(repository.attempts, PostingAttempt{
		ID: attemptID, RequestID: command.CommandID, AccountingScopeKey: command.AccountingScopeID.String(),
		SourceContext: command.SourceContext, SourceAggregateID: command.SourceAggregateID, SourceVersion: command.SourceVersion,
		IdempotencyKey: command.IdempotencyKey, RequestFingerprint: fingerprint, Outcome: storedOutcome, Operation: JournalReversalOperation,
		Issues: append([]PostingIssue(nil), issues...), CreatedAt: time.Now().UTC(),
	})
	repository.events = append(repository.events, PostingEvent{ID: uuid.New(), Type: storedOutcome, OccurredAt: time.Now().UTC(), Result: PostingResult{Outcome: storedOutcome, Issues: append([]PostingIssue(nil), issues...)}})
	return nil
}

func (repository *MemoryPostingRepository) recordReversalAuditLocked(commit JournalReversalCommit, attemptID, journalID uuid.UUID, outcome string, issues []PostingIssue) (uuid.UUID, error) {
	if repository.audit == nil {
		return uuid.Nil, ErrPostingAuditUnavailable
	}
	return repository.audit.RecordPostingMutation(context.Background(), PostingAuditRecord{
		AttemptID: attemptID, JournalID: journalID, ActorUserID: commit.Actor.UserID, ActorSubjectReference: commit.Actor.SubjectReference,
		Outcome: outcome, Operation: JournalReversalOperation, OriginalJournalID: commit.Command.OriginalJournalID, CorrectionReason: commit.Command.CorrectionReason,
		AccountingScope: commit.Original.Request.AccountingScope, AccountingScopeID: commit.Command.AccountingScopeID,
		SourceContext: commit.Command.SourceContext, SourceAggregateType: commit.Command.SourceAggregateType, SourceAggregateID: commit.Command.SourceAggregateID,
		SourceVersion: commit.Command.SourceVersion, RequestID: commit.Command.CommandID, IdempotencyKey: commit.Command.IdempotencyKey,
		RequestFingerprint: commit.RequestFingerprint, Permission: JournalReversalPermission, PolicyReference: commit.Authorization.PolicyReference,
		PolicyVersion: commit.Authorization.PolicyVersion, DecisionReference: commit.Authorization.DecisionReference, GateEvidence: commit.GateEvidence,
		CorrelationID: commit.Command.CorrelationID, CausationID: commit.Command.CausationID, Issues: issues,
	})
}

func (repository *MemoryPostingRepository) recordReversalAuditOnlyLocked(command ReverseJournalEntryCommand, actor Actor, attemptID uuid.UUID, outcome, fingerprint string, issues []PostingIssue) error {
	if repository.audit == nil {
		return ErrPostingAuditUnavailable
	}
	_, err := repository.audit.RecordPostingMutation(context.Background(), PostingAuditRecord{
		AttemptID: attemptID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference,
		Outcome: outcome, Operation: JournalReversalOperation, OriginalJournalID: command.OriginalJournalID, CorrectionReason: command.CorrectionReason,
		AccountingScopeID: command.AccountingScopeID, SourceContext: command.SourceContext, SourceAggregateType: command.SourceAggregateType,
		SourceAggregateID: command.SourceAggregateID, SourceVersion: command.SourceVersion, RequestID: command.CommandID,
		IdempotencyKey: command.IdempotencyKey, RequestFingerprint: fingerprint, Permission: JournalReversalPermission,
		CorrelationID: command.CorrelationID, CausationID: command.CausationID, Issues: issues,
	})
	return err
}

var _ JournalReversalRepository = (*MemoryPostingRepository)(nil)
var _ JournalReversalAuthorizer = MemoryJournalReversalAuthorizer{}

func hasReversalIssue(issues []PostingIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
