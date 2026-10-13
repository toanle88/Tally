package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

type ApprovalPolicyAuthorizer interface {
	AuthorizeApprovalPolicy(context.Context, Actor, ApprovalPolicyCommand, *ApprovalPolicy) (AuthorizationDecision, error)
}

type ApprovalPolicyAuditRecord struct {
	PolicyID                      uuid.UUID
	ActorUserID                   uuid.UUID
	ActorAuthenticationSubjectRef string
	ApproverUserID                uuid.UUID
	Action                        string
	ScopeIDs                      []string
	Reason                        string
	Permission                    string
	PolicyReference               string
	PolicyVersion                 string
	RevisionVersion               int64
	PreviousPolicyVersion         string
	DecisionReference             uuid.UUID
	ApprovalRequestID             uuid.UUID
	ApprovalDecisionID            uuid.UUID
	EffectiveFrom                 time.Time
	EffectiveTo                   *time.Time
	BeforeFingerprint             string
	AfterFingerprint              string
	CorrelationID                 string
	CausationID                   string
}

type ApprovalPolicyAuditRecorder interface {
	RecordApprovalPolicyMutation(context.Context, ApprovalPolicyAuditRecord) (uuid.UUID, error)
}

type ApprovalPolicyMutation struct {
	Before          *ApprovalPolicy
	After           ApprovalPolicy
	Audit           ApprovalPolicyAuditRecord
	ApprovalRequest *PreparedApprovalRequest
}

type ApprovalPolicyRepository interface {
	Get(context.Context, uuid.UUID) (ApprovalPolicy, error)
	List(context.Context) ([]ApprovalPolicy, error)
	ListSelectable(context.Context) ([]ApprovalPolicy, error)
	Commit(context.Context, ApprovalPolicyMutation) error
}

type ApprovalPolicyCommitResultRepository interface {
	CommitWithResult(context.Context, ApprovalPolicyMutation) (uuid.UUID, error)
}

type DurableApprovalPolicyServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type DurableApprovalPolicyMutationCommit struct {
	Coordinator   platformidempotency.DurableCoordinator
	Acquisition   platformidempotency.DurableAcquisition
	Result        platformidempotency.CommandResultMetadata
	ResultBuilder func(uuid.UUID) (platformidempotency.CommandResultMetadata, error)
}

type DurableApprovalPolicyRepository interface {
	CommitWithIdempotency(context.Context, ApprovalPolicyMutation, DurableApprovalPolicyMutationCommit) (uuid.UUID, error)
}

type ApprovalPolicyCommand struct {
	Action                  string
	PolicyID                uuid.UUID
	Code                    string
	Name                    string
	PolicyVersion           string
	Applicability           ApprovalApplicability
	Scope                   ApprovalPolicyScope
	Rules                   []ApprovalRule
	Steps                   []ApprovalStepDefinition
	EffectiveFrom           time.Time
	EffectiveTo             *time.Time
	Approval                *ApprovalDecisionReference
	ExpectedVersion         *aggregateversion.AggregateVersion
	SubmitForApproval       bool
	Reason                  string
	DependentWorkflowImpact string
	IdempotencyKey          string
	CorrelationID           string
	CausationID             string
}

func (command ApprovalPolicyCommand) normalizedAction() string {
	switch strings.TrimSpace(command.Action) {
	case ApprovalPolicyActionMaintain:
		return ApprovalPolicyActionUpdate
	case "submit_for_approval", "submit-for-approval", "submit":
		return ApprovalPolicyActionSubmitForApproval
	default:
		return strings.TrimSpace(command.Action)
	}
}

func (command ApprovalPolicyCommand) Validate() error {
	action := command.normalizedAction()
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidApprovalPolicyCommand)
	}
	switch action {
	case ApprovalPolicyActionCreate:
		if strings.TrimSpace(command.Code) == "" || strings.TrimSpace(command.Name) == "" || strings.TrimSpace(command.PolicyVersion) == "" {
			return fmt.Errorf("%w: policy identity is required", ErrInvalidApprovalPolicyCommand)
		}
		if err := validatePolicyContent(command.Applicability, command.Scope, command.Rules, command.Steps, command.EffectiveFrom, command.EffectiveTo); err != nil {
			return err
		}
	case ApprovalPolicyActionUpdate:
		if command.PolicyID == uuid.Nil || command.ExpectedVersion == nil {
			return fmt.Errorf("%w: update requires policy id and expected version", ErrInvalidApprovalPolicyCommand)
		}
		if strings.TrimSpace(command.PolicyVersion) == "" {
			return fmt.Errorf("%w: update requires a new policy version", ErrInvalidApprovalPolicyCommand)
		}
		if hasPolicyContent(command) {
			if err := validatePolicyContent(command.Applicability, command.Scope, command.Rules, command.Steps, command.EffectiveFrom, command.EffectiveTo); err != nil {
				return err
			}
		}
	case ApprovalPolicyActionSubmitForApproval, ApprovalPolicyActionActivate:
		if command.PolicyID == uuid.Nil || command.ExpectedVersion == nil {
			return fmt.Errorf("%w: lifecycle action requires policy id and expected version", ErrInvalidApprovalPolicyCommand)
		}
		if action == ApprovalPolicyActionActivate {
			if command.Approval == nil {
				return ErrApprovalPolicyApprovalRequired
			}
			if err := command.Approval.Validate(); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w: unsupported action %q", ErrInvalidApprovalPolicyCommand, command.Action)
	}
	return nil
}

