package gl

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/money"
)

const (
	LedgerManagementPermission         = "finance.gl.maintain.ledgers"
	AccountingBookManagementPermission = "finance.gl.maintain.accounting.books"

	LedgerActionCreate = "create"
	LedgerActionUpdate = "update"

	LedgerStatusDraft     = "draft"
	LedgerStatusActive    = "active"
	LedgerStatusSuspended = "suspended"
	LedgerStatusRetired   = "retired"
)

var (
	ErrInvalidLedger                          = errors.New("invalid ledger")
	ErrInvalidAccountingBook                  = errors.New("invalid accounting book")
	ErrInvalidLedgerCommand                   = errors.New("invalid ledger command")
	ErrInvalidAccountingBookCommand           = errors.New("invalid accounting-book command")
	ErrLedgerNotFound                         = errors.New("ledger not found")
	ErrAccountingBookNotFound                 = errors.New("accounting book not found")
	ErrLedgerVersionConflict                  = errors.New("ledger version conflict")
	ErrAccountingBookVersionConflict          = errors.New("accounting-book version conflict")
	ErrLedgerDuplicate                        = errors.New("duplicate ledger effective identity")
	ErrAccountingBookDuplicate                = errors.New("duplicate accounting-book effective identity")
	ErrLedgerAuthorizationDenied              = errors.New("ledger authorization denied")
	ErrAccountingBookAuthorizationDenied      = errors.New("accounting-book authorization denied")
	ErrLedgerAuthorizationUnavailable         = errors.New("ledger authorization unavailable")
	ErrAccountingBookAuthorizationUnavailable = errors.New("accounting-book authorization unavailable")
	ErrLedgerAuthorizationStale               = errors.New("ledger authorization policy is stale")
	ErrAccountingBookAuthorizationStale       = errors.New("accounting-book authorization policy is stale")
	ErrLedgerAuditUnavailable                 = errors.New("ledger audit unavailable")
	ErrAccountingBookAuditUnavailable         = errors.New("accounting-book audit unavailable")
	ErrLedgerReferenceInvalid                 = errors.New("ledger reference is invalid")
	ErrAccountingBookReferenceInvalid         = errors.New("accounting-book reference is invalid")
	ErrLedgerReferenceUnavailable             = errors.New("ledger reference unavailable")
	ErrAccountingBookReferenceUnavailable     = errors.New("accounting-book reference unavailable")
	ErrLedgerApprovalRequired                 = errors.New("ledger approval required")
	ErrAccountingBookApprovalRequired         = errors.New("accounting-book approval required")
	ErrLedgerApprovalInvalid                  = errors.New("ledger approval is invalid")
	ErrAccountingBookApprovalInvalid          = errors.New("accounting-book approval is invalid")
	ErrLedgerApprovalUnavailable              = errors.New("ledger approval validation unavailable")
	ErrAccountingBookApprovalUnavailable      = errors.New("accounting-book approval validation unavailable")
	ErrLedgerIdempotencyConflict              = errors.New("ledger idempotency conflict")
	ErrAccountingBookIdempotencyConflict      = errors.New("accounting-book idempotency conflict")
	ErrLedgerCommandInProgress                = errors.New("ledger command is already in progress")
	ErrAccountingBookCommandInProgress        = errors.New("accounting-book command is already in progress")
	ErrLedgerDurableCommandFailed             = errors.New("durable ledger command failed")
	ErrAccountingBookDurableCommandFailed     = errors.New("durable accounting-book command failed")
	ErrInvalidLedgerService                   = errors.New("invalid ledger service")
	ErrInvalidAccountingBookService           = errors.New("invalid accounting-book service")
	ErrInvalidReferenceValidator              = errors.New("invalid GL reference validator")
)

type Actor struct {
	UserID           uuid.UUID
	SubjectReference string
}

func (actor Actor) Validate() error {
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return ErrLedgerAuthorizationDenied
	}
	return nil
}

type ApprovalDecisionReference struct {
	ApprovalRequestID    uuid.UUID `json:"approvalRequestId"`
	DecisionID           uuid.UUID `json:"decisionId"`
	PolicyVersion        string    `json:"policyVersion"`
	DecisionVersion      int64     `json:"decisionVersion"`
	SubjectVersion       int64     `json:"subjectVersion"`
	CandidateFingerprint string    `json:"candidateFingerprint"`
	ApproverUserID       uuid.UUID `json:"approverUserId"`
}

