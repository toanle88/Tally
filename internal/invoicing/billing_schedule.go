package invoicing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
	"github.com/toanle88/Tally/internal/platform/money"
)

const (
	BillingScheduleManagementPermission = "finance.inv.configure.billing.schedules"

	BillingScheduleActionCreate = "create"
	BillingScheduleActionUpdate = "update"

	ScheduleStatusConfigured ScheduleStatus = "configured"
)

var (
	ErrInvalidBillingSchedule                   = errors.New("invalid billing schedule")
	ErrInvalidBillingScheduleCommand            = errors.New("invalid billing-schedule command")
	ErrBillingScheduleNotFound                  = errors.New("billing schedule not found")
	ErrBillingScheduleRevisionNotFound          = errors.New("billing schedule revision not found")
	ErrBillingScheduleVersionConflict           = errors.New("billing schedule version conflict")
	ErrBillingScheduleDuplicate                 = errors.New("duplicate billing-schedule effective identity")
	ErrBillingScheduleAuthorizationDenied       = errors.New("billing-schedule authorization denied")
	ErrBillingScheduleAuthorizationUnavailable  = errors.New("billing-schedule authorization unavailable")
	ErrBillingScheduleAuthorizationStale        = errors.New("billing-schedule authorization policy is stale")
	ErrBillingScheduleReferenceInvalid          = errors.New("billing-schedule reference is invalid")
	ErrBillingScheduleReferenceUnavailable      = errors.New("billing-schedule reference unavailable")
	ErrBillingScheduleAuditUnavailable          = errors.New("billing-schedule audit unavailable")
	ErrBillingScheduleIdempotencyConflict       = errors.New("billing-schedule idempotency conflict")
	ErrBillingScheduleCommandInProgress         = errors.New("billing-schedule command is already in progress")
	ErrBillingScheduleDurableCommandFailed      = errors.New("billing-schedule durable command previously failed")
	ErrBillingScheduleCommitAmbiguous           = errors.New("billing-schedule commit outcome is ambiguous")
	ErrInvalidBillingScheduleService            = errors.New("invalid billing-schedule service")
	ErrInvalidBillingScheduleReferenceValidator = errors.New("invalid billing-schedule reference validator")
	ErrInvalidBillingScheduleAuditRecorder      = errors.New("invalid billing-schedule audit recorder")
)

type ScheduleStatus string

// ScheduledCharge is deliberately linked to an invoice-template line and
// version. The source version is retained so later template maintenance does
// not rewrite the established billing input used by invoice generation.
type ScheduledCharge struct {
	ID                 uuid.UUID `json:"id"`
	TemplateID         uuid.UUID `json:"templateId"`
	TemplateVersion    int64     `json:"templateVersion"`
	TemplateLineItemID uuid.UUID `json:"templateLineItemId"`
	Quantity           string    `json:"quantity"`
	Amount             string    `json:"amount"`
	Currency           string    `json:"currency"`
}

func (charge ScheduledCharge) Canonical() ScheduledCharge {
	charge.Quantity = canonicalDecimalString(charge.Quantity)
	charge.Amount = canonicalDecimalString(charge.Amount)
	charge.Currency = strings.ToUpper(strings.TrimSpace(charge.Currency))
	return charge
}

func (charge ScheduledCharge) Validate() error {
	charge = charge.Canonical()
	if charge.ID == uuid.Nil || charge.TemplateID == uuid.Nil || charge.TemplateLineItemID == uuid.Nil || charge.TemplateVersion < 1 {
		return fmt.Errorf("%w: charge identity and source version are required", ErrInvalidBillingScheduleCommand)
	}
	if err := validateNonNegativeDecimal(charge.Quantity, false); err != nil {
		return fmt.Errorf("%w: quantity: %v", ErrInvalidBillingScheduleCommand, err)
	}
	quantity, err := decimal.NewFromString(charge.Quantity)
	if err != nil || !quantity.GreaterThan(decimal.Zero) {
		return fmt.Errorf("%w: quantity must be greater than zero", ErrInvalidBillingScheduleCommand)
	}
	if err := validateNonNegativeDecimal(charge.Amount, false); err != nil {
		return fmt.Errorf("%w: amount: %v", ErrInvalidBillingScheduleCommand, err)
	}
	if err := money.ValidateCurrencyCode(charge.Currency); err != nil {
		return fmt.Errorf("%w: currency: %v", ErrInvalidBillingScheduleCommand, err)
	}
	return nil
}

type BillingScheduleCommand struct {
	Action            string
	BillingScheduleID uuid.UUID
	AccountingScopeID uuid.UUID
	CustomerID        uuid.UUID
	ContractReference string
	Charges           []ScheduledCharge
	NextBillingDate   time.Time
	TimeZone          string
	Status            ScheduleStatus
	EffectiveDateFrom time.Time
	EffectiveDateTo   *time.Time
	ExpectedVersion   *aggregateversion.AggregateVersion
	IdempotencyKey    string
	CorrelationID     string
	CausationID       string
}