func validatePolicyContent(applicability ApprovalApplicability, scope ApprovalPolicyScope, rules []ApprovalRule, steps []ApprovalStepDefinition, effectiveFrom time.Time, effectiveTo *time.Time) error {
	policy, err := NewApprovalPolicy(uuid.New(), "policy-validation", "Policy validation", "validation", applicability, scope, rules, steps, effectiveFrom, effectiveTo, time.Now())
	if err != nil {
		return err
	}
	_ = policy
	return nil
}

func hasPolicyContent(command ApprovalPolicyCommand) bool {
	return strings.TrimSpace(command.Code) != "" ||
		strings.TrimSpace(command.Name) != "" ||
		command.Applicability.SourceContext != "" ||
		command.Applicability.SubjectType != "" ||
		command.Applicability.Operation != "" ||
		len(command.Applicability.TransactionTypes) > 0 ||
		len(command.Applicability.Currencies) > 0 ||
		len(command.Applicability.DataClassifications) > 0 ||
		len(command.Applicability.Dimensions) > 0 ||
		len(command.Scope.ScopeIDs) > 0 ||
		len(command.Scope.AccountingScopeIDs) > 0 ||
		len(command.Scope.LegalEntityIDs) > 0 ||
		len(command.Scope.BusinessUnitIDs) > 0 ||
		len(command.Scope.SegmentIDs) > 0 ||
		len(command.Scope.AccountIDs) > 0 ||
		len(command.Scope.FiscalPeriodIDs) > 0 ||
		len(command.Rules) > 0 ||
		len(command.Steps) > 0 ||
		!command.EffectiveFrom.IsZero() ||
		command.EffectiveTo != nil
}

type ApprovalPolicyCommandResult struct {
	Policy                  ApprovalPolicy
	ApprovalRequest         *ApprovalRequest
	ApprovalStatus          string
	ActivationOutcome       string
	DependentWorkflowImpact string
	DecisionReference       uuid.UUID
	AuditReference          uuid.UUID
	Replayed                bool
}

type ApprovalPolicyService struct {
	repository ApprovalPolicyRepository
	authorizer ApprovalPolicyAuthorizer
	workflow   ApprovalWorkflowPort
	audit      ApprovalPolicyAuditRecorder
	clock      func() time.Time
	durable    *DurableApprovalPolicyServiceConfig

	mu          sync.Mutex
	idempotency map[string]storedApprovalPolicyResult
}

type storedApprovalPolicyResult struct {
	fingerprint string
	result      ApprovalPolicyCommandResult
}

func NewApprovalPolicyService(repository ApprovalPolicyRepository, authorizer ApprovalPolicyAuthorizer, workflow ApprovalWorkflowPort, audit ApprovalPolicyAuditRecorder, clock func() time.Time) (*ApprovalPolicyService, error) {
	return newApprovalPolicyService(repository, authorizer, workflow, audit, clock, nil)
}

func NewApprovalPolicyServiceWithDurableIdempotency(repository ApprovalPolicyRepository, authorizer ApprovalPolicyAuthorizer, workflow ApprovalWorkflowPort, audit ApprovalPolicyAuditRecorder, clock func() time.Time, durable DurableApprovalPolicyServiceConfig) (*ApprovalPolicyService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidApprovalPolicyService
	}
	return newApprovalPolicyService(repository, authorizer, workflow, audit, clock, &durable)
}

