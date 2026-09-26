package identity

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
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

type SegregationRuleStatus string

const (
	SegregationRuleStatusActive  SegregationRuleStatus = "active"
	SegregationRuleStatusRetired SegregationRuleStatus = "retired"
)

type SegregationEnforcementMode string

const (
	SegregationEnforcementBlock             SegregationEnforcementMode = "block"
	SegregationEnforcementExceptionRequired SegregationEnforcementMode = "exception-required"
)

const (
	SegregationRulePaymentBatch         = "payment-batch-preparation-approval"
	SegregationRuleFiscalReopen         = "fiscal-period-reopen-request-approval"
	SegregationRuleVendorBankCooling    = "vendor-bank-detail-payment-release-cooling-off"
	SegregationRuleManualJournal        = "manual-journal-self-approval-threshold"
	SegregationRulePayrollSummary       = "payroll-detail-summary-ledger"
	SegregationRulePolicyAdministration = "independent-policy-approval"
)

const (
	SegregationActionPaymentBatchPrepare    = "payment-batch.prepare"
	SegregationActionPaymentBatchApprove    = "payment-batch.approve"
	SegregationActionFiscalReopenRequest    = "fiscal-reopen.request"
	SegregationActionFiscalReopenApprove    = "fiscal-reopen.approve"
	SegregationActionVendorBankDetailChange = "vendor-bank-detail.change"
	SegregationActionPaymentRelease         = "payment.release"
	SegregationActionManualJournalPrepare   = "manual-journal.prepare"
	SegregationActionManualJournalApprove   = "manual-journal.approve"
	SegregationActionPolicyPropose          = "policy.propose"
	SegregationActionPolicyApprove          = "policy.approve"
)

var (
	ErrInvalidSegregationRule       = errors.New("invalid identity segregation rule")
	ErrInvalidSegregationCommand    = errors.New("invalid identity segregation command")
	ErrSegregationRuleNotFound      = errors.New("identity segregation rule not found")
	ErrSegregationRuleRetired       = errors.New("identity segregation rule is retired")
	ErrSegregationRuleConflict      = errors.New("identity segregation rule conflict")
	ErrSegregationRuleUnavailable   = errors.New("identity segregation rule evaluation unavailable")
	ErrSegregationRuleStale         = errors.New("identity segregation rule is stale")
	ErrSegregationRuleAuthorization = errors.New("identity segregation rule authorization denied")
	ErrSegregationRuleApproval      = errors.New("identity segregation rule approval rejected")
	ErrSegregationRuleAudit         = errors.New("identity segregation rule audit unavailable")
	ErrSegregationRuleVersion       = errors.New("identity segregation rule version conflict")
	ErrSegregationRuleIdempotency   = errors.New("identity segregation rule idempotency conflict")
	ErrSegregationRuleInProgress    = errors.New("identity segregation rule command is in progress")
)

// SegregationRule is an immutable policy revision exposed as the current
// aggregate value. Historical revisions are retained by the repository.
type SegregationRule struct {
	ID                     uuid.UUID
	Code                   string
	Name                   string
	Status                 SegregationRuleStatus
	Version                aggregateversion.AggregateVersion
	ConflictingPermissions []string
	EnforcementMode        SegregationEnforcementMode
	ScopeIDs               []string
	AmountThreshold        *decimal.Decimal
	CoolingOff             time.Duration
	EffectiveFrom          time.Time
	EffectiveTo            *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	Approval               ApprovalDecisionReference
	AuditReference         uuid.UUID
}

func (rule SegregationRule) Validate() error {
	if rule.ID == uuid.Nil || strings.TrimSpace(rule.Code) == "" || strings.TrimSpace(rule.Name) == "" {
		return fmt.Errorf("%w: id, code, and name are required", ErrInvalidSegregationRule)
	}
	if rule.Status != SegregationRuleStatusActive && rule.Status != SegregationRuleStatusRetired {
		return fmt.Errorf("%w: unsupported status", ErrInvalidSegregationRule)
	}
	if rule.Version.Value() < 1 || rule.CreatedAt.IsZero() || rule.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: version and timestamps are required", ErrInvalidSegregationRule)
	}
	if rule.EnforcementMode != SegregationEnforcementBlock && rule.EnforcementMode != SegregationEnforcementExceptionRequired {
		return fmt.Errorf("%w: unsupported enforcement mode", ErrInvalidSegregationRule)
	}
	if len(rule.ConflictingPermissions) < 2 {
		return fmt.Errorf("%w: at least two conflicting permissions are required", ErrInvalidSegregationRule)
	}
	seen := make(map[string]struct{}, len(rule.ConflictingPermissions))
	for _, permission := range rule.ConflictingPermissions {
		permission = strings.TrimSpace(permission)
		if permission == "" {
			return fmt.Errorf("%w: conflicting permission is blank", ErrInvalidSegregationRule)
		}
		if _, exists := seen[permission]; exists {
			return fmt.Errorf("%w: duplicate conflicting permission", ErrInvalidSegregationRule)
		}
		seen[permission] = struct{}{}
	}
	seen = make(map[string]struct{}, len(rule.ScopeIDs))
	for _, scopeID := range rule.ScopeIDs {
		scopeID = strings.TrimSpace(scopeID)
		if scopeID == "" {
			return fmt.Errorf("%w: scope is blank", ErrInvalidSegregationRule)
		}
		if _, exists := seen[scopeID]; exists {
			return fmt.Errorf("%w: duplicate scope", ErrInvalidSegregationRule)
		}
		seen[scopeID] = struct{}{}
	}
	if rule.AmountThreshold != nil && rule.AmountThreshold.IsNegative() {
		return fmt.Errorf("%w: amount threshold cannot be negative", ErrInvalidSegregationRule)
	}
	if rule.CoolingOff < 0 {
		return fmt.Errorf("%w: cooling-off cannot be negative", ErrInvalidSegregationRule)
	}
	if rule.EffectiveFrom.IsZero() || (rule.EffectiveTo != nil && !rule.EffectiveTo.After(rule.EffectiveFrom)) {
		return fmt.Errorf("%w: invalid effective interval", ErrInvalidSegregationRule)
	}
	if err := rule.Approval.Validate(); err != nil {
		return err
	}
	return nil
}