func (command BillingScheduleCommand) Canonical() BillingScheduleCommand {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	command.ContractReference = strings.TrimSpace(command.ContractReference)
	command.Charges = append([]ScheduledCharge(nil), command.Charges...)
	for index := range command.Charges {
		command.Charges[index] = command.Charges[index].Canonical()
	}
	command.TimeZone = strings.TrimSpace(command.TimeZone)
	command.Status = ScheduleStatus(strings.ToLower(strings.TrimSpace(string(command.Status))))
	command.NextBillingDate = dateOnly(command.NextBillingDate)
	command.EffectiveDateFrom = dateOnly(command.EffectiveDateFrom)
	command.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command BillingScheduleCommand) Validate() error {
	command = command.Canonical()
	if command.Action != BillingScheduleActionCreate && command.Action != BillingScheduleActionUpdate {
		return fmt.Errorf("%w: unsupported action", ErrInvalidBillingScheduleCommand)
	}
	if command.AccountingScopeID == uuid.Nil || command.CustomerID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope and customer are required", ErrInvalidBillingScheduleCommand)
	}
	if command.ContractReference == "" || len([]rune(command.ContractReference)) > 240 {
		return fmt.Errorf("%w: contract reference is required", ErrInvalidBillingScheduleCommand)
	}
	if len(command.Charges) == 0 {
		return fmt.Errorf("%w: at least one scheduled charge is required", ErrInvalidBillingScheduleCommand)
	}
	seen := make(map[uuid.UUID]struct{}, len(command.Charges))
	for _, charge := range command.Charges {
		if _, exists := seen[charge.ID]; exists {
			return fmt.Errorf("%w: duplicate scheduled charge id", ErrInvalidBillingScheduleCommand)
		}
		seen[charge.ID] = struct{}{}
		if err := charge.Validate(); err != nil {
			return err
		}
	}
	if command.NextBillingDate.IsZero() || command.EffectiveDateFrom.IsZero() {
		return fmt.Errorf("%w: billing and effective dates are required", ErrInvalidBillingScheduleCommand)
	}
	if command.NextBillingDate.Before(command.EffectiveDateFrom) {
		return fmt.Errorf("%w: next billing date precedes effective date", ErrInvalidBillingScheduleCommand)
	}
	if command.EffectiveDateTo != nil && command.EffectiveDateTo.Before(command.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidBillingScheduleCommand)
	}
	if command.EffectiveDateTo != nil && command.NextBillingDate.After(*command.EffectiveDateTo) {
		return fmt.Errorf("%w: next billing date is after effective date interval", ErrInvalidBillingScheduleCommand)
	}
	if command.Status != ScheduleStatusConfigured {
		return fmt.Errorf("%w: unsupported schedule status", ErrInvalidBillingScheduleCommand)
	}
	if command.TimeZone == "" {
		return fmt.Errorf("%w: IANA time zone is required", ErrInvalidBillingScheduleCommand)
	}
	if _, err := time.LoadLocation(command.TimeZone); err != nil {
		return fmt.Errorf("%w: invalid IANA time zone", ErrInvalidBillingScheduleCommand)
	}
	if command.Action == BillingScheduleActionCreate {
		if command.BillingScheduleID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create cannot include schedule id or expected version", ErrInvalidBillingScheduleCommand)
		}
	} else if command.BillingScheduleID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
		return fmt.Errorf("%w: update requires schedule id and expected version", ErrInvalidBillingScheduleCommand)
	}
	if command.IdempotencyKey == "" || len([]rune(command.IdempotencyKey)) > 240 {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidBillingScheduleCommand)
	}
	return nil
}

type BillingSchedule struct {
	ID                 uuid.UUID                         `json:"id"`
	AccountingScopeID  uuid.UUID                         `json:"accountingScopeId"`
	CustomerID         uuid.UUID                         `json:"customerId"`
	ContractReference  string                            `json:"contractReference"`
	Charges            []ScheduledCharge                 `json:"charges"`
	NextBillingDate    time.Time                         `json:"nextBillingDate"`
	TimeZone           string                            `json:"timeZone"`
	Status             ScheduleStatus                    `json:"status"`
	EffectiveDateFrom  time.Time                         `json:"effectiveFrom"`
	EffectiveDateTo    *time.Time                        `json:"effectiveTo,omitempty"`
	Version            aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber     int64                             `json:"revisionNumber"`
	CreatedAt          time.Time                         `json:"createdAt"`
	UpdatedAt          time.Time                         `json:"updatedAt"`
	LastAuditReference uuid.UUID                         `json:"lastAuditReference,omitempty"`
	Revisions          []BillingScheduleRevision         `json:"revisions"`
}

type BillingScheduleSnapshot struct {
	ID                uuid.UUID                         `json:"id"`
	AccountingScopeID uuid.UUID                         `json:"accountingScopeId"`
	CustomerID        uuid.UUID                         `json:"customerId"`
	ContractReference string                            `json:"contractReference"`
	Charges           []ScheduledCharge                 `json:"charges"`
	NextBillingDate   time.Time                         `json:"nextBillingDate"`
	TimeZone          string                            `json:"timeZone"`
	Status            ScheduleStatus                    `json:"status"`
	EffectiveDateFrom time.Time                         `json:"effectiveFrom"`
	EffectiveDateTo   *time.Time                        `json:"effectiveTo,omitempty"`
	Version           aggregateversion.AggregateVersion `json:"version"`
}