func newApprovalPolicyService(repository ApprovalPolicyRepository, authorizer ApprovalPolicyAuthorizer, workflow ApprovalWorkflowPort, audit ApprovalPolicyAuditRecorder, clock func() time.Time, durable *DurableApprovalPolicyServiceConfig) (*ApprovalPolicyService, error) {
	if repository == nil || authorizer == nil || workflow == nil || audit == nil || clock == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	if binder, ok := repository.(interface {
		BindAuditRecorder(ApprovalPolicyAuditRecorder)
	}); ok {
		binder.BindAuditRecorder(audit)
	}
	return &ApprovalPolicyService{repository: repository, authorizer: authorizer, workflow: workflow, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedApprovalPolicyResult)}, nil
}

func (service *ApprovalPolicyService) Execute(ctx context.Context, actor Actor, command ApprovalPolicyCommand) (result ApprovalPolicyCommandResult, returnErr error) {
	var durableAcquisition platformidempotency.DurableAcquisition
	defer func() {
		if service != nil && returnErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, returnErr)
		}
	}()
	if service == nil {
		return ApprovalPolicyCommandResult{}, ErrInvalidApprovalPolicyService
	}
	if err := actor.Validate(); err != nil {
		return ApprovalPolicyCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return ApprovalPolicyCommandResult{}, err
	}
	commandFingerprint := approvalPolicyCommandFingerprint(command)
	key := actor.UserID.String() + ":" + command.IdempotencyKey
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scopeKey, err := json.Marshal(struct {
			Module  string    `json:"module"`
			ActorID uuid.UUID `json:"actorId"`
		}{Module: "workflow", ActorID: actor.UserID})
		if err != nil {
			return ApprovalPolicyCommandResult{}, err
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scopeKey), command.IdempotencyKey)
		if err != nil {
			return ApprovalPolicyCommandResult{}, err
		}
		durableAcquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(commandFingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return ApprovalPolicyCommandResult{}, ErrApprovalPolicyIdempotencyConflict
		}
		if err != nil {
			return ApprovalPolicyCommandResult{}, err
		}
		if durableAcquisition.Decision() == platformidempotency.DecisionReturn {
			switch durableAcquisition.Result().State() {
			case platformidempotency.StateFailed:
				return ApprovalPolicyCommandResult{}, durableApprovalPolicyFailureError(durableAcquisition.Result().ResultBody())
			case platformidempotency.StateInProgress:
				return ApprovalPolicyCommandResult{}, ErrApprovalPolicyCommandInProgress
			case platformidempotency.StateEstablished:
				stored, err := decodeApprovalPolicyCommandResult(durableAcquisition.Result().ResultBody())
				if err != nil {
					return ApprovalPolicyCommandResult{}, err
				}
				stored.Replayed = true
				return stored, nil
			}
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != commandFingerprint {
				return ApprovalPolicyCommandResult{}, ErrApprovalPolicyIdempotencyConflict
			}
			result := cloneApprovalPolicyResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}
	now := service.clock().UTC()
	action := command.normalizedAction()
	var current *ApprovalPolicy
	if action != ApprovalPolicyActionCreate {
		loaded, err := service.repository.Get(ctx, command.PolicyID)
		if err != nil {
			return ApprovalPolicyCommandResult{}, err
		}
		current = &loaded
		if command.ExpectedVersion == nil || !loaded.Version.Matches(*command.ExpectedVersion) {
			return ApprovalPolicyCommandResult{}, ErrApprovalPolicyVersionConflict
		}
	}
	decision, err := service.authorizer.AuthorizeApprovalPolicy(ctx, actor, command, cloneApprovalPolicyPtr(current))
	if err != nil {
		return ApprovalPolicyCommandResult{}, mapPolicyAuthorizationError(err)
	}
	scope := command.Scope
	if action == ApprovalPolicyActionSubmitForApproval || action == ApprovalPolicyActionActivate || (action == ApprovalPolicyActionUpdate && !hasPolicyContent(command)) {
		scope = current.Scope
	}
	if err := validatePolicyAuthorization(decision, scope, action); err != nil {
		return ApprovalPolicyCommandResult{}, err
	}

	var before *ApprovalPolicy
	var after ApprovalPolicy
	var approvalRequest *ApprovalRequest
	var preparedApprovalRequest *PreparedApprovalRequest
	var decisionReference uuid.UUID
	switch action {
	case ApprovalPolicyActionCreate:
		policyID := command.PolicyID
		if policyID == uuid.Nil {
			policyID = uuid.New()
		}
		after, err = NewApprovalPolicy(policyID, command.Code, command.Name, command.PolicyVersion, command.Applicability, command.Scope, command.Rules, command.Steps, command.EffectiveFrom, command.EffectiveTo, now)
		if err == nil && command.SubmitForApproval {
			err = service.submitCandidate(ctx, actor, &after, command, now, &approvalRequest, &preparedApprovalRequest)
		}
	case ApprovalPolicyActionUpdate:
		beforeValue := cloneApprovalPolicy(*current)
		before = &beforeValue
		if beforeValue.Status == ApprovalPolicyStatusSuperseded {
			err = fmt.Errorf("%w: superseded policy cannot be edited", ErrApprovalPolicyImmutable)
			break
		}
		if hasPolicyContent(command) {
			after = beforeValue
			after.Code = strings.TrimSpace(command.Code)
			after.Name = strings.TrimSpace(command.Name)
			after.Applicability = cloneApplicability(command.Applicability)
			after.Scope = cloneScope(command.Scope)
			after.Rules = cloneRules(command.Rules)
			after.Steps = cloneSteps(command.Steps)
			after.EffectiveFrom = command.EffectiveFrom.UTC()
			after.EffectiveTo = cloneTime(command.EffectiveTo)
			if err == nil {
				nextVersion, advanceErr := beforeValue.Version.Advance()
				err = advanceErr
				after.Version = nextVersion
			}
			after.PolicyVersion = strings.TrimSpace(command.PolicyVersion)
			if beforeValue.hasPolicyVersion(after.PolicyVersion) {
				err = fmt.Errorf("%w: a new policy revision requires a new policy version", ErrApprovalPolicyVersionConflict)
			}
			after.Status = ApprovalPolicyStatusDraft
			after.Approval = nil
			after.ApprovalRequestID = uuid.Nil
			after.ActivationOutcome = ApprovalPolicyActivationOutcomeNone
			after.UpdatedAt = now
			if err == nil {
				err = after.Validate()
			}
		} else {
			after, err = beforeValue.NewVersion(command.PolicyVersion, now)
		}
		if err == nil && command.SubmitForApproval {
			err = service.submitCandidate(ctx, actor, &after, command, now, &approvalRequest, &preparedApprovalRequest)
		}
	case ApprovalPolicyActionSubmitForApproval:
		beforeValue := cloneApprovalPolicy(*current)
		before = &beforeValue
		after = beforeValue
		requestID := uuid.New()
		err = after.SubmitForApproval(requestID, now)
		if err == nil {
			var created ApprovalRequest
			if preparer, ok := service.workflow.(ApprovalWorkflowPreparationPort); ok {
				prepared, prepareErr := preparer.PreparePolicyApproval(ctx, actor, after, command)
				err = prepareErr
				if err == nil {
					preparedApprovalRequest = &prepared
					created = prepared.Request
				}
			} else {
				created, err = service.workflow.CreatePolicyApproval(ctx, actor, after, command)
			}
			if err == nil {
				approvalRequest = &created
			}
		}
	case ApprovalPolicyActionActivate:
		beforeValue := cloneApprovalPolicy(*current)
		before = &beforeValue
		after = beforeValue
		if command.Approval == nil {
			err = ErrApprovalPolicyApprovalRequired
			break
		}
		if command.Approval.ApproverUserID == actor.UserID {
			err = ErrApprovalPolicySelfApproval
			break
		}
		err = service.workflow.ValidatePolicyApproval(ctx, actor, after, *command.Approval)
		if err == nil {
			err = after.Activate(*command.Approval, now)
			decisionReference = command.Approval.DecisionID
		}
	}
	if err != nil {
		return ApprovalPolicyCommandResult{}, err
	}
	if err := after.Validate(); err != nil {
		return ApprovalPolicyCommandResult{}, err
	}
	if command.DependentWorkflowImpact != "" {
		after.DependentWorkflowImpact = strings.TrimSpace(command.DependentWorkflowImpact)
	}
	if decision.DecisionReference != uuid.Nil && decisionReference == uuid.Nil {
		decisionReference = decision.DecisionReference
	}
	auditRecord := ApprovalPolicyAuditRecord{PolicyID: after.ID, ActorUserID: actor.UserID, ActorAuthenticationSubjectRef: actor.SubjectReference, Action: action, ScopeIDs: after.Scope.CanonicalScopeIDs(), Reason: command.Reason, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: after.PolicyVersion, RevisionVersion: after.Version.Value(), DecisionReference: decisionReference, ApprovalRequestID: after.ApprovalRequestID, EffectiveFrom: after.EffectiveFrom, EffectiveTo: cloneTime(after.EffectiveTo), BeforeFingerprint: fingerprintBeforePolicy(before), AfterFingerprint: policyStateFingerprint(after), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	if before != nil {
		auditRecord.PreviousPolicyVersion = before.PolicyVersion
	}
	if after.Approval != nil {
		auditRecord.ApprovalDecisionID = after.Approval.DecisionID
		auditRecord.ApproverUserID = after.Approval.ApproverUserID
	}
	mutation := ApprovalPolicyMutation{Before: before, After: after, Audit: auditRecord, ApprovalRequest: preparedApprovalRequest}
	var auditReference uuid.UUID
	if service.durable != nil {
		committer, ok := service.repository.(DurableApprovalPolicyRepository)
		if !ok {
			return ApprovalPolicyCommandResult{}, ErrInvalidApprovalPolicyService
		}
		result = ApprovalPolicyCommandResult{Policy: cloneApprovalPolicy(after), ApprovalRequest: cloneApprovalRequestPtr(approvalRequest), ApprovalStatus: after.Status, ActivationOutcome: after.ActivationOutcome, DependentWorkflowImpact: after.DependentWorkflowImpact, DecisionReference: decisionReference}
		aggregateID := after.ID
		resultForDurable := cloneApprovalPolicyResult(result)
		resultBuilder := func(auditReference uuid.UUID) (platformidempotency.CommandResultMetadata, error) {
			resultForDurable.AuditReference = auditReference
			body, marshalErr := json.Marshal(resultForDurable)
			if marshalErr != nil {
				return platformidempotency.CommandResultMetadata{}, marshalErr
			}
			status := httpStatusOK
			return platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(commandFingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		}
		metadata, metadataErr := resultBuilder(uuid.Nil)
		if metadataErr != nil {
			return ApprovalPolicyCommandResult{}, metadataErr
		}
		auditReference, err = committer.CommitWithIdempotency(ctx, mutation, DurableApprovalPolicyMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: durableAcquisition, Result: metadata, ResultBuilder: resultBuilder})
		if err != nil {
			return ApprovalPolicyCommandResult{}, err
		}
	} else if committer, ok := service.repository.(ApprovalPolicyCommitResultRepository); ok {
		auditReference, err = committer.CommitWithResult(ctx, mutation)
		if err != nil {
			return ApprovalPolicyCommandResult{}, err
		}
	} else if err := service.repository.Commit(ctx, mutation); err != nil {
		return ApprovalPolicyCommandResult{}, err
	}
	result.Policy = cloneApprovalPolicy(after)
	result.ApprovalRequest = cloneApprovalRequestPtr(approvalRequest)
	result.ApprovalStatus = after.Status
	result.ActivationOutcome = after.ActivationOutcome
	result.DependentWorkflowImpact = after.DependentWorkflowImpact
	result.DecisionReference = decisionReference
	result.AuditReference = auditReference
	if service.durable == nil {
		service.idempotency[key] = storedApprovalPolicyResult{fingerprint: commandFingerprint, result: cloneApprovalPolicyResult(result)}
	}
	return result, nil
}

