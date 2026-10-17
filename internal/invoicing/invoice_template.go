package invoicing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/money"
)

const (
	InvoiceTemplateManagementPermission = "finance.inv.configure.invoice.templates"

	InvoiceTemplateActionCreate = "create"
	InvoiceTemplateActionUpdate = "update"

	BillingFrequencyOneTime   BillingFrequency = "one_time"
	BillingFrequencyDaily     BillingFrequency = "daily"
	BillingFrequencyWeekly    BillingFrequency = "weekly"
	BillingFrequencyMonthly   BillingFrequency = "monthly"
	BillingFrequencyQuarterly BillingFrequency = "quarterly"
	BillingFrequencyAnnual    BillingFrequency = "annual"

	ChargeRuleFixed      ChargeRuleKind = "fixed"
	ChargeRulePerUnit    ChargeRuleKind = "per_unit"
	ChargeRulePercentage ChargeRuleKind = "percentage"

	PaymentMethodBankTransfer PaymentMethodKind = "bank_transfer"
	PaymentMethodCard         PaymentMethodKind = "card"
	PaymentMethodDirectDebit  PaymentMethodKind = "direct_debit"
	PaymentMethodCash         PaymentMethodKind = "cash"
	PaymentMethodOther        PaymentMethodKind = "other"

	ApplicabilityAll        ApplicabilityMode = "all"
	ApplicabilityDimensions ApplicabilityMode = "dimensions"
)

var (
	ErrInvalidInvoiceTemplate                   = errors.New("invalid invoice template")
	ErrInvalidInvoiceTemplateCommand            = errors.New("invalid invoice-template command")
	ErrInvoiceTemplateNotFound                  = errors.New("invoice template not found")
	ErrInvoiceTemplateRevisionNotFound          = errors.New("invoice template revision not found")
	ErrInvoiceTemplateVersionConflict           = errors.New("invoice template version conflict")
	ErrInvoiceTemplateDuplicate                 = errors.New("duplicate invoice-template effective identity")
	ErrInvoiceTemplateAuthorizationDenied       = errors.New("invoice-template authorization denied")
	ErrInvoiceTemplateAuthorizationUnavailable  = errors.New("invoice-template authorization unavailable")
	ErrInvoiceTemplateAuthorizationStale        = errors.New("invoice-template authorization policy is stale")
	ErrInvoiceTemplateReferenceInvalid          = errors.New("invoice-template reference is invalid")
	ErrInvoiceTemplateReferenceUnavailable      = errors.New("invoice-template reference unavailable")
	ErrInvoiceTemplateAuditUnavailable          = errors.New("invoice-template audit unavailable")
	ErrInvoiceTemplateIdempotencyConflict       = errors.New("invoice-template idempotency conflict")
	ErrInvoiceTemplateCommandInProgress         = errors.New("invoice-template command is already in progress")
	ErrInvoiceTemplateDurableCommandFailed      = errors.New("durable invoice-template command failed")
	ErrInvalidInvoiceTemplateService            = errors.New("invalid invoice-template service")
	ErrInvalidInvoiceTemplateReferenceValidator = errors.New("invalid invoice-template reference validator")
	ErrInvalidInvoiceTemplateAuditRecorder      = errors.New("invalid invoice-template audit recorder")
)

var exactDecimalPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

// TemplateID is the stable identity of an InvoiceTemplate aggregate.
// It is an alias so adapters can use the repository's UUID primitive without
// introducing a second identity representation.
type TemplateID = uuid.UUID

type BillingFrequency string

type ChargeRuleKind string

type PaymentMethodKind string

type ApplicabilityMode string

type Actor struct {
	UserID           uuid.UUID
	SubjectReference string
}

func (actor Actor) Validate() error {
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return ErrInvoiceTemplateAuthorizationDenied
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
}

type TaxCategory struct {
	Code string `json:"code"`
}

func (category TaxCategory) Canonical() TaxCategory {
	category.Code = strings.TrimSpace(category.Code)
	return category
}

