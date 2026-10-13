package workflow

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
)

const MaintainApprovalPoliciesPermission = "finance.wfa.maintain.approval.policies"

const (
	ApprovalPolicyActionCreate            = "create"
	ApprovalPolicyActionUpdate            = "update"
	ApprovalPolicyActionMaintain          = "maintain"
	ApprovalPolicyActionSubmitForApproval = "submitForApproval"
	ApprovalPolicyActionActivate          = "activate"
)

const (
	ApprovalPolicyStatusDraft           = "Draft"
	ApprovalPolicyStatusPendingApproval = "PendingApproval"
	ApprovalPolicyStatusActive          = "Active"
	ApprovalPolicyStatusSuperseded      = "Superseded"
)

const (
	ApprovalPolicyActivationOutcomePending   = "pending_approval"
	ApprovalPolicyActivationOutcomeActivated = "activated"
	ApprovalPolicyActivationOutcomeRejected  = "rejected"
	ApprovalPolicyActivationOutcomeNone      = "not_requested"
)

var (
	ErrInvalidApprovalPolicy                  = errors.New("invalid approval policy")
	ErrInvalidApprovalPolicyCommand           = errors.New("invalid approval policy command")
	ErrApprovalPolicyNotFound                 = errors.New("approval policy not found")
	ErrApprovalPolicyVersionConflict          = errors.New("approval policy version conflict")
	ErrApprovalPolicyAuthorizationDenied      = errors.New("approval policy authorization denied")
	ErrApprovalPolicyAuthorizationUnavailable = errors.New("approval policy authorization unavailable")
	ErrApprovalPolicyAuthorizationStale       = errors.New("approval policy authorization stale")
	ErrApprovalPolicyIdempotencyConflict      = errors.New("approval policy idempotency conflict")
	ErrApprovalPolicyCommandInProgress        = errors.New("approval policy command is already in progress")
	ErrApprovalPolicyAuditUnavailable         = errors.New("approval policy audit unavailable")
	ErrApprovalPolicyOverlap                  = errors.New("approval policies have an overlapping selectable interval")
	ErrApprovalPolicyApprovalRequired         = errors.New("approval policy approval is required")
	ErrApprovalPolicyApprovalRejected         = errors.New("approval policy approval was rejected")
	ErrApprovalPolicyApprovalUnavailable      = errors.New("approval policy approval dependency is unavailable")
	ErrApprovalPolicySelfApproval             = errors.New("approval policy cannot be self-approved")
	ErrApprovalPolicyImmutable                = errors.New("established approval policy version is immutable")
	ErrApprovalPolicyDuplicate                = errors.New("duplicate approval policy identity")
	ErrInvalidApprovalPolicyService           = errors.New("invalid approval policy service")
	ErrApprovalPolicyDurableCommandFailed     = errors.New("approval policy command previously failed")
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Actor is deliberately independent from IAM adapters. The workflow module
// receives only the stable user identity and authentication-subject evidence
// needed for authorization and audit.
type Actor struct {
	UserID           uuid.UUID `json:"userId"`
	SubjectReference string    `json:"subjectReference"`
}

func (actor Actor) Validate() error {
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return fmt.Errorf("%w: actor identity and subject reference are required", ErrInvalidApprovalPolicyCommand)
	}
	return nil
}

type ApprovalPolicyScope struct {
	ScopeIDs           []string    `json:"scopeIds"`
	AccountingScopeIDs []uuid.UUID `json:"accountingScopeIds,omitempty"`
	LegalEntityIDs     []uuid.UUID `json:"legalEntityIds,omitempty"`
	BusinessUnitIDs    []uuid.UUID `json:"businessUnitIds,omitempty"`
	SegmentIDs         []uuid.UUID `json:"segmentIds,omitempty"`
	AccountIDs         []uuid.UUID `json:"accountIds,omitempty"`
	FiscalPeriodIDs    []uuid.UUID `json:"fiscalPeriodIds,omitempty"`
}