func NewSegregationRule(id uuid.UUID, code, name string, permissions []string, mode SegregationEnforcementMode, scopes []string, threshold *decimal.Decimal, coolingOff time.Duration, effectiveFrom time.Time, approval ApprovalDecisionReference, now time.Time) (SegregationRule, error) {
	rule := SegregationRule{
		ID: id, Code: strings.TrimSpace(code), Name: strings.TrimSpace(name), Status: SegregationRuleStatusActive,
		Version: aggregateversion.Initial(), ConflictingPermissions: append([]string(nil), permissions...),
		EnforcementMode: mode, ScopeIDs: append([]string(nil), scopes...), AmountThreshold: cloneDecimal(threshold),
		CoolingOff: coolingOff, EffectiveFrom: effectiveFrom.UTC(), CreatedAt: now.UTC(), UpdatedAt: now.UTC(), Approval: approval,
	}
	sort.Strings(rule.ConflictingPermissions)
	sort.Strings(rule.ScopeIDs)
	if err := rule.Validate(); err != nil {
		return SegregationRule{}, err
	}
	return rule, nil
}

func (rule *SegregationRule) Replace(command SegregationRuleCommand, approval ApprovalDecisionReference, now time.Time) error {
	if rule == nil {
		return ErrInvalidSegregationRule
	}
	if rule.Status == SegregationRuleStatusRetired {
		return ErrSegregationRuleRetired
	}
	replacement, err := NewSegregationRule(rule.ID, command.Code, command.Name, command.ConflictingPermissions, command.EnforcementMode, command.ScopeIDs, command.AmountThreshold, command.CoolingOff, command.EffectiveFrom, approval, now)
	if err != nil {
		return err
	}
	replacement.Version = rule.Version
	replacement.CreatedAt = rule.CreatedAt
	replacement.UpdatedAt = now.UTC()
	replacement.EffectiveTo = cloneTime(command.EffectiveTo)
	*rule = replacement
	return nil
}

func (rule *SegregationRule) Retire(approval ApprovalDecisionReference, now time.Time) error {
	if rule == nil {
		return ErrInvalidSegregationRule
	}
	if rule.Status == SegregationRuleStatusRetired {
		return ErrSegregationRuleRetired
	}
	rule.Status = SegregationRuleStatusRetired
	rule.UpdatedAt = now.UTC()
	rule.Approval = approval
	return nil
}

type SegregationHistoryEntry struct {
	ActorID   uuid.UUID
	SubjectID uuid.UUID
	Action    string
	ScopeIDs  []string
	At        time.Time
}

type SegregationException struct {
	Active      bool
	Reason      string
	ApprovedBy  uuid.UUID
	ExpiresAt   time.Time
	RuleVersion aggregateversion.AggregateVersion
}

type SegregationDecisionInput struct {
	ActorID               uuid.UUID
	Action                string
	SubjectID             uuid.UUID
	ScopeIDs              []string
	Permissions           []string
	Amount                *decimal.Decimal
	History               []SegregationHistoryEntry
	Exception             *SegregationException
	ExpectedRuleVersion   *aggregateversion.AggregateVersion
	ExpectedPolicyVersion string
	ProposerID            uuid.UUID
	ApproverID            uuid.UUID
}

func (input SegregationDecisionInput) Validate() error {
	if input.ActorID == uuid.Nil {
		return fmt.Errorf("%w: actor is required", ErrInvalidSegregationCommand)
	}
	if strings.TrimSpace(input.Action) == "" {
		return fmt.Errorf("%w: action is required", ErrInvalidSegregationCommand)
	}
	if input.Amount != nil && input.Amount.IsNegative() {
		return fmt.Errorf("%w: amount cannot be negative", ErrInvalidSegregationCommand)
	}
	return nil
}

type SegregationDecision struct {
	Allowed           bool
	Outcome           AuthorizationOutcome
	RuleReference     uuid.UUID
	RuleCode          string
	RuleVersion       aggregateversion.AggregateVersion
	PolicyVersion     string
	DecisionReference uuid.UUID
	ReasonCode        string
	Reason            string
	Resolution        string
}

