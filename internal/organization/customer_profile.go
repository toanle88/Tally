package organization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/money"
)

const CustomerProfileManagementPermission = "finance.omd.maintain.customer.profiles"

const (
	CustomerProfileActionCreate   = "create"
	CustomerProfileActionMaintain = "maintain"

	CustomerProfileStatusDraft    = "draft"
	CustomerProfileStatusActive   = "active"
	CustomerProfileStatusEndDated = "end_dated"
)

var (
	ErrInvalidCustomerProfile                       = errors.New("invalid customer profile")
	ErrInvalidCustomerProfileCommand                = errors.New("invalid customer profile command")
	ErrCustomerProfileNotFound                      = errors.New("customer profile not found")
	ErrCustomerProfileVersionConflict               = errors.New("customer profile version conflict")
	ErrCustomerProfilePartyNotFound                 = errors.New("customer profile party not found")
	ErrCustomerProfilePartyInvalid                  = errors.New("customer profile party is invalid")
	ErrCustomerProfilePartyVersionConflict          = errors.New("customer profile party version conflict")
	ErrCustomerProfilePartyMismatch                 = errors.New("customer profile party reference mismatch")
	ErrCustomerProfileAuthorizationDenied           = errors.New("customer profile authorization denied")
	ErrCustomerProfileAuthorizationUnavailable      = errors.New("customer profile authorization unavailable")
	ErrCustomerProfileAuthorizationStale            = errors.New("customer profile authorization stale")
	ErrCustomerProfileFieldAuthorizationDenied      = errors.New("customer profile field authorization denied")
	ErrCustomerProfileFieldAuthorizationUnavailable = errors.New("customer profile field authorization unavailable")
	ErrCustomerProfileIdempotencyConflict           = errors.New("customer profile idempotency conflict")
	ErrCustomerProfileCommandInProgress             = errors.New("customer profile command is already in progress")
	ErrCustomerProfileAuditUnavailable              = errors.New("customer profile audit unavailable")
	ErrCustomerProfileApprovalRequired              = errors.New("customer profile approval required")
	ErrCustomerProfileApprovalRejected              = errors.New("customer profile approval rejected")
	ErrCustomerProfileDuplicate                     = errors.New("duplicate customer profile value")
	ErrInvalidCustomerProfileService                = errors.New("invalid customer profile service")
	ErrCustomerProfileDurableCommandFailed          = errors.New("customer profile command previously failed")
)

var customerProfileCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// CreditLimit is the exact-decimal money value accepted by the OMD customer
// profile contract. Currency master-data membership remains outside this
// aggregate; the platform money primitive still enforces ISO syntax, scale,
// precision, and non-floating-point parsing.
type CreditLimit struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func (limit CreditLimit) Canonicalize() (CreditLimit, error) {
	currency := strings.TrimSpace(limit.Currency)
	if currency != strings.ToUpper(currency) {
		return CreditLimit{}, fmt.Errorf("%w: credit-limit currency must be uppercase", ErrInvalidCustomerProfile)
	}
	registry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{{Code: currency, Scale: 12}})
	if err != nil {
		return CreditLimit{}, fmt.Errorf("%w: credit-limit currency is invalid", ErrInvalidCustomerProfile)
	}
	definition, err := registry.Lookup(currency)
	if err != nil {
		return CreditLimit{}, fmt.Errorf("%w: credit-limit currency is invalid", ErrInvalidCustomerProfile)
	}
	value, err := money.NewMoney(definition, strings.TrimSpace(limit.Amount))
	if err != nil {
		return CreditLimit{}, fmt.Errorf("%w: credit-limit amount is invalid", ErrInvalidCustomerProfile)
	}
	if value.Amount().IsNegative() {
		return CreditLimit{}, fmt.Errorf("%w: credit-limit amount cannot be negative", ErrInvalidCustomerProfile)
	}
	return CreditLimit{Amount: value.Amount().String(), Currency: currency}, nil
}