const httpStatusOK = 200

func approvalPolicyCommandFingerprint(command ApprovalPolicyCommand) string {
	command.IdempotencyKey = ""
	command.CorrelationID = ""
	command.CausationID = ""
	return fingerprint(command)
}

func (service *ApprovalPolicyService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code, ok := durableApprovalPolicyFailureCode(commandErr)
	if !ok {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{code})
	if err != nil {
		return
	}
	metadata, err := platformidempotency.NewCommandResultMetadata(acquisition.Result().Identity(), acquisition.Result().Fingerprint(), acquisition.Result().OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
	if err != nil {
		return
	}
	tx, err := service.durable.Database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := service.durable.Coordinator.Finalize(ctx, tx, acquisition, metadata); err != nil {
		return
	}
	if err := tx.Commit(ctx); err != nil {
		return
	}
	committed = true
}

func durableApprovalPolicyFailureCode(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrApprovalPolicyAuthorizationDenied):
		return "AUTHORIZATION_DENIED", true
	case errors.Is(err, ErrApprovalPolicyAuthorizationUnavailable):
		return "AUTHORIZATION_UNAVAILABLE", true
	case errors.Is(err, ErrApprovalPolicyAuthorizationStale):
		return "AUTHORIZATION_STALE", true
	case errors.Is(err, ErrApprovalPolicyVersionConflict):
		return "VERSION_CONFLICT", true
	case errors.Is(err, ErrApprovalPolicyOverlap):
		return "POLICY_OVERLAP", true
	case errors.Is(err, ErrApprovalPolicyNotFound):
		return "POLICY_NOT_FOUND", true
	case errors.Is(err, ErrInvalidApprovalPolicyCommand), errors.Is(err, ErrInvalidApprovalPolicy), errors.Is(err, ErrApprovalPolicyApprovalRequired), errors.Is(err, ErrApprovalPolicyApprovalRejected), errors.Is(err, ErrApprovalPolicySelfApproval):
		return "VALIDATION_FAILED", true
	default:
		return "", false
	}
}