func (category TaxCategory) Validate() error {
	category = category.Canonical()
	if category.Code == "" || len([]rune(category.Code)) > 160 {
		return fmt.Errorf("%w: tax category is required", ErrInvalidInvoiceTemplate)
	}
	return nil
}

type ChargeRule struct {
	Kind     ChargeRuleKind `json:"kind"`
	Amount   string         `json:"amount,omitempty"`
	Currency string         `json:"currency,omitempty"`
	Rate     string         `json:"rate,omitempty"`
}

func (rule ChargeRule) Canonical() ChargeRule {
	rule.Kind = ChargeRuleKind(strings.ToLower(strings.TrimSpace(string(rule.Kind))))
	rule.Amount = canonicalDecimalString(rule.Amount)
	rule.Currency = strings.ToUpper(strings.TrimSpace(rule.Currency))
	rule.Rate = canonicalDecimalString(rule.Rate)
	return rule
}

func (rule ChargeRule) Validate() error {
	rule = rule.Canonical()
	switch rule.Kind {
	case ChargeRuleFixed, ChargeRulePerUnit:
		if rule.Amount == "" || rule.Currency == "" || rule.Rate != "" {
			return fmt.Errorf("%w: fixed and per-unit rules require amount and currency only", ErrInvalidInvoiceTemplate)
		}
		if err := validateNonNegativeDecimal(rule.Amount, false); err != nil {
			return fmt.Errorf("%w: amount: %v", ErrInvalidInvoiceTemplate, err)
		}
		if err := money.ValidateCurrencyCode(rule.Currency); err != nil {
			return fmt.Errorf("%w: currency: %v", ErrInvalidInvoiceTemplate, err)
		}
	case ChargeRulePercentage:
		if rule.Amount != "" || rule.Currency != "" || rule.Rate == "" {
			return fmt.Errorf("%w: percentage rules require rate and no monetary amount", ErrInvalidInvoiceTemplate)
		}
		if err := validateNonNegativeDecimal(rule.Rate, true); err != nil {
			return fmt.Errorf("%w: rate: %v", ErrInvalidInvoiceTemplate, err)
		}
	default:
		return fmt.Errorf("%w: unsupported charge rule kind", ErrInvalidInvoiceTemplate)
	}
	return nil
}

type TemplateLineItem struct {
	ID          uuid.UUID   `json:"id"`
	Code        string      `json:"code"`
	Description string      `json:"description"`
	ChargeRule  ChargeRule  `json:"chargeRule"`
	TaxCategory TaxCategory `json:"taxCategory"`
}

func (item TemplateLineItem) Canonical() TemplateLineItem {
	item.Code = strings.TrimSpace(item.Code)
	item.Description = strings.TrimSpace(item.Description)
	item.ChargeRule = item.ChargeRule.Canonical()
	item.TaxCategory = item.TaxCategory.Canonical()
	return item
}

func (item TemplateLineItem) Validate() error {
	item = item.Canonical()
	if item.ID == uuid.Nil {
		return fmt.Errorf("%w: line-item id is required", ErrInvalidInvoiceTemplate)
	}
	if item.Code == "" || len([]rune(item.Code)) > 160 {
		return fmt.Errorf("%w: line-item code is required", ErrInvalidInvoiceTemplate)
	}
	if item.Description == "" || len([]rune(item.Description)) > 500 {
		return fmt.Errorf("%w: line-item description is required", ErrInvalidInvoiceTemplate)
	}
	if err := item.ChargeRule.Validate(); err != nil {
		return err
	}
	return item.TaxCategory.Validate()
}

type PaymentMethod struct {
	Method           PaymentMethodKind `json:"method"`
	TermsDays        int               `json:"termsDays"`
	PaymentReference string            `json:"paymentReference,omitempty"`
}

func (method PaymentMethod) Canonical() PaymentMethod {
	method.Method = PaymentMethodKind(strings.ToLower(strings.TrimSpace(string(method.Method))))
	method.PaymentReference = strings.TrimSpace(method.PaymentReference)
	return method
}