const (
	SegregationReasonAllowed          = "allowed"
	SegregationReasonConflict         = "segregation-conflict"
	SegregationReasonExpiredException = "exception-expired"
	SegregationReasonStale            = "segregation-policy-stale"
	SegregationReasonUnavailable      = "segregation-policy-unavailable"
)

type SegregationRuleStore interface {
	List(context.Context) ([]SegregationRule, error)
}

type SegregationDecisionPort interface {
	Evaluate(context.Context, SegregationDecisionInput) (SegregationDecision, error)
}

type SegregationDecisionObserver func(context.Context, SegregationDecision, error)

type SegregationEvaluator struct {
	store    SegregationRuleStore
	clock    func() time.Time
	observer SegregationDecisionObserver
}

func NewSegregationEvaluator(store SegregationRuleStore, clock func() time.Time, observers ...SegregationDecisionObserver) (*SegregationEvaluator, error) {
	if store == nil {
		return nil, ErrSegregationRuleUnavailable
	}
	if clock == nil {
		clock = time.Now
	}
	var observer SegregationDecisionObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &SegregationEvaluator{store: store, clock: clock, observer: observer}, nil
}

func (evaluator *SegregationEvaluator) Evaluate(ctx context.Context, input SegregationDecisionInput) (decision SegregationDecision, err error) {
	decision = SegregationDecision{Outcome: AuthorizationDenied, DecisionReference: uuid.New(), ReasonCode: SegregationReasonConflict, Resolution: "Use an independent actor or request an approved exception."}
	defer func() {
		if evaluator != nil && evaluator.observer != nil {
			evaluator.observer(ctx, decision, err)
		}
	}()
	if err := input.Validate(); err != nil {
		return decision, err
	}
	rules, err := evaluator.store.List(ctx)
	if err != nil {
		decision.Outcome = AuthorizationUnavailable
		decision.ReasonCode = SegregationReasonUnavailable
		decision.Reason = "The segregation policy could not be evaluated."
		decision.Resolution = "Retry after the identity policy dependency is available."
		return decision, nil
	}
	now := evaluator.clock().UTC()
	for _, rule := range rules {
		if rule.Status != SegregationRuleStatusActive || now.Before(rule.EffectiveFrom) || (rule.EffectiveTo != nil && !now.Before(*rule.EffectiveTo)) {
			continue
		}
		if !scopeMatches(rule.ScopeIDs, input.ScopeIDs) {
			continue
		}
		if input.ExpectedRuleVersion != nil && !rule.Version.Matches(*input.ExpectedRuleVersion) {
			decision.Outcome = AuthorizationStale
			decision.ReasonCode = SegregationReasonStale
			decision.RuleReference = rule.ID
			decision.RuleCode = rule.Code
			decision.RuleVersion = rule.Version
			decision.PolicyVersion = rule.Approval.PolicyVersion
			decision.Reason = "The segregation rule changed after the action was prepared."
			decision.Resolution = "Refresh the current rule and retry."
			return decision, nil
		}
		if input.ExpectedPolicyVersion != "" && input.ExpectedPolicyVersion != rule.Approval.PolicyVersion {
			decision.Outcome = AuthorizationStale
			decision.ReasonCode = SegregationReasonStale
			decision.RuleReference = rule.ID
			decision.RuleCode = rule.Code
			decision.RuleVersion = rule.Version
			decision.PolicyVersion = rule.Approval.PolicyVersion
			decision.Reason = "The authorization policy changed after the action was prepared."
			decision.Resolution = "Refresh policy state and retry."
			return decision, nil
		}
		if segregationHistoryRequired(rule, input) && (input.History == nil || input.SubjectID == uuid.Nil) {
			decision.Outcome = AuthorizationUnavailable
			decision.ReasonCode = SegregationReasonUnavailable
			decision.RuleReference = rule.ID
			decision.RuleCode = rule.Code
			decision.RuleVersion = rule.Version
			decision.PolicyVersion = rule.Approval.PolicyVersion
			decision.Reason = "The actor history required for this decision is unavailable."
			decision.Resolution = "Retry after the identity history dependency is available."
			return decision, nil
		}
		if !ruleConflicts(rule, input, now) {
			continue
		}
		decision.RuleReference = rule.ID
		decision.RuleCode = rule.Code
		decision.RuleVersion = rule.Version
		decision.PolicyVersion = rule.Approval.PolicyVersion
		if rule.EnforcementMode == SegregationEnforcementExceptionRequired && validSegregationException(input.Exception, rule, input.ActorID, now) {
			decision.Allowed = true
			decision.Outcome = AuthorizationAllowed
			decision.ReasonCode = SegregationReasonAllowed
			decision.Reason = "An approved, time-bound exception permits this conflict."
			decision.Resolution = "Continue under the recorded exception and retain the audit reference."
			return decision, nil
		}
		if input.Exception != nil && input.Exception.Active && !input.Exception.ExpiresAt.After(now) {
			decision.ReasonCode = SegregationReasonExpiredException
			decision.Reason = "The recorded exception has expired."
			decision.Resolution = "Request a new independently approved exception."
		}
		return decision, nil
	}
	decision.Allowed = true
	decision.Outcome = AuthorizationAllowed
	decision.ReasonCode = SegregationReasonAllowed
	decision.Reason = SegregationReasonAllowed
	decision.Resolution = "Continue with the requested action."
	return decision, nil
}

