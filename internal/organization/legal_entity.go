package organization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

const LegalEntityManagementPermission = "finance.omd.maintain.legal.entities"
const LegalEntityReadPermission = "finance.omd.read.legal.entities"

const (
	LegalEntityActionCreate   = "create"
	LegalEntityActionMaintain = "maintain"
	LegalEntityActionEndDate  = "end-date"

	LegalEntityStatusDraft    = "draft"
	LegalEntityStatusActive   = "active"
	LegalEntityStatusEndDated = "end_dated"
)

var (
	ErrInvalidLegalEntity                  = errors.New("invalid legal entity")
	ErrInvalidLegalEntityCommand           = errors.New("invalid legal entity command")
	ErrLegalEntityNotFound                 = errors.New("legal entity not found")
	ErrLegalEntityVersionConflict          = errors.New("legal entity version conflict")
	ErrLegalEntityAuthorizationDenied      = errors.New("legal entity authorization denied")
	ErrLegalEntityAuthorizationUnavailable = errors.New("legal entity authorization unavailable")
	ErrLegalEntityAuthorizationStale       = errors.New("legal entity authorization stale")
	ErrLegalEntityIdempotencyConflict      = errors.New("legal entity idempotency conflict")
	ErrLegalEntityCommandInProgress        = errors.New("legal entity command is already in progress")
	ErrLegalEntityAuditUnavailable         = errors.New("legal entity audit unavailable")
	ErrLegalEntityApprovalRequired         = errors.New("legal entity approval required")
	ErrLegalEntityApprovalRejected         = errors.New("legal entity approval rejected")
	ErrLegalEntityApprovalUnavailable      = errors.New("legal entity approval unavailable")
	ErrLegalEntityDuplicate                = errors.New("duplicate legal entity value")
	ErrInvalidLegalEntityService           = errors.New("invalid legal entity service")
	ErrLegalEntityDurableCommandFailed     = errors.New("legal entity command previously failed")
)

var (
	currencyPattern   = regexp.MustCompile(`^[A-Z]{3}$`)
	codePattern       = regexp.MustCompile(`^[A-Z]{2,3}$`)
	percentagePattern = regexp.MustCompile(`^(100|[0-9]{1,2})(\.[0-9]{1,6})?$`)
)

type LegalEntityRegistration struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Identifier    string     `json:"identifier"`
	Jurisdiction  string     `json:"jurisdiction"`
	EffectiveFrom time.Time  `json:"effectiveFrom"`
	EffectiveTo   *time.Time `json:"effectiveTo,omitempty"`
}

type LegalEntityAddress struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Line1         string     `json:"line1"`
	Line2         string     `json:"line2,omitempty"`
	Locality      string     `json:"locality"`
	Region        string     `json:"region,omitempty"`
	PostalCode    string     `json:"postalCode"`
	CountryCode   string     `json:"countryCode"`
	EffectiveFrom time.Time  `json:"effectiveFrom"`
	EffectiveTo   *time.Time `json:"effectiveTo,omitempty"`
}

type LegalEntityOwnershipInterest struct {
	ID             uuid.UUID  `json:"id"`
	OwnerReference string     `json:"ownerReference"`
	Percentage     string     `json:"percentage"`
	EffectiveFrom  time.Time  `json:"effectiveFrom"`
	EffectiveTo    *time.Time `json:"effectiveTo,omitempty"`
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
	if reference.ApprovalRequestID == uuid.Nil || reference.DecisionID == uuid.Nil || reference.ApproverUserID == uuid.Nil {
		return fmt.Errorf("%w: approval identifiers are required", ErrInvalidLegalEntity)
	}
	if strings.TrimSpace(reference.PolicyVersion) == "" || reference.DecisionVersion < 1 || reference.SubjectVersion < 1 || strings.TrimSpace(reference.CandidateFingerprint) == "" {
		return fmt.Errorf("%w: approval metadata is incomplete", ErrInvalidLegalEntity)
	}
	return nil
}