func (method PaymentMethod) Validate() error {
	method = method.Canonical()
	switch method.Method {
	case PaymentMethodBankTransfer, PaymentMethodCard, PaymentMethodDirectDebit, PaymentMethodCash, PaymentMethodOther:
	default:
		return fmt.Errorf("%w: unsupported payment method", ErrInvalidInvoiceTemplate)
	}
	if method.TermsDays < 0 {
		return fmt.Errorf("%w: payment terms cannot be negative", ErrInvalidInvoiceTemplate)
	}
	if len([]rune(method.PaymentReference)) > 240 {
		return fmt.Errorf("%w: payment reference is too long", ErrInvalidInvoiceTemplate)
	}
	return nil
}

type Applicability struct {
	Mode       ApplicabilityMode `json:"mode"`
	Dimensions map[string]string `json:"dimensions,omitempty"`
}

func (applicability Applicability) Canonical() Applicability {
	applicability.Mode = ApplicabilityMode(strings.ToLower(strings.TrimSpace(string(applicability.Mode))))
	if applicability.Dimensions == nil {
		return applicability
	}
	canonical := make(map[string]string, len(applicability.Dimensions))
	for key, value := range applicability.Dimensions {
		canonical[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	applicability.Dimensions = canonical
	return applicability
}

func (applicability Applicability) Validate() error {
	if err := validateApplicabilityDimensionKeys(applicability.Dimensions); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInvoiceTemplate, err)
	}
	applicability = applicability.Canonical()
	switch applicability.Mode {
	case ApplicabilityAll:
		if len(applicability.Dimensions) != 0 {
			return fmt.Errorf("%w: all applicability cannot contain dimensions", ErrInvalidInvoiceTemplate)
		}
	case ApplicabilityDimensions:
		if len(applicability.Dimensions) == 0 {
			return fmt.Errorf("%w: dimension applicability requires dimensions", ErrInvalidInvoiceTemplate)
		}
		for rawKey, rawValue := range applicability.Dimensions {
			key, value := strings.TrimSpace(rawKey), strings.TrimSpace(rawValue)
			if key == "" || value == "" || len([]rune(key)) > 160 || len([]rune(value)) > 160 {
				return fmt.Errorf("%w: applicability dimensions must be non-empty", ErrInvalidInvoiceTemplate)
			}
		}
	default:
		return fmt.Errorf("%w: unsupported applicability mode", ErrInvalidInvoiceTemplate)
	}
	return nil
}

type InvoiceTemplateCommand struct {
	Action            string
	TemplateID        uuid.UUID
	AccountingScopeID uuid.UUID
	Code              string
	Name              string
	BillingFrequency  BillingFrequency
	LineItems         []TemplateLineItem
	PaymentMethod     PaymentMethod
	Applicability     Applicability
	EffectiveDateFrom time.Time
	EffectiveDateTo   *time.Time
	ExpectedVersion   *aggregateversion.AggregateVersion
	IdempotencyKey    string
	CorrelationID     string
	CausationID       string
}

