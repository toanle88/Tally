package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ApprovalSubjectSnapshot is the bounded, review-oriented evidence stored on
// an approval request. Source contexts remain responsible for deciding which
// values are safe to publish into this projection.
type ApprovalSubjectSnapshot struct {
	Label     string           `json:"label"`
	Reference string           `json:"reference,omitempty"`
	Summary   string           `json:"summary,omitempty"`
	Amount    *decimal.Decimal `json:"amount,omitempty"`
	Currency  string           `json:"currency,omitempty"`
}

func (snapshot ApprovalSubjectSnapshot) Validate() error {
	if strings.TrimSpace(snapshot.Label) == "" || len(snapshot.Label) > 240 {
		return fmt.Errorf("%w: approval subject snapshot label is required", ErrInvalidApprovalPolicyCommand)
	}
	if len(snapshot.Reference) > 240 || len(snapshot.Summary) > 2000 {
		return fmt.Errorf("%w: approval subject snapshot text is too long", ErrInvalidApprovalPolicyCommand)
	}
	if snapshot.Amount != nil && snapshot.Amount.IsNegative() {
		return fmt.Errorf("%w: approval subject snapshot amount cannot be negative", ErrInvalidApprovalPolicyCommand)
	}
	if snapshot.Currency != "" && !currencyCodePattern.MatchString(snapshot.Currency) {
		return fmt.Errorf("%w: approval subject snapshot currency is invalid", ErrInvalidApprovalPolicyCommand)
	}
	if snapshot.Amount != nil && snapshot.Currency == "" {
		return fmt.Errorf("%w: approval subject snapshot amount requires currency", ErrInvalidApprovalPolicyCommand)
	}
	return nil
}

func (snapshot ApprovalSubjectSnapshot) Fingerprint() string {
	if strings.TrimSpace(snapshot.Label) == "" && snapshot.Reference == "" && snapshot.Summary == "" && snapshot.Amount == nil && snapshot.Currency == "" {
		return ""
	}
	return fingerprint(snapshot)
}

type ApprovalPolicySelection struct {
	Applicability ApprovalApplicability
	Scope         ApprovalPolicyScope
	Amount        *decimal.Decimal
	Currency      string
	AsOf          time.Time
}

type ApprovalPolicyResolver interface {
	Resolve(context.Context, ApprovalPolicySelection) (ApprovalPolicy, error)
}

type EffectiveApprovalPolicyResolver struct {
	repository ApprovalPolicyRepository
	clock      func() time.Time
}

func NewEffectiveApprovalPolicyResolver(repository ApprovalPolicyRepository, clock func() time.Time) (*EffectiveApprovalPolicyResolver, error) {
	if repository == nil || clock == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	return &EffectiveApprovalPolicyResolver{repository: repository, clock: clock}, nil
}

func (resolver *EffectiveApprovalPolicyResolver) Resolve(ctx context.Context, selection ApprovalPolicySelection) (ApprovalPolicy, error) {
	if resolver == nil || resolver.repository == nil {
		return ApprovalPolicy{}, ErrApprovalPolicySubjectUnavailable
	}
	if err := selection.Applicability.Validate(); err != nil {
		return ApprovalPolicy{}, err
	}
	if err := selection.Scope.Validate(); err != nil {
		return ApprovalPolicy{}, err
	}
	if selection.Amount == nil || selection.Amount.IsNegative() {
		return ApprovalPolicy{}, fmt.Errorf("%w: approval policy selection amount is required", ErrInvalidApprovalPolicyCommand)
	}
	if !currencyCodePattern.MatchString(selection.Currency) {
		return ApprovalPolicy{}, fmt.Errorf("%w: approval policy selection currency is invalid", ErrInvalidApprovalPolicyCommand)
	}
	asOf := selection.AsOf.UTC()
	if asOf.IsZero() {
		asOf = resolver.clock().UTC()
	}
	policies, err := resolver.repository.ListSelectable(ctx)
	if err != nil {
		return ApprovalPolicy{}, fmt.Errorf("%w: %v", ErrApprovalPolicyResolverUnavailable, err)
	}
	matches := make([]ApprovalPolicy, 0, len(policies))
	for _, policy := range policies {
		if approvalPolicyMatches(policy, selection, asOf) {
			matches = append(matches, policy)
		}
	}
	switch len(matches) {
	case 0:
		return ApprovalPolicy{}, ErrApprovalPolicyNotApplicable
	case 1:
		return cloneApprovalPolicy(matches[0]), nil
	default:
		return ApprovalPolicy{}, ErrApprovalPolicyAmbiguous
	}
}

func approvalPolicyMatches(policy ApprovalPolicy, selection ApprovalPolicySelection, asOf time.Time) bool {
	if policy.Status != ApprovalPolicyStatusActive || asOf.Before(policy.EffectiveFrom.UTC()) {
		return false
	}
	if policy.EffectiveTo != nil && asOf.After(policy.EffectiveTo.UTC()) {
		return false
	}
	if policy.Applicability.SourceContext != selection.Applicability.SourceContext || policy.Applicability.SubjectType != selection.Applicability.SubjectType || policy.Applicability.Operation != selection.Applicability.Operation {
		return false
	}
	if !selectorValuesMatch(policy.Applicability.TransactionTypes, selection.Applicability.TransactionTypes) || !selectorValuesMatch(policy.Applicability.Currencies, []string{selection.Currency}) || !selectorValuesMatch(policy.Applicability.DataClassifications, selection.Applicability.DataClassifications) {
		return false
	}
	for key, value := range policy.Applicability.Dimensions {
		if selection.Applicability.Dimensions[key] != value {
			return false
		}
	}
	if !policy.Scope.Contains(selection.Scope.CanonicalScopeIDs()) {
		return false
	}
	for _, rule := range policy.Rules {
		threshold := rule.Threshold.normalized()
		if threshold.Currency != selection.Currency {
			continue
		}
		if threshold.Minimum != nil && selection.Amount.LessThan(*threshold.Minimum) {
			continue
		}
		if threshold.Maximum != nil && selection.Amount.GreaterThan(*threshold.Maximum) {
			continue
		}
		return true
	}
	return false
}

func selectorValuesMatch(policyValues, requestedValues []string) bool {
	if len(policyValues) == 0 {
		return true
	}
	for _, policyValue := range policyValues {
		for _, requestedValue := range requestedValues {
			if policyValue == requestedValue {
				return true
			}
		}
	}
	return false
}

func cloneSubjectSnapshot(value ApprovalSubjectSnapshot) ApprovalSubjectSnapshot {
	value.Amount = cloneDecimal(value.Amount)
	return value
}

func cloneDecimal(value *decimal.Decimal) *decimal.Decimal {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