func (reference ApprovalDecisionReference) Validate() error {
	if reference.ApprovalRequestID == uuid.Nil || reference.DecisionID == uuid.Nil || reference.ApproverUserID == uuid.Nil ||
		reference.DecisionVersion < 1 || reference.SubjectVersion < 1 || strings.TrimSpace(reference.PolicyVersion) == "" ||
		strings.TrimSpace(reference.CandidateFingerprint) == "" {
		return errors.New("approval decision reference is incomplete")
	}
	return nil
}

type AuthorizationDecision struct {
	Allowed           bool
	Outcome           string
	Permission        string
	ApprovedScopeIDs  []uuid.UUID
	PolicyReference   string
	PolicyVersion     string
	DecisionReference uuid.UUID
	ApprovalRequired  bool
	Reason            string
}

type LedgerCommand struct {
	Action             string
	LedgerID           uuid.UUID
	AccountingScopeID  uuid.UUID
	LegalEntityID      uuid.UUID
	LedgerType         string
	FunctionalCurrency string
	FiscalCalendarID   uuid.UUID
	LifecycleStatus    string
	EffectiveDateFrom  time.Time
	EffectiveDateTo    *time.Time
	Approval           *ApprovalDecisionReference
	ExpectedVersion    *aggregateversion.AggregateVersion
	IdempotencyKey     string
	CorrelationID      string
	CausationID        string
}

func (command LedgerCommand) Canonical() LedgerCommand {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	command.LedgerType = strings.TrimSpace(command.LedgerType)
	command.FunctionalCurrency = strings.ToUpper(strings.TrimSpace(command.FunctionalCurrency))
	command.LifecycleStatus = strings.ToLower(strings.TrimSpace(command.LifecycleStatus))
	command.EffectiveDateFrom = dateOnly(command.EffectiveDateFrom)
	command.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Approval = cloneApproval(command.Approval)
	return command
}

func (command LedgerCommand) Validate() error {
	if command.Action != LedgerActionCreate && command.Action != LedgerActionUpdate {
		return fmt.Errorf("%w: unsupported action", ErrInvalidLedgerCommand)
	}
	if command.AccountingScopeID == uuid.Nil || command.LegalEntityID == uuid.Nil || command.FiscalCalendarID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope, legal entity, and fiscal calendar are required", ErrInvalidLedgerCommand)
	}
	if command.LedgerType == "" || len([]rune(command.LedgerType)) > 80 {
		return fmt.Errorf("%w: ledger type is required", ErrInvalidLedgerCommand)
	}
	if err := money.ValidateCurrencyCode(command.FunctionalCurrency); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidLedgerCommand, err)
	}
	if command.EffectiveDateFrom.IsZero() || command.EffectiveDateTo != nil && command.EffectiveDateTo.Before(command.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidLedgerCommand)
	}
	if !validLifecycleStatus(command.LifecycleStatus) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidLedgerCommand)
	}
	if command.Action == LedgerActionCreate {
		if command.LedgerID != uuid.Nil || command.ExpectedVersion != nil || command.LifecycleStatus == LedgerStatusRetired {
			return fmt.Errorf("%w: create cannot include an id, expected version, or retired status", ErrInvalidLedgerCommand)
		}
	} else if command.LedgerID == uuid.Nil || command.ExpectedVersion == nil {
		return fmt.Errorf("%w: update requires an id and expected version", ErrInvalidLedgerCommand)
	} else if command.ExpectedVersion.Value() < 1 {
		return fmt.Errorf("%w: expected version is invalid", ErrInvalidLedgerCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidLedgerCommand)
	}
	if command.Approval != nil {
		if err := command.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidLedgerCommand, err)
		}
	}
	return nil
}