func (command InvoiceTemplateCommand) Canonical() InvoiceTemplateCommand {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	command.Code = strings.TrimSpace(command.Code)
	command.Name = strings.TrimSpace(command.Name)
	command.BillingFrequency = BillingFrequency(strings.ToLower(strings.TrimSpace(string(command.BillingFrequency))))
	command.LineItems = cloneLineItems(command.LineItems)
	for index := range command.LineItems {
		command.LineItems[index] = command.LineItems[index].Canonical()
	}
	command.PaymentMethod = command.PaymentMethod.Canonical()
	command.Applicability = command.Applicability.Canonical()
	command.EffectiveDateFrom = dateOnly(command.EffectiveDateFrom)
	command.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command InvoiceTemplateCommand) Validate() error {
	if err := validateApplicabilityDimensionKeys(command.Applicability.Dimensions); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInvoiceTemplateCommand, err)
	}
	command = command.Canonical()
	if command.Action != InvoiceTemplateActionCreate && command.Action != InvoiceTemplateActionUpdate {
		return fmt.Errorf("%w: unsupported action", ErrInvalidInvoiceTemplateCommand)
	}
	if command.AccountingScopeID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope is required", ErrInvalidInvoiceTemplateCommand)
	}
	if command.Code == "" || len([]rune(command.Code)) > 160 || command.Name == "" || len([]rune(command.Name)) > 200 {
		return fmt.Errorf("%w: code and name are required", ErrInvalidInvoiceTemplateCommand)
	}
	if !validBillingFrequency(command.BillingFrequency) {
		return fmt.Errorf("%w: unsupported billing frequency", ErrInvalidInvoiceTemplateCommand)
	}
	if len(command.LineItems) == 0 {
		return fmt.Errorf("%w: at least one line item is required", ErrInvalidInvoiceTemplateCommand)
	}
	seenLineItems := make(map[uuid.UUID]struct{}, len(command.LineItems))
	for _, item := range command.LineItems {
		if _, exists := seenLineItems[item.ID]; exists {
			return fmt.Errorf("%w: duplicate line-item id", ErrInvalidInvoiceTemplateCommand)
		}
		seenLineItems[item.ID] = struct{}{}
		if err := item.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidInvoiceTemplateCommand, err)
		}
	}
	if err := command.PaymentMethod.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInvoiceTemplateCommand, err)
	}
	if err := command.Applicability.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInvoiceTemplateCommand, err)
	}
	if command.EffectiveDateFrom.IsZero() || command.EffectiveDateTo != nil && command.EffectiveDateTo.Before(command.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidInvoiceTemplateCommand)
	}
	if command.Action == InvoiceTemplateActionCreate {
		if command.TemplateID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create cannot include template id or expected version", ErrInvalidInvoiceTemplateCommand)
		}
	} else {
		if command.TemplateID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: update requires template id and expected version", ErrInvalidInvoiceTemplateCommand)
		}
	}
	if command.IdempotencyKey == "" || len([]rune(command.IdempotencyKey)) > 240 {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidInvoiceTemplateCommand)
	}
	return nil
}

type InvoiceTemplate struct {
	ID                 TemplateID                        `json:"id"`
	AccountingScopeID  uuid.UUID                         `json:"accountingScopeId"`
	Code               string                            `json:"code"`
	Name               string                            `json:"name"`
	BillingFrequency   BillingFrequency                  `json:"billingFrequency"`
	LineItems          []TemplateLineItem                `json:"lineItems"`
	PaymentMethod      PaymentMethod                     `json:"paymentMethod"`
	Applicability      Applicability                     `json:"applicability"`
	EffectiveDateFrom  time.Time                         `json:"effectiveFrom"`
	EffectiveDateTo    *time.Time                        `json:"effectiveTo,omitempty"`
	Version            aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber     int64                             `json:"revisionNumber"`
	CreatedAt          time.Time                         `json:"createdAt"`
	UpdatedAt          time.Time                         `json:"updatedAt"`
	LastAuditReference uuid.UUID                         `json:"lastAuditReference,omitempty"`
	Revisions          []InvoiceTemplateRevision         `json:"revisions"`
}

type InvoiceTemplateSnapshot struct {
	ID                TemplateID                        `json:"id"`
	AccountingScopeID uuid.UUID                         `json:"accountingScopeId"`
	Code              string                            `json:"code"`
	Name              string                            `json:"name"`
	BillingFrequency  BillingFrequency                  `json:"billingFrequency"`
	LineItems         []TemplateLineItem                `json:"lineItems"`
	PaymentMethod     PaymentMethod                     `json:"paymentMethod"`
	Applicability     Applicability                     `json:"applicability"`
	EffectiveDateFrom time.Time                         `json:"effectiveFrom"`
	EffectiveDateTo   *time.Time                        `json:"effectiveTo,omitempty"`
	Version           aggregateversion.AggregateVersion `json:"version"`
}

type InvoiceTemplateRevision struct {
	RevisionNumber int64                   `json:"revisionNumber"`
	CapturedAt     time.Time               `json:"capturedAt"`
	Snapshot       InvoiceTemplateSnapshot `json:"snapshot"`
	Fingerprint    string                  `json:"fingerprint"`
}