func durableApprovalPolicyFailureError(body []byte) error {
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrApprovalPolicyCommandInProgress
	}
	switch payload.Code {
	case "AUTHORIZATION_DENIED":
		return ErrApprovalPolicyAuthorizationDenied
	case "AUTHORIZATION_UNAVAILABLE":
		return ErrApprovalPolicyAuthorizationUnavailable
	case "AUTHORIZATION_STALE":
		return ErrApprovalPolicyAuthorizationStale
	case "VERSION_CONFLICT":
		return ErrApprovalPolicyVersionConflict
	case "POLICY_OVERLAP":
		return ErrApprovalPolicyOverlap
	case "POLICY_NOT_FOUND":
		return ErrApprovalPolicyNotFound
	case "VALIDATION_FAILED":
		return ErrApprovalPolicyDurableCommandFailed
	default:
		return ErrApprovalPolicyCommandInProgress
	}
}

func decodeApprovalPolicyCommandResult(body []byte) (ApprovalPolicyCommandResult, error) {
	var result ApprovalPolicyCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return ApprovalPolicyCommandResult{}, err
	}
	return result, nil
}

func (service *ApprovalPolicyService) submitCandidate(ctx context.Context, actor Actor, candidate *ApprovalPolicy, command ApprovalPolicyCommand, now time.Time, request **ApprovalRequest, preparedRequest **PreparedApprovalRequest) error {
	if candidate == nil {
		return ErrInvalidApprovalPolicy
	}
	requestID := uuid.New()
	if err := candidate.SubmitForApproval(requestID, now); err != nil {
		return err
	}
	if preparer, ok := service.workflow.(ApprovalWorkflowPreparationPort); ok {
		prepared, err := preparer.PreparePolicyApproval(ctx, actor, *candidate, command)
		if err != nil {
			return err
		}
		*preparedRequest = &prepared
		*request = &prepared.Request
		return nil
	}
	created, err := service.workflow.CreatePolicyApproval(ctx, actor, *candidate, command)
	if err != nil {
		return err
	}
	*request = &created
	return nil
}