type AccountingBookCommand struct {
	Action               string
	AccountingBookID     uuid.UUID
	AccountingScopeID    uuid.UUID
	LedgerID             uuid.UUID
	BookType             string
	AccountingBasis      string
	PostingPolicyVersion string
	LifecycleStatus      string
	EffectiveDateFrom    time.Time
	EffectiveDateTo      *time.Time
	Approval             *ApprovalDecisionReference
	ExpectedVersion      *aggregateversion.AggregateVersion
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

func (command AccountingBookCommand) Canonical() AccountingBookCommand {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	command.BookType = strings.TrimSpace(command.BookType)
	command.AccountingBasis = strings.TrimSpace(command.AccountingBasis)
	command.PostingPolicyVersion = strings.TrimSpace(command.PostingPolicyVersion)
	command.LifecycleStatus = strings.ToLower(strings.TrimSpace(command.LifecycleStatus))
	command.EffectiveDateFrom = dateOnly(command.EffectiveDateFrom)
	command.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Approval = cloneApproval(command.Approval)
	return command
}

func (command AccountingBookCommand) Validate() error {
	if command.Action != LedgerActionCreate && command.Action != LedgerActionUpdate {
		return fmt.Errorf("%w: unsupported action", ErrInvalidAccountingBookCommand)
	}
	if command.AccountingScopeID == uuid.Nil || command.LedgerID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope and ledger are required", ErrInvalidAccountingBookCommand)
	}
	if command.BookType == "" || len([]rune(command.BookType)) > 80 || command.AccountingBasis == "" || len([]rune(command.AccountingBasis)) > 80 || command.PostingPolicyVersion == "" || len([]rune(command.PostingPolicyVersion)) > 120 {
		return fmt.Errorf("%w: book type, accounting basis, and posting policy version are required", ErrInvalidAccountingBookCommand)
	}
	if command.EffectiveDateFrom.IsZero() || command.EffectiveDateTo != nil && command.EffectiveDateTo.Before(command.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidAccountingBookCommand)
	}
	if !validLifecycleStatus(command.LifecycleStatus) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidAccountingBookCommand)
	}
	if command.Action == LedgerActionCreate {
		if command.AccountingBookID != uuid.Nil || command.ExpectedVersion != nil || command.LifecycleStatus == LedgerStatusRetired {
			return fmt.Errorf("%w: create cannot include an id, expected version, or retired status", ErrInvalidAccountingBookCommand)
		}
	} else if command.AccountingBookID == uuid.Nil || command.ExpectedVersion == nil {
		return fmt.Errorf("%w: update requires an id and expected version", ErrInvalidAccountingBookCommand)
	} else if command.ExpectedVersion.Value() < 1 {
		return fmt.Errorf("%w: expected version is invalid", ErrInvalidAccountingBookCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidAccountingBookCommand)
	}
	if command.Approval != nil {
		if err := command.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccountingBookCommand, err)
		}
	}
	return nil
}

type Ledger struct {
	ID                 uuid.UUID
	AccountingScopeID  uuid.UUID
	LegalEntityID      uuid.UUID
	LedgerType         string
	FunctionalCurrency string
	FiscalCalendarID   uuid.UUID
	LifecycleStatus    string
	EffectiveDateFrom  time.Time
	EffectiveDateTo    *time.Time
	Approval           *ApprovalDecisionReference
	Version            aggregateversion.AggregateVersion
	RevisionNumber     int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastAuditReference uuid.UUID
	Revisions          []LedgerRevision
}

type LedgerRevision struct {
	RevisionNumber int64
	Version        aggregateversion.AggregateVersion
	Snapshot       LedgerSnapshot
	CreatedAt      time.Time
}

type LedgerSnapshot struct {
	AccountingScopeID  uuid.UUID                  `json:"accountingScopeId"`
	LegalEntityID      uuid.UUID                  `json:"legalEntityId"`
	LedgerType         string                     `json:"ledgerType"`
	FunctionalCurrency string                     `json:"functionalCurrency"`
	FiscalCalendarID   uuid.UUID                  `json:"fiscalCalendarId"`
	LifecycleStatus    string                     `json:"lifecycleStatus"`
	EffectiveDateFrom  time.Time                  `json:"effectiveDateFrom"`
	EffectiveDateTo    *time.Time                 `json:"effectiveDateTo,omitempty"`
	Approval           *ApprovalDecisionReference `json:"approval,omitempty"`
}