type BillingScheduleRevision struct {
	RevisionNumber int64                   `json:"revisionNumber"`
	CapturedAt     time.Time               `json:"capturedAt"`
	Snapshot       BillingScheduleSnapshot `json:"snapshot"`
	Fingerprint    string                  `json:"fingerprint"`
}

type SafeBillingSchedule struct {
	ID                 uuid.UUID                         `json:"id"`
	AccountingScopeID  uuid.UUID                         `json:"accountingScopeId"`
	CustomerID         uuid.UUID                         `json:"customerId"`
	ContractReference  string                            `json:"contractReference"`
	Charges            []ScheduledCharge                 `json:"charges"`
	NextBillingDate    time.Time                         `json:"nextBillingDate"`
	TimeZone           string                            `json:"timeZone"`
	Status             ScheduleStatus                    `json:"status"`
	EffectiveDateFrom  time.Time                         `json:"effectiveFrom"`
	EffectiveDateTo    *time.Time                        `json:"effectiveTo,omitempty"`
	Version            aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber     int64                             `json:"revisionNumber"`
	CreatedAt          time.Time                         `json:"createdAt"`
	UpdatedAt          time.Time                         `json:"updatedAt"`
	LastAuditReference uuid.UUID                         `json:"lastAuditReference,omitempty"`
	Revisions          []SafeBillingScheduleRevision     `json:"revisions"`
}

type SafeBillingScheduleRevision struct {
	RevisionNumber int64                             `json:"revisionNumber"`
	CapturedAt     time.Time                         `json:"capturedAt"`
	Fingerprint    string                            `json:"fingerprint"`
	Version        aggregateversion.AggregateVersion `json:"version"`
}

type BillingScheduleAuditRecord struct {
	BillingScheduleID     uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Action                string
	AccountingScopeID     uuid.UUID
	Permission            string
	PolicyReference       string
	PolicyVersion         string
	DecisionReference     uuid.UUID
	ExpectedVersion       *aggregateversion.AggregateVersion
	RevisionNumber        int64
	BeforeFingerprint     string
	AfterFingerprint      string
	CorrelationID         string
	CausationID           string
}

type BillingScheduleMutation struct {
	Before          BillingSchedule
	After           BillingSchedule
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           BillingScheduleAuditRecord
}

