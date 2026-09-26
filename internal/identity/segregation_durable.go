package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

type DurableSegregationRuleServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type SegregationRuleMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableSegregationRuleMutationRepository interface {
	CommitSegregationRuleMutationWithIdempotency(context.Context, SegregationRuleMutation, SegregationRuleMutationCommit) error
}

func NewSegregationRuleServiceWithDurableIdempotency(repository SegregationRuleRepository, authorizer SegregationRuleAuthorizer, approval SegregationRuleApprovalPort, audit SegregationRuleAuditRecorder, clock func() time.Time, durable DurableSegregationRuleServiceConfig) (*SegregationRuleService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrSegregationRuleUnavailable
	}
	service, err := NewSegregationRuleService(repository, authorizer, approval, audit, clock)
	if err != nil {
		return nil, err
	}
	service.durable = &DurableSegregationRuleServiceConfig{Database: durable.Database, Coordinator: durable.Coordinator, Policy: durable.Policy, OperationID: durable.OperationID}
	return service, nil
}

func (service *SegregationRuleService) executeDurable(ctx context.Context, actor ApplicationActor, command SegregationRuleCommand) (SegregationRuleCommandResult, error) {
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
	scopeKey, err := json.Marshal(struct {
		Module  string    `json:"module"`
		ActorID uuid.UUID `json:"actorId"`
	}{Module: "identity.segregation-rule", ActorID: actor.UserID})
	if err != nil {
		return SegregationRuleCommandResult{}, err
	}
	durableIdentity, err := platformidempotency.NewOpaqueIdentity(string(scopeKey), command.IdempotencyKey)
	if err != nil {
		return SegregationRuleCommandResult{}, err
	}
	acquisition, err := service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
	if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
		return SegregationRuleCommandResult{}, ErrSegregationRuleIdempotency
	}
	if err != nil {
		return SegregationRuleCommandResult{}, err
	}
	if acquisition.Decision() == platformidempotency.DecisionReturn {
		switch acquisition.Result().State() {
		case platformidempotency.StateFailed:
			return SegregationRuleCommandResult{}, segregationDurableFailureError(acquisition.Result().ResultBody())
		case platformidempotency.StateInProgress:
			return SegregationRuleCommandResult{}, ErrSegregationRuleInProgress
		case platformidempotency.StateEstablished:
			result, decodeErr := decodeSegregationRuleCommandResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return SegregationRuleCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	}

	var current *SegregationRule
	if command.Action != SegregationRuleActionCreate {
		value, getErr := service.repository.Get(ctx, command.RuleID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, getErr)
			return SegregationRuleCommandResult{}, getErr
		}
		if command.ExpectedVersion == nil || !value.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrSegregationRuleVersion)
			return SegregationRuleCommandResult{}, ErrSegregationRuleVersion
		}
		current = &value
	}
	decision, err := service.authorizer.AuthorizeSegregationRuleManagement(ctx, actor, command, cloneSegregationRulePtr(current))
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegregationRuleCommandResult{}, err
	}
	if decision.Outcome == AuthorizationUnavailable {
		service.finalizeDurableFailure(ctx, acquisition, ErrSegregationRuleUnavailable)
		return SegregationRuleCommandResult{}, ErrSegregationRuleUnavailable
	}
	if decision.Outcome == AuthorizationExpired {
		service.finalizeDurableFailure(ctx, acquisition, ErrAuthorizationExpired)
		return SegregationRuleCommandResult{}, ErrAuthorizationExpired
	}
	if decision.Outcome == AuthorizationStale {
		authorizationErr := ErrSegregationRuleStale
		service.finalizeDurableFailure(ctx, acquisition, authorizationErr)
		return SegregationRuleCommandResult{}, authorizationErr
	}
	if !decision.Allowed || decision.Permission != SegregationRuleManagementPermission {
		authorizationErr := ErrSegregationRuleAuthorization
		service.finalizeDurableFailure(ctx, acquisition, authorizationErr)
		return SegregationRuleCommandResult{}, authorizationErr
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
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegregationRuleCommandResult{}, err
	}
	if command.Action != SegregationRuleActionCreate {
		after.Version, err = after.Version.Advance()
		if err != nil {
			service.finalizeDurableFailure(ctx, acquisition, err)
			return SegregationRuleCommandResult{}, err
		}
	}
	if err := after.Validate(); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegregationRuleCommandResult{}, err
	}
	candidateFingerprint := candidateSegregationRuleFingerprint(command, after)
	if err := service.approval.ValidateSegregationRuleApproval(ctx, actor, command, cloneSegregationRulePtr(current), candidateFingerprint); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegregationRuleCommandResult{}, err
	}
	audit := SegregationRuleAuditRecord{RuleID: after.ID, ActorUserID: actor.UserID, ActorAuthenticationRef: fingerprintSubject(actor.Subject), Action: command.Action, Code: after.Code, ScopeIDs: append([]string(nil), after.ScopeIDs...), PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, ApprovalRequestID: command.Approval.ApprovalRequestID, ApprovalDecisionID: command.Approval.DecisionID, ApproverUserID: command.Approval.ApproverUserID, RuleVersion: after.Version.Value(), BeforeFingerprint: fingerprintSegregationRule(before), AfterFingerprint: candidateFingerprint, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	result := SegregationRuleCommandResult{Rule: cloneSegregationRule(after), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference}
	committer, ok := service.repository.(DurableSegregationRuleMutationRepository)
	if !ok {
		service.finalizeDurableFailure(ctx, acquisition, ErrSegregationRuleUnavailable)
		return SegregationRuleCommandResult{}, ErrSegregationRuleUnavailable
	}
	body, err := json.Marshal(result)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegregationRuleCommandResult{}, err
	}
	status := httpStatusOK
	aggregateID := after.ID
	metadata, err := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegregationRuleCommandResult{}, err
	}
	err = committer.CommitSegregationRuleMutationWithIdempotency(ctx, SegregationRuleMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: audit}, SegregationRuleMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegregationRuleCommandResult{}, err
	}
	return result, nil
}

