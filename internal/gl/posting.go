package gl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/platform/accountingscope"
	"github.com/toanle88/Tally/internal/platform/idempotency"
	"github.com/toanle88/Tally/internal/platform/money"
)

const (
	PostingPermission = "finance.gl.submit.posting.request"

	PostingContractVersion = 2

	PostingPurposeOrdinary          = "Ordinary"
	PostingPurposeClose             = "Close"
	PostingPurposeReopenCorrection  = "ReopenCorrection"
	PostingPurposeOperationalReopen = "OperationalReopen"
	PostingPurposePolicyAdjustment  = "PolicyAdjustment"

	PostingLineTransactionAndFunctional = "TransactionAndFunctional"
	PostingLineFunctionalOnlyAdjustment = "FunctionalOnlyAdjustment"
	PostingDebit                        = "debit"
	PostingCredit                       = "credit"

	PostingGateOpen              = "Open"
	PostingGateSoftClosePolicy   = "SoftClosePolicy"
	PostingGateCloseOnly         = "CloseOnly"
	PostingGateHardClosed        = "HardClosed"
	PostingGateScopedReopen      = "ScopedReopen"
	PostingGateOperationalReopen = "OperationalReopen"

	PostingStatusPendingApproval = "PendingApproval"
	PostingStatusPosted          = "Posted"

	PostingOutcomeJournalEntryPosted     = "JournalEntryPosted"
	PostingOutcomePostingRejected        = "PostingRejected"
	PostingOutcomePostingPendingApproval = "PostingPendingApproval"
	PostingOutcomeIdempotencyConflict    = "IdempotencyConflict"
)

var (
	ErrInvalidPostingRequest           = errors.New("invalid posting request")
	ErrPostingValidation               = errors.New("posting validation failed")
	ErrPostingAuthorizationDenied      = errors.New("posting authorization denied")
	ErrPostingAuthorizationUnavailable = errors.New("posting authorization unavailable")
	ErrPostingReferenceInvalid         = errors.New("posting reference is invalid")
	ErrPostingReferenceUnavailable     = errors.New("posting reference unavailable")
	ErrPostingPeriodConflict           = errors.New("posting period version conflict")
	ErrPostingGateConflict             = errors.New("posting gate version conflict")
	ErrPostingGateClosed               = errors.New("posting gate rejects this request")
	ErrPostingSourceDuplicate          = errors.New("posting source already owns an accounting effect")
	ErrPostingIdempotencyConflict      = errors.New("posting idempotency conflict")
	ErrPostingCommandInProgress        = errors.New("posting command is already in progress")
	ErrPostingApprovalUnavailable      = errors.New("posting approval policy unavailable")
	ErrPostingAuditUnavailable         = errors.New("posting audit unavailable")
	ErrPostingOutboxUnavailable        = errors.New("posting outbox unavailable")
	ErrInvalidPostingService           = errors.New("invalid posting service")
)

// PostingRequest is the version-2 GL posting contract after transport
// canonicalization. The idempotency key is supplied by the transport header;
// the fingerprint is computed by GL from the business fields.
type PostingRequest struct {
	ContractVersion int
	RequestID       uuid.UUID

	SourceContext       string
	SourceAggregateType string
	SourceAggregateID   uuid.UUID
	SourceVersion       int64

	AccountingScope    accountingscope.AccountingScope
	AccountingScopeID  uuid.UUID
	PostingDate        time.Time
	FiscalPeriodID     uuid.UUID
	PeriodStateVersion int64
	PostingGateVersion int64
	PostingPurpose     string

	AdjustmentPeriodIndicator  bool
	PostingAuthorizationID     *uuid.UUID
	CloseRunID                 *uuid.UUID
	ReopenRequestID            *uuid.UUID
	OperationalReopenRequestID *uuid.UUID
	ControlAuthorityEpoch      *int64

	TransactionCurrency string
	ConversionEvidence  *PostingConversionEvidence
	Description         string
	Lines               []PostingLine

	ReversalOfJournalEntryID *uuid.UUID
	AutomaticReversalDate    *time.Time

	IdempotencyKey     string
	RequestFingerprint string
	CorrelationID      string
	CausationID        string
}

type PostingConversionEvidence struct {
	RateSetID           uuid.UUID
	RateType            string
	ConversionDate      time.Time
	ConversionTimestamp time.Time
}

