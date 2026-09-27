package organization

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
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

// PartyManagementPermission is the OMD permission for the private party
// mutation surface. Party read APIs are deliberately not introduced by this
// slice; safe projections are returned only from an accepted command.
const PartyManagementPermission = "finance.omd.maintain.parties"

const (
	PartyActionCreate   = "create"
	PartyActionMaintain = "maintain"

	PartyBankControlNotRequired = "not-required"
	PartyBankControlApproved    = "approved"
	PartyBankControlPending     = "pending"
	PartyBankControlRejected    = "rejected"
	PartyBankControlStale       = "stale"
	PartyBankControlUnavailable = "unavailable"
)

var (
	ErrInvalidParty                       = errors.New("invalid party")
	ErrInvalidPartyCommand                = errors.New("invalid party command")
	ErrPartyNotFound                      = errors.New("party not found")
	ErrPartyVersionConflict               = errors.New("party version conflict")
	ErrPartyAuthorizationDenied           = errors.New("party authorization denied")
	ErrPartyAuthorizationUnavailable      = errors.New("party authorization unavailable")
	ErrPartyAuthorizationStale            = errors.New("party authorization stale")
	ErrPartyFieldAuthorizationDenied      = errors.New("party field authorization denied")
	ErrPartyFieldAuthorizationUnavailable = errors.New("party field authorization unavailable")
	ErrPartyIdempotencyConflict           = errors.New("party idempotency conflict")
	ErrPartyCommandInProgress             = errors.New("party command is already in progress")
	ErrPartyAuditUnavailable              = errors.New("party audit unavailable")
	ErrPartyBankReferenceInvalid          = errors.New("party bank reference is invalid")
	ErrPartyBankReferenceUnavailable      = errors.New("party bank-reference validation is unavailable")
	ErrPartyBankControlUnavailable        = errors.New("party bank control is unavailable")
	ErrPartyBankApprovalRejected          = errors.New("party bank change approval rejected")
	ErrPartyBankApprovalStale             = errors.New("party bank change approval is stale")
	ErrPartyDuplicate                     = errors.New("duplicate party value")
	ErrInvalidPartyService                = errors.New("invalid party service")
	ErrPartyDurableCommandFailed          = errors.New("party command previously failed")
)

var partyCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var partyCountryPattern = regexp.MustCompile(`^[A-Z]{2,3}$`)
var partyOpaqueReferencePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,159}$`)

// PartyType and PartyStatus intentionally remain open canonical codes. The
// approved Party contract does not define a closed vocabulary, so OMD must
// not invent one in the domain model.
type PartyType string
type PartyStatus string

type PartyContactMethod struct {
	ID    uuid.UUID `json:"id"`
	Type  string    `json:"type"`
	Value string    `json:"value"`
	Label string    `json:"label,omitempty"`
}

type PartyAddress struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Line1       string    `json:"line1"`
	Line2       string    `json:"line2,omitempty"`
	Locality    string    `json:"locality"`
	Region      string    `json:"region,omitempty"`
	PostalCode  string    `json:"postalCode"`
	CountryCode string    `json:"countryCode"`
}

type PartyClassification struct {
	ID    uuid.UUID `json:"id"`
	Code  string    `json:"code"`
	Value string    `json:"value"`
}

// PartyBankDetailReference contains only an approved opaque reference. It is
// deliberately impossible to represent an account number, credential, or
// provider token in this type.
type PartyBankDetailReference struct {
	ID               uuid.UUID `json:"id"`
	Reference        string    `json:"reference"`
	ProviderCode     string    `json:"providerCode,omitempty"`
	ConsentReference string    `json:"consentReference,omitempty"`
}

type BankDetailControlState struct {
	Status            string     `json:"status"`
	CoolingOffUntil   *time.Time `json:"coolingOffUntil,omitempty"`
	DecisionReference uuid.UUID  `json:"decisionReference,omitempty"`
}

func (state BankDetailControlState) Validate() error {
	switch state.Status {
	case PartyBankControlNotRequired, PartyBankControlApproved, PartyBankControlPending,
		PartyBankControlRejected, PartyBankControlStale, PartyBankControlUnavailable:
		return nil
	default:
		return fmt.Errorf("%w: unsupported bank-control status", ErrInvalidParty)
	}
}

type Party struct {
	ID                   uuid.UUID                         `json:"id"`
	ScopeID              uuid.UUID                         `json:"scopeId"`
	Name                 string                            `json:"name"`
	PartyType            PartyType                         `json:"partyType"`
	Status               PartyStatus                       `json:"status"`
	TaxIdentifier        string                            `json:"-"`
	ContactMethods       []PartyContactMethod              `json:"contactMethods"`
	Addresses            []PartyAddress                    `json:"addresses"`
	Classifications      []PartyClassification             `json:"classifications"`
	BankDetailReferences []PartyBankDetailReference        `json:"bankDetailReferences"`
	BankControl          BankDetailControlState            `json:"bankControl"`
	Version              aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber       int64                             `json:"revisionNumber"`
	CreatedAt            time.Time                         `json:"createdAt"`
	UpdatedAt            time.Time                         `json:"updatedAt"`
	Revisions            []PartyRevision                   `json:"-"`
}

type PartyRevision struct {
	RevisionNumber int64                             `json:"revisionNumber"`
	Version        aggregateversion.AggregateVersion `json:"version"`
	Snapshot       PartySnapshot                     `json:"snapshot"`
	CreatedAt      time.Time                         `json:"createdAt"`
}

// PartySnapshot is an internal OMD history value. It may contain the
// restricted tax identifier, but is never part of a safe projection or API
// response.
type PartySnapshot struct {
	Name                 string                     `json:"name"`
	PartyType            PartyType                  `json:"partyType"`
	Status               PartyStatus                `json:"status"`
	TaxIdentifier        string                     `json:"taxIdentifier,omitempty"`
	ContactMethods       []PartyContactMethod       `json:"contactMethods"`
	Addresses            []PartyAddress             `json:"addresses"`
	Classifications      []PartyClassification      `json:"classifications"`
	BankDetailReferences []PartyBankDetailReference `json:"bankDetailReferences"`
	BankControl          BankDetailControlState     `json:"bankControl"`
}

type SafePartyBankDetailReference struct {
	ID           uuid.UUID `json:"id"`
	Reference    string    `json:"reference"`
	ProviderCode string    `json:"providerCode,omitempty"`
	Status       string    `json:"status"`
}

// SafeParty is the only Party value intended for an unrestricted command
// response. Restricted tax data is represented only by a mask and bank data
// is represented by opaque provider references plus bounded control state.
type SafeParty struct {
	ID                   uuid.UUID                         `json:"id"`
	ScopeID              uuid.UUID                         `json:"scopeId"`
	Name                 string                            `json:"name,omitempty"`
	PartyType            PartyType                         `json:"partyType,omitempty"`
	Status               PartyStatus                       `json:"status"`
	TaxIdentifierMasked  string                            `json:"taxIdentifierMasked,omitempty"`
	ContactMethods       []PartyContactMethod              `json:"contactMethods,omitempty"`
	Addresses            []PartyAddress                    `json:"addresses,omitempty"`
	Classifications      []PartyClassification             `json:"classifications,omitempty"`
	BankDetailReferences []SafePartyBankDetailReference    `json:"bankDetailReferences,omitempty"`
	BankControl          BankDetailControlState            `json:"bankControl"`
	Version              aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber       int64                             `json:"revisionNumber"`
}

// PartySafeProjection returns the broad safe projection used by internal
// callers. Services should prefer SafeProjectionWithAccess when a field-level
// decision is available.
func (party Party) SafeProjection() SafeParty {
	return party.SafeProjectionWithAccess(PartyFieldAuthorization{Identity: true, RestrictedTaxIdentifier: true, PersonalData: true, Classifications: true, BankDetailReferences: true})
}

func (party Party) SafeProjectionWithAccess(access PartyFieldAuthorization) SafeParty {
	result := SafeParty{
		ID: party.ID, ScopeID: party.ScopeID, Status: party.Status, BankControl: cloneBankControl(party.BankControl),
		Version: party.Version, RevisionNumber: party.RevisionNumber,
	}
	if access.Identity {
		result.Name, result.PartyType = party.Name, party.PartyType
	}
	if access.RestrictedTaxIdentifier && party.TaxIdentifier != "" {
		result.TaxIdentifierMasked = maskPartyIdentifier(party.TaxIdentifier)
	}
	if access.PersonalData {
		result.ContactMethods = clonePartyContacts(party.ContactMethods)
		result.Addresses = clonePartyAddresses(party.Addresses)
	}
	if access.Classifications {
		result.Classifications = clonePartyClassifications(party.Classifications)
	}
	if access.BankDetailReferences {
		result.BankDetailReferences = make([]SafePartyBankDetailReference, 0, len(party.BankDetailReferences))
		for _, reference := range party.BankDetailReferences {
			result.BankDetailReferences = append(result.BankDetailReferences, SafePartyBankDetailReference{
				ID: reference.ID, Reference: reference.Reference, ProviderCode: reference.ProviderCode, Status: party.BankControl.Status,
			})
		}
	}
	return result
}

func (party Party) Snapshot() PartySnapshot {
	return PartySnapshot{
		Name: party.Name, PartyType: party.PartyType, Status: party.Status,
		TaxIdentifier:        party.TaxIdentifier,
		ContactMethods:       clonePartyContacts(party.ContactMethods),
		Addresses:            clonePartyAddresses(party.Addresses),
		Classifications:      clonePartyClassifications(party.Classifications),
		BankDetailReferences: clonePartyBankReferences(party.BankDetailReferences),
		BankControl:          cloneBankControl(party.BankControl),
	}
}

func (party Party) Validate() error {
	if party.ID == uuid.Nil || party.ScopeID == uuid.Nil || party.Version.Value() < 1 || party.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidParty)
	}
	if party.Name != canonicalPartyText(party.Name) || len([]rune(party.Name)) == 0 || len([]rune(party.Name)) > 200 {
		return fmt.Errorf("%w: name is required and must be canonical and at most 200 characters", ErrInvalidParty)
	}
	if err := validatePartyCode(string(party.PartyType), "party type"); err != nil {
		return err
	}
	if err := validatePartyCode(string(party.Status), "party status"); err != nil {
		return err
	}
	if party.TaxIdentifier != "" && (party.TaxIdentifier != strings.TrimSpace(party.TaxIdentifier) || len([]rune(party.TaxIdentifier)) > 160) {
		return fmt.Errorf("%w: restricted tax identifier is invalid", ErrInvalidParty)
	}
	if err := validatePartyContacts(party.ContactMethods); err != nil {
		return err
	}
	if err := validatePartyAddresses(party.Addresses); err != nil {
		return err
	}
	if err := validatePartyClassifications(party.Classifications); err != nil {
		return err
	}
	if err := validatePartyBankReferences(party.BankDetailReferences); err != nil {
		return err
	}
	if err := party.BankControl.Validate(); err != nil {
		return err
	}
	if party.CreatedAt.IsZero() || party.UpdatedAt.IsZero() || party.UpdatedAt.Before(party.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidParty)
	}
	return nil
}

func NewParty(id, scopeID uuid.UUID, name string, partyType PartyType, status PartyStatus, taxIdentifier string, contacts []PartyContactMethod, addresses []PartyAddress, classifications []PartyClassification, bankReferences []PartyBankDetailReference, bankControl BankDetailControlState, now time.Time) (Party, error) {
	party := Party{
		ID: id, ScopeID: scopeID, Name: canonicalPartyText(name), PartyType: PartyType(canonicalPartyCode(string(partyType))), Status: PartyStatus(canonicalPartyCode(string(status))),
		TaxIdentifier: strings.TrimSpace(taxIdentifier), ContactMethods: newPartyContactIDs(contacts), Addresses: newPartyAddressIDs(addresses), Classifications: newPartyClassificationIDs(classifications), BankDetailReferences: newPartyBankReferenceIDs(bankReferences), BankControl: cloneBankControl(bankControl), Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if party.BankControl.Status == "" {
		party.BankControl.Status = PartyBankControlNotRequired
	}
	if err := party.Validate(); err != nil {
		return Party{}, err
	}
	party.Revisions = []PartyRevision{{RevisionNumber: party.RevisionNumber, Version: party.Version, Snapshot: party.Snapshot(), CreatedAt: party.UpdatedAt}}
	return party, nil
}

func (party *Party) Replace(current Party, name string, partyType PartyType, status PartyStatus, taxIdentifier string, contacts []PartyContactMethod, addresses []PartyAddress, classifications []PartyClassification, bankReferences []PartyBankDetailReference, bankControl BankDetailControlState, now time.Time) error {
	if party == nil || current.ID == uuid.Nil || party.ID != current.ID {
		return ErrInvalidParty
	}
	nextVersion, err := current.Version.Advance()
	if err != nil {
		return err
	}
	party.ID, party.ScopeID = current.ID, current.ScopeID
	party.Name, party.PartyType, party.Status = canonicalPartyText(name), PartyType(canonicalPartyCode(string(partyType))), PartyStatus(canonicalPartyCode(string(status)))
	party.TaxIdentifier = strings.TrimSpace(taxIdentifier)
	party.ContactMethods, party.Addresses, party.Classifications, party.BankDetailReferences = newPartyContactIDs(contacts), newPartyAddressIDs(addresses), newPartyClassificationIDs(classifications), newPartyBankReferenceIDs(bankReferences)
	party.BankControl = cloneBankControl(bankControl)
	party.Version, party.RevisionNumber, party.CreatedAt, party.UpdatedAt = nextVersion, current.RevisionNumber+1, current.CreatedAt, now.UTC()
	party.Revisions = append(clonePartyRevisions(current.Revisions), PartyRevision{RevisionNumber: party.RevisionNumber, Version: party.Version, Snapshot: party.Snapshot(), CreatedAt: party.UpdatedAt})
	return party.Validate()
}

func validatePartyCode(value, label string) error {
	canonical := canonicalPartyCode(value)
	if canonical != value || !partyCodePattern.MatchString(value) {
		return fmt.Errorf("%w: %s must be a canonical non-empty code", ErrInvalidParty, label)
	}
	return nil
}

func validatePartyContacts(values []PartyContactMethod) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ID == uuid.Nil || value.Type != canonicalPartyCode(value.Type) || !partyCodePattern.MatchString(value.Type) || strings.TrimSpace(value.Value) == "" || value.Value != strings.TrimSpace(value.Value) || len([]rune(value.Value)) > 320 || value.Label != canonicalPartyText(value.Label) {
			return fmt.Errorf("%w: contact method is invalid", ErrInvalidParty)
		}
		key := strings.ToLower(value.Type) + "\x00" + strings.ToLower(value.Value)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: duplicate contact method", ErrPartyDuplicate)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validatePartyAddresses(values []PartyAddress) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ID == uuid.Nil || value.Type != canonicalPartyCode(value.Type) || !partyCodePattern.MatchString(value.Type) || value.Line1 != canonicalPartyText(value.Line1) || value.Line2 != canonicalPartyText(value.Line2) || value.Locality != canonicalPartyText(value.Locality) || value.Region != canonicalPartyText(value.Region) || value.PostalCode != canonicalPartyText(value.PostalCode) || strings.TrimSpace(value.Line1) == "" || strings.TrimSpace(value.Locality) == "" || strings.TrimSpace(value.PostalCode) == "" || value.CountryCode != strings.ToUpper(strings.TrimSpace(value.CountryCode)) || !partyCountryPattern.MatchString(value.CountryCode) {
			return fmt.Errorf("%w: address is invalid", ErrInvalidParty)
		}
		key := strings.ToLower(value.Type) + "\x00" + strings.ToLower(canonicalPartyText(value.Line1)) + "\x00" + strings.ToUpper(strings.TrimSpace(value.CountryCode)) + "\x00" + strings.ToLower(canonicalPartyText(value.PostalCode))
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: duplicate address", ErrPartyDuplicate)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validatePartyClassifications(values []PartyClassification) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ID == uuid.Nil || value.Code != canonicalPartyCode(value.Code) || !partyCodePattern.MatchString(value.Code) || strings.TrimSpace(value.Value) == "" || value.Value != strings.TrimSpace(value.Value) || value.Value != canonicalPartyText(value.Value) {
			return fmt.Errorf("%w: classification is invalid", ErrInvalidParty)
		}
		key := strings.ToLower(value.Code) + "\x00" + strings.ToLower(value.Value)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: duplicate classification", ErrPartyDuplicate)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validatePartyBankReferences(values []PartyBankDetailReference) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value.ID == uuid.Nil {
			return fmt.Errorf("%w: bank reference identifier is required", ErrInvalidParty)
		}
		if err := validateOpaquePartyReference(value.Reference, "bank reference"); err != nil {
			return err
		}
		if value.ProviderCode != "" && (value.ProviderCode != canonicalPartyCode(value.ProviderCode) || !partyCodePattern.MatchString(value.ProviderCode) || containsPartySensitiveWord(value.ProviderCode)) {
			return fmt.Errorf("%w: bank provider code is invalid", ErrPartyBankReferenceInvalid)
		}
		if value.ConsentReference != "" {
			if err := validateOpaquePartyReference(value.ConsentReference, "bank consent reference"); err != nil {
				return err
			}
		}
		key := strings.ToLower(value.Reference)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: duplicate bank reference", ErrPartyDuplicate)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateOpaquePartyReference(value, label string) error {
	if !partyOpaqueReferencePattern.MatchString(value) || numericPartyReference(value) || containsPartySensitiveWord(value) {
		return fmt.Errorf("%w: %s must be an opaque approved reference", ErrPartyBankReferenceInvalid, label)
	}
	return nil
}

func numericPartyReference(value string) bool {
	digitCount := 0
	for _, runeValue := range value {
		switch runeValue {
		case '.', '-', '_', ':':
			continue
		default:
			if runeValue < '0' || runeValue > '9' {
				return false
			}
			digitCount++
		}
	}
	return digitCount > 0
}

func containsPartySensitiveWord(value string) bool {
	value = strings.ToLower(value)
	for _, word := range []string{"account", "accountnumber", "account_number", "iban", "routing", "swift", "credential", "password", "provider-token", "providertoken", "secret", "token", "raw-account"} {
		if strings.Contains(value, word) {
			return true
		}
	}
	return false
}

func canonicalPartyText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
func canonicalPartyCode(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func maskPartyIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 4 {
		return "••••"
	}
	return "••••" + string(runes[len(runes)-4:])
}

func FingerprintParty(party Party) string {
	data, err := json.Marshal(party.Snapshot())
	if err != nil {
		return "sha256:invalid"
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneParty(party Party) Party {
	party.ContactMethods = clonePartyContacts(party.ContactMethods)
	party.Addresses = clonePartyAddresses(party.Addresses)
	party.Classifications = clonePartyClassifications(party.Classifications)
	party.BankDetailReferences = clonePartyBankReferences(party.BankDetailReferences)
	party.BankControl = cloneBankControl(party.BankControl)
	party.Revisions = clonePartyRevisions(party.Revisions)
	return party
}

func clonePartyContacts(values []PartyContactMethod) []PartyContactMethod {
	return append([]PartyContactMethod(nil), values...)
}
func clonePartyAddresses(values []PartyAddress) []PartyAddress {
	return append([]PartyAddress(nil), values...)
}
func clonePartyClassifications(values []PartyClassification) []PartyClassification {
	return append([]PartyClassification(nil), values...)
}
func clonePartyBankReferences(values []PartyBankDetailReference) []PartyBankDetailReference {
	return append([]PartyBankDetailReference(nil), values...)
}
func clonePartyRevisions(values []PartyRevision) []PartyRevision {
	result := make([]PartyRevision, 0, len(values))
	for _, value := range values {
		value.Snapshot.ContactMethods = clonePartyContacts(value.Snapshot.ContactMethods)
		value.Snapshot.Addresses = clonePartyAddresses(value.Snapshot.Addresses)
		value.Snapshot.Classifications = clonePartyClassifications(value.Snapshot.Classifications)
		value.Snapshot.BankDetailReferences = clonePartyBankReferences(value.Snapshot.BankDetailReferences)
		value.Snapshot.BankControl = cloneBankControl(value.Snapshot.BankControl)
		result = append(result, value)
	}
	return result
}
func cloneBankControl(value BankDetailControlState) BankDetailControlState {
	if value.CoolingOffUntil != nil {
		copied := value.CoolingOffUntil.UTC()
		value.CoolingOffUntil = &copied
	}
	return value
}
func newPartyContactIDs(values []PartyContactMethod) []PartyContactMethod {
	result := clonePartyContacts(values)
	for index := range result {
		result[index].ID = ensurePartyID(result[index].ID)
		result[index].Type = canonicalPartyCode(result[index].Type)
		result[index].Value = strings.TrimSpace(result[index].Value)
		result[index].Label = canonicalPartyText(result[index].Label)
	}
	return result
}
func newPartyAddressIDs(values []PartyAddress) []PartyAddress {
	result := clonePartyAddresses(values)
	for index := range result {
		result[index].ID = ensurePartyID(result[index].ID)
		result[index].Type = canonicalPartyCode(result[index].Type)
		result[index].Line1 = canonicalPartyText(result[index].Line1)
		result[index].Line2 = canonicalPartyText(result[index].Line2)
		result[index].Locality = canonicalPartyText(result[index].Locality)
		result[index].Region = canonicalPartyText(result[index].Region)
		result[index].PostalCode = canonicalPartyText(result[index].PostalCode)
		result[index].CountryCode = strings.ToUpper(strings.TrimSpace(result[index].CountryCode))
	}
	return result
}
func newPartyClassificationIDs(values []PartyClassification) []PartyClassification {
	result := clonePartyClassifications(values)
	for index := range result {
		result[index].ID = ensurePartyID(result[index].ID)
		result[index].Code = canonicalPartyCode(result[index].Code)
		result[index].Value = strings.TrimSpace(result[index].Value)
	}
	return result
}
func newPartyBankReferenceIDs(values []PartyBankDetailReference) []PartyBankDetailReference {
	result := clonePartyBankReferences(values)
	for index := range result {
		result[index].ID = ensurePartyID(result[index].ID)
		result[index].Reference = strings.TrimSpace(result[index].Reference)
		result[index].ProviderCode = canonicalPartyCode(result[index].ProviderCode)
		result[index].ConsentReference = strings.TrimSpace(result[index].ConsentReference)
	}
	return result
}
func ensurePartyID(value uuid.UUID) uuid.UUID {
	if value == uuid.Nil {
		return uuid.New()
	}
	return value
}

func sortParties(values []Party) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].ID.String() < values[j].ID.String()
	})
}