type SafeInvoiceTemplate struct {
	ID                 TemplateID                        `json:"id"`
	AccountingScopeID  uuid.UUID                         `json:"accountingScopeId"`
	Code               string                            `json:"code"`
	Name               string                            `json:"name"`
	BillingFrequency   BillingFrequency                  `json:"billingFrequency"`
	LineItems          []TemplateLineItem                `json:"lineItems"`
	PaymentMethod      PaymentMethod                     `json:"paymentMethod"`
	Applicability      Applicability                     `json:"applicability"`
	EffectiveDateFrom  time.Time                         `json:"effectiveFrom"`
	EffectiveDateTo    *time.Time                        `json:"effectiveTo,omitempty"`
	Version            aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber     int64                             `json:"revisionNumber"`
	CreatedAt          time.Time                         `json:"createdAt"`
	UpdatedAt          time.Time                         `json:"updatedAt"`
	LastAuditReference uuid.UUID                         `json:"lastAuditReference,omitempty"`
	Revisions          []SafeInvoiceTemplateRevision     `json:"revisions"`
}

type SafeInvoiceTemplateRevision struct {
	RevisionNumber int64                             `json:"revisionNumber"`
	CapturedAt     time.Time                         `json:"capturedAt"`
	Fingerprint    string                            `json:"fingerprint"`
	Version        aggregateversion.AggregateVersion `json:"version"`
}