type PostingLine struct {
	AccountID            uuid.UUID
	DebitOrCredit        string
	LineCurrencyMode     string
	TransactionAmount    string
	FunctionalAmount     string
	SegmentCombinationID uuid.UUID
	LineReference        string
}

type PostingGateEvidence struct {
	FiscalPeriodID     uuid.UUID `json:"fiscalPeriodId"`
	PeriodStateVersion int64     `json:"periodStateVersion"`
	PostingGateVersion int64     `json:"postingGateVersion"`
	GateMode           string    `json:"gateMode"`
}

type PostingApproval struct {
	Required          bool
	ApprovalRequestID uuid.UUID
	PolicyReference   string
	PolicyVersion     string
}

type PostingIssue struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type PostingValidationError struct {
	Issues []PostingIssue
}

func (e *PostingValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return ErrPostingValidation.Error()
	}
	return fmt.Sprintf("%s: %s", ErrPostingValidation, e.Issues[0].Message)
}

func (e *PostingValidationError) Unwrap() error { return ErrPostingValidation }

type PostingJournal struct {
	ID                 uuid.UUID           `json:"journalId"`
	Number             string              `json:"journalNumber"`
	Version            int64               `json:"journalVersion"`
	LedgerPosition     int64               `json:"ledgerPosition"`
	Status             string              `json:"lifecycleStatus"`
	Approval           *PostingApproval    `json:"approval,omitempty"`
	SourceReference    string              `json:"sourceReference"`
	RequestFingerprint string              `json:"-"`
	GateEvidence       PostingGateEvidence `json:"gateEvidence"`
	AuditReference     uuid.UUID           `json:"auditReference"`
}

type PostingResult struct {
	Outcome           string
	Journal           *PostingJournal
	ApprovalRequestID uuid.UUID
	NextAction        string
	ValidationOutcome string
	ApprovalStatus    string
	SourceReference   string
	GateEvidence      PostingGateEvidence
	AuditReference    uuid.UUID
	Issues            []PostingIssue
	Replayed          bool
}

func (result PostingResult) AggregateID() uuid.UUID {
	if result.Journal == nil {
		return uuid.Nil
	}
	return result.Journal.ID
}

func (result PostingResult) AggregateVersion() int64 {
	if result.Journal == nil {
		return 0
	}
	return result.Journal.Version
}

func (result PostingResult) LifecycleStatus() string {
	if result.Journal == nil {
		return "Rejected"
	}
	return result.Journal.Status
}

type PostingAuthorizationDecision struct {
	Allowed           bool
	ApprovalRequired  bool
	PolicyReference   string
	PolicyVersion     string
	DecisionReference uuid.UUID
	Reason            string
}

type PostingAuthorizer interface {
	AuthorizePosting(context.Context, Actor, PostingRequest) (PostingAuthorizationDecision, error)
}

type PostingReferenceValidator interface {
	ValidatePostingReferences(context.Context, Actor, PostingRequest) error
}

type PostingGateValidator interface {
	ValidatePostingGate(context.Context, PostingRequest) (PostingGateEvidence, error)
}

type PostingApprovalValidator interface {
	EvaluatePostingApproval(context.Context, Actor, PostingRequest, PostingAuthorizationDecision) (PostingApproval, error)
}

type PostingAuditRecord struct {
	AttemptID             uuid.UUID
	JournalID             uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Outcome               string
	AccountingScope       accountingscope.AccountingScope
	AccountingScopeID     uuid.UUID
	SourceContext         string
	SourceAggregateType   string
	SourceAggregateID     uuid.UUID
	SourceVersion         int64
	RequestID             uuid.UUID
	IdempotencyKey        string
	RequestFingerprint    string
	Permission            string
	PolicyReference       string
	PolicyVersion         string
	DecisionReference     uuid.UUID
	CorrelationID         string
	CausationID           string
	Issues                []PostingIssue
}

type PostingAuditRecorder interface {
	RecordPostingMutation(context.Context, PostingAuditRecord) (uuid.UUID, error)
}

type PostingCommit struct {
	Request       PostingRequest
	Actor         Actor
	Authorization PostingAuthorizationDecision
	Approval      PostingApproval
	GateEvidence  PostingGateEvidence
	Now           time.Time
}

type PostingRepository interface {
	CommitPosting(context.Context, PostingCommit) (PostingResult, error)
	RecordPostingAttempt(context.Context, PostingRequest, Actor, string, []PostingIssue) error
}