type LegalEntity struct {
	ID                   uuid.UUID                         `json:"id"`
	ScopeID              uuid.UUID                         `json:"scopeId"`
	LegalName            string                            `json:"legalName"`
	FunctionalCurrency   string                            `json:"functionalCurrency"`
	PresentationCurrency string                            `json:"presentationCurrency"`
	TaxRegistrationID    string                            `json:"taxRegistrationId,omitempty"`
	Status               string                            `json:"status"`
	EffectiveFrom        time.Time                         `json:"effectiveFrom"`
	EffectiveTo          *time.Time                        `json:"effectiveTo,omitempty"`
	Registrations        []LegalEntityRegistration         `json:"registrations"`
	Addresses            []LegalEntityAddress              `json:"addresses"`
	OwnershipInterests   []LegalEntityOwnershipInterest    `json:"ownershipInterests"`
	Approval             *ApprovalDecisionReference        `json:"approval,omitempty"`
	Version              aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber       int64                             `json:"revisionNumber"`
	CreatedAt            time.Time                         `json:"createdAt"`
	UpdatedAt            time.Time                         `json:"updatedAt"`
	Revisions            []LegalEntityRevision             `json:"revisions,omitempty"`
}

type LegalEntityRevision struct {
	RevisionNumber int64                             `json:"revisionNumber"`
	Version        aggregateversion.AggregateVersion `json:"version"`
	Snapshot       LegalEntitySnapshot               `json:"snapshot"`
	CreatedAt      time.Time                         `json:"createdAt"`
}

type LegalEntitySnapshot struct {
	LegalName            string                         `json:"legalName"`
	FunctionalCurrency   string                         `json:"functionalCurrency"`
	PresentationCurrency string                         `json:"presentationCurrency"`
	TaxRegistrationID    string                         `json:"taxRegistrationId,omitempty"`
	Status               string                         `json:"status"`
	EffectiveFrom        time.Time                      `json:"effectiveFrom"`
	EffectiveTo          *time.Time                     `json:"effectiveTo,omitempty"`
	Registrations        []LegalEntityRegistration      `json:"registrations"`
	Addresses            []LegalEntityAddress           `json:"addresses"`
	OwnershipInterests   []LegalEntityOwnershipInterest `json:"ownershipInterests"`
	Approval             *ApprovalDecisionReference     `json:"approval,omitempty"`
}

func (entity LegalEntity) Snapshot() LegalEntitySnapshot {
	return LegalEntitySnapshot{
		LegalName: entity.LegalName, FunctionalCurrency: entity.FunctionalCurrency,
		PresentationCurrency: entity.PresentationCurrency, TaxRegistrationID: entity.TaxRegistrationID,
		Status: entity.Status, EffectiveFrom: entity.EffectiveFrom, EffectiveTo: cloneTime(entity.EffectiveTo),
		Registrations:      append([]LegalEntityRegistration(nil), entity.Registrations...),
		Addresses:          append([]LegalEntityAddress(nil), entity.Addresses...),
		OwnershipInterests: append([]LegalEntityOwnershipInterest(nil), entity.OwnershipInterests...),
		Approval:           cloneApproval(entity.Approval),
	}
}