func (ledger Ledger) Snapshot() LedgerSnapshot {
	return LedgerSnapshot{AccountingScopeID: ledger.AccountingScopeID, LegalEntityID: ledger.LegalEntityID, LedgerType: ledger.LedgerType, FunctionalCurrency: ledger.FunctionalCurrency, FiscalCalendarID: ledger.FiscalCalendarID, LifecycleStatus: ledger.LifecycleStatus, EffectiveDateFrom: ledger.EffectiveDateFrom, EffectiveDateTo: cloneDate(ledger.EffectiveDateTo), Approval: cloneApproval(ledger.Approval)}
}

func (ledger Ledger) Validate() error {
	if ledger.ID == uuid.Nil || ledger.AccountingScopeID == uuid.Nil || ledger.LegalEntityID == uuid.Nil || ledger.FiscalCalendarID == uuid.Nil || ledger.Version.Value() < 1 || ledger.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidLedger)
	}
	if ledger.LedgerType == "" || len([]rune(ledger.LedgerType)) > 80 {
		return fmt.Errorf("%w: ledger type is required", ErrInvalidLedger)
	}
	if err := money.ValidateCurrencyCode(ledger.FunctionalCurrency); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidLedger, err)
	}
	if !validLifecycleStatus(ledger.LifecycleStatus) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidLedger)
	}
	if ledger.EffectiveDateFrom.IsZero() || ledger.EffectiveDateTo != nil && ledger.EffectiveDateTo.Before(ledger.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidLedger)
	}
	if ledger.CreatedAt.IsZero() || ledger.UpdatedAt.IsZero() || ledger.UpdatedAt.Before(ledger.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidLedger)
	}
	if ledger.Approval != nil {
		if err := ledger.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidLedger, err)
		}
	}
	return nil
}

func NewLedger(id, scopeID, legalEntityID uuid.UUID, ledgerType, functionalCurrency string, fiscalCalendarID uuid.UUID, status string, from time.Time, to *time.Time, approval *ApprovalDecisionReference, now time.Time) (Ledger, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == LedgerStatusRetired {
		return Ledger{}, fmt.Errorf("%w: a new ledger cannot start retired", ErrInvalidLedger)
	}
	now = now.UTC()
	ledger := Ledger{ID: id, AccountingScopeID: scopeID, LegalEntityID: legalEntityID, LedgerType: strings.TrimSpace(ledgerType), FunctionalCurrency: strings.ToUpper(strings.TrimSpace(functionalCurrency)), FiscalCalendarID: fiscalCalendarID, LifecycleStatus: status, EffectiveDateFrom: dateOnly(from), EffectiveDateTo: cloneDate(to), Approval: cloneApproval(approval), Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now, UpdatedAt: now}
	if err := ledger.Validate(); err != nil {
		return Ledger{}, err
	}
	ledger.Revisions = []LedgerRevision{{RevisionNumber: 1, Version: ledger.Version, Snapshot: ledger.Snapshot(), CreatedAt: now}}
	return ledger, nil
}

func (ledger *Ledger) Replace(current Ledger, command LedgerCommand, now time.Time) error {
	if ledger == nil || current.ID == uuid.Nil || ledger.ID != current.ID {
		return ErrInvalidLedger
	}
	if current.LifecycleStatus == LedgerStatusRetired {
		return fmt.Errorf("%w: a retired ledger cannot be maintained", ErrInvalidLedger)
	}
	if !validLifecycleTransition(current.LifecycleStatus, command.LifecycleStatus) {
		return fmt.Errorf("%w: lifecycle transition from %s to %s is not allowed", ErrInvalidLedger, current.LifecycleStatus, command.LifecycleStatus)
	}
	next, err := current.Version.Advance()
	if err != nil {
		return err
	}
	*ledger = Ledger{ID: current.ID, AccountingScopeID: current.AccountingScopeID, LegalEntityID: command.LegalEntityID, LedgerType: strings.TrimSpace(command.LedgerType), FunctionalCurrency: strings.ToUpper(strings.TrimSpace(command.FunctionalCurrency)), FiscalCalendarID: command.FiscalCalendarID, LifecycleStatus: command.LifecycleStatus, EffectiveDateFrom: dateOnly(command.EffectiveDateFrom), EffectiveDateTo: cloneDate(command.EffectiveDateTo), Approval: cloneApproval(command.Approval), Version: next, RevisionNumber: current.RevisionNumber + 1, CreatedAt: current.CreatedAt, UpdatedAt: now.UTC(), LastAuditReference: current.LastAuditReference}
	if err := ledger.Validate(); err != nil {
		return err
	}
	ledger.Revisions = append(cloneLedgerRevisions(current.Revisions), LedgerRevision{RevisionNumber: ledger.RevisionNumber, Version: ledger.Version, Snapshot: ledger.Snapshot(), CreatedAt: ledger.UpdatedAt})
	return nil
}