// PostingReplayResolver lets a repository return an established result before
// current gate/reference validation runs. A retry must remain replayable even
// after the period gate has advanced or closed; CommitPosting repeats the
// lookup under its transaction to protect the race between the lookup and the
// new-admission path.
type PostingReplayResolver interface {
	ResolveExistingPosting(context.Context, PostingRequest) (PostingResult, bool, error)
}

type PostingService struct {
	repository PostingRepository
	authorizer PostingAuthorizer
	references PostingReferenceValidator
	gate       PostingGateValidator
	approval   PostingApprovalValidator
	audit      PostingAuditRecorder
	currency   money.CurrencyRegistry
	clock      func() time.Time
}

func NewPostingService(
	repository PostingRepository,
	authorizer PostingAuthorizer,
	references PostingReferenceValidator,
	gate PostingGateValidator,
	approval PostingApprovalValidator,
	audit PostingAuditRecorder,
	currency money.CurrencyRegistry,
	clock func() time.Time,
) (*PostingService, error) {
	if repository == nil || authorizer == nil || references == nil || gate == nil || approval == nil || audit == nil || clock == nil {
		return nil, ErrInvalidPostingService
	}
	if binder, ok := repository.(interface{ BindPostingAuditRecorder(PostingAuditRecorder) }); ok {
		binder.BindPostingAuditRecorder(audit)
	}
	return &PostingService{repository: repository, authorizer: authorizer, references: references, gate: gate, approval: approval, audit: audit, currency: currency, clock: clock}, nil
}

func (service *PostingService) Execute(ctx context.Context, actor Actor, request PostingRequest) (PostingResult, error) {
	if service == nil {
		return PostingResult{}, ErrInvalidPostingService
	}
	request = request.Canonical()
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return PostingResult{}, ErrPostingAuthorizationDenied
	}
	if err := request.Validate(service.currency); err != nil {
		if auditErr := service.recordRejected(ctx, request, actor, postingIssues(err)); auditErr != nil {
			return PostingResult{}, auditErr
		}
		return PostingResult{}, err
	}

	authorization, err := service.authorizer.AuthorizePosting(ctx, actor, request)
	if err != nil {
		if isPostingRejection(err) {
			if auditErr := service.recordRejected(ctx, request, actor, postingIssues(err)); auditErr != nil {
				return PostingResult{}, auditErr
			}
		}
		return PostingResult{}, err
	}
	if !authorization.Allowed {
		if auditErr := service.recordRejected(ctx, request, actor, []PostingIssue{{Code: "authorization_denied", Field: "authorization", Message: "the actor is not authorized for this posting purpose"}}); auditErr != nil {
			return PostingResult{}, auditErr
		}
		return PostingResult{}, ErrPostingAuthorizationDenied
	}
	request.RequestFingerprint, err = postingRequestFingerprint(request)
	if err != nil {
		return PostingResult{}, fmt.Errorf("%w: fingerprint: %v", ErrInvalidPostingRequest, err)
	}
	if resolver, ok := service.repository.(PostingReplayResolver); ok {
		result, found, err := resolver.ResolveExistingPosting(ctx, request)
		if err != nil {
			if auditErr := service.recordPostingConflict(ctx, request, actor, err); auditErr != nil {
				return PostingResult{}, auditErr
			}
			return PostingResult{}, err
		}
		if found {
			result.Replayed = true
			return result, nil
		}
	}
	if err := service.references.ValidatePostingReferences(ctx, actor, request); err != nil {
		if isPostingRejection(err) {
			if auditErr := service.recordRejected(ctx, request, actor, postingIssues(err)); auditErr != nil {
				return PostingResult{}, auditErr
			}
		}
		return PostingResult{}, err
	}

	gateEvidence, err := service.gate.ValidatePostingGate(ctx, request)
	if err != nil {
		if isPostingRejection(err) {
			if auditErr := service.recordRejected(ctx, request, actor, postingIssues(err)); auditErr != nil {
				return PostingResult{}, auditErr
			}
		}
		return PostingResult{}, err
	}
	if err := validatePostingGateEvidence(request, gateEvidence); err != nil {
		if auditErr := service.recordRejected(ctx, request, actor, postingIssues(err)); auditErr != nil {
			return PostingResult{}, auditErr
		}
		return PostingResult{}, err
	}

	approval, err := service.approval.EvaluatePostingApproval(ctx, actor, request, authorization)
	if err != nil {
		return PostingResult{}, err
	}
	if authorization.ApprovalRequired && !approval.Required {
		return PostingResult{}, ErrPostingApprovalUnavailable
	}
	if approval.Required && approval.ApprovalRequestID == uuid.Nil {
		return PostingResult{}, ErrPostingApprovalUnavailable
	}

	result, err := service.repository.CommitPosting(ctx, PostingCommit{Request: request, Actor: actor, Authorization: authorization, Approval: approval, GateEvidence: gateEvidence, Now: service.clock().UTC()})
	if err != nil {
		if auditErr := service.recordPostingConflict(ctx, request, actor, err); auditErr != nil {
			return PostingResult{}, auditErr
		}
		return PostingResult{}, err
	}
	return result, nil
}