func validatePolicyAuthorization(decision AuthorizationDecision, scope ApprovalPolicyScope, action string) error {
	switch decision.Outcome {
	case AuthorizationUnavailable:
		return ErrApprovalPolicyAuthorizationUnavailable
	case AuthorizationStale:
		return ErrApprovalPolicyAuthorizationStale
	case AuthorizationExpired:
		return ErrApprovalPolicyAuthorizationDenied
	}
	if !decision.Allowed || decision.Permission != MaintainApprovalPoliciesPermission {
		return ErrApprovalPolicyAuthorizationDenied
	}
	if decision.DecisionReference == uuid.Nil {
		return fmt.Errorf("%w: authorization decision reference is required", ErrApprovalPolicyAuthorizationDenied)
	}
	requested := scope.CanonicalScopeIDs()
	approved := make(map[string]struct{}, len(decision.ApprovedScopeIDs))
	for _, value := range decision.ApprovedScopeIDs {
		approved[strings.TrimSpace(value)] = struct{}{}
	}
	for _, value := range requested {
		if _, ok := approved["*"]; ok {
			break
		}
		if _, ok := approved[value]; !ok {
			return fmt.Errorf("%w: scope %q is outside the administering actor scope", ErrApprovalPolicyAuthorizationDenied, value)
		}
	}
	return nil
}

func mapPolicyAuthorizationError(err error) error {
	switch {
	case errors.Is(err, ErrApprovalPolicyAuthorizationUnavailable), errors.Is(err, ErrApprovalPolicyAuthorizationStale), errors.Is(err, ErrApprovalPolicyAuthorizationDenied):
		return err
	default:
		return ErrApprovalPolicyAuthorizationUnavailable
	}
}

type MemoryApprovalPolicyRepository struct {
	mu       sync.RWMutex
	policies map[uuid.UUID]ApprovalPolicy
	audit    ApprovalPolicyAuditRecorder
}

func NewMemoryApprovalPolicyRepository() *MemoryApprovalPolicyRepository {
	return &MemoryApprovalPolicyRepository{policies: make(map[uuid.UUID]ApprovalPolicy)}
}