func NewBillingSchedule(id, scopeID uuid.UUID, command BillingScheduleCommand, now time.Time) (BillingSchedule, error) {
	if id == uuid.Nil {
		return BillingSchedule{}, ErrInvalidBillingSchedule
	}
	if err := command.Validate(); err != nil {
		return BillingSchedule{}, err
	}
	command = command.Canonical()
	if command.Action != BillingScheduleActionCreate || scopeID != command.AccountingScopeID {
		return BillingSchedule{}, ErrInvalidBillingScheduleCommand
	}
	schedule := BillingSchedule{
		ID: id, AccountingScopeID: scopeID, CustomerID: command.CustomerID, ContractReference: command.ContractReference,
		Charges: cloneScheduledCharges(command.Charges), NextBillingDate: command.NextBillingDate, TimeZone: command.TimeZone,
		Status: command.Status, EffectiveDateFrom: command.EffectiveDateFrom, EffectiveDateTo: cloneDate(command.EffectiveDateTo),
		Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	schedule.Revisions = []BillingScheduleRevision{{RevisionNumber: 1, CapturedAt: schedule.UpdatedAt, Snapshot: schedule.Snapshot(), Fingerprint: FingerprintBillingSchedule(schedule)}}
	return schedule, nil
}

func (schedule *BillingSchedule) Replace(command BillingScheduleCommand, now time.Time) error {
	if schedule == nil {
		return ErrInvalidBillingSchedule
	}
	if err := command.Validate(); err != nil {
		return err
	}
	command = command.Canonical()
	if command.Action != BillingScheduleActionUpdate || command.BillingScheduleID != schedule.ID || command.AccountingScopeID != schedule.AccountingScopeID {
		return ErrInvalidBillingScheduleCommand
	}
	if command.ExpectedVersion == nil || !schedule.Version.Matches(*command.ExpectedVersion) {
		return ErrBillingScheduleVersionConflict
	}
	next, err := schedule.Version.Advance()
	if err != nil {
		return err
	}
	schedule.CustomerID, schedule.ContractReference = command.CustomerID, command.ContractReference
	schedule.Charges = cloneScheduledCharges(command.Charges)
	schedule.NextBillingDate, schedule.TimeZone, schedule.Status = command.NextBillingDate, command.TimeZone, command.Status
	schedule.EffectiveDateFrom, schedule.EffectiveDateTo = command.EffectiveDateFrom, cloneDate(command.EffectiveDateTo)
	schedule.Version, schedule.RevisionNumber, schedule.UpdatedAt = next, schedule.RevisionNumber+1, now.UTC()
	schedule.Revisions = append(schedule.Revisions, BillingScheduleRevision{RevisionNumber: schedule.RevisionNumber, CapturedAt: schedule.UpdatedAt, Snapshot: schedule.Snapshot(), Fingerprint: FingerprintBillingSchedule(*schedule)})
	return nil
}

func (schedule BillingSchedule) Validate() error {
	if schedule.ID == uuid.Nil || schedule.AccountingScopeID == uuid.Nil || schedule.CustomerID == uuid.Nil || schedule.Version.Value() < 1 || schedule.RevisionNumber < 1 {
		return ErrInvalidBillingSchedule
	}
	command := BillingScheduleCommand{Action: BillingScheduleActionUpdate, BillingScheduleID: schedule.ID, AccountingScopeID: schedule.AccountingScopeID, CustomerID: schedule.CustomerID, ContractReference: schedule.ContractReference, Charges: schedule.Charges, NextBillingDate: schedule.NextBillingDate, TimeZone: schedule.TimeZone, Status: schedule.Status, EffectiveDateFrom: schedule.EffectiveDateFrom, EffectiveDateTo: schedule.EffectiveDateTo, ExpectedVersion: &schedule.Version, IdempotencyKey: "billing-schedule-validation"}
	return command.Validate()
}

func (schedule BillingSchedule) Snapshot() BillingScheduleSnapshot {
	return BillingScheduleSnapshot{
		ID: schedule.ID, AccountingScopeID: schedule.AccountingScopeID, CustomerID: schedule.CustomerID,
		ContractReference: schedule.ContractReference, Charges: cloneScheduledCharges(schedule.Charges), NextBillingDate: schedule.NextBillingDate,
		TimeZone: schedule.TimeZone, Status: schedule.Status, EffectiveDateFrom: schedule.EffectiveDateFrom,
		EffectiveDateTo: cloneDate(schedule.EffectiveDateTo), Version: schedule.Version,
	}
}

func (schedule BillingSchedule) SafeProjection() SafeBillingSchedule {
	revisions := make([]SafeBillingScheduleRevision, 0, len(schedule.Revisions))
	for _, revision := range schedule.Revisions {
		revisions = append(revisions, SafeBillingScheduleRevision{RevisionNumber: revision.RevisionNumber, CapturedAt: revision.CapturedAt, Fingerprint: revision.Fingerprint, Version: revision.Snapshot.Version})
	}
	return SafeBillingSchedule{
		ID: schedule.ID, AccountingScopeID: schedule.AccountingScopeID, CustomerID: schedule.CustomerID,
		ContractReference: schedule.ContractReference, Charges: cloneScheduledCharges(schedule.Charges), NextBillingDate: schedule.NextBillingDate,
		TimeZone: schedule.TimeZone, Status: schedule.Status, EffectiveDateFrom: schedule.EffectiveDateFrom,
		EffectiveDateTo: cloneDate(schedule.EffectiveDateTo), Version: schedule.Version, RevisionNumber: schedule.RevisionNumber,
		CreatedAt: schedule.CreatedAt, UpdatedAt: schedule.UpdatedAt, LastAuditReference: schedule.LastAuditReference, Revisions: revisions,
	}
}

func FingerprintBillingSchedule(schedule BillingSchedule) string {
	data, _ := json.Marshal(schedule.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func billingScheduleCommandFingerprint(command BillingScheduleCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func cloneScheduledCharges(charges []ScheduledCharge) []ScheduledCharge {
	if charges == nil {
		return nil
	}
	result := make([]ScheduledCharge, len(charges))
	copy(result, charges)
	return result
}

func billingScheduleRangesOverlap(first, second BillingSchedule) bool {
	return rangesOverlap(first.EffectiveDateFrom, first.EffectiveDateTo, second.EffectiveDateFrom, second.EffectiveDateTo)
}

type BillingScheduleAuthorizer interface {
	AuthorizeBillingSchedule(context.Context, Actor, BillingScheduleCommand, *BillingSchedule) (AuthorizationDecision, error)
}

type BillingScheduleReferenceValidator interface {
	ValidateBillingScheduleReferences(context.Context, Actor, BillingScheduleCommand) error
}

type BillingScheduleAuditRecorder interface {
	RecordBillingScheduleMutation(context.Context, BillingScheduleAuditRecord) error
}

type BillingScheduleRepository interface {
	GetBillingSchedule(context.Context, uuid.UUID) (BillingSchedule, error)
	ListBillingSchedules(context.Context, *uuid.UUID) ([]BillingSchedule, error)
	CommitBillingScheduleMutation(context.Context, BillingScheduleMutation) error
}

type DurableBillingScheduleRepository interface {
	CommitBillingScheduleMutationWithIdempotency(context.Context, BillingScheduleMutation, DurableBillingScheduleMutationCommit) error
}

type DurableBillingScheduleMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableBillingScheduleServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type BillingScheduleCommandResult struct {
	Schedule          SafeBillingSchedule `json:"billingSchedule"`
	DecisionReference uuid.UUID           `json:"decisionReference"`
	PolicyReference   string              `json:"policyReference"`
	ValidationOutcome string              `json:"validationOutcome"`
	Replayed          bool                `json:"replayed,omitempty"`
}

type BillingScheduleService struct {
	repository  BillingScheduleRepository
	authorizer  BillingScheduleAuthorizer
	references  BillingScheduleReferenceValidator
	audit       BillingScheduleAuditRecorder
	clock       func() time.Time
	durable     *DurableBillingScheduleServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedBillingScheduleCommand
}

type storedBillingScheduleCommand struct {
	fingerprint string
	result      BillingScheduleCommandResult
}

func NewBillingScheduleService(repository BillingScheduleRepository, authorizer BillingScheduleAuthorizer, references BillingScheduleReferenceValidator, audit BillingScheduleAuditRecorder, clock func() time.Time) (*BillingScheduleService, error) {
	return newBillingScheduleService(repository, authorizer, references, audit, clock, nil)
}

func NewBillingScheduleServiceWithDurableIdempotency(repository BillingScheduleRepository, authorizer BillingScheduleAuthorizer, references BillingScheduleReferenceValidator, audit BillingScheduleAuditRecorder, clock func() time.Time, durable DurableBillingScheduleServiceConfig) (*BillingScheduleService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidBillingScheduleService
	}
	return newBillingScheduleService(repository, authorizer, references, audit, clock, &durable)
}

func newBillingScheduleService(repository BillingScheduleRepository, authorizer BillingScheduleAuthorizer, references BillingScheduleReferenceValidator, audit BillingScheduleAuditRecorder, clock func() time.Time, durable *DurableBillingScheduleServiceConfig) (*BillingScheduleService, error) {
	if repository == nil || authorizer == nil || references == nil || audit == nil || clock == nil {
		if references == nil {
			return nil, ErrInvalidBillingScheduleReferenceValidator
		}
		if audit == nil {
			return nil, ErrInvalidBillingScheduleAuditRecorder
		}
		return nil, ErrInvalidBillingScheduleService
	}
	if binder, ok := repository.(interface {
		BindBillingScheduleAuditRecorder(BillingScheduleAuditRecorder)
	}); ok {
		binder.BindBillingScheduleAuditRecorder(audit)
	}
	return &BillingScheduleService{repository: repository, authorizer: authorizer, references: references, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedBillingScheduleCommand)}, nil
}

func (service *BillingScheduleService) Execute(ctx context.Context, actor Actor, command BillingScheduleCommand) (BillingScheduleCommandResult, error) {
	if service == nil {
		return BillingScheduleCommandResult{}, ErrInvalidBillingScheduleService
	}
	if err := actor.Validate(); err != nil {
		return BillingScheduleCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return BillingScheduleCommandResult{}, err
	}
	command = command.Canonical()
	fingerprint, err := billingScheduleCommandFingerprint(command)
	if err != nil {
		return BillingScheduleCommandResult{}, err
	}
	key := actor.UserID.String() + ":" + command.AccountingScopeID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		durableIdentity, err = newBillingScheduleDurableIdentity(actor, command.AccountingScopeID, command.IdempotencyKey)
		if err != nil {
			return BillingScheduleCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return BillingScheduleCommandResult{}, ErrBillingScheduleIdempotencyConflict
		}
		if err != nil {
			return BillingScheduleCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return BillingScheduleCommandResult{}, ErrBillingScheduleCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return BillingScheduleCommandResult{}, billingScheduleDurableFailureError(acquisition.Result().ResultBody())
			}
			result, decodeErr := decodeBillingScheduleResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return BillingScheduleCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return BillingScheduleCommandResult{}, ErrBillingScheduleIdempotencyConflict
			}
			result := cloneBillingScheduleCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *BillingSchedule
	if command.Action == BillingScheduleActionUpdate {
		loaded, getErr := service.repository.GetBillingSchedule(ctx, command.BillingScheduleID)
		if getErr != nil {
			service.finalizeFailure(ctx, acquisition, getErr)
			return BillingScheduleCommandResult{}, getErr
		}
		current = &loaded
		if current.AccountingScopeID != command.AccountingScopeID {
			service.finalizeFailure(ctx, acquisition, ErrBillingScheduleAuthorizationDenied)
			return BillingScheduleCommandResult{}, ErrBillingScheduleAuthorizationDenied
		}
	}
	decision, err := service.authorizer.AuthorizeBillingSchedule(ctx, actor, command, current)
	if err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return BillingScheduleCommandResult{}, err
	}
	if err := authorizeBillingScheduleDecision(decision, command.AccountingScopeID); err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return BillingScheduleCommandResult{}, err
	}
	if current != nil && (command.ExpectedVersion == nil || !current.Version.Matches(*command.ExpectedVersion)) {
		service.finalizeFailure(ctx, acquisition, ErrBillingScheduleVersionConflict)
		return BillingScheduleCommandResult{}, ErrBillingScheduleVersionConflict
	}
	if err := normalizeBillingScheduleReferenceError(service.references.ValidateBillingScheduleReferences(ctx, actor, command)); err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return BillingScheduleCommandResult{}, err
	}

	now := service.clock().UTC()
	var before, after BillingSchedule
	switch command.Action {
	case BillingScheduleActionCreate:
		id, idErr := uuid.NewV7()
		if idErr != nil {
			service.finalizeFailure(ctx, acquisition, idErr)
			return BillingScheduleCommandResult{}, idErr
		}
		after, err = NewBillingSchedule(id, command.AccountingScopeID, command, now)
	case BillingScheduleActionUpdate:
		before = cloneBillingSchedule(*current)
		after = cloneBillingSchedule(*current)
		err = after.Replace(command, now)
	}
	if err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return BillingScheduleCommandResult{}, err
	}
	audit := BillingScheduleAuditRecord{BillingScheduleID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, AccountingScopeID: after.AccountingScopeID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, ExpectedVersion: cloneVersion(command.ExpectedVersion), RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintBillingSchedule(before), AfterFingerprint: FingerprintBillingSchedule(after), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	result := BillingScheduleCommandResult{Schedule: after.SafeProjection(), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "valid"}
	mutation := BillingScheduleMutation{Before: before, After: after, ExpectedVersion: cloneVersion(command.ExpectedVersion), Audit: audit}
	if service.durable != nil {
		committer, ok := service.repository.(DurableBillingScheduleRepository)
		if !ok {
			service.finalizeFailure(ctx, acquisition, ErrInvalidBillingScheduleService)
			return BillingScheduleCommandResult{}, ErrInvalidBillingScheduleService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeFailure(ctx, acquisition, marshalErr)
			return BillingScheduleCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeFailure(ctx, acquisition, metadataErr)
			return BillingScheduleCommandResult{}, metadataErr
		}
		err = committer.CommitBillingScheduleMutationWithIdempotency(ctx, mutation, DurableBillingScheduleMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitBillingScheduleMutation(ctx, mutation)
	}
	if err != nil {
		if !errors.Is(err, ErrBillingScheduleCommitAmbiguous) {
			service.finalizeFailure(ctx, acquisition, err)
		}
		return BillingScheduleCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedBillingScheduleCommand{fingerprint: fingerprint, result: cloneBillingScheduleCommandResult(result)}
	}
	return result, nil
}

func (service *BillingScheduleService) ListSafe(ctx context.Context, actor Actor, scopeID *uuid.UUID) ([]SafeBillingSchedule, error) {
	if service == nil {
		return nil, ErrInvalidBillingScheduleService
	}
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	command := BillingScheduleCommand{Action: "read"}
	if scopeID != nil {
		command.AccountingScopeID = *scopeID
	}
	decision, err := service.authorizer.AuthorizeBillingSchedule(ctx, actor, command, nil)
	if err != nil {
		return nil, err
	}
	if scopeID != nil {
		if err := authorizeBillingScheduleDecision(decision, *scopeID); err != nil {
			return nil, err
		}
	} else if !decision.Allowed || decision.Permission != BillingScheduleManagementPermission || decision.DecisionReference == uuid.Nil || !containsWildcardScope(decision.ApprovedScopeIDs) {
		return nil, ErrBillingScheduleAuthorizationDenied
	}
	values, err := service.repository.ListBillingSchedules(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	result := make([]SafeBillingSchedule, 0, len(values))
	for _, value := range values {
		result = append(result, value.SafeProjection())
	}
	return result, nil
}

func (service *BillingScheduleService) GetSafe(ctx context.Context, actor Actor, id uuid.UUID, scopeID *uuid.UUID) (SafeBillingSchedule, error) {
	if service == nil {
		return SafeBillingSchedule{}, ErrInvalidBillingScheduleService
	}
	if err := actor.Validate(); err != nil {
		return SafeBillingSchedule{}, err
	}
	schedule, err := service.repository.GetBillingSchedule(ctx, id)
	if err != nil {
		return SafeBillingSchedule{}, err
	}
	if scopeID != nil && *scopeID != schedule.AccountingScopeID {
		return SafeBillingSchedule{}, ErrBillingScheduleAuthorizationDenied
	}
	decision, err := service.authorizer.AuthorizeBillingSchedule(ctx, actor, BillingScheduleCommand{Action: "read", AccountingScopeID: schedule.AccountingScopeID}, &schedule)
	if err != nil {
		return SafeBillingSchedule{}, err
	}
	if err := authorizeBillingScheduleDecision(decision, schedule.AccountingScopeID); err != nil {
		return SafeBillingSchedule{}, err
	}
	return schedule.SafeProjection(), nil
}

func authorizeBillingScheduleDecision(decision AuthorizationDecision, scopeID uuid.UUID) error {
	if !decision.Allowed || decision.Permission != BillingScheduleManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrBillingScheduleAuthorizationDenied
	}
	if scopeID == uuid.Nil {
		if containsWildcardScope(decision.ApprovedScopeIDs) {
			return nil
		}
		return ErrBillingScheduleAuthorizationDenied
	}
	for _, approved := range decision.ApprovedScopeIDs {
		if approved == uuid.Nil || approved == scopeID {
			return nil
		}
	}
	return ErrBillingScheduleAuthorizationDenied
}

func normalizeBillingScheduleReferenceError(err error) error {
	if err == nil || errors.Is(err, ErrBillingScheduleReferenceInvalid) || errors.Is(err, ErrBillingScheduleReferenceUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrBillingScheduleReferenceInvalid, err)
}

func newBillingScheduleDurableIdentity(actor Actor, scopeID uuid.UUID, key string) (platformidempotency.IdempotencyIdentity, error) {
	identity, err := json.Marshal(struct {
		Module    string    `json:"module"`
		Aggregate string    `json:"aggregate"`
		ActorID   uuid.UUID `json:"actorId"`
		ScopeID   uuid.UUID `json:"scopeId"`
	}{"invoicing", "billing-schedule", actor.UserID, scopeID})
	if err != nil {
		return platformidempotency.IdempotencyIdentity{}, err
	}
	return platformidempotency.NewOpaqueIdentity(string(identity), key)
}

func decodeBillingScheduleResult(body []byte) (BillingScheduleCommandResult, error) {
	var result BillingScheduleCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return BillingScheduleCommandResult{}, ErrBillingScheduleCommandInProgress
	}
	return result, nil
}