func (scope ApprovalPolicyScope) Validate() error {
	if len(scope.ScopeIDs) == 0 && len(scope.AccountingScopeIDs) == 0 && len(scope.LegalEntityIDs) == 0 && len(scope.BusinessUnitIDs) == 0 && len(scope.SegmentIDs) == 0 && len(scope.AccountIDs) == 0 && len(scope.FiscalPeriodIDs) == 0 {
		return fmt.Errorf("%w: at least one scope dimension is required", ErrInvalidApprovalPolicy)
	}
	if err := validateUniqueStrings(scope.ScopeIDs, "scope id"); err != nil {
		return err
	}
	for _, value := range scope.ScopeIDs {
		if value == "*" && (len(scope.ScopeIDs) > 1 || len(scope.AccountingScopeIDs) > 0 || len(scope.LegalEntityIDs) > 0 || len(scope.BusinessUnitIDs) > 0 || len(scope.SegmentIDs) > 0 || len(scope.AccountIDs) > 0 || len(scope.FiscalPeriodIDs) > 0) {
			return fmt.Errorf("%w: wildcard scope cannot be combined with narrower scope identifiers", ErrInvalidApprovalPolicy)
		}
	}
	for _, values := range [][]uuid.UUID{scope.AccountingScopeIDs, scope.LegalEntityIDs, scope.BusinessUnitIDs, scope.SegmentIDs, scope.AccountIDs, scope.FiscalPeriodIDs} {
		seen := make(map[uuid.UUID]struct{}, len(values))
		for _, value := range values {
			if value == uuid.Nil {
				return fmt.Errorf("%w: scope identity cannot be nil", ErrInvalidApprovalPolicy)
			}
			if _, ok := seen[value]; ok {
				return fmt.Errorf("%w: duplicate scope identity", ErrInvalidApprovalPolicy)
			}
			seen[value] = struct{}{}
		}
	}
	return nil
}

func (scope ApprovalPolicyScope) CanonicalScopeIDs() []string {
	values := append([]string(nil), scope.ScopeIDs...)
	for _, value := range scope.AccountingScopeIDs {
		values = append(values, value.String())
	}
	for _, value := range scope.LegalEntityIDs {
		values = append(values, value.String())
	}
	for _, value := range scope.BusinessUnitIDs {
		values = append(values, value.String())
	}
	for _, value := range scope.SegmentIDs {
		values = append(values, value.String())
	}
	for _, value := range scope.AccountIDs {
		values = append(values, value.String())
	}
	for _, value := range scope.FiscalPeriodIDs {
		values = append(values, value.String())
	}
	for index := range values {
		values[index] = strings.TrimSpace(values[index])
	}
	sort.Strings(values)
	return values
}

// Contains reports whether the policy scope contains every requested opaque
// scope. A wildcard is an explicit, auditable scope rather than an empty one.
func (scope ApprovalPolicyScope) Contains(requested []string) bool {
	if err := scope.Validate(); err != nil {
		return false
	}
	allowed := make(map[string]struct{}, len(scope.CanonicalScopeIDs()))
	for _, value := range scope.CanonicalScopeIDs() {
		allowed[strings.TrimSpace(value)] = struct{}{}
	}
	if _, ok := allowed["*"]; ok {
		return true
	}
	for _, value := range requested {
		if _, ok := allowed[strings.TrimSpace(value)]; !ok {
			return false
		}
	}
	return true
}

type ApprovalApplicability struct {
	SourceContext       string            `json:"sourceContext"`
	SubjectType         string            `json:"subjectType"`
	Operation           string            `json:"operation"`
	TransactionTypes    []string          `json:"transactionTypes,omitempty"`
	Currencies          []string          `json:"currencies,omitempty"`
	DataClassifications []string          `json:"dataClassifications,omitempty"`
	Dimensions          map[string]string `json:"dimensions,omitempty"`
}

func (applicability ApprovalApplicability) Validate() error {
	for _, value := range map[string]string{
		"source context": applicability.SourceContext,
		"subject type":   applicability.SubjectType,
		"operation":      applicability.Operation,
	} {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			return fmt.Errorf("%w: applicability value is required and must not contain surrounding whitespace", ErrInvalidApprovalPolicy)
		}
	}
	if err := validateUniqueStrings(applicability.TransactionTypes, "transaction type"); err != nil {
		return err
	}
	for _, currency := range applicability.Currencies {
		if !currencyCodePattern.MatchString(currency) {
			return fmt.Errorf("%w: currency must be an uppercase three-letter code", ErrInvalidApprovalPolicy)
		}
	}
	if err := validateUniqueStrings(applicability.Currencies, "currency"); err != nil {
		return err
	}
	if err := validateUniqueStrings(applicability.DataClassifications, "data classification"); err != nil {
		return err
	}
	for key, value := range applicability.Dimensions {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" || strings.TrimSpace(key) != key || strings.TrimSpace(value) != value {
			return fmt.Errorf("%w: applicability dimensions must have non-blank keys and values", ErrInvalidApprovalPolicy)
		}
	}
	return nil
}