const httpStatusOK = 200

func (service *SegregationRuleService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code, ok := segregationDurableFailureCode(commandErr)
	if !ok {
		return
	}
	body, err := json.Marshal(durableFailurePayload{Code: code})
	if err != nil {
		return
	}
	metadata, err := platformidempotency.NewCommandResultMetadata(acquisition.Result().Identity(), acquisition.Result().Fingerprint(), acquisition.Result().OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
	if err != nil {
		return
	}
	transaction, err := service.durable.Database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback(ctx)
		}
	}()
	if err := service.durable.Coordinator.Finalize(ctx, transaction, acquisition, metadata); err != nil {
		return
	}
	if err := transaction.Commit(ctx); err != nil {
		return
	}
	committed = true
}

func segregationDurableFailureCode(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrSegregationRuleAuthorization):
		return "AUTHORIZATION_DENIED", true
	case errors.Is(err, ErrAuthorizationExpired):
		return "POLICY_EXPIRED", true
	case errors.Is(err, ErrSegregationRuleUnavailable), errors.Is(err, ErrSegregationRuleAudit), errors.Is(err, ErrAuthorizationUnavailable), errors.Is(err, ErrApprovalUnavailable):
		return "POLICY_UNAVAILABLE", true
	case errors.Is(err, ErrSegregationRuleStale), errors.Is(err, ErrAuthorizationStale):
		return "POLICY_STALE", true
	case errors.Is(err, ErrSegregationRuleVersion):
		return "VERSION_CONFLICT", true
	case errors.Is(err, ErrSegregationRuleIdempotency):
		return "IDEMPOTENCY_CONFLICT", true
	case errors.Is(err, ErrSegregationRuleNotFound):
		return "RULE_NOT_FOUND", true
	case errors.Is(err, ErrSegregationRuleApproval), errors.Is(err, ErrInvalidSegregationCommand), errors.Is(err, ErrInvalidSegregationRule), errors.Is(err, ErrSegregationRuleRetired), errors.Is(err, ErrApprovalRequired):
		return "VALIDATION_FAILED", true
	default:
		return "", false
	}
}

func segregationDurableFailureError(body []byte) error {
	var payload durableFailurePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrSegregationRuleInProgress
	}
	switch payload.Code {
	case "AUTHORIZATION_DENIED":
		return ErrSegregationRuleAuthorization
	case "POLICY_EXPIRED":
		return ErrAuthorizationExpired
	case "POLICY_UNAVAILABLE":
		return ErrSegregationRuleUnavailable
	case "POLICY_STALE":
		return ErrSegregationRuleStale
	case "VERSION_CONFLICT":
		return ErrSegregationRuleVersion
	case "IDEMPOTENCY_CONFLICT":
		return ErrSegregationRuleIdempotency
	case "RULE_NOT_FOUND":
		return ErrSegregationRuleNotFound
	case "VALIDATION_FAILED":
		return fmt.Errorf("%w: %s", ErrSegregationRuleApproval, payload.Code)
	default:
		return ErrSegregationRuleInProgress
	}
}

func decodeSegregationRuleCommandResult(body []byte) (SegregationRuleCommandResult, error) {
	var result SegregationRuleCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return SegregationRuleCommandResult{}, err
	}
	return result, nil
}