func (service *BillingScheduleService) finalizeFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{billingScheduleFailureCode(commandErr)})
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

func billingScheduleFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidBillingScheduleCommand):
		return "INVALID_REQUEST"
	case errors.Is(err, ErrBillingScheduleAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrBillingScheduleAuthorizationStale):
		return "POLICY_STALE"
	case errors.Is(err, ErrBillingScheduleAuthorizationUnavailable):
		return "AUTHORIZATION_UNAVAILABLE"
	case errors.Is(err, ErrBillingScheduleReferenceUnavailable):
		return "REFERENCE_UNAVAILABLE"
	case errors.Is(err, ErrBillingScheduleAuditUnavailable):
		return "AUDIT_UNAVAILABLE"
	case errors.Is(err, ErrBillingScheduleReferenceInvalid):
		return "REFERENCE_INVALID"
	case errors.Is(err, ErrBillingScheduleVersionConflict):
		return "VERSION_CONFLICT"
	case errors.Is(err, ErrBillingScheduleNotFound):
		return "BILLING_SCHEDULE_NOT_FOUND"
	case errors.Is(err, ErrBillingScheduleDuplicate):
		return "DUPLICATE_BILLING_SCHEDULE"
	case errors.Is(err, ErrBillingScheduleIdempotencyConflict):
		return "IDEMPOTENCY_CONFLICT"
	default:
		return "VALIDATION_FAILED"
	}
}