func (repository *MemoryApprovalPolicyRepository) BindAuditRecorder(audit ApprovalPolicyAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryApprovalPolicyRepository) Get(_ context.Context, id uuid.UUID) (ApprovalPolicy, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	policy, ok := repository.policies[id]
	if !ok {
		return ApprovalPolicy{}, ErrApprovalPolicyNotFound
	}
	return cloneApprovalPolicy(policy), nil
}

func (repository *MemoryApprovalPolicyRepository) List(_ context.Context) ([]ApprovalPolicy, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]ApprovalPolicy, 0, len(repository.policies))
	for _, policy := range repository.policies {
		result = append(result, cloneApprovalPolicy(policy))
	}
	return sortPolicies(result), nil
}

func (repository *MemoryApprovalPolicyRepository) ListSelectable(_ context.Context) ([]ApprovalPolicy, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]ApprovalPolicy, 0)
	for _, current := range repository.policies {
		result = append(result, selectablePolicyVersions(current)...)
	}
	return sortPolicies(result), nil
}

func (repository *MemoryApprovalPolicyRepository) Commit(ctx context.Context, mutation ApprovalPolicyMutation) error {
	_, err := repository.CommitWithResult(ctx, mutation)
	return err
}

func (repository *MemoryApprovalPolicyRepository) CommitWithResult(ctx context.Context, mutation ApprovalPolicyMutation) (uuid.UUID, error) {
	if repository == nil {
		return uuid.Nil, ErrInvalidApprovalPolicyService
	}
	if err := mutation.After.Validate(); err != nil {
		return uuid.Nil, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	current, exists := repository.policies[mutation.After.ID]
	if mutation.Before == nil {
		if exists {
			return uuid.Nil, ErrApprovalPolicyDuplicate
		}
	} else {
		if !exists || current.Version.Value() != mutation.Before.Version.Value() {
			return uuid.Nil, ErrApprovalPolicyVersionConflict
		}
	}
	for policyID, candidate := range repository.policies {
		if policyID != mutation.After.ID && candidate.Code == mutation.After.Code {
			return uuid.Nil, ErrApprovalPolicyDuplicate
		}
	}
	if mutation.After.Status == ApprovalPolicyStatusActive {
		for _, candidate := range repository.policies {
				if candidate.Status == ApprovalPolicyStatusSuperseded {
					continue
				}
			if PoliciesOverlap(candidate, mutation.After) {
				return uuid.Nil, ErrApprovalPolicyOverlap
			}
			for _, revision := range candidate.History {
				if revision.Snapshot.Status == ApprovalPolicyStatusActive && PoliciesOverlap(snapshotToPolicy(revision), mutation.After) {
					return uuid.Nil, ErrApprovalPolicyOverlap
				}
			}
		}
	}
	if repository.audit == nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}
	auditReference, err := repository.audit.RecordApprovalPolicyMutation(ctx, mutation.Audit)
	if err != nil {
		return uuid.Nil, err
	}
	if exists {
		current.History = append(current.History, ApprovalPolicyRevision{PolicyVersion: current.PolicyVersion, Version: current.Version, Snapshot: current.Snapshot(), CreatedAt: current.UpdatedAt})
	}
	mutation.After.History = append([]ApprovalPolicyRevision(nil), current.History...)
	repository.policies[mutation.After.ID] = cloneApprovalPolicy(mutation.After)
	return auditReference, nil
}

type MemoryApprovalPolicyAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryApprovalPolicyAuthorizer) AuthorizeApprovalPolicy(context.Context, Actor, ApprovalPolicyCommand, *ApprovalPolicy) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	decision := authorizer.Decision
	decision.ApprovedScopeIDs = append([]string(nil), decision.ApprovedScopeIDs...)
	decision.ApplicableDimensions = append([]string(nil), decision.ApplicableDimensions...)
	return decision, nil
}

type MemoryApprovalPolicyAuditRecorder struct {
	mu      sync.Mutex
	Records []ApprovalPolicyAuditRecord
	Err     error
}

func (recorder *MemoryApprovalPolicyAuditRecorder) RecordApprovalPolicyMutation(_ context.Context, record ApprovalPolicyAuditRecord) (uuid.UUID, error) {
	if recorder == nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return uuid.Nil, recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return uuid.New(), nil
}