// ValidateRoleGrants is the identity-owned adapter used by RoleService. A
// role revision cannot introduce a known conflicting permission set.
func (evaluator *SegregationEvaluator) ValidateRoleGrants(ctx context.Context, grants []PermissionGrant) error {
	permissions := make([]string, 0, len(grants))
	scopes := make([]string, 0)
	for _, grant := range grants {
		permissions = append(permissions, grant.Permission)
		scopes = append(scopes, grant.ScopeIDs...)
	}
	rules, err := evaluator.store.List(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSegregationUnavailable, ErrSegregationRuleUnavailable)
	}
	now := evaluator.clock().UTC()
	for _, rule := range rules {
		if rule.Status != SegregationRuleStatusActive || now.Before(rule.EffectiveFrom) || (rule.EffectiveTo != nil && !now.Before(*rule.EffectiveTo)) {
			continue
		}
		if scopeMatches(rule.ScopeIDs, scopes) && containsAll(permissions, rule.ConflictingPermissions) {
			return fmt.Errorf("%w: %w: %s", ErrSegregationConflict, ErrSegregationRuleConflict, rule.Code)
		}
	}
	return nil
}

func segregationHistoryRequired(rule SegregationRule, input SegregationDecisionInput) bool {
	switch rule.Code {
	case SegregationRulePaymentBatch:
		return actionMatches(input.Action, SegregationActionPaymentBatchApprove)
	case SegregationRuleFiscalReopen:
		return actionMatches(input.Action, SegregationActionFiscalReopenApprove)
	case SegregationRuleVendorBankCooling:
		return actionMatches(input.Action, SegregationActionPaymentRelease)
	case SegregationRuleManualJournal:
		return actionMatches(input.Action, SegregationActionManualJournalApprove)
	case SegregationRulePolicyAdministration:
		return actionMatches(input.Action, SegregationActionPolicyApprove)
	default:
		return false
	}
}

func ruleConflicts(rule SegregationRule, input SegregationDecisionInput, now time.Time) bool {
	if containsAll(input.Permissions, rule.ConflictingPermissions) {
		return true
	}
	switch rule.Code {
	case SegregationRulePaymentBatch:
		return historyConflict(input, SegregationActionPaymentBatchPrepare, SegregationActionPaymentBatchApprove)
	case SegregationRuleFiscalReopen:
		return historyConflict(input, SegregationActionFiscalReopenRequest, SegregationActionFiscalReopenApprove)
	case SegregationRuleVendorBankCooling:
		if !actionMatches(input.Action, SegregationActionPaymentRelease) {
			return false
		}
		for _, event := range input.History {
			if event.ActorID == input.ActorID && event.SubjectID == input.SubjectID && actionMatches(event.Action, SegregationActionVendorBankDetailChange) && scopeMatches(event.ScopeIDs, input.ScopeIDs) && !event.At.Add(rule.CoolingOff).Before(now) {
				return true
			}
		}
	case SegregationRuleManualJournal:
		if input.Amount == nil || rule.AmountThreshold == nil || input.Amount.LessThanOrEqual(*rule.AmountThreshold) || !actionMatches(input.Action, SegregationActionManualJournalApprove) {
			return false
		}
		return historyContains(input, SegregationActionManualJournalPrepare)
	case SegregationRulePolicyAdministration:
		if input.ApproverID != uuid.Nil && input.ProposerID != uuid.Nil && input.ApproverID == input.ProposerID {
			return true
		}
		return historyConflict(input, SegregationActionPolicyPropose, SegregationActionPolicyApprove)
	case SegregationRulePayrollSummary:
		return containsAll(input.Permissions, rule.ConflictingPermissions)
	}
	return false
}

func historyConflict(input SegregationDecisionInput, prepare, approve string) bool {
	if !actionMatches(input.Action, approve) {
		return false
	}
	return historyContains(input, prepare)
}

func historyContains(input SegregationDecisionInput, action string) bool {
	for _, event := range input.History {
		if event.ActorID == input.ActorID && event.SubjectID == input.SubjectID && actionMatches(event.Action, action) && scopeMatches(event.ScopeIDs, input.ScopeIDs) {
			return true
		}
	}
	return false
}

func actionMatches(actual, expected string) bool {
	return strings.EqualFold(strings.TrimSpace(actual), strings.TrimSpace(expected))
}

func scopeMatches(ruleScopes, inputScopes []string) bool {
	if len(ruleScopes) == 0 {
		return true
	}
	for _, ruleScope := range ruleScopes {
		for _, inputScope := range inputScopes {
			if strings.TrimSpace(ruleScope) == strings.TrimSpace(inputScope) {
				return true
			}
		}
	}
	return false
}

func containsAll(values, required []string) bool {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[strings.TrimSpace(value)] = struct{}{}
	}
	for _, value := range required {
		if _, exists := set[strings.TrimSpace(value)]; !exists {
			return false
		}
	}
	return true
}