func (entity LegalEntity) Validate() error {
	if entity.ID == uuid.Nil || entity.ScopeID == uuid.Nil || entity.Version.Value() < 1 || entity.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidLegalEntity)
	}
	if len(strings.TrimSpace(entity.LegalName)) == 0 || len([]rune(entity.LegalName)) > 200 {
		return fmt.Errorf("%w: legal name is required and must be at most 200 characters", ErrInvalidLegalEntity)
	}
	if !currencyPattern.MatchString(entity.FunctionalCurrency) || !currencyPattern.MatchString(entity.PresentationCurrency) {
		return fmt.Errorf("%w: currency must be an uppercase three-letter code", ErrInvalidLegalEntity)
	}
	if entity.TaxRegistrationID != "" && (len([]rune(entity.TaxRegistrationID)) > 120 || strings.TrimSpace(entity.TaxRegistrationID) != entity.TaxRegistrationID) {
		return fmt.Errorf("%w: tax registration id is invalid", ErrInvalidLegalEntity)
	}
	switch entity.Status {
	case LegalEntityStatusDraft, LegalEntityStatusActive:
		if entity.EffectiveTo != nil {
			return fmt.Errorf("%w: non-ended entity cannot have an end date", ErrInvalidLegalEntity)
		}
	case LegalEntityStatusEndDated:
		if entity.EffectiveTo == nil {
			return fmt.Errorf("%w: ended entity requires an end date", ErrInvalidLegalEntity)
		}
	default:
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidLegalEntity)
	}
	if err := validateInterval(entity.EffectiveFrom, entity.EffectiveTo); err != nil {
		return err
	}
	if err := validateRegistrations(entity.Registrations); err != nil {
		return err
	}
	if err := validateAddresses(entity.Addresses); err != nil {
		return err
	}
	if err := validateOwnership(entity.OwnershipInterests); err != nil {
		return err
	}
	if entity.Approval != nil {
		if err := entity.Approval.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func NewLegalEntity(id, scopeID uuid.UUID, name, functionalCurrency, presentationCurrency, taxID string, from time.Time, registrations []LegalEntityRegistration, addresses []LegalEntityAddress, ownership []LegalEntityOwnershipInterest, approval *ApprovalDecisionReference, status string, now time.Time) (LegalEntity, error) {
	entity := LegalEntity{ID: id, ScopeID: scopeID, LegalName: strings.TrimSpace(name), FunctionalCurrency: functionalCurrency, PresentationCurrency: presentationCurrency, TaxRegistrationID: taxID, Status: status, EffectiveFrom: dateOnly(from), Registrations: registrations, Addresses: addresses, OwnershipInterests: ownership, Approval: cloneApproval(approval), Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
	if err := entity.Validate(); err != nil {
		return LegalEntity{}, err
	}
	entity.Revisions = []LegalEntityRevision{{RevisionNumber: 1, Version: entity.Version, Snapshot: entity.Snapshot(), CreatedAt: now.UTC()}}
	return entity, nil
}

func (entity *LegalEntity) Replace(current LegalEntity, name, functionalCurrency, presentationCurrency, taxID string, from time.Time, to *time.Time, registrations []LegalEntityRegistration, addresses []LegalEntityAddress, ownership []LegalEntityOwnershipInterest, approval *ApprovalDecisionReference, now time.Time) error {
	if entity == nil {
		return ErrInvalidLegalEntity
	}
	if current.Status == LegalEntityStatusEndDated {
		return fmt.Errorf("%w: an end-dated legal entity cannot be maintained", ErrInvalidLegalEntity)
	}
	entity.LegalName, entity.FunctionalCurrency, entity.PresentationCurrency, entity.TaxRegistrationID = strings.TrimSpace(name), functionalCurrency, presentationCurrency, taxID
	entity.EffectiveFrom, entity.EffectiveTo = dateOnly(from), cloneTime(to)
	entity.Registrations, entity.Addresses, entity.OwnershipInterests = registrations, addresses, ownership
	entity.Approval, entity.UpdatedAt = cloneApproval(approval), now.UTC()
	if entity.EffectiveTo != nil {
		entity.Status = LegalEntityStatusEndDated
	} else if entity.Status == LegalEntityStatusDraft {
		entity.Status = LegalEntityStatusActive
	}
	next, err := current.Version.Advance()
	if err != nil {
		return err
	}
	entity.Version, entity.RevisionNumber = next, current.RevisionNumber+1
	if err := entity.Validate(); err != nil {
		return err
	}
	entity.Revisions = append(cloneRevisions(current.Revisions), LegalEntityRevision{RevisionNumber: entity.RevisionNumber, Version: entity.Version, Snapshot: entity.Snapshot(), CreatedAt: entity.UpdatedAt})
	return nil
}

func (entity *LegalEntity) EndDate(current LegalEntity, to time.Time, now time.Time) error {
	if entity == nil {
		return ErrInvalidLegalEntity
	}
	if current.Status == LegalEntityStatusEndDated {
		return fmt.Errorf("%w: legal entity is already end-dated", ErrInvalidLegalEntity)
	}
	copy := cloneLegalEntity(current)
	copy.Status, copy.EffectiveTo, copy.UpdatedAt = LegalEntityStatusEndDated, ptrDate(to), now.UTC()
	next, err := current.Version.Advance()
	if err != nil {
		return err
	}
	copy.Version, copy.RevisionNumber = next, current.RevisionNumber+1
	if err := copy.Validate(); err != nil {
		return err
	}
	copy.Revisions = append(cloneRevisions(current.Revisions), LegalEntityRevision{RevisionNumber: copy.RevisionNumber, Version: copy.Version, Snapshot: copy.Snapshot(), CreatedAt: copy.UpdatedAt})
	*entity = copy
	return nil
}

type SafeLegalEntity struct {
	ID                   uuid.UUID                         `json:"id"`
	ScopeID              uuid.UUID                         `json:"scopeId"`
	LegalName            string                            `json:"legalName"`
	FunctionalCurrency   string                            `json:"functionalCurrency"`
	PresentationCurrency string                            `json:"presentationCurrency"`
	Status               string                            `json:"status"`
	EffectiveFrom        time.Time                         `json:"effectiveFrom"`
	EffectiveTo          *time.Time                        `json:"effectiveTo,omitempty"`
	Registrations        []SafeRegistration                `json:"registrations"`
	Addresses            []LegalEntityAddress              `json:"addresses"`
	OwnershipInterests   []LegalEntityOwnershipInterest    `json:"ownershipInterests"`
	ApprovalStatus       string                            `json:"approvalStatus"`
	Version              aggregateversion.AggregateVersion `json:"version"`
	NextAction           string                            `json:"nextAction"`
}

type SafeRegistration struct {
	Type             string     `json:"type"`
	Jurisdiction     string     `json:"jurisdiction"`
	IdentifierMasked string     `json:"identifierMasked"`
	EffectiveFrom    time.Time  `json:"effectiveFrom"`
	EffectiveTo      *time.Time `json:"effectiveTo,omitempty"`
}

func (entity LegalEntity) SafeProjection() SafeLegalEntity {
	registrations := make([]SafeRegistration, 0, len(entity.Registrations))
	for _, registration := range entity.Registrations {
		registrations = append(registrations, SafeRegistration{Type: registration.Type, Jurisdiction: registration.Jurisdiction, IdentifierMasked: maskIdentifier(registration.Identifier), EffectiveFrom: registration.EffectiveFrom, EffectiveTo: cloneTime(registration.EffectiveTo)})
	}
	approvalStatus, nextAction := "not-required", "maintain"
	if entity.Approval != nil {
		approvalStatus, nextAction = "approved", "ready"
	}
	if entity.Status == LegalEntityStatusDraft {
		approvalStatus, nextAction = "pending", "submit for approval"
	}
	if entity.Status == LegalEntityStatusEndDated {
		nextAction = "view history"
	}
	return SafeLegalEntity{ID: entity.ID, ScopeID: entity.ScopeID, LegalName: entity.LegalName, FunctionalCurrency: entity.FunctionalCurrency, PresentationCurrency: entity.PresentationCurrency, Status: entity.Status, EffectiveFrom: entity.EffectiveFrom, EffectiveTo: cloneTime(entity.EffectiveTo), Registrations: registrations, Addresses: append([]LegalEntityAddress(nil), entity.Addresses...), OwnershipInterests: append([]LegalEntityOwnershipInterest(nil), entity.OwnershipInterests...), ApprovalStatus: approvalStatus, Version: entity.Version, NextAction: nextAction}
}

func Fingerprint(entity LegalEntity) string {
	data, _ := json.Marshal(entity.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validateRegistrations(values []LegalEntityRegistration) error {
	for index, value := range values {
		if value.ID == uuid.Nil {
			return fmt.Errorf("%w: registration %d id is required", ErrInvalidLegalEntity, index)
		}
		if strings.TrimSpace(value.Type) == "" || len([]rune(value.Type)) > 64 || strings.TrimSpace(value.Identifier) == "" || len([]rune(value.Identifier)) > 120 || !codePattern.MatchString(value.Jurisdiction) {
			return fmt.Errorf("%w: registration %d is invalid", ErrInvalidLegalEntity, index)
		}
		if err := validateInterval(value.EffectiveFrom, value.EffectiveTo); err != nil {
			return err
		}
		for previous := 0; previous < index; previous++ {
			other := values[previous]
			if value.Type == other.Type && value.Identifier == other.Identifier && value.Jurisdiction == other.Jurisdiction && intervalsOverlap(value.EffectiveFrom, value.EffectiveTo, other.EffectiveFrom, other.EffectiveTo) {
				return fmt.Errorf("%w: overlapping registration", ErrLegalEntityDuplicate)
			}
		}
	}
	return nil
}

func validateAddresses(values []LegalEntityAddress) error {
	for index, value := range values {
		if value.ID == uuid.Nil || strings.TrimSpace(value.Type) == "" || strings.TrimSpace(value.Line1) == "" || strings.TrimSpace(value.Locality) == "" || strings.TrimSpace(value.PostalCode) == "" || !codePattern.MatchString(value.CountryCode) {
			return fmt.Errorf("%w: address %d is invalid", ErrInvalidLegalEntity, index)
		}
		if err := validateInterval(value.EffectiveFrom, value.EffectiveTo); err != nil {
			return err
		}
		for previous := 0; previous < index; previous++ {
			other := values[previous]
			if value.Type == other.Type && intervalsOverlap(value.EffectiveFrom, value.EffectiveTo, other.EffectiveFrom, other.EffectiveTo) {
				return fmt.Errorf("%w: overlapping address type", ErrLegalEntityDuplicate)
			}
		}
	}
	return nil
}

func validateOwnership(values []LegalEntityOwnershipInterest) error {
	total := new(big.Rat)
	owners := map[string][]LegalEntityOwnershipInterest{}
	for index, value := range values {
		if value.ID == uuid.Nil || strings.TrimSpace(value.OwnerReference) == "" || !percentagePattern.MatchString(value.Percentage) {
			return fmt.Errorf("%w: ownership interest %d is invalid", ErrInvalidLegalEntity, index)
		}
		if err := validateInterval(value.EffectiveFrom, value.EffectiveTo); err != nil {
			return err
		}
		fraction, ok := new(big.Rat).SetString(value.Percentage)
		if !ok {
			return fmt.Errorf("%w: ownership percentage is invalid", ErrInvalidLegalEntity)
		}
		total.Add(total, fraction)
		for _, other := range owners[value.OwnerReference] {
			if intervalsOverlap(value.EffectiveFrom, value.EffectiveTo, other.EffectiveFrom, other.EffectiveTo) {
				return fmt.Errorf("%w: overlapping ownership interest", ErrLegalEntityDuplicate)
			}
		}
		owners[value.OwnerReference] = append(owners[value.OwnerReference], value)
	}
	if total.Cmp(big.NewRat(100, 1)) > 0 {
		return fmt.Errorf("%w: ownership total exceeds 100", ErrInvalidLegalEntity)
	}
	return nil
}

func validateInterval(from time.Time, to *time.Time) error {
	if from.IsZero() {
		return fmt.Errorf("%w: effective from is required", ErrInvalidLegalEntity)
	}
	if to != nil && !to.After(from) {
		return fmt.Errorf("%w: effective interval must end after it starts", ErrInvalidLegalEntity)
	}
	return nil
}
func intervalsOverlap(from time.Time, to *time.Time, otherFrom time.Time, otherTo *time.Time) bool {
	left := from
	right := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	if to != nil {
		right = *to
	}
	otherRight := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	if otherTo != nil {
		otherRight = *otherTo
	}
	return left.Before(otherRight) && otherFrom.Before(right)
}
func maskIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= 4 {
		return "••••"
	}
	runes := []rune(value)
	return "••••" + string(runes[len(runes)-4:])
}
func dateOnly(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
func ptrDate(value time.Time) *time.Time { result := dateOnly(value); return &result }
func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
func cloneApproval(value *ApprovalDecisionReference) *ApprovalDecisionReference {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
func cloneRevisions(values []LegalEntityRevision) []LegalEntityRevision {
	result := make([]LegalEntityRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.Registrations = append([]LegalEntityRegistration(nil), values[index].Snapshot.Registrations...)
		result[index].Snapshot.Addresses = append([]LegalEntityAddress(nil), values[index].Snapshot.Addresses...)
		result[index].Snapshot.OwnershipInterests = append([]LegalEntityOwnershipInterest(nil), values[index].Snapshot.OwnershipInterests...)
		result[index].Snapshot.EffectiveTo = cloneTime(values[index].Snapshot.EffectiveTo)
		result[index].Snapshot.Approval = cloneApproval(values[index].Snapshot.Approval)
	}
	return result
}
func cloneLegalEntity(value LegalEntity) LegalEntity {
	value.Registrations = append([]LegalEntityRegistration(nil), value.Registrations...)
	value.Addresses = append([]LegalEntityAddress(nil), value.Addresses...)
	value.OwnershipInterests = append([]LegalEntityOwnershipInterest(nil), value.OwnershipInterests...)
	value.EffectiveTo = cloneTime(value.EffectiveTo)
	value.Approval = cloneApproval(value.Approval)
	value.Revisions = cloneRevisions(value.Revisions)
	return value
}
func sortLegalEntities(values []LegalEntity) {
	sort.Slice(values, func(left, right int) bool { return values[left].ID.String() < values[right].ID.String() })
}