type SafeLedger struct {
	ID                 uuid.UUID                         `json:"id"`
	AccountingScopeID  uuid.UUID                         `json:"accountingScopeId"`
	LegalEntityID      uuid.UUID                         `json:"legalEntityId"`
	LedgerType         string                            `json:"ledgerType"`
	FunctionalCurrency string                            `json:"functionalCurrency"`
	FiscalCalendarID   uuid.UUID                         `json:"fiscalCalendarId"`
	LifecycleStatus    string                            `json:"lifecycleStatus"`
	EffectiveDateFrom  time.Time                         `json:"effectiveDateFrom"`
	EffectiveDateTo    *time.Time                        `json:"effectiveDateTo,omitempty"`
	ApprovalStatus     string                            `json:"approvalStatus"`
	ValidationOutcome  string                            `json:"validationOutcome"`
	NextAction         string                            `json:"nextAction"`
	Version            aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber     int64                             `json:"revisionNumber"`
}

func (ledger Ledger) SafeProjection() SafeLedger {
	nextAction := "maintain"
	if ledger.LifecycleStatus == LedgerStatusRetired {
		nextAction = "view history"
	}
	approvalStatus := "not-required"
	if ledger.Approval != nil {
		approvalStatus = "approved"
	}
	return SafeLedger{ID: ledger.ID, AccountingScopeID: ledger.AccountingScopeID, LegalEntityID: ledger.LegalEntityID, LedgerType: ledger.LedgerType, FunctionalCurrency: ledger.FunctionalCurrency, FiscalCalendarID: ledger.FiscalCalendarID, LifecycleStatus: ledger.LifecycleStatus, EffectiveDateFrom: ledger.EffectiveDateFrom, EffectiveDateTo: cloneDate(ledger.EffectiveDateTo), ApprovalStatus: approvalStatus, ValidationOutcome: "valid", NextAction: nextAction, Version: ledger.Version, RevisionNumber: ledger.RevisionNumber}
}

type AccountingBook struct {
	ID                   uuid.UUID
	AccountingScopeID    uuid.UUID
	LedgerID             uuid.UUID
	BookType             string
	AccountingBasis      string
	PostingPolicyVersion string
	LifecycleStatus      string
	EffectiveDateFrom    time.Time
	EffectiveDateTo      *time.Time
	Approval             *ApprovalDecisionReference
	Version              aggregateversion.AggregateVersion
	RevisionNumber       int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
	LastAuditReference   uuid.UUID
	Revisions            []AccountingBookRevision
}

type AccountingBookRevision struct {
	RevisionNumber int64
	Version        aggregateversion.AggregateVersion
	Snapshot       AccountingBookSnapshot
	CreatedAt      time.Time
}

type AccountingBookSnapshot struct {
	AccountingScopeID    uuid.UUID                  `json:"accountingScopeId"`
	LedgerID             uuid.UUID                  `json:"ledgerId"`
	BookType             string                     `json:"bookType"`
	AccountingBasis      string                     `json:"accountingBasis"`
	PostingPolicyVersion string                     `json:"postingPolicyVersion"`
	LifecycleStatus      string                     `json:"lifecycleStatus"`
	EffectiveDateFrom    time.Time                  `json:"effectiveDateFrom"`
	EffectiveDateTo      *time.Time                 `json:"effectiveDateTo,omitempty"`
	Approval             *ApprovalDecisionReference `json:"approval,omitempty"`
}

