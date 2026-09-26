package identity

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AuthorizationOutcome is deliberately closed: callers must not infer an
// authorization result from an error string or from a missing policy.
type AuthorizationOutcome string

const (
	AuthorizationAllowed     AuthorizationOutcome = "allowed"
	AuthorizationDenied      AuthorizationOutcome = "denied"
	AuthorizationExpired     AuthorizationOutcome = "expired"
	AuthorizationUnavailable AuthorizationOutcome = "unavailable"
	AuthorizationStale       AuthorizationOutcome = "stale"
)

const (
	AuthorizationReasonAllowed             = "allowed"
	AuthorizationReasonDefaultDeny         = "default-deny"
	AuthorizationReasonMissingDimension    = "missing-required-dimension"
	AuthorizationReasonScopeMismatch       = "scope-mismatch"
	AuthorizationReasonPolicyExpired       = "policy-expired"
	AuthorizationReasonPolicyUnavailable   = "policy-unavailable"
	AuthorizationReasonPolicyVersionStale  = "policy-version-stale"
	AuthorizationReasonPolicyNotApplicable = "policy-not-applicable"
)

const (
	AuthorizationDimensionAction             = "action"
	AuthorizationDimensionActor              = "actor"
	AuthorizationDimensionAccountingScope    = "accounting-scope"
	AuthorizationDimensionLegalEntity        = "legal-entity"
	AuthorizationDimensionSegment            = "segment-business-unit"
	AuthorizationDimensionAccount            = "account"
	AuthorizationDimensionAccountClass       = "account-class"
	AuthorizationDimensionTransactionType    = "transaction-type"
	AuthorizationDimensionAmount             = "amount"
	AuthorizationDimensionCurrency           = "currency"
	AuthorizationDimensionFiscalPeriod       = "fiscal-period"
	AuthorizationDimensionDataClassification = "data-classification"
	AuthorizationDimensionSubjectActor       = "subject-actor"
	AuthorizationDimensionScope              = "scope"
)

var (
	ErrInvalidAuthorizationInput = errors.New("invalid authorization decision input")
	ErrInvalidAccessPolicy       = errors.New("invalid identity access policy")
	ErrAccessPolicyRevision      = errors.New("identity access policy revision is immutable")
	ErrPolicyUnavailable         = errors.New("identity access policy dependency unavailable")
	ErrAuthorizationExpired      = errors.New("identity authorization policy expired")
	ErrAuthorizationStale        = errors.New("identity authorization policy is stale")
	ErrAuthorizationUnavailable  = errors.New("identity authorization policy is unavailable")
)

// DecisionInput is the single typed input contract used by identity-owned
// authorization evaluation. Scope IDs remain opaque to identity.
type DecisionInput struct {
	ActorID               uuid.UUID
	RoleIDs               []uuid.UUID
	Permission            string
	RequestedScopeIDs     []string
	AccountingScopeID     *uuid.UUID
	LegalEntityID         *uuid.UUID
	SegmentIDs            []uuid.UUID
	AccountID             *uuid.UUID
	AccountClass          *string
	TransactionType       *string
	Amount                *decimal.Decimal
	Currency              *string
	FiscalPeriodID        *uuid.UUID
	DataClassification    string
	SubjectActorIDs       []uuid.UUID
	ExpectedPolicyVersion string
}

func (input DecisionInput) Validate() error {
	if input.ActorID == uuid.Nil {
		return fmt.Errorf("%w: actor is required", ErrInvalidAuthorizationInput)
	}
	if strings.TrimSpace(input.Permission) == "" {
		return fmt.Errorf("%w: permission is required", ErrInvalidAuthorizationInput)
	}
	if input.Amount != nil && input.Amount.IsNegative() {
		return fmt.Errorf("%w: amount cannot be negative", ErrInvalidAuthorizationInput)
	}
	if input.Currency != nil {
		if value := strings.TrimSpace(*input.Currency); value == "" {
			return fmt.Errorf("%w: currency cannot be blank", ErrInvalidAuthorizationInput)
		}
	}
	for _, scopeID := range input.RequestedScopeIDs {
		if strings.TrimSpace(scopeID) == "" {
			return fmt.Errorf("%w: requested scope cannot be blank", ErrInvalidAuthorizationInput)
		}
	}
	return nil
}