func cloneApprovalPolicy(policy ApprovalPolicy) ApprovalPolicy {
	policy.Applicability = cloneApplicability(policy.Applicability)
	policy.Scope = cloneScope(policy.Scope)
	policy.Rules = cloneRules(policy.Rules)
	policy.Steps = cloneSteps(policy.Steps)
	policy.EffectiveTo = cloneTime(policy.EffectiveTo)
	policy.Approval = cloneApproval(policy.Approval)
	policy.History = append([]ApprovalPolicyRevision(nil), policy.History...)
	for index := range policy.History {
		policy.History[index].Snapshot.Applicability = cloneApplicability(policy.History[index].Snapshot.Applicability)
		policy.History[index].Snapshot.Scope = cloneScope(policy.History[index].Snapshot.Scope)
		policy.History[index].Snapshot.Rules = cloneRules(policy.History[index].Snapshot.Rules)
		policy.History[index].Snapshot.Steps = cloneSteps(policy.History[index].Snapshot.Steps)
		policy.History[index].Snapshot.EffectiveTo = cloneTime(policy.History[index].Snapshot.EffectiveTo)
		policy.History[index].Snapshot.Approval = cloneApproval(policy.History[index].Snapshot.Approval)
	}
	return policy
}

func cloneApprovalPolicyPtr(policy *ApprovalPolicy) *ApprovalPolicy {
	if policy == nil {
		return nil
	}
	copy := cloneApprovalPolicy(*policy)
	return &copy
}

func cloneApprovalPolicyResult(result ApprovalPolicyCommandResult) ApprovalPolicyCommandResult {
	result.Policy = cloneApprovalPolicy(result.Policy)
	result.ApprovalRequest = cloneApprovalRequestPtr(result.ApprovalRequest)
	return result
}

func cloneApprovalRequestPtr(request *ApprovalRequest) *ApprovalRequest {
	if request == nil {
		return nil
	}
	copy := cloneApprovalRequest(*request)
	return &copy
}

func fingerprintBeforePolicy(policy *ApprovalPolicy) string {
	if policy == nil {
		return ""
	}
	return policyStateFingerprint(*policy)
}

func policyStateFingerprint(policy ApprovalPolicy) string {
	return fingerprint(struct {
		Snapshot         ApprovalPolicySnapshot `json:"snapshot"`
		AggregateVersion int64                  `json:"aggregateVersion"`
	}{
		Snapshot:         policy.Snapshot(),
		AggregateVersion: policy.Version.Value(),
	})
}

func sortPolicies(values []ApprovalPolicy) []ApprovalPolicy {
	for index := range values {
		values[index] = cloneApprovalPolicy(values[index])
	}
	// UUID ordering is stable and avoids exposing map iteration order through
	// API results and tests.
	for left := 0; left < len(values); left++ {
		for right := left + 1; right < len(values); right++ {
			if values[right].ID.String() < values[left].ID.String() {
				values[left], values[right] = values[right], values[left]
			}
		}
	}
	return values
}

func snapshotToPolicy(revision ApprovalPolicyRevision) ApprovalPolicy {
	snapshot := revision.Snapshot
	return ApprovalPolicy{ID: snapshot.ID, Code: snapshot.Code, Name: snapshot.Name, PolicyVersion: snapshot.PolicyVersion, Status: snapshot.Status, Applicability: snapshot.Applicability, Scope: snapshot.Scope, Rules: snapshot.Rules, Steps: snapshot.Steps, EffectiveFrom: snapshot.EffectiveFrom, EffectiveTo: snapshot.EffectiveTo, Approval: snapshot.Approval, ApprovalRequestID: snapshot.ApprovalRequestID, ActivationOutcome: snapshot.ActivationOutcome, DependentWorkflowImpact: snapshot.DependentWorkflowImpact, Version: revision.Version}
}

func selectablePolicyVersions(policy ApprovalPolicy) []ApprovalPolicy {
	result := make([]ApprovalPolicy, 0, len(policy.History)+1)
	if policy.Status == ApprovalPolicyStatusSuperseded {
		return result
	}
	if policy.Status == ApprovalPolicyStatusActive {
		result = append(result, cloneApprovalPolicy(policy))
	}
	for _, revision := range policy.History {
		if revision.Snapshot.Status != ApprovalPolicyStatusActive {
			continue
		}
		historical := snapshotToPolicy(revision)
		if supersededByCurrentPolicy(policy, historical) {
			continue
		}
		result = append(result, historical)
	}
	return result
}

func supersededByCurrentPolicy(current, historical ApprovalPolicy) bool {
	return current.Status == ApprovalPolicyStatusActive && current.ID == historical.ID && selectorFingerprint(current.Applicability, current.Scope) == selectorFingerprint(historical.Applicability, historical.Scope) && intervalsOverlap(current.EffectiveFrom, current.EffectiveTo, historical.EffectiveFrom, historical.EffectiveTo)
}

var _ ApprovalPolicyRepository = (*MemoryApprovalPolicyRepository)(nil)