func (book AccountingBook) Snapshot() AccountingBookSnapshot {
	return AccountingBookSnapshot{AccountingScopeID: book.AccountingScopeID, LedgerID: book.LedgerID, BookType: book.BookType, AccountingBasis: book.AccountingBasis, PostingPolicyVersion: book.PostingPolicyVersion, LifecycleStatus: book.LifecycleStatus, EffectiveDateFrom: book.EffectiveDateFrom, EffectiveDateTo: cloneDate(book.EffectiveDateTo), Approval: cloneApproval(book.Approval)}
}

func (book AccountingBook) Validate() error {
	if book.ID == uuid.Nil || book.AccountingScopeID == uuid.Nil || book.LedgerID == uuid.Nil || book.Version.Value() < 1 || book.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidAccountingBook)
	}
	if book.BookType == "" || len([]rune(book.BookType)) > 80 || book.AccountingBasis == "" || len([]rune(book.AccountingBasis)) > 80 || book.PostingPolicyVersion == "" || len([]rune(book.PostingPolicyVersion)) > 120 {
		return fmt.Errorf("%w: book type, accounting basis, and posting policy version are required", ErrInvalidAccountingBook)
	}
	if !validLifecycleStatus(book.LifecycleStatus) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidAccountingBook)
	}
	if book.EffectiveDateFrom.IsZero() || book.EffectiveDateTo != nil && book.EffectiveDateTo.Before(book.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidAccountingBook)
	}
	if book.CreatedAt.IsZero() || book.UpdatedAt.IsZero() || book.UpdatedAt.Before(book.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidAccountingBook)
	}
	if book.Approval != nil {
		if err := book.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccountingBook, err)
		}
	}
	return nil
}

func NewAccountingBook(id, scopeID, ledgerID uuid.UUID, bookType, basis, policyVersion, status string, from time.Time, to *time.Time, approval *ApprovalDecisionReference, now time.Time) (AccountingBook, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == LedgerStatusRetired {
		return AccountingBook{}, fmt.Errorf("%w: a new accounting book cannot start retired", ErrInvalidAccountingBook)
	}
	now = now.UTC()
	book := AccountingBook{ID: id, AccountingScopeID: scopeID, LedgerID: ledgerID, BookType: strings.TrimSpace(bookType), AccountingBasis: strings.TrimSpace(basis), PostingPolicyVersion: strings.TrimSpace(policyVersion), LifecycleStatus: status, EffectiveDateFrom: dateOnly(from), EffectiveDateTo: cloneDate(to), Approval: cloneApproval(approval), Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now, UpdatedAt: now}
	if err := book.Validate(); err != nil {
		return AccountingBook{}, err
	}
	book.Revisions = []AccountingBookRevision{{RevisionNumber: 1, Version: book.Version, Snapshot: book.Snapshot(), CreatedAt: now}}
	return book, nil
}

func (book *AccountingBook) Replace(current AccountingBook, command AccountingBookCommand, now time.Time) error {
	if book == nil || current.ID == uuid.Nil || book.ID != current.ID {
		return ErrInvalidAccountingBook
	}
	if current.LifecycleStatus == LedgerStatusRetired {
		return fmt.Errorf("%w: a retired accounting book cannot be maintained", ErrInvalidAccountingBook)
	}
	if !validLifecycleTransition(current.LifecycleStatus, command.LifecycleStatus) {
		return fmt.Errorf("%w: lifecycle transition from %s to %s is not allowed", ErrInvalidAccountingBook, current.LifecycleStatus, command.LifecycleStatus)
	}
	next, err := current.Version.Advance()
	if err != nil {
		return err
	}
	*book = AccountingBook{ID: current.ID, AccountingScopeID: current.AccountingScopeID, LedgerID: command.LedgerID, BookType: strings.TrimSpace(command.BookType), AccountingBasis: strings.TrimSpace(command.AccountingBasis), PostingPolicyVersion: strings.TrimSpace(command.PostingPolicyVersion), LifecycleStatus: command.LifecycleStatus, EffectiveDateFrom: dateOnly(command.EffectiveDateFrom), EffectiveDateTo: cloneDate(command.EffectiveDateTo), Approval: cloneApproval(command.Approval), Version: next, RevisionNumber: current.RevisionNumber + 1, CreatedAt: current.CreatedAt, UpdatedAt: now.UTC(), LastAuditReference: current.LastAuditReference}
	if err := book.Validate(); err != nil {
		return err
	}
	book.Revisions = append(cloneAccountingBookRevisions(current.Revisions), AccountingBookRevision{RevisionNumber: book.RevisionNumber, Version: book.Version, Snapshot: book.Snapshot(), CreatedAt: book.UpdatedAt})
	return nil
}