type AmountThreshold struct {
	Minimum  *decimal.Decimal `json:"minimum,omitempty"`
	Maximum  *decimal.Decimal `json:"maximum,omitempty"`
	Currency string           `json:"currency"`

	// These aliases keep the domain convenient for callers that already use
	// IAM's MinAmount/MaxAmount vocabulary. They are folded into Minimum and
	// Maximum before validation and fingerprinting.
	MinAmount *decimal.Decimal `json:"-"`
	MaxAmount *decimal.Decimal `json:"-"`
}

func (threshold AmountThreshold) normalized() AmountThreshold {
	if threshold.Minimum == nil {
		threshold.Minimum = threshold.MinAmount
	}
	if threshold.Maximum == nil {
		threshold.Maximum = threshold.MaxAmount
	}
	threshold.Currency = strings.TrimSpace(threshold.Currency)
	return threshold
}

func (threshold AmountThreshold) Validate() error {
	threshold = threshold.normalized()
	if threshold.Minimum == nil && threshold.Maximum == nil {
		return fmt.Errorf("%w: at least one amount threshold is required", ErrInvalidApprovalPolicy)
	}
	if threshold.Minimum != nil && threshold.Minimum.IsNegative() {
		return fmt.Errorf("%w: minimum amount cannot be negative", ErrInvalidApprovalPolicy)
	}
	if threshold.Maximum != nil && threshold.Maximum.IsNegative() {
		return fmt.Errorf("%w: maximum amount cannot be negative", ErrInvalidApprovalPolicy)
	}
	if threshold.Minimum != nil && threshold.Maximum != nil && threshold.Minimum.GreaterThan(*threshold.Maximum) {
		return fmt.Errorf("%w: minimum amount exceeds maximum amount", ErrInvalidApprovalPolicy)
	}
	if !currencyCodePattern.MatchString(threshold.Currency) {
		return fmt.Errorf("%w: threshold currency must be an uppercase three-letter code", ErrInvalidApprovalPolicy)
	}
	return nil
}

type RoleRequirement struct {
	Role             string `json:"role"`
	MinimumApprovals int    `json:"minimumApprovals"`
	AllowDelegation  bool   `json:"allowDelegation"`
}

func (requirement RoleRequirement) Validate() error {
	if strings.TrimSpace(requirement.Role) == "" || strings.TrimSpace(requirement.Role) != requirement.Role {
		return fmt.Errorf("%w: role requirement is invalid", ErrInvalidApprovalPolicy)
	}
	if requirement.MinimumApprovals < 1 {
		return fmt.Errorf("%w: role requirement must require at least one approval", ErrInvalidApprovalPolicy)
	}
	return nil
}

type DelegationConfiguration struct {
	Enabled                     bool     `json:"enabled"`
	DelegateToRoles             []string `json:"delegateToRoles,omitempty"`
	MaximumDepth                int      `json:"maximumDepth,omitempty"`
	RequiresIndependentApproval bool     `json:"requiresIndependentApproval"`
}

func (configuration DelegationConfiguration) Validate() error {
	if !configuration.Enabled {
		return nil
	}
	if configuration.MaximumDepth < 1 {
		return fmt.Errorf("%w: enabled delegation requires a positive maximum depth", ErrInvalidApprovalPolicy)
	}
	if len(configuration.DelegateToRoles) == 0 {
		return fmt.Errorf("%w: enabled delegation requires a delegate role", ErrInvalidApprovalPolicy)
	}
	return validateUniqueStrings(configuration.DelegateToRoles, "delegation role")
}

type EscalationConfiguration struct {
	Enabled         bool     `json:"enabled"`
	AfterSeconds    int64    `json:"afterSeconds,omitempty"`
	EscalateToRoles []string `json:"escalateToRoles,omitempty"`
}

func (configuration EscalationConfiguration) Validate() error {
	if !configuration.Enabled {
		return nil
	}
	if configuration.AfterSeconds < 1 {
		return fmt.Errorf("%w: enabled escalation requires a positive delay", ErrInvalidApprovalPolicy)
	}
	if len(configuration.EscalateToRoles) == 0 {
		return fmt.Errorf("%w: enabled escalation requires an escalation role", ErrInvalidApprovalPolicy)
	}
	return validateUniqueStrings(configuration.EscalateToRoles, "escalation role")
}