// AccessRule is one conjunctive rule. Every non-empty constraint on a rule
// must match; an omitted input never satisfies a constrained dimension.
type AccessRule struct {
	ScopeIDs            []string
	AccountingScopeIDs  []uuid.UUID
	LegalEntityIDs      []uuid.UUID
	SegmentIDs          []uuid.UUID
	AccountIDs          []uuid.UUID
	AccountClasses      []string
	TransactionTypes    []string
	MinAmount           *decimal.Decimal
	MaxAmount           *decimal.Decimal
	Currencies          []string
	FiscalPeriodIDs     []uuid.UUID
	DataClassifications []string
	SubjectActorIDs     []uuid.UUID
}

func (rule AccessRule) Validate() error {
	if rule.MinAmount != nil && rule.MinAmount.IsNegative() {
		return fmt.Errorf("%w: minimum amount cannot be negative", ErrInvalidAccessPolicy)
	}
	if rule.MaxAmount != nil && rule.MaxAmount.IsNegative() {
		return fmt.Errorf("%w: maximum amount cannot be negative", ErrInvalidAccessPolicy)
	}
	if rule.MinAmount != nil && rule.MaxAmount != nil && rule.MinAmount.GreaterThan(*rule.MaxAmount) {
		return fmt.Errorf("%w: minimum amount exceeds maximum amount", ErrInvalidAccessPolicy)
	}
	for _, value := range append(append(append(append(append([]uuid.UUID{}, rule.AccountingScopeIDs...), rule.LegalEntityIDs...), rule.SegmentIDs...), rule.AccountIDs...), rule.FiscalPeriodIDs...) {
		if value == uuid.Nil {
			return fmt.Errorf("%w: constrained identity reference cannot be nil", ErrInvalidAccessPolicy)
		}
	}
	for _, value := range append(append(append(append([]string{}, rule.ScopeIDs...), rule.AccountClasses...), rule.TransactionTypes...), rule.Currencies...) {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: constrained value cannot be blank", ErrInvalidAccessPolicy)
		}
	}
	for _, value := range rule.DataClassifications {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: data classification cannot be blank", ErrInvalidAccessPolicy)
		}
	}
	return nil
}

// AccessPolicy is an immutable policy revision. A later revision is a new
// value with the same ID and a different Version; callers must never mutate a
// revision already returned by a policy store.
type AccessPolicy struct {
	ID              uuid.UUID
	Version         string
	Status          string
	SubjectActorIDs []uuid.UUID
	SubjectRoleIDs  []uuid.UUID
	Permissions     []string
	EffectiveFrom   time.Time
	EffectiveTo     *time.Time
	Rules           []AccessRule
}

const (
	AccessPolicyStatusActive  = "active"
	AccessPolicyStatusRetired = "retired"
)