func billingScheduleDurableFailureError(body []byte) error {
	var failure struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &failure); err != nil {
		return ErrBillingScheduleDurableCommandFailed
	}
	switch failure.Code {
	case "INVALID_REQUEST":
		return ErrInvalidBillingScheduleCommand
	case "AUTHORIZATION_DENIED":
		return ErrBillingScheduleAuthorizationDenied
	case "POLICY_STALE":
		return ErrBillingScheduleAuthorizationStale
	case "AUTHORIZATION_UNAVAILABLE":
		return ErrBillingScheduleAuthorizationUnavailable
	case "REFERENCE_UNAVAILABLE":
		return ErrBillingScheduleReferenceUnavailable
	case "AUDIT_UNAVAILABLE":
		return ErrBillingScheduleAuditUnavailable
	case "REFERENCE_INVALID":
		return ErrBillingScheduleReferenceInvalid
	case "VERSION_CONFLICT":
		return ErrBillingScheduleVersionConflict
	case "BILLING_SCHEDULE_NOT_FOUND":
		return ErrBillingScheduleNotFound
	case "DUPLICATE_BILLING_SCHEDULE":
		return ErrBillingScheduleDuplicate
	case "IDEMPOTENCY_CONFLICT":
		return ErrBillingScheduleIdempotencyConflict
	case "VALIDATION_FAILED":
		return ErrInvalidBillingSchedule
	default:
		return ErrBillingScheduleDurableCommandFailed
	}
}