type ApprovalRule struct {
	ID            uuid.UUID               `json:"id"`
	Name          string                  `json:"name,omitempty"`
	Threshold     AmountThreshold         `json:"threshold"`
	MinAmount     *decimal.Decimal        `json:"-"`
	MaxAmount     *decimal.Decimal        `json:"-"`
	Currency      string                  `json:"-"`
	RequiredRoles []RoleRequirement       `json:"requiredRoles"`
	Delegation    DelegationConfiguration `json:"delegation"`
	Escalation    EscalationConfiguration `json:"escalation"`
}

func (rule ApprovalRule) normalized() ApprovalRule {
	threshold := rule.Threshold
	if threshold.Minimum == nil {
		threshold.Minimum = rule.MinAmount
	}
	if threshold.Maximum == nil {
		threshold.Maximum = rule.MaxAmount
	}
	if threshold.Currency == "" {
		threshold.Currency = rule.Currency
	}
	rule.Threshold = threshold.normalized()
	return rule
}

func (rule ApprovalRule) Validate() error {
	rule = rule.normalized()
	if rule.ID == uuid.Nil {
		return fmt.Errorf("%w: approval rule id is required", ErrInvalidApprovalPolicy)
	}
	if err := rule.Threshold.Validate(); err != nil {
		return err
	}
	if len(rule.RequiredRoles) == 0 {
		return fmt.Errorf("%w: every rule requires an approver role", ErrInvalidApprovalPolicy)
	}
	for _, requirement := range rule.RequiredRoles {
		if err := requirement.Validate(); err != nil {
			return err
		}
	}
	if err := rule.Delegation.Validate(); err != nil {
		return err
	}
	return rule.Escalation.Validate()
}

type ApprovalStepDefinition struct {
	ID               uuid.UUID               `json:"id"`
	Sequence         int                     `json:"sequence"`
	Name             string                  `json:"name"`
	RequiredRoles    []RoleRequirement       `json:"requiredRoles"`
	MinimumApprovals int                     `json:"minimumApprovals"`
	Delegation       DelegationConfiguration `json:"delegation"`
	Escalation       EscalationConfiguration `json:"escalation"`
}

func (step ApprovalStepDefinition) Validate() error {
	if step.ID == uuid.Nil || step.Sequence < 1 || strings.TrimSpace(step.Name) == "" {
		return fmt.Errorf("%w: approval step identity and sequence are required", ErrInvalidApprovalPolicy)
	}
	if step.MinimumApprovals < 1 {
		return fmt.Errorf("%w: approval step must require at least one approval", ErrInvalidApprovalPolicy)
	}
	if len(step.RequiredRoles) == 0 {
		return fmt.Errorf("%w: every approval step requires a role", ErrInvalidApprovalPolicy)
	}
	for _, requirement := range step.RequiredRoles {
		if err := requirement.Validate(); err != nil {
			return err
		}
	}
	if err := step.Delegation.Validate(); err != nil {
		return err
	}
	return step.Escalation.Validate()
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
		return fmt.Errorf("%w: approval identifiers are required", ErrInvalidApprovalPolicy)
	}
	if strings.TrimSpace(reference.PolicyVersion) == "" || reference.DecisionVersion < 1 || reference.SubjectVersion < 1 || strings.TrimSpace(reference.CandidateFingerprint) == "" {
		return fmt.Errorf("%w: approval evidence is incomplete", ErrInvalidApprovalPolicy)
	}
	return nil
}

type ApprovalPolicySnapshot struct {
	ID                      uuid.UUID                  `json:"id"`
	Code                    string                     `json:"code"`
	Name                    string                     `json:"name"`
	PolicyVersion           string                     `json:"policyVersion"`
	Status                  string                     `json:"status"`
	Applicability           ApprovalApplicability      `json:"applicability"`
	Scope                   ApprovalPolicyScope        `json:"scope"`
	Rules                   []ApprovalRule             `json:"rules"`
	Steps                   []ApprovalStepDefinition   `json:"steps"`
	EffectiveFrom           time.Time                  `json:"effectiveFrom"`
	EffectiveTo             *time.Time                 `json:"effectiveTo,omitempty"`
	Approval                *ApprovalDecisionReference `json:"approval,omitempty"`
	ApprovalRequestID       uuid.UUID                  `json:"approvalRequestId,omitempty"`
	ActivationOutcome       string                     `json:"activationOutcome"`
	DependentWorkflowImpact string                     `json:"dependentWorkflowImpact,omitempty"`
}