func (request PostingRequest) Canonical() PostingRequest {
	request.SourceContext = strings.TrimSpace(request.SourceContext)
	request.SourceAggregateType = strings.TrimSpace(request.SourceAggregateType)
	request.PostingPurpose = strings.TrimSpace(request.PostingPurpose)
	request.TransactionCurrency = strings.ToUpper(strings.TrimSpace(request.TransactionCurrency))
	request.Description = strings.TrimSpace(request.Description)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.CausationID = strings.TrimSpace(request.CausationID)
	request.PostingDate = dateOnly(request.PostingDate)
	request.AutomaticReversalDate = cloneDate(request.AutomaticReversalDate)
	request.Lines = append([]PostingLine(nil), request.Lines...)
	for i := range request.Lines {
		request.Lines[i].DebitOrCredit = strings.ToLower(strings.TrimSpace(request.Lines[i].DebitOrCredit))
		request.Lines[i].LineCurrencyMode = strings.TrimSpace(request.Lines[i].LineCurrencyMode)
		request.Lines[i].TransactionAmount = strings.TrimSpace(request.Lines[i].TransactionAmount)
		request.Lines[i].FunctionalAmount = strings.TrimSpace(request.Lines[i].FunctionalAmount)
		request.Lines[i].LineReference = strings.TrimSpace(request.Lines[i].LineReference)
	}
	if request.ConversionEvidence != nil {
		evidence := *request.ConversionEvidence
		evidence.RateType = strings.TrimSpace(evidence.RateType)
		evidence.ConversionDate = dateOnly(evidence.ConversionDate)
		evidence.ConversionTimestamp = evidence.ConversionTimestamp.UTC()
		request.ConversionEvidence = &evidence
	}
	return request
}