func cloneBillingSchedule(value BillingSchedule) BillingSchedule {
	value.Charges = cloneScheduledCharges(value.Charges)
	value.EffectiveDateTo = cloneDate(value.EffectiveDateTo)
	if value.Revisions != nil {
		revisions := value.Revisions
		value.Revisions = make([]BillingScheduleRevision, len(revisions))
		for index, revision := range revisions {
			revision.Snapshot.Charges = cloneScheduledCharges(revision.Snapshot.Charges)
			revision.Snapshot.EffectiveDateTo = cloneDate(revision.Snapshot.EffectiveDateTo)
			value.Revisions[index] = revision
		}
	}
	return value
}

func cloneBillingScheduleCommandResult(result BillingScheduleCommandResult) BillingScheduleCommandResult {
	result.Schedule.Charges = cloneScheduledCharges(result.Schedule.Charges)
	if result.Schedule.Revisions != nil {
		result.Schedule.Revisions = append([]SafeBillingScheduleRevision(nil), result.Schedule.Revisions...)
	}
	return result
}

type MemoryBillingScheduleRepository struct {
	mu        sync.RWMutex
	schedules map[uuid.UUID]BillingSchedule
	audit     BillingScheduleAuditRecorder
}

func NewMemoryBillingScheduleRepository() *MemoryBillingScheduleRepository {
	return &MemoryBillingScheduleRepository{schedules: make(map[uuid.UUID]BillingSchedule)}
}