func validSegregationException(exception *SegregationException, rule SegregationRule, actor uuid.UUID, now time.Time) bool {
	return exception != nil && exception.Active && exception.ApprovedBy != uuid.Nil && exception.ApprovedBy != actor && strings.TrimSpace(exception.Reason) != "" && exception.ExpiresAt.After(now) && (exception.RuleVersion.Value() == 0 || exception.RuleVersion.Matches(rule.Version))
}

func cloneDecimal(value *decimal.Decimal) *decimal.Decimal {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func cloneSegregationRule(rule SegregationRule) SegregationRule {
	rule.ConflictingPermissions = append([]string(nil), rule.ConflictingPermissions...)
	rule.ScopeIDs = append([]string(nil), rule.ScopeIDs...)
	rule.AmountThreshold = cloneDecimal(rule.AmountThreshold)
	rule.EffectiveTo = cloneTime(rule.EffectiveTo)
	return rule
}

func DefaultSegregationRules(now time.Time) []SegregationRule {
	return []SegregationRule{
		defaultSegregationRule("00000000-0000-0000-0000-000000000501", SegregationRulePaymentBatch, "Payment batch preparer and approver", []string{"finance.pcm.prepare.payment.batch", "finance.pcm.apply.payment.batch.approval.decision"}, SegregationEnforcementExceptionRequired, now),
		defaultSegregationRule("00000000-0000-0000-0000-000000000502", SegregationRuleFiscalReopen, "Fiscal-period reopen requester and approver", []string{"finance.fpm.request.reopen", "finance.fpm.apply.reopen.approval.decision"}, SegregationEnforcementBlock, now),
		defaultSegregationRule("00000000-0000-0000-0000-000000000503", SegregationRuleVendorBankCooling, "Vendor bank-detail cooling-off", []string{"finance.omd.maintain.vendor.profiles", "finance.pcm.submit.payment.instruction"}, SegregationEnforcementBlock, now),
		defaultSegregationRule("00000000-0000-0000-0000-000000000504", SegregationRuleManualJournal, "Manual-journal self approval threshold", []string{"finance.gl.submit.posting.request", "finance.gl.apply.journal.approval.decision"}, SegregationEnforcementBlock, now),
		defaultSegregationRule("00000000-0000-0000-0000-000000000505", SegregationRulePayrollSummary, "Payroll detail and summary ledger", []string{"finance.payr.maintain.employee.payroll.profiles", "finance.rpt.generate.and.publish.ledger.financial.statements"}, SegregationEnforcementBlock, now),
		defaultSegregationRule("00000000-0000-0000-0000-000000000506", SegregationRulePolicyAdministration, "Independent policy approval", []string{"finance.iam.manage.access.policies", "finance.wfa.decide.approval.request"}, SegregationEnforcementBlock, now),
	}
}

func defaultSegregationRule(id, code, name string, permissions []string, mode SegregationEnforcementMode, now time.Time) SegregationRule {
	rule := SegregationRule{ID: uuid.MustParse(id), Code: code, Name: name, Status: SegregationRuleStatusActive, Version: aggregateversion.Initial(), ConflictingPermissions: permissions, EnforcementMode: mode, CoolingOff: 24 * time.Hour, EffectiveFrom: now.UTC().Add(-time.Hour), CreatedAt: now.UTC(), UpdatedAt: now.UTC(), Approval: ApprovalDecisionReference{ApprovalRequestID: deterministicSegregationUUID(code + ":request"), DecisionID: deterministicSegregationUUID(code + ":decision"), ApproverUserID: deterministicSegregationUUID(code + ":approver"), PolicyVersion: "iam-segregation-baseline-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:baseline-" + code}}
	if code == SegregationRuleManualJournal {
		threshold := decimal.NewFromInt(10000)
		rule.AmountThreshold = &threshold
	}
	return rule
}

func deterministicSegregationUUID(value string) uuid.UUID {
	return uuid.NewSHA1(uuid.Nil, []byte(value))
}

type SegregationRuleCommand struct {
	Action                 string
	RuleID                 uuid.UUID
	Code                   string
	Name                   string
	ConflictingPermissions []string
	EnforcementMode        SegregationEnforcementMode
	ScopeIDs               []string
	AmountThreshold        *decimal.Decimal
	CoolingOff             time.Duration
	EffectiveFrom          time.Time
	EffectiveTo            *time.Time
	Approval               ApprovalDecisionReference
	ExpectedVersion        *aggregateversion.AggregateVersion
	IdempotencyKey         string
	CorrelationID          string
	CausationID            string
}

const (
	SegregationRuleActionCreate         = "create"
	SegregationRuleActionUpdate         = "update"
	SegregationRuleActionRetire         = "retire"
	SegregationRuleManagementPermission = "finance.iam.manage.segregation.rules"
)

func (command SegregationRuleCommand) Validate() error {
	if command.Action != SegregationRuleActionCreate && command.Action != SegregationRuleActionUpdate && command.Action != SegregationRuleActionRetire {
		return fmt.Errorf("%w: unsupported action", ErrInvalidSegregationCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidSegregationCommand)
	}
	if command.Action == SegregationRuleActionCreate && command.RuleID != uuid.Nil {
		return fmt.Errorf("%w: create cannot include rule id", ErrInvalidSegregationCommand)
	}
	if command.Action == SegregationRuleActionCreate && command.ExpectedVersion != nil {
		return fmt.Errorf("%w: create cannot include expected version", ErrInvalidSegregationCommand)
	}
	if command.Action != SegregationRuleActionCreate {
		if command.RuleID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: existing rule id and expected version are required", ErrInvalidSegregationCommand)
		}
	}
	if command.Action != SegregationRuleActionRetire {
		if strings.TrimSpace(command.Code) == "" || strings.TrimSpace(command.Name) == "" || command.EffectiveFrom.IsZero() {
			return fmt.Errorf("%w: rule identity and effective date are required", ErrInvalidSegregationCommand)
		}
		if len(command.ConflictingPermissions) < 2 {
			return fmt.Errorf("%w: conflicting permissions are required", ErrInvalidSegregationCommand)
		}
	}
	return nil
}

