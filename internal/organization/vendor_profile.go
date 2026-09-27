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
)

const VendorProfileManagementPermission = "finance.omd.maintain.vendor.profiles"

const (
	VendorProfileActionCreate   = "create"
	VendorProfileActionMaintain = "maintain"

	VendorProfileStatusDraft    = "draft"
	VendorProfileStatusActive   = "active"
	VendorProfileStatusEndDated = "end_dated"
)

var (
	ErrInvalidVendorProfile                       = errors.New("invalid vendor profile")
	ErrInvalidVendorProfileCommand                = errors.New("invalid vendor profile command")
	ErrVendorProfileNotFound                      = errors.New("vendor profile not found")
	ErrVendorProfileVersionConflict               = errors.New("vendor profile version conflict")
	ErrVendorProfilePartyNotFound                 = errors.New("vendor profile party not found")
	ErrVendorProfilePartyInvalid                  = errors.New("vendor profile party is invalid")
	ErrVendorProfilePartyVersionConflict          = errors.New("vendor profile party version conflict")
	ErrVendorProfilePartyMismatch                 = errors.New("vendor profile party reference mismatch")
	ErrVendorProfileAuthorizationDenied           = errors.New("vendor profile authorization denied")
	ErrVendorProfileAuthorizationUnavailable      = errors.New("vendor profile authorization unavailable")
	ErrVendorProfileAuthorizationStale            = errors.New("vendor profile authorization stale")
	ErrVendorProfileFieldAuthorizationDenied      = errors.New("vendor profile field authorization denied")
	ErrVendorProfileFieldAuthorizationUnavailable = errors.New("vendor profile field authorization unavailable")
	ErrVendorProfileIdempotencyConflict           = errors.New("vendor profile idempotency conflict")
	ErrVendorProfileCommandInProgress             = errors.New("vendor profile command is already in progress")
	ErrVendorProfileAuditUnavailable              = errors.New("vendor profile audit unavailable")
	ErrVendorProfileApprovalRequired              = errors.New("vendor profile approval required")
	ErrVendorProfileApprovalRejected              = errors.New("vendor profile approval rejected")
	ErrVendorProfileDuplicate                     = errors.New("duplicate vendor profile value")
	ErrInvalidVendorProfileService                = errors.New("invalid vendor profile service")
	ErrVendorProfileDurableCommandFailed          = errors.New("vendor profile command previously failed")
)

var vendorProfileCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// VendorProfile contains OMD-owned vendor settlement policy. Bank details
// remain Party-owned and are intentionally not represented here.
type VendorProfile struct {
	ID                   uuid.UUID                         `json:"id"`
	ScopeID              uuid.UUID                         `json:"scopeId"`
	PartyID              uuid.UUID                         `json:"partyId"`
	PartyVersion         aggregateversion.AggregateVersion `json:"partyVersion"`
	PaymentTerms         string                            `json:"paymentTerms"`
	WithholdingTreatment string                            `json:"withholdingTreatment"`
	RemittancePreference string                            `json:"remittancePreference"`
	Status               string                            `json:"status"`
	EffectiveFrom        time.Time                         `json:"effectiveFrom"`
	EffectiveTo          *time.Time                        `json:"effectiveTo,omitempty"`
	Approval             *ApprovalDecisionReference        `json:"approval,omitempty"`
	Version              aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber       int64                             `json:"revisionNumber"`
	CreatedAt            time.Time                         `json:"createdAt"`
	UpdatedAt            time.Time                         `json:"updatedAt"`
	Revisions            []VendorProfileRevision           `json:"revisions,omitempty"`
}

type VendorProfileRevision struct {
	RevisionNumber int64                             `json:"revisionNumber"`
	Version        aggregateversion.AggregateVersion `json:"version"`
	Snapshot       VendorProfileSnapshot             `json:"snapshot"`
	CreatedAt      time.Time                         `json:"createdAt"`
}

type VendorProfileSnapshot struct {
	PartyID              uuid.UUID                         `json:"partyId"`
	PartyVersion         aggregateversion.AggregateVersion `json:"partyVersion"`
	PaymentTerms         string                            `json:"paymentTerms"`
	WithholdingTreatment string                            `json:"withholdingTreatment"`
	RemittancePreference string                            `json:"remittancePreference"`
	Status               string                            `json:"status"`
	EffectiveFrom        time.Time                         `json:"effectiveFrom"`
	EffectiveTo          *time.Time                        `json:"effectiveTo,omitempty"`
	Approval             *ApprovalDecisionReference        `json:"approval,omitempty"`
}