type InvoiceTemplateAuditRecord struct {
	TemplateID            uuid.UUID
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

type InvoiceTemplateMutation struct {
	Before          InvoiceTemplate
	After           InvoiceTemplate
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           InvoiceTemplateAuditRecord
}

func NewInvoiceTemplate(id, scopeID uuid.UUID, command InvoiceTemplateCommand, now time.Time) (InvoiceTemplate, error) {
	if id == uuid.Nil {
		return InvoiceTemplate{}, ErrInvalidInvoiceTemplate
	}
	if err := command.Validate(); err != nil {
		return InvoiceTemplate{}, err
	}
	command = command.Canonical()
	if command.Action != InvoiceTemplateActionCreate {
		return InvoiceTemplate{}, ErrInvalidInvoiceTemplateCommand
	}
	if scopeID != command.AccountingScopeID {
		return InvoiceTemplate{}, fmt.Errorf("%w: accounting scope mismatch", ErrInvalidInvoiceTemplate)
	}
	version := aggregateversion.Initial()
	template := InvoiceTemplate{
		ID: id, AccountingScopeID: scopeID, Code: command.Code, Name: command.Name,
		BillingFrequency: command.BillingFrequency, LineItems: cloneLineItems(command.LineItems),
		PaymentMethod: command.PaymentMethod, Applicability: cloneApplicability(command.Applicability),
		EffectiveDateFrom: command.EffectiveDateFrom, EffectiveDateTo: cloneDate(command.EffectiveDateTo),
		Version: version, RevisionNumber: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(), Revisions: nil,
	}
	template.Revisions = []InvoiceTemplateRevision{{RevisionNumber: 1, CapturedAt: now.UTC(), Snapshot: template.Snapshot(), Fingerprint: FingerprintInvoiceTemplate(template)}}
	return template, nil
}

func (template *InvoiceTemplate) Replace(command InvoiceTemplateCommand, now time.Time) error {
	if template == nil {
		return ErrInvalidInvoiceTemplate
	}
	if err := command.Validate(); err != nil {
		return err
	}
	command = command.Canonical()
	if command.Action != InvoiceTemplateActionUpdate || command.TemplateID != template.ID || command.AccountingScopeID != template.AccountingScopeID {
		return ErrInvalidInvoiceTemplateCommand
	}
	if command.ExpectedVersion == nil || !template.Version.Matches(*command.ExpectedVersion) {
		return ErrInvoiceTemplateVersionConflict
	}
	next, err := template.Version.Advance()
	if err != nil {
		return err
	}
	template.Code, template.Name = command.Code, command.Name
	template.BillingFrequency = command.BillingFrequency
	template.LineItems = cloneLineItems(command.LineItems)
	template.PaymentMethod = command.PaymentMethod
	template.Applicability = cloneApplicability(command.Applicability)
	template.EffectiveDateFrom = command.EffectiveDateFrom
	template.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	template.Version, template.RevisionNumber = next, template.RevisionNumber+1
	template.UpdatedAt = now.UTC()
	if template.Revisions == nil {
		template.Revisions = make([]InvoiceTemplateRevision, 0, 1)
	}
	// Keep every established snapshot in the immutable revision lineage.
	template.Revisions = append(template.Revisions, InvoiceTemplateRevision{
		RevisionNumber: template.RevisionNumber,
		CapturedAt:     template.UpdatedAt,
		Snapshot:       template.Snapshot(),
		Fingerprint:    FingerprintInvoiceTemplate(*template),
	})
	return nil
}

func (template InvoiceTemplate) Validate() error {
	if template.ID == uuid.Nil || template.AccountingScopeID == uuid.Nil || template.Version.Value() < 1 || template.RevisionNumber < 1 {
		return ErrInvalidInvoiceTemplate
	}
	command := InvoiceTemplateCommand{
		Action: InvoiceTemplateActionUpdate, TemplateID: template.ID, AccountingScopeID: template.AccountingScopeID,
		Code: template.Code, Name: template.Name, BillingFrequency: template.BillingFrequency, LineItems: template.LineItems,
		PaymentMethod: template.PaymentMethod, Applicability: template.Applicability,
		EffectiveDateFrom: template.EffectiveDateFrom, EffectiveDateTo: template.EffectiveDateTo,
		ExpectedVersion: &template.Version, IdempotencyKey: "template-validation",
	}
	return command.Validate()
}

func (template InvoiceTemplate) Snapshot() InvoiceTemplateSnapshot {
	return InvoiceTemplateSnapshot{
		ID: template.ID, AccountingScopeID: template.AccountingScopeID, Code: template.Code, Name: template.Name,
		BillingFrequency: template.BillingFrequency, LineItems: cloneLineItems(template.LineItems),
		PaymentMethod: template.PaymentMethod, Applicability: cloneApplicability(template.Applicability),
		EffectiveDateFrom: template.EffectiveDateFrom, EffectiveDateTo: cloneDate(template.EffectiveDateTo), Version: template.Version,
	}
}

func (template InvoiceTemplate) SafeProjection() SafeInvoiceTemplate {
	revisions := make([]SafeInvoiceTemplateRevision, 0, len(template.Revisions))
	for _, revision := range template.Revisions {
		revisions = append(revisions, SafeInvoiceTemplateRevision{RevisionNumber: revision.RevisionNumber, CapturedAt: revision.CapturedAt, Fingerprint: revision.Fingerprint, Version: revision.Snapshot.Version})
	}
	return SafeInvoiceTemplate{
		ID: template.ID, AccountingScopeID: template.AccountingScopeID, Code: template.Code, Name: template.Name,
		BillingFrequency: template.BillingFrequency, LineItems: cloneLineItems(template.LineItems), PaymentMethod: template.PaymentMethod,
		Applicability: cloneApplicability(template.Applicability), EffectiveDateFrom: template.EffectiveDateFrom,
		EffectiveDateTo: cloneDate(template.EffectiveDateTo), Version: template.Version, RevisionNumber: template.RevisionNumber,
		CreatedAt: template.CreatedAt, UpdatedAt: template.UpdatedAt, LastAuditReference: template.LastAuditReference, Revisions: revisions,
	}
}

func FingerprintInvoiceTemplate(template InvoiceTemplate) string {
	data, _ := json.Marshal(template.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func invoiceTemplateCommandFingerprint(command InvoiceTemplateCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func applicabilityOverlaps(first, second Applicability) bool {
	first, second = first.Canonical(), second.Canonical()
	if first.Mode == ApplicabilityAll || second.Mode == ApplicabilityAll {
		return true
	}
	for key, firstValue := range first.Dimensions {
		if secondValue, exists := second.Dimensions[key]; exists && firstValue != secondValue {
			return false
		}
	}
	return true
}

func validBillingFrequency(value BillingFrequency) bool {
	switch value {
	case BillingFrequencyOneTime, BillingFrequencyDaily, BillingFrequencyWeekly, BillingFrequencyMonthly, BillingFrequencyQuarterly, BillingFrequencyAnnual:
		return true
	default:
		return false
	}
}

func validateApplicabilityDimensionKeys(dimensions map[string]string) error {
	seen := make(map[string]struct{}, len(dimensions))
	for rawKey := range dimensions {
		key := strings.TrimSpace(rawKey)
		if _, exists := seen[key]; exists {
			return errors.New("applicability dimension keys must be unique after canonicalization")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateNonNegativeDecimal(value string, percentage bool) error {
	if value == "" || !exactDecimalPattern.MatchString(value) {
		return errors.New("malformed exact decimal")
	}
	parts := strings.SplitN(value, ".", 2)
	integer := strings.TrimLeft(parts[0], "0")
	if integer == "" {
		integer = "0"
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = strings.TrimRight(parts[1], "0")
	}
	if len(integer)+len(fraction) > 38 || len(fraction) > 12 {
		return errors.New("decimal precision or scale exceeds the supported limit")
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil || parsed.IsNegative() {
		return errors.New("decimal must be non-negative")
	}
	if percentage && parsed.GreaterThan(decimal.NewFromInt(100)) {
		return errors.New("percentage rate must not exceed 100")
	}
	return nil
}

func canonicalDecimalString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !exactDecimalPattern.MatchString(value) {
		return value
	}
	parts := strings.SplitN(value, ".", 2)
	integer := strings.TrimLeft(parts[0], "0")
	if integer == "" {
		integer = "0"
	}
	if len(parts) == 1 {
		return integer
	}
	fraction := strings.TrimRight(parts[1], "0")
	if fraction == "" {
		return integer
	}
	return integer + "." + fraction
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
	result := dateOnly(*value)
	return &result
}

func cloneLineItems(items []TemplateLineItem) []TemplateLineItem {
	if items == nil {
		return nil
	}
	result := make([]TemplateLineItem, len(items))
	for index, item := range items {
		result[index] = item
		result[index].ChargeRule = item.ChargeRule.Canonical()
		result[index].TaxCategory = item.TaxCategory.Canonical()
	}
	return result
}

func cloneApplicability(value Applicability) Applicability {
	canonical := value.Canonical()
	if canonical.Dimensions == nil {
		return canonical
	}
	dimensions := make(map[string]string, len(canonical.Dimensions))
	for key, item := range canonical.Dimensions {
		dimensions[key] = item
	}
	canonical.Dimensions = dimensions
	return canonical
}

func cloneInvoiceTemplate(value InvoiceTemplate) InvoiceTemplate {
	value.LineItems = cloneLineItems(value.LineItems)
	value.Applicability = cloneApplicability(value.Applicability)
	value.EffectiveDateTo = cloneDate(value.EffectiveDateTo)
	if value.Revisions != nil {
		revisions := value.Revisions
		value.Revisions = make([]InvoiceTemplateRevision, len(revisions))
		for index, revision := range revisions {
			value.Revisions[index] = revision
			revision.Snapshot.LineItems = cloneLineItems(revision.Snapshot.LineItems)
			revision.Snapshot.Applicability = cloneApplicability(revision.Snapshot.Applicability)
			revision.Snapshot.EffectiveDateTo = cloneDate(revision.Snapshot.EffectiveDateTo)
			value.Revisions[index].Snapshot = revision.Snapshot
		}
	}
	return value
}

func rangesOverlap(fromA time.Time, toA *time.Time, fromB time.Time, toB *time.Time) bool {
	if toA != nil && toA.Before(fromB) {
		return false
	}
	if toB != nil && toB.Before(fromA) {
		return false
	}
	return true
}

func sortInvoiceTemplates(values []InvoiceTemplate) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code == values[j].Code {
			return values[i].ID.String() < values[j].ID.String()
		}
		return values[i].Code < values[j].Code
	})
}