type SegregationRuleAuthorizer interface {
	AuthorizeSegregationRuleManagement(context.Context, ApplicationActor, SegregationRuleCommand, *SegregationRule) (AuthorizationDecision, error)
}

type SegregationRuleApprovalPort interface {
	ValidateSegregationRuleApproval(context.Context, ApplicationActor, SegregationRuleCommand, *SegregationRule, string) error
}

type AllowAllSegregationRuleApprovalPort struct{}

func (AllowAllSegregationRuleApprovalPort) ValidateSegregationRuleApproval(_ context.Context, actor ApplicationActor, command SegregationRuleCommand, current *SegregationRule, fingerprint string) error {
	if err := command.Approval.Validate(); err != nil {
		return err
	}
	expectedVersion := int64(1)
	if current != nil {
		expectedVersion = current.Version.Value() + 1
	}
	if command.Approval.SubjectVersion != expectedVersion {
		return fmt.Errorf("%w: subject version does not match candidate revision", ErrSegregationRuleApproval)
	}
	if command.Approval.ApproverUserID == actor.UserID {
		return fmt.Errorf("%w: requesting actor cannot approve the same rule change", ErrSegregationRuleApproval)
	}
	if command.Approval.CandidateFingerprint != fingerprint {
		return fmt.Errorf("%w: candidate fingerprint does not match", ErrSegregationRuleApproval)
	}
	return nil
}

type SegregationRuleAuditRecord struct {
	RuleID                 uuid.UUID
	ActorUserID            uuid.UUID
	ActorAuthenticationRef string
	Action                 string
	Code                   string
	ScopeIDs               []string
	PolicyReference        string
	PolicyVersion          string
	DecisionReference      uuid.UUID
	ApprovalRequestID      uuid.UUID
	ApprovalDecisionID     uuid.UUID
	ApproverUserID         uuid.UUID
	RuleVersion            int64
	BeforeFingerprint      string
	AfterFingerprint       string
	CorrelationID          string
	CausationID            string
}

type SegregationRuleAuditRecorder interface {
	RecordSegregationRuleMutation(context.Context, SegregationRuleAuditRecord) error
}

type SegregationRuleRepository interface {
	SegregationRuleStore
	Get(context.Context, uuid.UUID) (SegregationRule, error)
	CommitSegregationRuleMutation(context.Context, SegregationRuleMutation) error
}

type SegregationRuleMutation struct {
	Before          SegregationRule
	After           SegregationRule
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           SegregationRuleAuditRecord
}

type SegregationRuleCommandResult struct {
	Rule              SegregationRule
	DecisionReference uuid.UUID
	PolicyReference   string
	Replayed          bool
}

type SegregationRuleService struct {
	repository  SegregationRuleRepository
	authorizer  SegregationRuleAuthorizer
	approval    SegregationRuleApprovalPort
	audit       SegregationRuleAuditRecorder
	clock       func() time.Time
	durable     *DurableSegregationRuleServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedSegregationRuleCommand
}

type storedSegregationRuleCommand struct {
	fingerprint string
	result      SegregationRuleCommandResult
}

func NewSegregationRuleService(repository SegregationRuleRepository, authorizer SegregationRuleAuthorizer, approval SegregationRuleApprovalPort, audit SegregationRuleAuditRecorder, clock func() time.Time) (*SegregationRuleService, error) {
	if repository == nil || authorizer == nil || approval == nil || audit == nil || clock == nil {
		return nil, ErrSegregationRuleUnavailable
	}
	if binder, ok := repository.(interface {
		BindSegregationRuleAuditRecorder(SegregationRuleAuditRecorder)
	}); ok {
		binder.BindSegregationRuleAuditRecorder(audit)
	}
	return &SegregationRuleService{repository: repository, authorizer: authorizer, approval: approval, audit: audit, clock: clock, idempotency: make(map[string]storedSegregationRuleCommand)}, nil
}