type VendorProfileFieldAuthorization struct {
	PaymentTerms         bool
	WithholdingTreatment bool
	RemittancePreference bool
}

func (authorization VendorProfileFieldAuthorization) all() bool {
	return authorization.PaymentTerms && authorization.WithholdingTreatment && authorization.RemittancePreference
}

func (profile VendorProfile) Snapshot() VendorProfileSnapshot {
	return VendorProfileSnapshot{
		PartyID: profile.PartyID, PartyVersion: profile.PartyVersion,
		PaymentTerms: profile.PaymentTerms, WithholdingTreatment: profile.WithholdingTreatment,
		RemittancePreference: profile.RemittancePreference, Status: profile.Status,
		EffectiveFrom: profile.EffectiveFrom, EffectiveTo: cloneTime(profile.EffectiveTo), Approval: cloneApproval(profile.Approval),
	}
}

func (profile VendorProfile) Validate() error {
	if profile.ID == uuid.Nil || profile.ScopeID == uuid.Nil || profile.PartyID == uuid.Nil || profile.PartyVersion.Value() < 1 || profile.Version.Value() < 1 || profile.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidVendorProfile)
	}
	if err := validateVendorProfileCode(profile.PaymentTerms, "payment terms"); err != nil {
		return err
	}
	if err := validateVendorProfileCode(profile.WithholdingTreatment, "withholding treatment"); err != nil {
		return err
	}
	if err := validateVendorProfileCode(profile.RemittancePreference, "remittance preference"); err != nil {
		return err
	}
	switch profile.Status {
	case VendorProfileStatusDraft, VendorProfileStatusActive:
		if profile.EffectiveTo != nil {
			return fmt.Errorf("%w: non-ended profile cannot have an end date", ErrInvalidVendorProfile)
		}
	case VendorProfileStatusEndDated:
		if profile.EffectiveTo == nil {
			return fmt.Errorf("%w: ended profile requires an end date", ErrInvalidVendorProfile)
		}
	default:
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidVendorProfile)
	}
	if profile.EffectiveFrom.IsZero() || (profile.EffectiveTo != nil && !profile.EffectiveTo.After(profile.EffectiveFrom)) {
		return fmt.Errorf("%w: effective interval is invalid", ErrInvalidVendorProfile)
	}
	if profile.Approval != nil {
		if err := profile.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: approval reference is invalid", ErrInvalidVendorProfile)
		}
	}
	if profile.CreatedAt.IsZero() || profile.UpdatedAt.IsZero() || profile.UpdatedAt.Before(profile.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidVendorProfile)
	}
	return nil
}