func (policy AccessPolicy) Validate() error {
	if policy.ID == uuid.Nil {
		return fmt.Errorf("%w: policy id is required", ErrInvalidAccessPolicy)
	}
	if strings.TrimSpace(policy.Version) == "" {
		return fmt.Errorf("%w: policy version is required", ErrInvalidAccessPolicy)
	}
	if policy.Status != AccessPolicyStatusActive && policy.Status != AccessPolicyStatusRetired {
		return fmt.Errorf("%w: unsupported policy status", ErrInvalidAccessPolicy)
	}
	if policy.EffectiveFrom.IsZero() {
		return fmt.Errorf("%w: effective-from is required", ErrInvalidAccessPolicy)
	}
	if policy.EffectiveTo != nil && !policy.EffectiveTo.After(policy.EffectiveFrom) {
		return fmt.Errorf("%w: effective-to must be after effective-from", ErrInvalidAccessPolicy)
	}
	if len(policy.Permissions) == 0 {
		return fmt.Errorf("%w: at least one permission is required", ErrInvalidAccessPolicy)
	}
	for _, permission := range policy.Permissions {
		if strings.TrimSpace(permission) == "" {
			return fmt.Errorf("%w: permission cannot be blank", ErrInvalidAccessPolicy)
		}
	}
	if len(policy.Rules) == 0 {
		return fmt.Errorf("%w: at least one rule is required", ErrInvalidAccessPolicy)
	}
	for _, rule := range policy.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type FieldAccessInput struct {
	DecisionInput
	FieldClassification string
	FieldAction         string
}

type FieldAccessDecision struct {
	Reveal               bool
	Export               bool
	Outcome              AuthorizationOutcome
	PolicyReference      string
	PolicyVersion        string
	DecisionReference    uuid.UUID
	ReasonCode           string
	ApplicableDimensions []string
}

// AccessPolicyStore is intentionally read-only for the evaluator. Policy
// administration and revision creation belong to a later story.
type AccessPolicyStore interface {
	List(context.Context) ([]AccessPolicy, error)
}

type AuthorizationEvaluator interface {
	Evaluate(context.Context, DecisionInput) (AuthorizationDecision, error)
	EvaluateField(context.Context, FieldAccessInput) (FieldAccessDecision, error)
}

type AuthorizationDecisionObserver func(context.Context, AuthorizationDecision, error)

type PolicyEvaluator struct {
	store    AccessPolicyStore
	clock    func() time.Time
	observer AuthorizationDecisionObserver
}

func NewPolicyEvaluator(store AccessPolicyStore, clock func() time.Time, observers ...AuthorizationDecisionObserver) (*PolicyEvaluator, error) {
	if store == nil {
		return nil, ErrPolicyUnavailable
	}
	if clock == nil {
		clock = time.Now
	}
	var observer AuthorizationDecisionObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &PolicyEvaluator{store: store, clock: clock, observer: observer}, nil
}

func (evaluator *PolicyEvaluator) Evaluate(ctx context.Context, input DecisionInput) (decision AuthorizationDecision, err error) {
	if evaluator != nil && evaluator.observer != nil {
		defer func() { evaluator.observer(ctx, cloneAuthorizationDecision(decision), err) }()
	}
	decision = AuthorizationDecision{
		Permission:        strings.TrimSpace(input.Permission),
		DecisionReference: uuid.New(),
		Outcome:           AuthorizationDenied,
		ReasonCode:        AuthorizationReasonDefaultDeny,
	}
	if err := input.Validate(); err != nil {
		decision.ReasonCode = AuthorizationReasonDefaultDeny
		return decision, err
	}
	policies, err := evaluator.store.List(ctx)
	if err != nil {
		decision.Outcome = AuthorizationUnavailable
		decision.ReasonCode = AuthorizationReasonPolicyUnavailable
		return decision, nil
	}
	now := evaluator.clock().UTC()
	matchedPermission := false
	matchedSubject := false
	matchedCurrentVersion := false
	matchedExpired := false
	for _, policy := range policies {
		if !containsString(policy.Permissions, input.Permission) {
			continue
		}
		matchedPermission = true
		if !subjectMatches(policy, input) {
			continue
		}
		matchedSubject = true
		if input.ExpectedPolicyVersion != "" && policy.Version == input.ExpectedPolicyVersion {
			matchedCurrentVersion = true
		}
		if input.ExpectedPolicyVersion != "" && policy.Version != input.ExpectedPolicyVersion {
			if decision.PolicyReference == "" {
				decision.PolicyReference = policy.ID.String()
				decision.PolicyVersion = policy.Version
			}
			continue
		}
		if policy.Status != AccessPolicyStatusActive || now.Before(policy.EffectiveFrom) || (policy.EffectiveTo != nil && !now.Before(*policy.EffectiveTo)) {
			matchedExpired = true
			if decision.PolicyReference == "" {
				decision.PolicyReference = policy.ID.String()
				decision.PolicyVersion = policy.Version
			}
			continue
		}
		for _, rule := range policy.Rules {
			matches, dimensions, reason := rule.matches(input)
			if !matches {
				if len(decision.ApplicableDimensions) == 0 {
					decision.ApplicableDimensions = dimensions
					decision.ReasonCode = reason
				}
				continue
			}
			decision.Allowed = true
			decision.Outcome = AuthorizationAllowed
			decision.PolicyReference = policy.ID.String()
			decision.PolicyVersion = policy.Version
			if len(rule.ScopeIDs) == 0 {
				decision.ApprovedScopeIDs = []string{"*"}
			} else {
				decision.ApprovedScopeIDs = append([]string(nil), rule.ScopeIDs...)
			}
			decision.ApplicableDimensions = dimensions
			decision.ReasonCode = AuthorizationReasonAllowed
			decision.Reason = AuthorizationReasonAllowed
			return decision, nil
		}
	}
	if input.ExpectedPolicyVersion != "" && matchedSubject && matchedPermission && !matchedCurrentVersion {
		decision.Outcome = AuthorizationStale
		decision.ReasonCode = AuthorizationReasonPolicyVersionStale
		return decision, nil
	}
	if matchedExpired {
		decision.Outcome = AuthorizationExpired
		decision.ReasonCode = AuthorizationReasonPolicyExpired
		return decision, nil
	}
	if matchedSubject && matchedPermission {
		decision.ReasonCode = AuthorizationReasonPolicyNotApplicable
	}
	return decision, nil
}

func (evaluator *PolicyEvaluator) EvaluateField(ctx context.Context, input FieldAccessInput) (FieldAccessDecision, error) {
	if strings.TrimSpace(input.FieldClassification) == "" || strings.TrimSpace(input.FieldAction) == "" {
		return FieldAccessDecision{Outcome: AuthorizationDenied, DecisionReference: uuid.New(), ReasonCode: AuthorizationReasonDefaultDeny}, fmt.Errorf("%w: field classification and action are required", ErrInvalidAuthorizationInput)
	}
	input.FieldAction = strings.ToLower(strings.TrimSpace(input.FieldAction))
	input.DecisionInput.Permission = input.FieldAction
	input.DecisionInput.DataClassification = strings.TrimSpace(input.FieldClassification)
	decision, err := evaluator.Evaluate(ctx, input.DecisionInput)
	return FieldAccessDecision{
		Reveal:               decision.Allowed && input.FieldAction == "reveal",
		Export:               decision.Allowed && input.FieldAction == "export",
		Outcome:              decision.Outcome,
		PolicyReference:      decision.PolicyReference,
		PolicyVersion:        decision.PolicyVersion,
		DecisionReference:    decision.DecisionReference,
		ReasonCode:           decision.ReasonCode,
		ApplicableDimensions: append([]string(nil), decision.ApplicableDimensions...),
	}, err
}

func (rule AccessRule) matches(input DecisionInput) (bool, []string, string) {
	dimensions := make([]string, 0, 12)
	match := func(ok bool, dimension string) bool {
		dimensions = append(dimensions, dimension)
		return ok
	}
	if len(rule.ScopeIDs) > 0 && !match(scopesMatch(rule.ScopeIDs, input.RequestedScopeIDs), AuthorizationDimensionScope) {
		return false, dimensions, AuthorizationReasonScopeMismatch
	}
	if len(rule.AccountingScopeIDs) > 0 && !match(uuidPointerIn(rule.AccountingScopeIDs, input.AccountingScopeID), AuthorizationDimensionAccountingScope) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(rule.LegalEntityIDs) > 0 && !match(uuidPointerIn(rule.LegalEntityIDs, input.LegalEntityID), AuthorizationDimensionLegalEntity) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(rule.SegmentIDs) > 0 && !match(uuidSetIn(rule.SegmentIDs, input.SegmentIDs), AuthorizationDimensionSegment) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(rule.AccountIDs) > 0 && !match(uuidPointerIn(rule.AccountIDs, input.AccountID), AuthorizationDimensionAccount) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(rule.AccountClasses) > 0 && !match(stringPointerIn(rule.AccountClasses, input.AccountClass), AuthorizationDimensionAccountClass) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(rule.TransactionTypes) > 0 && !match(stringPointerIn(rule.TransactionTypes, input.TransactionType), AuthorizationDimensionTransactionType) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if rule.MinAmount != nil || rule.MaxAmount != nil {
		amountMatches := input.Amount != nil && (rule.MinAmount == nil || !input.Amount.LessThan(*rule.MinAmount)) && (rule.MaxAmount == nil || !input.Amount.GreaterThan(*rule.MaxAmount))
		if !match(amountMatches, AuthorizationDimensionAmount) {
			return false, dimensions, AuthorizationReasonMissingDimension
		}
	}
	if len(rule.Currencies) > 0 && !match(stringPointerInFold(rule.Currencies, input.Currency), AuthorizationDimensionCurrency) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(rule.FiscalPeriodIDs) > 0 && !match(uuidPointerIn(rule.FiscalPeriodIDs, input.FiscalPeriodID), AuthorizationDimensionFiscalPeriod) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(rule.DataClassifications) > 0 && !match(stringIn(rule.DataClassifications, input.DataClassification), AuthorizationDimensionDataClassification) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(rule.SubjectActorIDs) > 0 && !match(uuidSetIn(rule.SubjectActorIDs, input.SubjectActorIDs), AuthorizationDimensionSubjectActor) {
		return false, dimensions, AuthorizationReasonMissingDimension
	}
	if len(dimensions) == 0 {
		dimensions = []string{AuthorizationDimensionAction, AuthorizationDimensionActor}
	}
	sort.Strings(dimensions)
	return true, dimensions, AuthorizationReasonAllowed
}

func subjectMatches(policy AccessPolicy, input DecisionInput) bool {
	if len(policy.SubjectActorIDs) > 0 && !containsUUID(policy.SubjectActorIDs, input.ActorID) {
		return false
	}
	if len(policy.SubjectRoleIDs) > 0 && !uuidSetIn(policy.SubjectRoleIDs, input.RoleIDs) {
		return false
	}
	return true
}

func scopesMatch(allowed, requested []string) bool {
	if containsString(allowed, "*") {
		return true
	}
	if len(requested) == 0 {
		return false
	}
	for _, requestedScope := range requested {
		if !containsString(allowed, requestedScope) {
			return false
		}
	}
	return true
}

func uuidPointerIn(allowed []uuid.UUID, value *uuid.UUID) bool {
	return value != nil && containsUUID(allowed, *value)
}

func uuidSetIn(allowed, values []uuid.UUID) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !containsUUID(allowed, value) {
			return false
		}
	}
	return true
}