func (request PostingRequest) Validate(registry money.CurrencyRegistry) error {
	var issues []PostingIssue
	issue := func(code, field, message string) {
		issues = append(issues, PostingIssue{Code: code, Field: field, Message: message})
	}
	if request.ContractVersion != PostingContractVersion {
		issue("unsupported_contract_version", "contractVersion", "contractVersion must be 2")
	}
	if _, err := request.AccountingScope.MarshalJSON(); err != nil {
		issue("invalid_scope", "accountingScope", "accountingScope must contain valid tenant, legal-entity, ledger, book, and functional-currency components")
	}
	for field, value := range map[string]uuid.UUID{
		"requestId": request.RequestID, "accountingScopeId": request.AccountingScopeID, "sourceAggregateId": request.SourceAggregateID,
		"fiscalPeriodId": request.FiscalPeriodID,
	} {
		if value == uuid.Nil {
			issue("required", field, field+" is required")
		}
	}
	if request.SourceContext == "" {
		issue("required", "sourceContext", "sourceContext is required")
	}
	if request.SourceAggregateType == "" {
		issue("required", "sourceAggregateType", "sourceAggregateType is required")
	}
	if request.SourceVersion < 1 {
		issue("invalid_version", "sourceVersion", "sourceVersion must be positive")
	}
	if request.PostingDate.IsZero() {
		issue("required", "postingDate", "postingDate is required")
	}
	if request.PeriodStateVersion < 1 {
		issue("invalid_version", "periodStateVersion", "periodStateVersion must be positive")
	}
	if request.PostingGateVersion < 1 {
		issue("invalid_version", "postingGateVersion", "postingGateVersion must be positive")
	}
	if request.IdempotencyKey == "" {
		issue("required", "Idempotency-Key", "Idempotency-Key is required")
	}
	if request.CorrelationID == "" || request.CausationID == "" {
		issue("required", "correlation", "correlation and causation references are required")
	}
	if err := validatePostingPurpose(request, issue); err != nil {
		issues = append(issues, postingIssues(err)...)
	}
	if err := money.ValidateCurrencyCode(request.TransactionCurrency); err != nil {
		issue("invalid_currency", "transactionCurrency", "transactionCurrency is invalid")
	}
	functionalCurrency := request.AccountingScope.FunctionalCurrency()
	if err := money.ValidateCurrencyCode(functionalCurrency); err != nil {
		issue("invalid_currency", "functionalCurrency", "functionalCurrency is invalid")
	}
	transaction, transactionErr := registry.Lookup(request.TransactionCurrency)
	functional, functionalErr := registry.Lookup(functionalCurrency)
	if transactionErr != nil {
		issue("unknown_currency", "transactionCurrency", "transactionCurrency is not configured")
	}
	if functionalErr != nil {
		issue("unknown_currency", "functionalCurrency", "functionalCurrency is not configured")
	}
	if request.TransactionCurrency != functionalCurrency && request.ConversionEvidence == nil {
		issue("conversion_evidence_required", "conversionEvidence", "conversion evidence is required when transaction and functional currencies differ")
	}
	if request.ConversionEvidence != nil {
		if request.ConversionEvidence.RateSetID == uuid.Nil || request.ConversionEvidence.RateType == "" || request.ConversionEvidence.ConversionDate.IsZero() || request.ConversionEvidence.ConversionTimestamp.IsZero() {
			issue("invalid_conversion_evidence", "conversionEvidence", "conversion evidence is incomplete")
		}
	}
	if len(request.Lines) < 2 {
		issue("minimum_lines", "lines", "at least two posting lines are required")
	}
	hasFunctionalOnlyLine := false
	var transactionDebit, transactionCredit, functionalDebit, functionalCredit decimal.Decimal
	for index, line := range request.Lines {
		field := fmt.Sprintf("lines[%d]", index)
		if line.AccountID == uuid.Nil {
			issue("required", field+".accountId", "accountId is required")
		}
		if line.SegmentCombinationID == uuid.Nil {
			issue("required", field+".segmentCombinationId", "segmentCombinationId is required")
		}
		if line.DebitOrCredit != PostingDebit && line.DebitOrCredit != PostingCredit {
			issue("invalid_direction", field+".debitOrCredit", "debitOrCredit must be debit or credit")
		}
		if line.LineCurrencyMode != PostingLineTransactionAndFunctional && line.LineCurrencyMode != PostingLineFunctionalOnlyAdjustment {
			issue("invalid_currency_mode", field+".lineCurrencyMode", "lineCurrencyMode is unsupported")
		}
		if transactionErr != nil || functionalErr != nil {
			continue
		}
		txMoney, txErr := money.NewMoney(transaction, line.TransactionAmount)
		functionalMoney, functionalErr := money.NewMoney(functional, line.FunctionalAmount)
		if txErr != nil {
			issue("invalid_amount", field+".transactionAmount", "transactionAmount is not valid for the transaction currency")
		}
		if functionalErr != nil {
			issue("invalid_amount", field+".functionalAmount", "functionalAmount is not valid for the functional currency")
		}
		if txErr != nil || functionalErr != nil {
			continue
		}
		txAmount := txMoney.Amount()
		functionalAmount := functionalMoney.Amount()
		if txAmount.IsNegative() || functionalAmount.IsNegative() {
			issue("negative_amount", field, "posting amounts cannot be negative")
		}
		if line.LineCurrencyMode == PostingLineFunctionalOnlyAdjustment {
			hasFunctionalOnlyLine = true
			if !txAmount.IsZero() || functionalAmount.IsZero() {
				issue("invalid_functional_only_line", field, "functional-only adjustments require zero transaction amount and non-zero functional amount")
			}
		} else if txAmount.IsZero() || functionalAmount.IsZero() {
			issue("zero_amount", field, "transaction and functional posting lines require non-zero amounts")
		}
		if line.DebitOrCredit == PostingDebit {
			transactionDebit = transactionDebit.Add(txAmount)
			functionalDebit = functionalDebit.Add(functionalAmount)
		} else if line.DebitOrCredit == PostingCredit {
			transactionCredit = transactionCredit.Add(txAmount)
			functionalCredit = functionalCredit.Add(functionalAmount)
		}
	}
	if hasFunctionalOnlyLine {
		if request.PostingAuthorizationID == nil || *request.PostingAuthorizationID == uuid.Nil {
			issue("functional_only_authorization_required", "postingAuthorizationId", "functional-only adjustments require an authorization reference")
		}
		if request.ConversionEvidence == nil {
			issue("functional_only_evidence_required", "conversionEvidence", "functional-only adjustments require immutable conversion evidence")
		}
	}
	if !transactionDebit.Equal(transactionCredit) {
		issue("transaction_unbalanced", "lines", "transaction debit and credit totals must balance")
	}
	if !functionalDebit.Equal(functionalCredit) {
		issue("functional_unbalanced", "lines", "functional debit and credit totals must balance")
	}
	if len(issues) > 0 {
		return &PostingValidationError{Issues: issues}
	}
	return nil
}