func (service *SegregationRuleService) Execute(ctx context.Context, actor ApplicationActor, command SegregationRuleCommand) (SegregationRuleCommandResult, error) {
	if service == nil {
		return SegregationRuleCommandResult{}, ErrSegregationRuleUnavailable
	}
	if service.durable != nil {
		return service.executeDurable(ctx, actor, command)
	}
	if err := actor.Validate(); err != nil {
		return SegregationRuleCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return SegregationRuleCommandResult{}, err
	}
	fingerprint, err := fingerprintSegregationRuleCommand(command)
	if err != nil {
		return SegregationRuleCommandResult{}, err
	}
	key := actor.UserID.String() + ":" + command.IdempotencyKey
	service.mu.Lock()
	defer service.mu.Unlock()
	if previous, exists := service.idempotency[key]; exists {
		if previous.fingerprint != fingerprint {
			return SegregationRuleCommandResult{}, ErrSegregationRuleIdempotency
		}
		result := cloneSegregationRuleCommandResult(previous.result)
		result.Replayed = true
		return result, nil
	}
	var current *SegregationRule
	if command.Action != SegregationRuleActionCreate {
		value, getErr := service.repository.Get(ctx, command.RuleID)
		if getErr != nil {
			return SegregationRuleCommandResult{}, getErr
		}
		if command.ExpectedVersion == nil || !value.Version.Matches(*command.ExpectedVersion) {
			return SegregationRuleCommandResult{}, ErrSegregationRuleVersion
		}
		current = &value
	}
	decision, err := service.authorizer.AuthorizeSegregationRuleManagement(ctx, actor, command, cloneSegregationRulePtr(current))
	if err != nil {
		return SegregationRuleCommandResult{}, err
	}
	if decision.Outcome == AuthorizationUnavailable {
		return SegregationRuleCommandResult{}, ErrSegregationRuleUnavailable
	}
	if decision.Outcome == AuthorizationExpired {
		return SegregationRuleCommandResult{}, ErrAuthorizationExpired
	}
	if decision.Outcome == AuthorizationStale {
		return SegregationRuleCommandResult{}, ErrSegregationRuleStale
	}
	if !decision.Allowed || decision.Permission != SegregationRuleManagementPermission {
		return SegregationRuleCommandResult{}, ErrSegregationRuleAuthorization
	}
	now := service.clock().UTC()
	var before, after SegregationRule
	if command.Action == SegregationRuleActionCreate {
		after, err = NewSegregationRule(uuid.New(), command.Code, command.Name, command.ConflictingPermissions, command.EnforcementMode, command.ScopeIDs, command.AmountThreshold, command.CoolingOff, command.EffectiveFrom, command.Approval, now)
		after.EffectiveTo = cloneTime(command.EffectiveTo)
	} else {
		before = cloneSegregationRule(*current)
		after = cloneSegregationRule(*current)
		if command.Action == SegregationRuleActionRetire {
			err = after.Retire(command.Approval, now)
		} else {
			err = after.Replace(command, command.Approval, now)
		}
	}
	if err != nil {
		return SegregationRuleCommandResult{}, err
	}
	if command.Action != SegregationRuleActionCreate {
		after.Version, err = after.Version.Advance()
		if err != nil {
			return SegregationRuleCommandResult{}, err
		}
	}
	if err := after.Validate(); err != nil {
		return SegregationRuleCommandResult{}, err
	}
	candidateFingerprint := candidateSegregationRuleFingerprint(command, after)
	if err := service.approval.ValidateSegregationRuleApproval(ctx, actor, command, cloneSegregationRulePtr(current), candidateFingerprint); err != nil {
		return SegregationRuleCommandResult{}, err
	}
	audit := SegregationRuleAuditRecord{RuleID: after.ID, ActorUserID: actor.UserID, ActorAuthenticationRef: fingerprintSubject(actor.Subject), Action: command.Action, Code: after.Code, ScopeIDs: append([]string(nil), after.ScopeIDs...), PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, ApprovalRequestID: command.Approval.ApprovalRequestID, ApprovalDecisionID: command.Approval.DecisionID, ApproverUserID: command.Approval.ApproverUserID, RuleVersion: after.Version.Value(), BeforeFingerprint: fingerprintSegregationRule(before), AfterFingerprint: candidateFingerprint, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	result := SegregationRuleCommandResult{Rule: cloneSegregationRule(after), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference}
	if err := service.repository.CommitSegregationRuleMutation(ctx, SegregationRuleMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: audit}); err != nil {
		return SegregationRuleCommandResult{}, err
	}
	service.idempotency[key] = storedSegregationRuleCommand{fingerprint: fingerprint, result: cloneSegregationRuleCommandResult(result)}
	return result, nil
}

type MemorySegregationRuleRepository struct {
	mu    sync.RWMutex
	rules map[uuid.UUID]SegregationRule
	audit SegregationRuleAuditRecorder
}

func NewMemorySegregationRuleRepository(rules ...SegregationRule) (*MemorySegregationRuleRepository, error) {
	repository := &MemorySegregationRuleRepository{rules: make(map[uuid.UUID]SegregationRule)}
	for _, rule := range rules {
		if err := rule.Validate(); err != nil {
			return nil, err
		}
		repository.rules[rule.ID] = cloneSegregationRule(rule)
	}
	return repository, nil
}