func stringPointerIn(allowed []string, value *string) bool {
	return value != nil && stringIn(allowed, *value)
}

func stringPointerInFold(allowed []string, value *string) bool {
	if value == nil {
		return false
	}
	for _, candidate := range allowed {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(*value)) {
			return true
		}
	}
	return false
}

func stringIn(allowed []string, value string) bool {
	return containsString(allowed, value)
}

func containsString(values []string, wanted string) bool {
	wanted = strings.TrimSpace(wanted)
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}

func containsUUID(values []uuid.UUID, wanted uuid.UUID) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

type MemoryAccessPolicyStore struct {
	mu       sync.RWMutex
	policies map[string]AccessPolicy
	err      error
}

func NewMemoryAccessPolicyStore(policies ...AccessPolicy) (*MemoryAccessPolicyStore, error) {
	store := &MemoryAccessPolicyStore{policies: make(map[string]AccessPolicy)}
	for _, policy := range policies {
		if err := store.Add(policy); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (store *MemoryAccessPolicyStore) Add(policy AccessPolicy) error {
	if store == nil {
		return ErrPolicyUnavailable
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	key := policy.ID.String() + "|" + policy.Version
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.policies[key]; exists {
		return ErrAccessPolicyRevision
	}
	store.policies[key] = cloneAccessPolicy(policy)
	return nil
}

func (store *MemoryAccessPolicyStore) SetError(err error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.err = err
}

func (store *MemoryAccessPolicyStore) List(_ context.Context) ([]AccessPolicy, error) {
	if store == nil {
		return nil, ErrPolicyUnavailable
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	if store.err != nil {
		return nil, store.err
	}
	result := make([]AccessPolicy, 0, len(store.policies))
	for _, policy := range store.policies {
		result = append(result, cloneAccessPolicy(policy))
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].ID.String()+result[left].Version < result[right].ID.String()+result[right].Version
	})
	return result, nil
}

func cloneAccessPolicy(policy AccessPolicy) AccessPolicy {
	if policy.EffectiveTo != nil {
		value := *policy.EffectiveTo
		policy.EffectiveTo = &value
	}
	policy.SubjectActorIDs = append([]uuid.UUID(nil), policy.SubjectActorIDs...)
	policy.SubjectRoleIDs = append([]uuid.UUID(nil), policy.SubjectRoleIDs...)
	policy.Permissions = append([]string(nil), policy.Permissions...)
	policy.Rules = append([]AccessRule(nil), policy.Rules...)
	for index := range policy.Rules {
		policy.Rules[index].ScopeIDs = append([]string(nil), policy.Rules[index].ScopeIDs...)
		policy.Rules[index].AccountingScopeIDs = append([]uuid.UUID(nil), policy.Rules[index].AccountingScopeIDs...)
		policy.Rules[index].LegalEntityIDs = append([]uuid.UUID(nil), policy.Rules[index].LegalEntityIDs...)
		policy.Rules[index].SegmentIDs = append([]uuid.UUID(nil), policy.Rules[index].SegmentIDs...)
		policy.Rules[index].AccountIDs = append([]uuid.UUID(nil), policy.Rules[index].AccountIDs...)
		policy.Rules[index].AccountClasses = append([]string(nil), policy.Rules[index].AccountClasses...)
		policy.Rules[index].TransactionTypes = append([]string(nil), policy.Rules[index].TransactionTypes...)
		policy.Rules[index].Currencies = append([]string(nil), policy.Rules[index].Currencies...)
		policy.Rules[index].FiscalPeriodIDs = append([]uuid.UUID(nil), policy.Rules[index].FiscalPeriodIDs...)
		policy.Rules[index].DataClassifications = append([]string(nil), policy.Rules[index].DataClassifications...)
		policy.Rules[index].SubjectActorIDs = append([]uuid.UUID(nil), policy.Rules[index].SubjectActorIDs...)
		if policy.Rules[index].MinAmount != nil {
			value := policy.Rules[index].MinAmount.Copy()
			policy.Rules[index].MinAmount = &value
		}
		if policy.Rules[index].MaxAmount != nil {
			value := policy.Rules[index].MaxAmount.Copy()
			policy.Rules[index].MaxAmount = &value
		}
	}
	return policy
}