func validatePostingPurpose(request PostingRequest, issue func(string, string, string)) error {
	switch request.PostingPurpose {
	case PostingPurposeOrdinary:
	case PostingPurposeClose:
		if request.CloseRunID == nil || *request.CloseRunID == uuid.Nil {
			issue("required", "closeRunId", "closeRunId is required for close postings")
		}
	case PostingPurposeReopenCorrection:
		if request.ReopenRequestID == nil || *request.ReopenRequestID == uuid.Nil {
			issue("required", "reopenRequestId", "reopenRequestId is required for reopen corrections")
		}
	case PostingPurposeOperationalReopen:
		if request.OperationalReopenRequestID == nil || *request.OperationalReopenRequestID == uuid.Nil || request.ControlAuthorityEpoch == nil || *request.ControlAuthorityEpoch < 1 {
			issue("required", "operationalReopenRequestId", "operational reopen evidence is required")
		}
	case PostingPurposePolicyAdjustment:
		if request.PostingAuthorizationID == nil || *request.PostingAuthorizationID == uuid.Nil {
			issue("required", "postingAuthorizationId", "postingAuthorizationId is required for policy adjustments")
		}
	default:
		issue("invalid_purpose", "postingPurpose", "postingPurpose is unsupported")
	}
	if request.ReversalOfJournalEntryID != nil || request.AutomaticReversalDate != nil {
		return &PostingValidationError{Issues: []PostingIssue{{Code: "reversal_out_of_scope", Field: "reversalOfJournalEntryId", Message: "journal reversal is handled by the reversal workflow"}}}
	}
	return nil
}

func validatePostingGateEvidence(request PostingRequest, evidence PostingGateEvidence) error {
	if evidence.FiscalPeriodID != request.FiscalPeriodID {
		return fmt.Errorf("%w: fiscal period does not match", ErrPostingPeriodConflict)
	}
	if evidence.PeriodStateVersion != request.PeriodStateVersion {
		return fmt.Errorf("%w: expected %d, current %d", ErrPostingPeriodConflict, request.PeriodStateVersion, evidence.PeriodStateVersion)
	}
	if evidence.PostingGateVersion != request.PostingGateVersion {
		return fmt.Errorf("%w: expected %d, current %d", ErrPostingGateConflict, request.PostingGateVersion, evidence.PostingGateVersion)
	}
	switch request.PostingPurpose {
	case PostingPurposeOrdinary:
		if evidence.GateMode != PostingGateOpen && evidence.GateMode != PostingGateSoftClosePolicy {
			return fmt.Errorf("%w: gate mode %s rejects ordinary postings", ErrPostingGateClosed, evidence.GateMode)
		}
	case PostingPurposeClose:
		if evidence.GateMode != PostingGateCloseOnly {
			return fmt.Errorf("%w: gate mode %s does not admit close postings", ErrPostingGateClosed, evidence.GateMode)
		}
	case PostingPurposeReopenCorrection:
		if evidence.GateMode != PostingGateScopedReopen {
			return fmt.Errorf("%w: gate mode %s does not admit scoped reopen corrections", ErrPostingGateClosed, evidence.GateMode)
		}
	case PostingPurposeOperationalReopen:
		if evidence.GateMode != PostingGateOperationalReopen {
			return fmt.Errorf("%w: gate mode %s does not admit operational reopen postings", ErrPostingGateClosed, evidence.GateMode)
		}
	case PostingPurposePolicyAdjustment:
		if evidence.GateMode != PostingGateOpen && evidence.GateMode != PostingGateSoftClosePolicy {
			return fmt.Errorf("%w: gate mode %s does not admit policy adjustments", ErrPostingGateClosed, evidence.GateMode)
		}
	}
	return nil
}