type ApprovalPolicyRevision struct {
	PolicyVersion string                            `json:"policyVersion"`
	Version       aggregateversion.AggregateVersion `json:"version"`
	Snapshot      ApprovalPolicySnapshot            `json:"snapshot"`
	CreatedAt     time.Time                         `json:"createdAt"`
}

type ApprovalPolicy struct {
	ID                      uuid.UUID                         `json:"id"`
	Code                    string                            `json:"code"`
	Name                    string                            `json:"name"`
	PolicyVersion           string                            `json:"policyVersion"`
	Status                  string                            `json:"status"`
	Applicability           ApprovalApplicability             `json:"applicability"`
	Scope                   ApprovalPolicyScope               `json:"scope"`
	Rules                   []ApprovalRule                    `json:"rules"`
	Steps                   []ApprovalStepDefinition          `json:"steps"`
	EffectiveFrom           time.Time                         `json:"effectiveFrom"`
	EffectiveTo             *time.Time                        `json:"effectiveTo,omitempty"`
	Approval                *ApprovalDecisionReference        `json:"approval,omitempty"`
	ApprovalRequestID       uuid.UUID                         `json:"approvalRequestId,omitempty"`
	ActivationOutcome       string                            `json:"activationOutcome"`
	DependentWorkflowImpact string                            `json:"dependentWorkflowImpact,omitempty"`
	Version                 aggregateversion.AggregateVersion `json:"version"`
	CreatedAt               time.Time                         `json:"createdAt"`
	UpdatedAt               time.Time                         `json:"updatedAt"`
	History                 []ApprovalPolicyRevision          `json:"history,omitempty"`
}