func (repository *MemorySegregationRuleRepository) BindSegregationRuleAuditRecorder(audit SegregationRuleAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemorySegregationRuleRepository) List(_ context.Context) ([]SegregationRule, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]SegregationRule, 0, len(repository.rules))
	for _, rule := range repository.rules {
		result = append(result, cloneSegregationRule(rule))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID.String() < result[right].ID.String() })
	return result, nil
}

func (repository *MemorySegregationRuleRepository) Get(_ context.Context, id uuid.UUID) (SegregationRule, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	rule, ok := repository.rules[id]
	if !ok {
		return SegregationRule{}, ErrSegregationRuleNotFound
	}
	return cloneSegregationRule(rule), nil
}

func (repository *MemorySegregationRuleRepository) CommitSegregationRuleMutation(ctx context.Context, mutation SegregationRuleMutation) error {
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.rules[mutation.After.ID]; exists {
			return ErrSegregationRuleIdempotency
		}
	} else {
		current, exists := repository.rules[mutation.After.ID]
		if !exists || mutation.ExpectedVersion == nil || !current.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 {
			return ErrSegregationRuleVersion
		}
	}
	if repository.audit == nil {
		return ErrSegregationRuleAudit
	}
	if err := repository.audit.RecordSegregationRuleMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.rules[mutation.After.ID] = cloneSegregationRule(mutation.After)
	return nil
}

type MemorySegregationRuleAuditRecorder struct {
	mu      sync.Mutex
	Records []SegregationRuleAuditRecord
	Err     error
}

func (recorder *MemorySegregationRuleAuditRecorder) RecordSegregationRuleMutation(_ context.Context, record SegregationRuleAuditRecord) error {
	if recorder == nil {
		return ErrSegregationRuleAudit
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

type MemorySegregationRuleAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemorySegregationRuleAuthorizer) AuthorizeSegregationRuleManagement(context.Context, ApplicationActor, SegregationRuleCommand, *SegregationRule) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return cloneAuthorizationDecision(authorizer.Decision), nil
}

func cloneSegregationRulePtr(rule *SegregationRule) *SegregationRule {
	if rule == nil {
		return nil
	}
	copy := cloneSegregationRule(*rule)
	return &copy
}

func cloneSegregationRuleCommandResult(result SegregationRuleCommandResult) SegregationRuleCommandResult {
	result.Rule = cloneSegregationRule(result.Rule)
	return result
}

func fingerprintSegregationRuleCommand(command SegregationRuleCommand) (string, error) {
	payload := struct {
		Action          string
		RuleID          uuid.UUID
		Code            string
		Name            string
		Permissions     []string
		Mode            SegregationEnforcementMode
		Scopes          []string
		Threshold       *decimal.Decimal
		CoolingOff      int64
		EffectiveFrom   time.Time
		EffectiveTo     *time.Time
		Approval        ApprovalDecisionReference
		ExpectedVersion int64
	}{command.Action, command.RuleID, strings.TrimSpace(command.Code), strings.TrimSpace(command.Name), command.ConflictingPermissions, command.EnforcementMode, command.ScopeIDs, command.AmountThreshold, int64(command.CoolingOff), command.EffectiveFrom, command.EffectiveTo, command.Approval, 0}
	if command.ExpectedVersion != nil {
		payload.ExpectedVersion = command.ExpectedVersion.Value()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func candidateSegregationRuleFingerprint(command SegregationRuleCommand, rule SegregationRule) string {
	data, _ := json.Marshal(struct {
		Action        string
		ID            uuid.UUID
		Code          string
		Name          string
		Permissions   []string
		Mode          SegregationEnforcementMode
		Scopes        []string
		Threshold     *decimal.Decimal
		CoolingOff    int64
		EffectiveFrom time.Time
		EffectiveTo   *time.Time
		Version       int64
	}{
		Action: command.Action, ID: segregationCandidateID(command, rule), Code: rule.Code, Name: rule.Name,
		Permissions: rule.ConflictingPermissions, Mode: rule.EnforcementMode, Scopes: rule.ScopeIDs,
		Threshold: rule.AmountThreshold, CoolingOff: int64(rule.CoolingOff), EffectiveFrom: rule.EffectiveFrom,
		EffectiveTo: rule.EffectiveTo, Version: rule.Version.Value(),
	})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func segregationCandidateID(command SegregationRuleCommand, rule SegregationRule) uuid.UUID {
	if command.Action == SegregationRuleActionCreate {
		return uuid.Nil
	}
	return rule.ID
}

func fingerprintSegregationRule(rule SegregationRule) string {
	if rule.ID == uuid.Nil {
		return ""
	}
	data, _ := json.Marshal(struct {
		ID          uuid.UUID
		Code        string
		Name        string
		Status      SegregationRuleStatus
		Permissions []string
		Mode        SegregationEnforcementMode
		Version     int64
	}{rule.ID, rule.Code, rule.Name, rule.Status, rule.ConflictingPermissions, rule.EnforcementMode, rule.Version.Value()})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

var _ SegregationRuleStore = (*MemorySegregationRuleRepository)(nil)
var _ SegregationDecisionPort = (*SegregationEvaluator)(nil)
var _ SegregationRuleRepository = (*MemorySegregationRuleRepository)(nil)
var _ RoleSegregationPort = (*SegregationEvaluator)(nil)