func postingRequestFingerprint(request PostingRequest) (string, error) {
	type fingerprintLine struct {
		AccountID            uuid.UUID `json:"accountId"`
		DebitOrCredit        string    `json:"debitOrCredit"`
		LineCurrencyMode     string    `json:"lineCurrencyMode"`
		TransactionAmount    string    `json:"transactionAmount"`
		FunctionalAmount     string    `json:"functionalAmount"`
		SegmentCombinationID uuid.UUID `json:"segmentCombinationId"`
		LineReference        string    `json:"lineReference,omitempty"`
	}
	type fingerprintRequest struct {
		ContractVersion            int                             `json:"contractVersion"`
		RequestID                  uuid.UUID                       `json:"requestId"`
		SourceContext              string                          `json:"sourceContext"`
		SourceAggregateType        string                          `json:"sourceAggregateType"`
		SourceAggregateID          uuid.UUID                       `json:"sourceAggregateId"`
		SourceVersion              int64                           `json:"sourceVersion"`
		AccountingScope            accountingscope.AccountingScope `json:"accountingScope"`
		AccountingScopeID          uuid.UUID                       `json:"accountingScopeId"`
		PostingDate                time.Time                       `json:"postingDate"`
		FiscalPeriodID             uuid.UUID                       `json:"fiscalPeriodId"`
		PeriodStateVersion         int64                           `json:"periodStateVersion"`
		PostingGateVersion         int64                           `json:"postingGateVersion"`
		PostingPurpose             string                          `json:"postingPurpose"`
		AdjustmentPeriodIndicator  bool                            `json:"adjustmentPeriodIndicator"`
		PostingAuthorizationID     *uuid.UUID                      `json:"postingAuthorizationId,omitempty"`
		CloseRunID                 *uuid.UUID                      `json:"closeRunId,omitempty"`
		ReopenRequestID            *uuid.UUID                      `json:"reopenRequestId,omitempty"`
		OperationalReopenRequestID *uuid.UUID                      `json:"operationalReopenRequestId,omitempty"`
		ControlAuthorityEpoch      *int64                          `json:"controlAuthorityEpoch,omitempty"`
		TransactionCurrency        string                          `json:"transactionCurrency"`
		ConversionEvidence         *PostingConversionEvidence      `json:"conversionEvidence,omitempty"`
		Description                string                          `json:"description,omitempty"`
		Lines                      []fingerprintLine               `json:"lines"`
	}
	payload := fingerprintRequest{
		ContractVersion: request.ContractVersion, RequestID: request.RequestID, SourceContext: request.SourceContext,
		SourceAggregateType: request.SourceAggregateType, SourceAggregateID: request.SourceAggregateID, SourceVersion: request.SourceVersion,
		AccountingScope: request.AccountingScope, AccountingScopeID: request.AccountingScopeID, PostingDate: request.PostingDate, FiscalPeriodID: request.FiscalPeriodID,
		PeriodStateVersion: request.PeriodStateVersion, PostingGateVersion: request.PostingGateVersion, PostingPurpose: request.PostingPurpose,
		AdjustmentPeriodIndicator: request.AdjustmentPeriodIndicator, PostingAuthorizationID: request.PostingAuthorizationID,
		CloseRunID: request.CloseRunID, ReopenRequestID: request.ReopenRequestID, OperationalReopenRequestID: request.OperationalReopenRequestID,
		ControlAuthorityEpoch: request.ControlAuthorityEpoch, TransactionCurrency: request.TransactionCurrency, ConversionEvidence: request.ConversionEvidence,
		Description: request.Description,
	}
	payload.Lines = make([]fingerprintLine, len(request.Lines))
	for i, line := range request.Lines {
		payload.Lines[i] = fingerprintLine{AccountID: line.AccountID, DebitOrCredit: line.DebitOrCredit, LineCurrencyMode: line.LineCurrencyMode, TransactionAmount: line.TransactionAmount, FunctionalAmount: line.FunctionalAmount, SegmentCombinationID: line.SegmentCombinationID, LineReference: line.LineReference}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	fingerprint, err := idempotency.ComputeFingerprint(data)
	return string(fingerprint), err
}

func postingIssues(err error) []PostingIssue {
	var validation *PostingValidationError
	if errors.As(err, &validation) {
		return append([]PostingIssue(nil), validation.Issues...)
	}
	return []PostingIssue{{Code: "validation_failed", Field: "posting", Message: err.Error()}}
}

func isPostingRejection(err error) bool {
	return errors.Is(err, ErrPostingAuthorizationDenied) ||
		errors.Is(err, ErrPostingReferenceInvalid) ||
		errors.Is(err, ErrPostingPeriodConflict) ||
		errors.Is(err, ErrPostingGateConflict) ||
		errors.Is(err, ErrPostingGateClosed) ||
		errors.Is(err, ErrPostingValidation)
}

func (service *PostingService) recordRejected(ctx context.Context, request PostingRequest, actor Actor, issues []PostingIssue) error {
	return service.recordPostingAttempt(ctx, request, actor, PostingOutcomePostingRejected, issues)
}

func (service *PostingService) recordPostingConflict(ctx context.Context, request PostingRequest, actor Actor, err error) error {
	if errors.Is(err, ErrPostingIdempotencyConflict) {
		return service.recordPostingAttempt(ctx, request, actor, PostingOutcomeIdempotencyConflict, []PostingIssue{{Code: "idempotency_conflict", Field: "Idempotency-Key", Message: "the idempotency key was already used for different posting data"}})
	}
	if errors.Is(err, ErrPostingSourceDuplicate) {
		return service.recordPostingAttempt(ctx, request, actor, PostingOutcomePostingRejected, []PostingIssue{{Code: "source_duplicate", Field: "sourceAggregateId", Message: "the source aggregate already owns an accounting effect"}})
	}
	return nil
}

func (service *PostingService) recordPostingAttempt(ctx context.Context, request PostingRequest, actor Actor, outcome string, issues []PostingIssue) error {
	if service == nil || service.repository == nil || len(issues) == 0 {
		return nil
	}
	if request.RequestFingerprint == "" {
		request.RequestFingerprint, _ = postingRequestFingerprint(request)
	}
	return service.repository.RecordPostingAttempt(ctx, request, actor, outcome, issues)
}

type MemoryPostingAuthorizer struct {
	Decision PostingAuthorizationDecision
	Err      error
}

func (authorizer MemoryPostingAuthorizer) AuthorizePosting(context.Context, Actor, PostingRequest) (PostingAuthorizationDecision, error) {
	if authorizer.Err != nil {
		return PostingAuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllPostingReferenceValidator struct{}

func (AllowAllPostingReferenceValidator) ValidatePostingReferences(context.Context, Actor, PostingRequest) error {
	return nil
}

type UnavailablePostingReferenceValidator struct{}

func (UnavailablePostingReferenceValidator) ValidatePostingReferences(context.Context, Actor, PostingRequest) error {
	return ErrPostingReferenceUnavailable
}

type MemoryPostingGateValidator struct {
	Evidence PostingGateEvidence
	Err      error
}

func (validator MemoryPostingGateValidator) ValidatePostingGate(_ context.Context, request PostingRequest) (PostingGateEvidence, error) {
	if validator.Err != nil {
		return PostingGateEvidence{}, validator.Err
	}
	if validator.Evidence.FiscalPeriodID == uuid.Nil {
		return PostingGateEvidence{FiscalPeriodID: request.FiscalPeriodID, PeriodStateVersion: request.PeriodStateVersion, PostingGateVersion: request.PostingGateVersion, GateMode: PostingGateOpen}, nil
	}
	return validator.Evidence, nil
}

type AllowAllPostingApprovalValidator struct{}

func (AllowAllPostingApprovalValidator) EvaluatePostingApproval(context.Context, Actor, PostingRequest, PostingAuthorizationDecision) (PostingApproval, error) {
	return PostingApproval{}, nil
}

type UnavailablePostingApprovalValidator struct{}

func (UnavailablePostingApprovalValidator) EvaluatePostingApproval(context.Context, Actor, PostingRequest, PostingAuthorizationDecision) (PostingApproval, error) {
	return PostingApproval{}, ErrPostingApprovalUnavailable
}