func NewApprovalPolicy(id uuid.UUID, code, name, policyVersion string, applicability ApprovalApplicability, scope ApprovalPolicyScope, rules []ApprovalRule, steps []ApprovalStepDefinition, effectiveFrom time.Time, effectiveTo *time.Time, now time.Time) (ApprovalPolicy, error) {
	policy := ApprovalPolicy{
		ID: id, Code: strings.TrimSpace(code), Name: strings.TrimSpace(name), PolicyVersion: strings.TrimSpace(policyVersion),
		Status: ApprovalPolicyStatusDraft, Applicability: cloneApplicability(applicability), Scope: cloneScope(scope),
		Rules: cloneRules(rules), Steps: cloneSteps(steps), EffectiveFrom: effectiveFrom.UTC(), EffectiveTo: cloneTime(effectiveTo),
		ActivationOutcome: ApprovalPolicyActivationOutcomeNone, Version: aggregateversion.Initial(), CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if err := policy.Validate(); err != nil {
		return ApprovalPolicy{}, err
	}
	return policy, nil
}

func (policy ApprovalPolicy) Validate() error {
	if policy.ID == uuid.Nil || strings.TrimSpace(policy.Code) == "" || strings.TrimSpace(policy.Name) == "" || strings.TrimSpace(policy.PolicyVersion) == "" {
		return fmt.Errorf("%w: policy identity, code, name, and version are required", ErrInvalidApprovalPolicy)
	}
	if policy.Version.Value() < 1 {
		return fmt.Errorf("%w: aggregate version is required", ErrInvalidApprovalPolicy)
	}
	switch policy.Status {
	case ApprovalPolicyStatusDraft, ApprovalPolicyStatusPendingApproval, ApprovalPolicyStatusActive, ApprovalPolicyStatusSuperseded:
	default:
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidApprovalPolicy)
	}
	if err := policy.Applicability.Validate(); err != nil {
		return err
	}
	if err := policy.Scope.Validate(); err != nil {
		return err
	}
	if policy.EffectiveFrom.IsZero() {
		return fmt.Errorf("%w: effective-from is required", ErrInvalidApprovalPolicy)
	}
	if policy.EffectiveTo != nil && !policy.EffectiveTo.After(policy.EffectiveFrom) {
		return fmt.Errorf("%w: effective-to must be after effective-from", ErrInvalidApprovalPolicy)
	}
	if len(policy.Rules) == 0 {
		return fmt.Errorf("%w: at least one approval rule is required", ErrInvalidApprovalPolicy)
	}
	ruleIDs := make(map[uuid.UUID]struct{}, len(policy.Rules))
	for _, rule := range policy.Rules {
		if _, exists := ruleIDs[rule.ID]; exists {
			return fmt.Errorf("%w: duplicate approval rule", ErrInvalidApprovalPolicy)
		}
		ruleIDs[rule.ID] = struct{}{}
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	if len(policy.Steps) == 0 {
		return fmt.Errorf("%w: at least one approval step is required", ErrInvalidApprovalPolicy)
	}
	stepIDs := make(map[uuid.UUID]struct{}, len(policy.Steps))
	sequences := make(map[int]struct{}, len(policy.Steps))
	for _, step := range policy.Steps {
		if _, exists := stepIDs[step.ID]; exists {
			return fmt.Errorf("%w: duplicate approval step", ErrInvalidApprovalPolicy)
		}
		if _, exists := sequences[step.Sequence]; exists {
			return fmt.Errorf("%w: duplicate approval step sequence", ErrInvalidApprovalPolicy)
		}
		stepIDs[step.ID], sequences[step.Sequence] = struct{}{}, struct{}{}
		if err := step.Validate(); err != nil {
			return err
		}
	}
	for sequence := 1; sequence <= len(policy.Steps); sequence++ {
		if _, exists := sequences[sequence]; !exists {
			return fmt.Errorf("%w: approval steps must have contiguous sequences", ErrInvalidApprovalPolicy)
		}
	}
	if policy.Status == ApprovalPolicyStatusPendingApproval && policy.ApprovalRequestID == uuid.Nil {
		return fmt.Errorf("%w: pending policy requires an approval request", ErrInvalidApprovalPolicy)
	}
	if policy.Status == ApprovalPolicyStatusActive {
		if policy.Approval == nil {
			return fmt.Errorf("%w: active policy requires approval evidence", ErrInvalidApprovalPolicy)
		}
		if err := policy.Approval.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (policy ApprovalPolicy) Snapshot() ApprovalPolicySnapshot {
	return ApprovalPolicySnapshot{
		ID: policy.ID, Code: policy.Code, Name: policy.Name, PolicyVersion: policy.PolicyVersion, Status: policy.Status,
		Applicability: cloneApplicability(policy.Applicability), Scope: cloneScope(policy.Scope), Rules: cloneRules(policy.Rules), Steps: cloneSteps(policy.Steps),
		EffectiveFrom: policy.EffectiveFrom, EffectiveTo: cloneTime(policy.EffectiveTo), Approval: cloneApproval(policy.Approval), ApprovalRequestID: policy.ApprovalRequestID, ActivationOutcome: policy.ActivationOutcome, DependentWorkflowImpact: policy.DependentWorkflowImpact,
	}
}

func (policy ApprovalPolicy) CandidateFingerprint() string {
	return fingerprint(policy.SnapshotForFingerprint())
}

func (policy ApprovalPolicy) SnapshotForFingerprint() ApprovalPolicySnapshot {
	snapshot := policy.Snapshot()
	snapshot.Applicability = canonicalApplicability(snapshot.Applicability)
	snapshot.Scope = canonicalScope(snapshot.Scope)
	snapshot.Status = "candidate"
	snapshot.Approval = nil
	snapshot.ApprovalRequestID = uuid.Nil
	snapshot.ActivationOutcome = ""
	snapshot.DependentWorkflowImpact = ""
	return snapshot
}

// NewVersion creates a new draft revision without mutating the established
// value. The content is copied so callers cannot accidentally rewrite the old
// revision through shared slices or pointers.
func (policy ApprovalPolicy) NewVersion(policyVersion string, now time.Time) (ApprovalPolicy, error) {
	if err := policy.Validate(); err != nil {
		return ApprovalPolicy{}, err
	}
	if policy.Status == ApprovalPolicyStatusSuperseded {
		return ApprovalPolicy{}, fmt.Errorf("%w: superseded policy cannot be edited", ErrApprovalPolicyImmutable)
	}
	policyVersion = strings.TrimSpace(policyVersion)
	if policyVersion == "" || policy.hasPolicyVersion(policyVersion) {
		return ApprovalPolicy{}, fmt.Errorf("%w: a new policy revision requires a new policy version", ErrApprovalPolicyVersionConflict)
	}
	nextVersion, err := policy.Version.Advance()
	if err != nil {
		return ApprovalPolicy{}, err
	}
	next := policy
	next.PolicyVersion = policyVersion
	next.Status = ApprovalPolicyStatusDraft
	next.Approval = nil
	next.ApprovalRequestID = uuid.Nil
	next.ActivationOutcome = ApprovalPolicyActivationOutcomeNone
	next.DependentWorkflowImpact = ""
	next.Version = nextVersion
	next.UpdatedAt = now.UTC()
	next.Rules = cloneRules(policy.Rules)
	next.Steps = cloneSteps(policy.Steps)
	next.Applicability = cloneApplicability(policy.Applicability)
	next.Scope = cloneScope(policy.Scope)
	next.EffectiveTo = cloneTime(policy.EffectiveTo)
	if err := next.Validate(); err != nil {
		return ApprovalPolicy{}, err
	}
	return next, nil
}

// hasPolicyVersion prevents a new revision from reusing a historical policy
// label. The current label is included so callers get the same typed conflict
// whether they are updating from the current or a historical value.
func (policy ApprovalPolicy) hasPolicyVersion(policyVersion string) bool {
	policyVersion = strings.TrimSpace(policyVersion)
	if policyVersion == "" || policyVersion == policy.PolicyVersion {
		return true
	}
	for _, revision := range policy.History {
		if strings.TrimSpace(revision.PolicyVersion) == policyVersion {
			return true
		}
	}
	return false
}

func (policy *ApprovalPolicy) SubmitForApproval(requestID uuid.UUID, now time.Time) error {
	if policy == nil {
		return ErrInvalidApprovalPolicy
	}
	if policy.Status != ApprovalPolicyStatusDraft || requestID == uuid.Nil {
		return fmt.Errorf("%w: only a draft can be submitted with an approval request", ErrInvalidApprovalPolicy)
	}
	next, err := policy.Version.Advance()
	if err != nil {
		return err
	}
	policy.Status = ApprovalPolicyStatusPendingApproval
	policy.ApprovalRequestID = requestID
	policy.ActivationOutcome = ApprovalPolicyActivationOutcomePending
	policy.Version = next
	policy.UpdatedAt = now.UTC()
	return policy.Validate()
}

func (policy *ApprovalPolicy) Activate(reference ApprovalDecisionReference, now time.Time) error {
	if policy == nil {
		return ErrInvalidApprovalPolicy
	}
	if policy.Status != ApprovalPolicyStatusPendingApproval {
		return fmt.Errorf("%w: only a pending policy can be activated", ErrInvalidApprovalPolicy)
	}
	if err := reference.Validate(); err != nil {
		return err
	}
	if reference.ApprovalRequestID != policy.ApprovalRequestID || reference.PolicyVersion != policy.PolicyVersion || reference.CandidateFingerprint != policy.CandidateFingerprint() || reference.SubjectVersion != policy.Version.Value() {
		return fmt.Errorf("%w: approval evidence does not match the candidate", ErrApprovalPolicyApprovalRejected)
	}
	next, err := policy.Version.Advance()
	if err != nil {
		return err
	}
	policy.Status = ApprovalPolicyStatusActive
	policy.Approval = cloneApproval(&reference)
	policy.ActivationOutcome = ApprovalPolicyActivationOutcomeActivated
	policy.Version = next
	policy.UpdatedAt = now.UTC()
	return policy.Validate()
}

func (policy *ApprovalPolicy) Supersede(now time.Time) error {
	if policy == nil || policy.Status != ApprovalPolicyStatusActive {
		return fmt.Errorf("%w: only an active policy can be superseded", ErrInvalidApprovalPolicy)
	}
	next, err := policy.Version.Advance()
	if err != nil {
		return err
	}
	policy.Status = ApprovalPolicyStatusSuperseded
	policy.Version = next
	policy.UpdatedAt = now.UTC()
	return policy.Validate()
}

// PoliciesOverlap is intentionally conservative: interval endpoints are
// inclusive, and only policies with equal applicability and scope selectors
// can conflict. Draft and pending candidates are not selectable, so they do
// not hide or rebind an active version.
func PoliciesOverlap(left, right ApprovalPolicy) bool {
	if left.Status != ApprovalPolicyStatusActive || right.Status != ApprovalPolicyStatusActive {
		return false
	}
	if left.ID == right.ID {
		return false
	}
	if selectorFingerprint(left.Applicability, left.Scope) != selectorFingerprint(right.Applicability, right.Scope) {
		return false
	}
	return intervalsOverlap(left.EffectiveFrom, left.EffectiveTo, right.EffectiveFrom, right.EffectiveTo)
}

func selectorFingerprint(applicability ApprovalApplicability, scope ApprovalPolicyScope) string {
	return fingerprint(struct {
		Applicability ApprovalApplicability `json:"applicability"`
		Scope         ApprovalPolicyScope   `json:"scope"`
	}{canonicalApplicability(applicability), canonicalScope(scope)})
}

func canonicalApplicability(value ApprovalApplicability) ApprovalApplicability {
	value = cloneApplicability(value)
	sort.Strings(value.TransactionTypes)
	sort.Strings(value.Currencies)
	sort.Strings(value.DataClassifications)
	return value
}

func canonicalScope(value ApprovalPolicyScope) ApprovalPolicyScope {
	value = cloneScope(value)
	sort.Strings(value.ScopeIDs)
	sort.Slice(value.AccountingScopeIDs, func(left, right int) bool {
		return value.AccountingScopeIDs[left].String() < value.AccountingScopeIDs[right].String()
	})
	sort.Slice(value.LegalEntityIDs, func(left, right int) bool {
		return value.LegalEntityIDs[left].String() < value.LegalEntityIDs[right].String()
	})
	sort.Slice(value.BusinessUnitIDs, func(left, right int) bool {
		return value.BusinessUnitIDs[left].String() < value.BusinessUnitIDs[right].String()
	})
	sort.Slice(value.SegmentIDs, func(left, right int) bool { return value.SegmentIDs[left].String() < value.SegmentIDs[right].String() })
	sort.Slice(value.AccountIDs, func(left, right int) bool { return value.AccountIDs[left].String() < value.AccountIDs[right].String() })
	sort.Slice(value.FiscalPeriodIDs, func(left, right int) bool {
		return value.FiscalPeriodIDs[left].String() < value.FiscalPeriodIDs[right].String()
	})
	return value
}

func intervalsOverlap(leftFrom time.Time, leftTo *time.Time, rightFrom time.Time, rightTo *time.Time) bool {
	if leftTo != nil && rightFrom.After(leftTo.UTC()) {
		return false
	}
	if rightTo != nil && leftFrom.After(rightTo.UTC()) {
		return false
	}
	return true
}

func validateUniqueStrings(values []string, label string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			return fmt.Errorf("%w: %s must be non-blank and must not contain surrounding whitespace", ErrInvalidApprovalPolicy, label)
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("%w: duplicate %s", ErrInvalidApprovalPolicy, label)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func fingerprint(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "sha256:invalid"
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func cloneApproval(value *ApprovalDecisionReference) *ApprovalDecisionReference {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneApplicability(value ApprovalApplicability) ApprovalApplicability {
	value.TransactionTypes = append([]string(nil), value.TransactionTypes...)
	value.Currencies = append([]string(nil), value.Currencies...)
	value.DataClassifications = append([]string(nil), value.DataClassifications...)
	if value.Dimensions != nil {
		value.Dimensions = map[string]string{}
		for key, item := range value.Dimensions {
			value.Dimensions[key] = item
		}
	}
	return value
}

func cloneScope(value ApprovalPolicyScope) ApprovalPolicyScope {
	value.ScopeIDs = append([]string(nil), value.ScopeIDs...)
	value.AccountingScopeIDs = append([]uuid.UUID(nil), value.AccountingScopeIDs...)
	value.LegalEntityIDs = append([]uuid.UUID(nil), value.LegalEntityIDs...)
	value.BusinessUnitIDs = append([]uuid.UUID(nil), value.BusinessUnitIDs...)
	value.SegmentIDs = append([]uuid.UUID(nil), value.SegmentIDs...)
	value.AccountIDs = append([]uuid.UUID(nil), value.AccountIDs...)
	value.FiscalPeriodIDs = append([]uuid.UUID(nil), value.FiscalPeriodIDs...)
	return value
}

func cloneRules(values []ApprovalRule) []ApprovalRule {
	result := append([]ApprovalRule(nil), values...)
	for index := range result {
		result[index] = result[index].normalized()
		result[index].RequiredRoles = append([]RoleRequirement(nil), result[index].RequiredRoles...)
		result[index].Delegation.DelegateToRoles = append([]string(nil), result[index].Delegation.DelegateToRoles...)
		result[index].Escalation.EscalateToRoles = append([]string(nil), result[index].Escalation.EscalateToRoles...)
	}
	return result
}

func cloneSteps(values []ApprovalStepDefinition) []ApprovalStepDefinition {
	result := append([]ApprovalStepDefinition(nil), values...)
	for index := range result {
		result[index].RequiredRoles = append([]RoleRequirement(nil), result[index].RequiredRoles...)
		result[index].Delegation.DelegateToRoles = append([]string(nil), result[index].Delegation.DelegateToRoles...)
		result[index].Escalation.EscalateToRoles = append([]string(nil), result[index].Escalation.EscalateToRoles...)
	}
	return result
}