func NewVendorProfile(id, scopeID, partyID uuid.UUID, partyVersion aggregateversion.AggregateVersion, paymentTerms, withholdingTreatment, remittancePreference string, effectiveFrom time.Time, approval *ApprovalDecisionReference, status string, now time.Time) (VendorProfile, error) {
	profile := VendorProfile{
		ID: id, ScopeID: scopeID, PartyID: partyID, PartyVersion: partyVersion,
		PaymentTerms: canonicalVendorProfileCode(paymentTerms), WithholdingTreatment: canonicalVendorProfileCode(withholdingTreatment), RemittancePreference: canonicalVendorProfileCode(remittancePreference),
		Status: status, EffectiveFrom: dateOnly(effectiveFrom), Approval: cloneApproval(approval),
		Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if err := profile.Validate(); err != nil {
		return VendorProfile{}, err
	}
	profile.Revisions = []VendorProfileRevision{{RevisionNumber: profile.RevisionNumber, Version: profile.Version, Snapshot: profile.Snapshot(), CreatedAt: profile.UpdatedAt}}
	return profile, nil
}

func (profile *VendorProfile) Replace(current VendorProfile, partyVersion aggregateversion.AggregateVersion, paymentTerms, withholdingTreatment, remittancePreference string, effectiveFrom time.Time, effectiveTo *time.Time, approval *ApprovalDecisionReference, now time.Time) error {
	if profile == nil || current.ID == uuid.Nil || profile.ID != current.ID {
		return ErrInvalidVendorProfile
	}
	if current.Status == VendorProfileStatusEndDated {
		return fmt.Errorf("%w: an end-dated vendor profile cannot be maintained", ErrInvalidVendorProfile)
	}
	nextVersion, err := current.Version.Advance()
	if err != nil {
		return err
	}
	profile.ID, profile.ScopeID, profile.PartyID = current.ID, current.ScopeID, current.PartyID
	profile.PartyVersion = partyVersion
	profile.PaymentTerms = canonicalVendorProfileCode(paymentTerms)
	profile.WithholdingTreatment = canonicalVendorProfileCode(withholdingTreatment)
	profile.RemittancePreference = canonicalVendorProfileCode(remittancePreference)
	profile.EffectiveFrom, profile.EffectiveTo = dateOnly(effectiveFrom), cloneTime(effectiveTo)
	profile.Approval, profile.UpdatedAt = cloneApproval(approval), now.UTC()
	if profile.EffectiveTo != nil {
		profile.Status = VendorProfileStatusEndDated
	} else if current.Status == VendorProfileStatusDraft {
		profile.Status = VendorProfileStatusActive
	} else {
		profile.Status = current.Status
	}
	profile.Version, profile.RevisionNumber = nextVersion, current.RevisionNumber+1
	if err := profile.Validate(); err != nil {
		return err
	}
	profile.Revisions = append(cloneVendorProfileRevisions(current.Revisions), VendorProfileRevision{RevisionNumber: profile.RevisionNumber, Version: profile.Version, Snapshot: profile.Snapshot(), CreatedAt: profile.UpdatedAt})
	return nil
}

type SafeVendorProfile struct {
	ID                   uuid.UUID                         `json:"id"`
	ScopeID              uuid.UUID                         `json:"scopeId"`
	PartyID              uuid.UUID                         `json:"partyId"`
	PartyVersion         aggregateversion.AggregateVersion `json:"partyVersion"`
	PaymentTerms         string                            `json:"paymentTerms,omitempty"`
	WithholdingTreatment string                            `json:"withholdingTreatment,omitempty"`
	RemittancePreference string                            `json:"remittancePreference,omitempty"`
	Status               string                            `json:"status"`
	EffectiveFrom        time.Time                         `json:"effectiveFrom"`
	EffectiveTo          *time.Time                        `json:"effectiveTo,omitempty"`
	ApprovalStatus       string                            `json:"approvalStatus"`
	Version              aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber       int64                             `json:"revisionNumber"`
	NextAction           string                            `json:"nextAction"`
}

func (profile VendorProfile) SafeProjection() SafeVendorProfile {
	return profile.SafeProjectionWithAccess(VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true})
}

func (profile VendorProfile) SafeProjectionWithAccess(access VendorProfileFieldAuthorization) SafeVendorProfile {
	approvalStatus, nextAction := "not-required", "maintain"
	if profile.Approval != nil {
		approvalStatus, nextAction = "approved", "ready"
	}
	if profile.Status == VendorProfileStatusDraft {
		approvalStatus, nextAction = "pending", "submit for approval"
	}
	if profile.Status == VendorProfileStatusEndDated {
		nextAction = "view history"
	}
	result := SafeVendorProfile{ID: profile.ID, ScopeID: profile.ScopeID, PartyID: profile.PartyID, PartyVersion: profile.PartyVersion, Status: profile.Status, EffectiveFrom: profile.EffectiveFrom, EffectiveTo: cloneTime(profile.EffectiveTo), ApprovalStatus: approvalStatus, Version: profile.Version, RevisionNumber: profile.RevisionNumber, NextAction: nextAction}
	if access.PaymentTerms {
		result.PaymentTerms = profile.PaymentTerms
	}
	if access.WithholdingTreatment {
		result.WithholdingTreatment = profile.WithholdingTreatment
	}
	if access.RemittancePreference {
		result.RemittancePreference = profile.RemittancePreference
	}
	return result
}

func FingerprintVendorProfile(profile VendorProfile) string {
	data, _ := json.Marshal(profile.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validateVendorProfileCode(value, label string) error {
	if value != canonicalVendorProfileCode(value) || !vendorProfileCodePattern.MatchString(value) {
		return fmt.Errorf("%w: %s must be a canonical non-empty code", ErrInvalidVendorProfile, label)
	}
	return nil
}

func canonicalVendorProfileCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cloneVendorProfileRevisions(values []VendorProfileRevision) []VendorProfileRevision {
	result := make([]VendorProfileRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.EffectiveTo = cloneTime(values[index].Snapshot.EffectiveTo)
		result[index].Snapshot.Approval = cloneApproval(values[index].Snapshot.Approval)
	}
	return result
}

func cloneVendorProfile(value VendorProfile) VendorProfile {
	value.EffectiveTo = cloneTime(value.EffectiveTo)
	value.Approval = cloneApproval(value.Approval)
	value.Revisions = cloneVendorProfileRevisions(value.Revisions)
	return value
}