func (limit CreditLimit) Validate() error {
	canonical, err := limit.Canonicalize()
	if err != nil {
		return err
	}
	if canonical != limit {
		return fmt.Errorf("%w: credit-limit value must be canonical", ErrInvalidCustomerProfile)
	}
	return nil
}

type CustomerProfile struct {
	ID                uuid.UUID                         `json:"id"`
	ScopeID           uuid.UUID                         `json:"scopeId"`
	PartyID           uuid.UUID                         `json:"partyId"`
	PartyVersion      aggregateversion.AggregateVersion `json:"partyVersion"`
	CreditTerms       string                            `json:"creditTerms"`
	CreditLimit       CreditLimit                       `json:"creditLimit"`
	BillingPreference string                            `json:"billingPreference"`
	TaxTreatment      string                            `json:"taxTreatment"`
	Status            string                            `json:"status"`
	EffectiveFrom     time.Time                         `json:"effectiveFrom"`
	EffectiveTo       *time.Time                        `json:"effectiveTo,omitempty"`
	Approval          *ApprovalDecisionReference        `json:"approval,omitempty"`
	Version           aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber    int64                             `json:"revisionNumber"`
	CreatedAt         time.Time                         `json:"createdAt"`
	UpdatedAt         time.Time                         `json:"updatedAt"`
	Revisions         []CustomerProfileRevision         `json:"revisions,omitempty"`
}

type CustomerProfileRevision struct {
	RevisionNumber int64                             `json:"revisionNumber"`
	Version        aggregateversion.AggregateVersion `json:"version"`
	Snapshot       CustomerProfileSnapshot           `json:"snapshot"`
	CreatedAt      time.Time                         `json:"createdAt"`
}

type CustomerProfileSnapshot struct {
	PartyID           uuid.UUID                         `json:"partyId"`
	PartyVersion      aggregateversion.AggregateVersion `json:"partyVersion"`
	CreditTerms       string                            `json:"creditTerms"`
	CreditLimit       CreditLimit                       `json:"creditLimit"`
	BillingPreference string                            `json:"billingPreference"`
	TaxTreatment      string                            `json:"taxTreatment"`
	Status            string                            `json:"status"`
	EffectiveFrom     time.Time                         `json:"effectiveFrom"`
	EffectiveTo       *time.Time                        `json:"effectiveTo,omitempty"`
	Approval          *ApprovalDecisionReference        `json:"approval,omitempty"`
}

type CustomerProfileFieldAuthorization struct {
	CreditTerms       bool
	CreditLimit       bool
	BillingPreference bool
	TaxTreatment      bool
}

func (authorization CustomerProfileFieldAuthorization) all() bool {
	return authorization.CreditTerms && authorization.CreditLimit && authorization.BillingPreference && authorization.TaxTreatment
}

func (profile CustomerProfile) Snapshot() CustomerProfileSnapshot {
	return CustomerProfileSnapshot{
		PartyID: profile.PartyID, PartyVersion: profile.PartyVersion,
		CreditTerms: profile.CreditTerms, CreditLimit: profile.CreditLimit,
		BillingPreference: profile.BillingPreference, TaxTreatment: profile.TaxTreatment,
		Status: profile.Status, EffectiveFrom: profile.EffectiveFrom, EffectiveTo: cloneTime(profile.EffectiveTo), Approval: cloneApproval(profile.Approval),
	}
}