type SafeAccountingBook struct {
	ID                   uuid.UUID                         `json:"id"`
	AccountingScopeID    uuid.UUID                         `json:"accountingScopeId"`
	LedgerID             uuid.UUID                         `json:"ledgerId"`
	BookType             string                            `json:"bookType"`
	AccountingBasis      string                            `json:"accountingBasis"`
	PostingPolicyVersion string                            `json:"postingPolicyVersion"`
	LifecycleStatus      string                            `json:"lifecycleStatus"`
	EffectiveDateFrom    time.Time                         `json:"effectiveDateFrom"`
	EffectiveDateTo      *time.Time                        `json:"effectiveDateTo,omitempty"`
	ApprovalStatus       string                            `json:"approvalStatus"`
	ValidationOutcome    string                            `json:"validationOutcome"`
	NextAction           string                            `json:"nextAction"`
	Version              aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber       int64                             `json:"revisionNumber"`
}

func (book AccountingBook) SafeProjection() SafeAccountingBook {
	nextAction := "maintain"
	if book.LifecycleStatus == LedgerStatusRetired {
		nextAction = "view history"
	}
	approvalStatus := "not-required"
	if book.Approval != nil {
		approvalStatus = "approved"
	}
	return SafeAccountingBook{ID: book.ID, AccountingScopeID: book.AccountingScopeID, LedgerID: book.LedgerID, BookType: book.BookType, AccountingBasis: book.AccountingBasis, PostingPolicyVersion: book.PostingPolicyVersion, LifecycleStatus: book.LifecycleStatus, EffectiveDateFrom: book.EffectiveDateFrom, EffectiveDateTo: cloneDate(book.EffectiveDateTo), ApprovalStatus: approvalStatus, ValidationOutcome: "valid", NextAction: nextAction, Version: book.Version, RevisionNumber: book.RevisionNumber}
}

func validLifecycleStatus(status string) bool {
	switch status {
	case LedgerStatusDraft, LedgerStatusActive, LedgerStatusSuspended, LedgerStatusRetired:
		return true
	default:
		return false
	}
}

func validLifecycleTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case LedgerStatusDraft:
		return to == LedgerStatusActive || to == LedgerStatusRetired
	case LedgerStatusActive:
		return to == LedgerStatusSuspended || to == LedgerStatusRetired
	case LedgerStatusSuspended:
		return to == LedgerStatusActive || to == LedgerStatusRetired
	case LedgerStatusRetired:
		return false
	default:
		return false
	}
}

func dateOnly(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func cloneDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := dateOnly(*value)
	return &cloned
}

func cloneApproval(value *ApprovalDecisionReference) *ApprovalDecisionReference {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneLedgerRevisions(values []LedgerRevision) []LedgerRevision {
	result := make([]LedgerRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.EffectiveDateTo = cloneDate(values[index].Snapshot.EffectiveDateTo)
		result[index].Snapshot.Approval = cloneApproval(values[index].Snapshot.Approval)
	}
	return result
}

func cloneAccountingBookRevisions(values []AccountingBookRevision) []AccountingBookRevision {
	result := make([]AccountingBookRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.EffectiveDateTo = cloneDate(values[index].Snapshot.EffectiveDateTo)
		result[index].Snapshot.Approval = cloneApproval(values[index].Snapshot.Approval)
	}
	return result
}

func rangesOverlap(fromA time.Time, toA *time.Time, fromB time.Time, toB *time.Time) bool {
	endA := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	endB := endA
	if toA != nil {
		endA = dateOnly(*toA)
	}
	if toB != nil {
		endB = dateOnly(*toB)
	}
	return !dateOnly(fromA).After(endB) && !dateOnly(fromB).After(endA)
}