func (repository *MemoryBillingScheduleRepository) BindBillingScheduleAuditRecorder(audit BillingScheduleAuditRecorder) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryBillingScheduleRepository) GetBillingSchedule(_ context.Context, id uuid.UUID) (BillingSchedule, error) {
	if repository == nil {
		return BillingSchedule{}, ErrInvalidBillingScheduleService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	schedule, ok := repository.schedules[id]
	if !ok {
		return BillingSchedule{}, ErrBillingScheduleNotFound
	}
	return cloneBillingSchedule(schedule), nil
}

func (repository *MemoryBillingScheduleRepository) ListBillingSchedules(_ context.Context, scopeID *uuid.UUID) ([]BillingSchedule, error) {
	if repository == nil {
		return nil, ErrInvalidBillingScheduleService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]BillingSchedule, 0, len(repository.schedules))
	for _, schedule := range repository.schedules {
		if scopeID != nil && schedule.AccountingScopeID != *scopeID {
			continue
		}
		result = append(result, cloneBillingSchedule(schedule))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].NextBillingDate.Equal(result[j].NextBillingDate) {
			return result[i].ID.String() < result[j].ID.String()
		}
		return result[i].NextBillingDate.Before(result[j].NextBillingDate)
	})
	return result, nil
}

func (repository *MemoryBillingScheduleRepository) CommitBillingScheduleMutation(ctx context.Context, mutation BillingScheduleMutation) error {
	if repository == nil {
		return ErrInvalidBillingScheduleService
	}
	if err := validateBillingScheduleMutation(mutation); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.schedules[mutation.After.ID]; exists {
			return ErrBillingScheduleDuplicate
		}
	} else {
		current, ok := repository.schedules[mutation.After.ID]
		if !ok {
			return ErrBillingScheduleNotFound
		}
		if current.AccountingScopeID != mutation.After.AccountingScopeID || mutation.ExpectedVersion == nil || !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrBillingScheduleVersionConflict
		}
	}
	for _, existing := range repository.schedules {
		if existing.ID == mutation.After.ID || existing.AccountingScopeID != mutation.After.AccountingScopeID || existing.CustomerID != mutation.After.CustomerID || existing.ContractReference != mutation.After.ContractReference {
			continue
		}
		if billingScheduleRangesOverlap(existing, mutation.After) {
			return ErrBillingScheduleDuplicate
		}
	}
	if repository.audit == nil {
		return ErrBillingScheduleAuditUnavailable
	}
	if err := repository.audit.RecordBillingScheduleMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.schedules[mutation.After.ID] = cloneBillingSchedule(mutation.After)
	return nil
}

type MemoryBillingScheduleAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryBillingScheduleAuthorizer) AuthorizeBillingSchedule(context.Context, Actor, BillingScheduleCommand, *BillingSchedule) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllBillingScheduleReferenceValidator struct{}

func (AllowAllBillingScheduleReferenceValidator) ValidateBillingScheduleReferences(context.Context, Actor, BillingScheduleCommand) error {
	return nil
}

type UnavailableBillingScheduleReferenceValidator struct{}

func (UnavailableBillingScheduleReferenceValidator) ValidateBillingScheduleReferences(context.Context, Actor, BillingScheduleCommand) error {
	return ErrBillingScheduleReferenceUnavailable
}

type MemoryBillingScheduleAuditRecorder struct {
	mu      sync.Mutex
	Records []BillingScheduleAuditRecord
	Err     error
}

func (recorder *MemoryBillingScheduleAuditRecorder) RecordBillingScheduleMutation(_ context.Context, record BillingScheduleAuditRecord) error {
	if recorder == nil {
		return ErrBillingScheduleAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func validateBillingScheduleMutation(mutation BillingScheduleMutation) error {
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.After.Version.Value() != 1 || mutation.After.RevisionNumber != 1 || mutation.ExpectedVersion != nil {
			return ErrInvalidBillingSchedule
		}
		return nil
	}
	if mutation.Before.ID != mutation.After.ID || mutation.ExpectedVersion == nil || !mutation.Before.Version.Matches(*mutation.ExpectedVersion) {
		return ErrBillingScheduleVersionConflict
	}
	if mutation.After.Version.Value() != mutation.Before.Version.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrBillingScheduleVersionConflict
	}
	return nil
}

var _ BillingScheduleRepository = (*MemoryBillingScheduleRepository)(nil)
var _ BillingScheduleAuthorizer = MemoryBillingScheduleAuthorizer{}
var _ BillingScheduleReferenceValidator = AllowAllBillingScheduleReferenceValidator{}
var _ BillingScheduleReferenceValidator = UnavailableBillingScheduleReferenceValidator{}
var _ BillingScheduleAuditRecorder = (*MemoryBillingScheduleAuditRecorder)(nil)
