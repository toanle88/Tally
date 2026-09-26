package identity

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

// AccessObservationOperation is the closed set of sensitive-access attempts
// that IAM can describe through its audit boundary. It is intentionally not a
// public API or an event name.
type AccessObservationOperation string

const (
	AccessObservationView             AccessObservationOperation = "view"
	AccessObservationReveal           AccessObservationOperation = "reveal"
	AccessObservationExport           AccessObservationOperation = "export"
	AccessObservationPrivilegedSearch AccessObservationOperation = "privileged-search"
)

const (
	AccessObservationClassificationPublic           = "public"
	AccessObservationClassificationInternal         = "internal"
	AccessObservationClassificationConfidential     = "confidential"
	AccessObservationClassificationHighlyRestricted = "highly_restricted"
)

var (
	ErrInvalidAccessObservation  = errors.New("invalid identity access observation")
	ErrAccessEvidenceUnavailable = errors.New("identity access evidence unavailable")
	ErrAccessEvidenceRecorder    = errors.New("identity access evidence recorder unavailable")
	accessReferencePattern       = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/-]{0,199}$`)
	fingerprintPattern           = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

// AccessObservation contains metadata only. Raw values, credentials, token
// claims, and policy payloads are not fields in this contract.
type AccessObservation struct {
	ActorUserID            uuid.UUID
	ActorAuthenticationRef string
	Operation              AccessObservationOperation
	TargetReference        string
	EvidenceReference      string
	ScopeReference         string
	Purpose                string
	FilterReference        string
	DataClassification     string
	Permission             string
	Outcome                AuthorizationOutcome
	PolicyReference        string
	PolicyVersion          string
	DecisionReference      uuid.UUID
	GrantReference         uuid.UUID
	CorrelationReference   string
	CausationReference     string
	BeforeFingerprint      string
	AfterFingerprint       string
}

// AccessObservationRecorder is the IAM-owned port into the approved audit
// boundary. The implementation remains outside this package and can be
// transactional without IAM reaching another module's schema.
type AccessObservationRecorder interface {
	RecordAccessObservation(context.Context, AccessObservation) (uuid.UUID, error)
}

type AccessEvidenceService struct {
	recorder AccessObservationRecorder
}

func NewAccessEvidenceService(recorder AccessObservationRecorder) (*AccessEvidenceService, error) {
	if recorder == nil {
		return nil, ErrAccessEvidenceRecorder
	}
	return &AccessEvidenceService{recorder: recorder}, nil
}

// RecordAccess validates and records an attempt before a sensitive value may
// be revealed or exported. It fails closed when the audit dependency fails.
func (service *AccessEvidenceService) RecordAccess(ctx context.Context, actor ApplicationActor, observation AccessObservation) (uuid.UUID, error) {
	if service == nil || service.recorder == nil {
		return uuid.Nil, ErrAccessEvidenceUnavailable
	}
	if err := actor.Validate(); err != nil {
		return uuid.Nil, err
	}
	observation.ActorUserID = actor.UserID
	observation.ActorAuthenticationRef = fingerprintSubject(actor.Subject)
	if err := observation.Validate(); err != nil {
		return uuid.Nil, err
	}
	auditReference, err := service.recorder.RecordAccessObservation(ctx, observation)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %w", ErrAccessEvidenceUnavailable, err)
	}
	if auditReference == uuid.Nil {
		return uuid.Nil, ErrAccessEvidenceUnavailable
	}
	return auditReference, nil
}

func (observation AccessObservation) Validate() error {
	if observation.ActorUserID == uuid.Nil {
		return fmt.Errorf("%w: actor is required", ErrInvalidAccessObservation)
	}
	if !fingerprintPattern.MatchString(observation.ActorAuthenticationRef) {
		return fmt.Errorf("%w: actor authentication reference must be a fingerprint", ErrInvalidAccessObservation)
	}
	switch observation.Operation {
	case AccessObservationView, AccessObservationReveal, AccessObservationExport, AccessObservationPrivilegedSearch:
	default:
		return fmt.Errorf("%w: unsupported operation", ErrInvalidAccessObservation)
	}
	for name, value := range map[string]string{
		"target reference":   observation.TargetReference,
		"evidence reference": observation.EvidenceReference,
		"scope reference":    observation.ScopeReference,
		"purpose":            observation.Purpose,
		"permission":         observation.Permission,
	} {
		if err := validateSafeReference(name, value); err != nil {
			return err
		}
	}
	if observation.FilterReference != "" {
		if err := validateSafeReference("filter reference", observation.FilterReference); err != nil {
			return err
		}
	}
	if observation.DataClassification != AccessObservationClassificationPublic &&
		observation.DataClassification != AccessObservationClassificationInternal &&
		observation.DataClassification != AccessObservationClassificationConfidential &&
		observation.DataClassification != AccessObservationClassificationHighlyRestricted {
		return fmt.Errorf("%w: unsupported data classification", ErrInvalidAccessObservation)
	}
	switch observation.Outcome {
	case AuthorizationAllowed, AuthorizationDenied, AuthorizationExpired, AuthorizationStale, AuthorizationUnavailable:
	default:
		return fmt.Errorf("%w: unsupported outcome", ErrInvalidAccessObservation)
	}
	if observation.DecisionReference == uuid.Nil {
		return fmt.Errorf("%w: decision reference is required", ErrInvalidAccessObservation)
	}
	for name, value := range map[string]string{
		"policy reference":      observation.PolicyReference,
		"policy version":        observation.PolicyVersion,
		"correlation reference": observation.CorrelationReference,
		"causation reference":   observation.CausationReference,
	} {
		if value != "" {
			if err := validateSafeReference(name, value); err != nil {
				return err
			}
		}
	}
	for name, value := range map[string]string{
		"before fingerprint": observation.BeforeFingerprint,
		"after fingerprint":  observation.AfterFingerprint,
	} {
		if value != "" && !fingerprintPattern.MatchString(value) {
			return fmt.Errorf("%w: %s must be a fingerprint", ErrInvalidAccessObservation, name)
		}
	}
	return nil
}

func validateSafeReference(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" || !accessReferencePattern.MatchString(value) || containsSensitiveMarker(value) {
		return fmt.Errorf("%w: %s must be a bounded safe reference", ErrInvalidAccessObservation, name)
	}
	return nil
}

func containsSensitiveMarker(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"password", "secret", "token", "credential", "bank", "payroll", "tax", "raw-policy"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// AccessDecisionExplanation is the safe projection shown to callers. The
// evaluator's human reason text is intentionally not copied into it.
type AccessDecisionExplanation struct {
	Outcome              AuthorizationOutcome
	Permission           string
	PolicyReference      string
	PolicyVersion        string
	DecisionReference    uuid.UUID
	ReasonCode           string
	ApplicableDimensions []string
	NextAction           string
}

func ExplainAuthorizationDecision(decision AuthorizationDecision) AccessDecisionExplanation {
	return AccessDecisionExplanation{
		Outcome:              decision.Outcome,
		Permission:           safeExplanationReference(decision.Permission),
		PolicyReference:      safeExplanationReference(decision.PolicyReference),
		PolicyVersion:        safeExplanationReference(decision.PolicyVersion),
		DecisionReference:    decision.DecisionReference,
		ReasonCode:           safeAuthorizationReasonCode(decision.ReasonCode),
		ApplicableDimensions: safeAuthorizationDimensions(decision.ApplicableDimensions),
		NextAction:           authorizationNextAction(decision.Outcome, decision.ReasonCode),
	}
}

func safeExplanationReference(value string) string {
	if value == "" || validateSafeReference("explanation reference", value) != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func safeAuthorizationReasonCode(value string) string {
	switch value {
	case AuthorizationReasonAllowed, AuthorizationReasonDefaultDeny, AuthorizationReasonMissingDimension,
		AuthorizationReasonScopeMismatch, AuthorizationReasonPolicyExpired, AuthorizationReasonPolicyUnavailable,
		AuthorizationReasonPolicyVersionStale, AuthorizationReasonPolicyNotApplicable:
		return value
	default:
		return ""
	}
}

func safeAuthorizationDimensions(values []string) []string {
	allowed := map[string]struct{}{
		AuthorizationDimensionAction: {}, AuthorizationDimensionActor: {}, AuthorizationDimensionAccountingScope: {},
		AuthorizationDimensionLegalEntity: {}, AuthorizationDimensionSegment: {}, AuthorizationDimensionAccount: {},
		AuthorizationDimensionAccountClass: {}, AuthorizationDimensionTransactionType: {}, AuthorizationDimensionAmount: {},
		AuthorizationDimensionCurrency: {}, AuthorizationDimensionFiscalPeriod: {}, AuthorizationDimensionDataClassification: {},
		AuthorizationDimensionSubjectActor: {}, AuthorizationDimensionScope: {},
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; ok {
			result = append(result, value)
		}
	}
	return result
}

func authorizationNextAction(outcome AuthorizationOutcome, reasonCode string) string {
	switch outcome {
	case AuthorizationAllowed:
		return "Continue with the permitted action."
	case AuthorizationExpired:
		return "Request a current authorization decision."
	case AuthorizationStale:
		return "Refresh policy state and retry."
	case AuthorizationUnavailable:
		return "Retry after the authorization dependency is available."
	case AuthorizationDenied:
		if reasonCode == AuthorizationReasonScopeMismatch {
			return "Request an approved scope or use an authorized scope."
		}
		return "Request an approved permission or scope."
	default:
		return "Review the current authorization state."
	}
}

type SegregationDecisionExplanation struct {
	Outcome           AuthorizationOutcome
	RuleReference     uuid.UUID
	RuleCode          string
	RuleVersion       aggregateversion.AggregateVersion
	PolicyVersion     string
	DecisionReference uuid.UUID
	ReasonCode        string
	NextAction        string
}

func ExplainSegregationDecision(decision SegregationDecision) SegregationDecisionExplanation {
	return SegregationDecisionExplanation{
		Outcome: decision.Outcome, RuleReference: decision.RuleReference, RuleCode: safeExplanationReference(decision.RuleCode),
		RuleVersion: decision.RuleVersion, PolicyVersion: safeExplanationReference(decision.PolicyVersion),
		DecisionReference: decision.DecisionReference, ReasonCode: safeSegregationReasonCode(decision.ReasonCode),
		NextAction: segregationNextAction(decision.Outcome, decision.ReasonCode),
	}
}

func safeSegregationReasonCode(value string) string {
	switch value {
	case SegregationReasonAllowed, SegregationReasonConflict, SegregationReasonExpiredException,
		SegregationReasonStale, SegregationReasonUnavailable:
		return value
	default:
		return ""
	}
}

func segregationNextAction(outcome AuthorizationOutcome, reasonCode string) string {
	switch outcome {
	case AuthorizationAllowed:
		return "Continue with the permitted action."
	case AuthorizationStale:
		return "Refresh the current segregation rule and retry."
	case AuthorizationUnavailable:
		return "Retry after the segregation dependency is available."
	case AuthorizationExpired:
		return "Request a new independently approved exception."
	case AuthorizationDenied:
		if reasonCode == SegregationReasonExpiredException {
			return "Request a new independently approved exception."
		}
		return "Use an independent actor or request an approved exception."
	default:
		return "Review the current segregation state."
	}
}

// MemoryAccessObservationRecorder is a test/local fixture implementation. A
// production audit adapter belongs to the audit context and must preserve its
// transaction boundary.
type MemoryAccessObservationRecorder struct {
	mu      sync.Mutex
	records []AccessObservation
	Err     error
}

func (recorder *MemoryAccessObservationRecorder) RecordAccessObservation(_ context.Context, observation AccessObservation) (uuid.UUID, error) {
	if recorder == nil {
		return uuid.Nil, ErrAccessEvidenceUnavailable
	}
	if recorder.Err != nil {
		return uuid.Nil, recorder.Err
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.records = append(recorder.records, observation)
	return uuid.New(), nil
}

func (recorder *MemoryAccessObservationRecorder) Records() []AccessObservation {
	if recorder == nil {
		return nil
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return append([]AccessObservation(nil), recorder.records...)
}