func (profile CustomerProfile) Validate() error {
	if profile.ID == uuid.Nil || profile.ScopeID == uuid.Nil || profile.PartyID == uuid.Nil || profile.PartyVersion.Value() < 1 || profile.Version.Value() < 1 || profile.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidCustomerProfile)
	}
	if err := validateCustomerProfileCode(profile.CreditTerms, "credit terms"); err != nil {
		return err
	}
	if err := profile.CreditLimit.Validate(); err != nil {
		return err
	}
	if err := validateCustomerProfileCode(profile.BillingPreference, "billing preference"); err != nil {
		return err
	}
	if err := validateCustomerProfileCode(profile.TaxTreatment, "tax treatment"); err != nil {
		return err
	}
	switch profile.Status {
	case CustomerProfileStatusDraft, CustomerProfileStatusActive:
		if profile.EffectiveTo != nil {
			return fmt.Errorf("%w: non-ended profile cannot have an end date", ErrInvalidCustomerProfile)
		}
	case CustomerProfileStatusEndDated:
		if profile.EffectiveTo == nil {
			return fmt.Errorf("%w: ended profile requires an end date", ErrInvalidCustomerProfile)
		}
	default:
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidCustomerProfile)
	}
	if profile.EffectiveFrom.IsZero() || (profile.EffectiveTo != nil && !profile.EffectiveTo.After(profile.EffectiveFrom)) {
		return fmt.Errorf("%w: effective interval is invalid", ErrInvalidCustomerProfile)
	}
	if profile.Approval != nil {
		if err := profile.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: approval reference is invalid", ErrInvalidCustomerProfile)
		}
	}
	if profile.CreatedAt.IsZero() || profile.UpdatedAt.IsZero() || profile.UpdatedAt.Before(profile.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidCustomerProfile)
	}
	return nil
}

func NewCustomerProfile(id, scopeID, partyID uuid.UUID, partyVersion aggregateversion.AggregateVersion, creditTerms string, creditLimit CreditLimit, billingPreference, taxTreatment string, effectiveFrom time.Time, approval *ApprovalDecisionReference, status string, now time.Time) (CustomerProfile, error) {
	canonicalLimit, err := creditLimit.Canonicalize()
	if err != nil {
		return CustomerProfile{}, err
	}
	profile := CustomerProfile{
		ID: id, ScopeID: scopeID, PartyID: partyID, PartyVersion: partyVersion,
		CreditTerms: canonicalCustomerProfileCode(creditTerms), CreditLimit: canonicalLimit,
		BillingPreference: canonicalCustomerProfileCode(billingPreference), TaxTreatment: canonicalCustomerProfileCode(taxTreatment),
		Status: status, EffectiveFrom: dateOnly(effectiveFrom), Approval: cloneApproval(approval),
		Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if err := profile.Validate(); err != nil {
		return CustomerProfile{}, err
	}
	profile.Revisions = []CustomerProfileRevision{{RevisionNumber: profile.RevisionNumber, Version: profile.Version, Snapshot: profile.Snapshot(), CreatedAt: profile.UpdatedAt}}
	return profile, nil
}

func (profile *CustomerProfile) Replace(current CustomerProfile, partyVersion aggregateversion.AggregateVersion, creditTerms string, creditLimit CreditLimit, billingPreference, taxTreatment string, effectiveFrom time.Time, effectiveTo *time.Time, approval *ApprovalDecisionReference, now time.Time) error {
	if profile == nil || current.ID == uuid.Nil || profile.ID != current.ID {
		return ErrInvalidCustomerProfile
	}
	if current.Status == CustomerProfileStatusEndDated {
		return fmt.Errorf("%w: an end-dated customer profile cannot be maintained", ErrInvalidCustomerProfile)
	}
	canonicalLimit, err := creditLimit.Canonicalize()
	if err != nil {
		return err
	}
	nextVersion, err := current.Version.Advance()
	if err != nil {
		return err
	}
	profile.ID, profile.ScopeID, profile.PartyID = current.ID, current.ScopeID, current.PartyID
	profile.PartyVersion = partyVersion
	profile.CreditTerms, profile.CreditLimit = canonicalCustomerProfileCode(creditTerms), canonicalLimit
	profile.BillingPreference, profile.TaxTreatment = canonicalCustomerProfileCode(billingPreference), canonicalCustomerProfileCode(taxTreatment)
	profile.EffectiveFrom, profile.EffectiveTo = dateOnly(effectiveFrom), cloneTime(effectiveTo)
	profile.Approval, profile.UpdatedAt = cloneApproval(approval), now.UTC()
	if profile.EffectiveTo != nil {
		profile.Status = CustomerProfileStatusEndDated
	} else if current.Status == CustomerProfileStatusDraft {
		profile.Status = CustomerProfileStatusActive
	} else {
		profile.Status = current.Status
	}
	profile.Version, profile.RevisionNumber = nextVersion, current.RevisionNumber+1
	if err := profile.Validate(); err != nil {
		return err
	}
	profile.Revisions = append(cloneCustomerProfileRevisions(current.Revisions), CustomerProfileRevision{RevisionNumber: profile.RevisionNumber, Version: profile.Version, Snapshot: profile.Snapshot(), CreatedAt: profile.UpdatedAt})
	return nil
}

type SafeCustomerProfile struct {
	ID                uuid.UUID                         `json:"id"`
	ScopeID           uuid.UUID                         `json:"scopeId"`
	PartyID           uuid.UUID                         `json:"partyId"`
	PartyVersion      aggregateversion.AggregateVersion `json:"partyVersion"`
	CreditTerms       string                            `json:"creditTerms,omitempty"`
	CreditLimit       *CreditLimit                      `json:"creditLimit,omitempty"`
	BillingPreference string                            `json:"billingPreference,omitempty"`
	TaxTreatment      string                            `json:"taxTreatment,omitempty"`
	Status            string                            `json:"status"`
	EffectiveFrom     time.Time                         `json:"effectiveFrom"`
	EffectiveTo       *time.Time                        `json:"effectiveTo,omitempty"`
	ApprovalStatus    string                            `json:"approvalStatus"`
	Version           aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber    int64                             `json:"revisionNumber"`
	NextAction        string                            `json:"nextAction"`
}

func (profile CustomerProfile) SafeProjection() SafeCustomerProfile {
	return profile.SafeProjectionWithAccess(CustomerProfileFieldAuthorization{CreditTerms: true, CreditLimit: true, BillingPreference: true, TaxTreatment: true})
}

func (profile CustomerProfile) SafeProjectionWithAccess(access CustomerProfileFieldAuthorization) SafeCustomerProfile {
	approvalStatus, nextAction := "not-required", "maintain"
	if profile.Approval != nil {
		approvalStatus, nextAction = "approved", "ready"
	}
	if profile.Status == CustomerProfileStatusDraft {
		approvalStatus, nextAction = "pending", "submit for approval"
	}
	if profile.Status == CustomerProfileStatusEndDated {
		nextAction = "view history"
	}
	result := SafeCustomerProfile{
		ID: profile.ID, ScopeID: profile.ScopeID, PartyID: profile.PartyID, PartyVersion: profile.PartyVersion,
		Status: profile.Status, EffectiveFrom: profile.EffectiveFrom, EffectiveTo: cloneTime(profile.EffectiveTo),
		ApprovalStatus: approvalStatus, Version: profile.Version, RevisionNumber: profile.RevisionNumber, NextAction: nextAction,
	}
	if access.CreditTerms {
		result.CreditTerms = profile.CreditTerms
	}
	if access.CreditLimit {
		limit := profile.CreditLimit
		result.CreditLimit = &limit
	}
	if access.BillingPreference {
		result.BillingPreference = profile.BillingPreference
	}
	if access.TaxTreatment {
		result.TaxTreatment = profile.TaxTreatment
	}
	return result
}

func FingerprintCustomerProfile(profile CustomerProfile) string {
	data, _ := json.Marshal(profile.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validateCustomerProfileCode(value, label string) error {
	if value != canonicalCustomerProfileCode(value) || !customerProfileCodePattern.MatchString(value) {
		return fmt.Errorf("%w: %s must be a canonical non-empty code", ErrInvalidCustomerProfile, label)
	}
	return nil
}

func canonicalCustomerProfileCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cloneCustomerProfileRevisions(values []CustomerProfileRevision) []CustomerProfileRevision {
	result := make([]CustomerProfileRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.EffectiveTo = cloneTime(values[index].Snapshot.EffectiveTo)
		result[index].Snapshot.Approval = cloneApproval(values[index].Snapshot.Approval)
	}
	return result
}

func cloneCustomerProfile(value CustomerProfile) CustomerProfile {
	value.EffectiveTo = cloneTime(value.EffectiveTo)
	value.Approval = cloneApproval(value.Approval)
	value.Revisions = cloneCustomerProfileRevisions(value.Revisions)
	return value
}
