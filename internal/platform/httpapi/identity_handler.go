package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-faster/jx"
	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type IdentityHandler struct {
	generated.UnimplementedHandler
	Service                *identity.UserService
	RoleService            *identity.RoleService
	SegregationRuleService *identity.SegregationRuleService
	EmergencyAccessService *identity.EmergencyAccessService
}

func (handler IdentityHandler) IamManageUsers(ctx context.Context, request *generated.IamManageUsersCommandRequest, params generated.IamManageUsersParams) (generated.IamManageUsersRes, error) {
	if handler.Service == nil {
		return iamProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Identity user service is unavailable.", correlationFromParams(params), nil)
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return iamProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationFromParams(params), nil)
	}

	action := string(request.Data.Action)
	userID := uuid.Nil
	if request.Data.UserId.Set {
		userID = uuid.UUID(request.Data.UserId.Value)
	}
	authenticationSubject := (*identity.AuthenticationSubject)(nil)
	if request.Data.AuthenticationSubject.Set {
		subject := request.Data.AuthenticationSubject.Value
		authenticationSubject = &identity.AuthenticationSubject{OID: subject.Oid, TID: subject.Tid, Sub: subject.Sub}
	}
	var assignments []identity.RoleAssignment
	if request.Data.Assignments != nil {
		assignments = make([]identity.RoleAssignment, len(request.Data.Assignments))
		for index, assignment := range request.Data.Assignments {
			scopes := make([]identity.EntityAccessScope, len(assignment.ScopeIds))
			for scopeIndex, scopeID := range assignment.ScopeIds {
				scopes[scopeIndex] = identity.EntityAccessScope{ScopeID: scopeID}
			}
			assignments[index] = identity.RoleAssignment{RoleID: uuid.UUID(assignment.RoleId), Scopes: scopes}
		}
	}

	var expectedVersion *aggregateversion.AggregateVersion
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return iamProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected version is invalid.", correlationFromParams(params), nil)
		}
		expectedVersion = &version
	}

	if request.Data.UserId.Set && params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return iamProblem(http.StatusBadRequest, "INVALID_REQUEST", "The If-Match version is invalid.", correlationFromParams(params), nil)
		}
		if expectedVersion != nil && expectedVersion.Value() != headerVersion.Value() {
			return iamProblem(http.StatusConflict, "VERSION_CONFLICT", "The body version and If-Match version do not agree.", correlationFromParams(params), nil)
		}
		expectedVersion = &headerVersion
	}
	command := identity.UserCommand{
		Action:                action,
		UserID:                userID,
		AuthenticationSubject: authenticationSubject,
		Assignments:           assignments,
		ExpectedVersion:       expectedVersion,
		IdempotencyKey:        params.IdempotencyKey,
		CorrelationID:         correlationFromParams(params).String(),
		CausationID:           uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.Service.Execute(ctx, actor, command)
	if err != nil {
		return mapIdentityError(err, correlationFromParams(params))
	}
	return establishedIdentityResult(result, correlationFromParams(params)), nil
}

func (handler IdentityHandler) IamGrantEmergencyAccess(ctx context.Context, request *generated.IamGrantEmergencyAccessCommandRequest, params generated.IamGrantEmergencyAccessParams) (generated.IamGrantEmergencyAccessRes, error) {
	if handler.EmergencyAccessService == nil {
		return emergencyGrantProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The emergency access service is unavailable.", emergencyGrantCorrelationFromParams(params))
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return emergencyGrantProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", emergencyGrantCorrelationFromParams(params))
	}
	data := request.Data
	correlationID := emergencyGrantCorrelationFromParams(params)
	command := identity.EmergencyAccessGrantCommand{
		Action: identity.EmergencyAccessActionGrant, TargetActorID: uuid.UUID(data.TargetActorId),
		Permissions: append([]string(nil), data.Permissions...), ScopeIDs: append([]string(nil), data.ScopeIds...),
		ReasonCode: data.ReasonCode, StartsAt: data.StartsAt, ExpiresAt: data.ExpiresAt,
		Approval: approvalFromGenerated(data.Approval), IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.EmergencyAccessService.Execute(ctx, actor, command)
	if err != nil {
		return mapEmergencyGrantError(err, correlationID)
	}
	return establishedEmergencyAccessResult(result, correlationID, "/api/v1/identity-access/actions/grant-emergency-access"), nil
}

func (handler IdentityHandler) IamRevokeEmergencyAccess(ctx context.Context, request *generated.IamRevokeEmergencyAccessCommandRequest, params generated.IamRevokeEmergencyAccessParams) (generated.IamRevokeEmergencyAccessRes, error) {
	if handler.EmergencyAccessService == nil {
		return emergencyRevokeProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The emergency access service is unavailable.", emergencyRevokeCorrelationFromParams(params))
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return emergencyRevokeProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", emergencyRevokeCorrelationFromParams(params))
	}
	correlationID := emergencyRevokeCorrelationFromParams(params)
	expectedVersion, err := aggregateversion.FromInt64(int64(request.ExpectedVersion))
	if err != nil {
		return emergencyRevokeProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected version is invalid.", correlationID)
	}
	if params.IfMatch.Set {
		headerVersion, parseErr := parseIfMatchVersion(params.IfMatch.Value)
		if parseErr != nil {
			return emergencyRevokeProblem(http.StatusBadRequest, "INVALID_REQUEST", "The If-Match version is invalid.", correlationID)
		}
		if expectedVersion.Value() != headerVersion.Value() {
			return emergencyRevokeProblem(http.StatusConflict, "VERSION_CONFLICT", "The body version and If-Match version do not agree.", correlationID)
		}
		expectedVersion = headerVersion
	}
	data := request.Data
	reviewStatus := ""
	if data.ReviewStatus.Set {
		reviewStatus = string(data.ReviewStatus.Value)
	}
	reviewOutcomeCode := ""
	if data.ReviewOutcomeCode.Set {
		reviewOutcomeCode = data.ReviewOutcomeCode.Value
	}
	reviewReference := ""
	if data.ReviewReference.Set {
		reviewReference = data.ReviewReference.Value
	}
	command := identity.EmergencyAccessGrantCommand{
		Action: identity.EmergencyAccessActionRevoke, GrantID: uuid.UUID(data.GrantId), ReasonCode: data.ReasonCode,
		ReviewStatus: reviewStatus, ReviewOutcomeCode: reviewOutcomeCode, ReviewReference: reviewReference,
		ExpectedVersion: &expectedVersion, IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.EmergencyAccessService.Execute(ctx, actor, command)
	if err != nil {
		return mapEmergencyRevokeError(err, correlationID)
	}
	return establishedEmergencyAccessResult(result, correlationID, "/api/v1/identity-access/actions/revoke-emergency-access"), nil
}

func approvalFromGenerated(value generated.IamApprovalDecisionReference) identity.ApprovalDecisionReference {
	return identity.ApprovalDecisionReference{
		ApprovalRequestID: uuid.UUID(value.ApprovalRequestId), DecisionID: uuid.UUID(value.DecisionId), PolicyVersion: value.PolicyVersion,
		DecisionVersion: int64(value.DecisionVersion), SubjectVersion: int64(value.SubjectVersion), CandidateFingerprint: value.CandidateFingerprint,
		ApproverUserID: uuid.UUID(value.ApproverUserId),
	}
}

func (handler IdentityHandler) IamManageRoles(ctx context.Context, request *generated.IamManageRolesCommandRequest, params generated.IamManageRolesParams) (generated.IamManageRolesRes, error) {
	if handler.RoleService == nil {
		return iamRoleProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Identity role service is unavailable.", roleCorrelationFromParams(params), nil)
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return iamRoleProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", roleCorrelationFromParams(params), nil)
	}

	data := request.Data
	roleID := uuid.Nil
	if data.RoleId.Set {
		roleID = uuid.UUID(data.RoleId.Value)
	}
	var expectedVersion *aggregateversion.AggregateVersion
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return iamRoleProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected version is invalid.", roleCorrelationFromParams(params), nil)
		}
		expectedVersion = &version
	}
	if data.RoleId.Set && params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return iamRoleProblem(http.StatusBadRequest, "INVALID_REQUEST", "The If-Match version is invalid.", roleCorrelationFromParams(params), nil)
		}
		if expectedVersion != nil && expectedVersion.Value() != headerVersion.Value() {
			return iamRoleProblem(http.StatusConflict, "VERSION_CONFLICT", "The body version and If-Match version do not agree.", roleCorrelationFromParams(params), nil)
		}
		expectedVersion = &headerVersion
	}

	grants := make([]identity.PermissionGrant, len(data.Grants))
	for index, grant := range data.Grants {
		var effectiveTo *time.Time
		if grant.EffectiveTo.Set {
			value := grant.EffectiveTo.Value
			effectiveTo = &value
		}
		grants[index] = identity.PermissionGrant{
			Permission:    grant.Permission,
			ScopeIDs:      append([]string(nil), grant.ScopeIds...),
			EffectiveFrom: grant.EffectiveFrom,
			EffectiveTo:   effectiveTo,
		}
	}
	approval := identity.ApprovalDecisionReference{
		ApprovalRequestID:    uuid.UUID(data.Approval.ApprovalRequestId),
		DecisionID:           uuid.UUID(data.Approval.DecisionId),
		PolicyVersion:        data.Approval.PolicyVersion,
		DecisionVersion:      int64(data.Approval.DecisionVersion),
		SubjectVersion:       int64(data.Approval.SubjectVersion),
		CandidateFingerprint: data.Approval.CandidateFingerprint,
		ApproverUserID:       uuid.UUID(data.Approval.ApproverUserId),
	}
	correlationID := roleCorrelationFromParams(params)
	command := identity.RoleCommand{
		Action:          string(data.Action),
		RoleID:          roleID,
		Name:            data.Name,
		Grants:          grants,
		Approval:        approval,
		ExpectedVersion: expectedVersion,
		IdempotencyKey:  params.IdempotencyKey,
		CorrelationID:   correlationID.String(),
		CausationID:     uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.RoleService.Execute(ctx, actor, command)
	if err != nil {
		return mapRoleError(err, correlationID)
	}
	return establishedRoleResult(result, correlationID), nil
}

func parseIfMatchVersion(value string) (aggregateversion.AggregateVersion, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "W/")
	value = strings.Trim(value, "\"")
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return aggregateversion.FromInt64(parsed)
}

func correlationFromParams(params generated.IamManageUsersParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedIdentityResult(result identity.UserCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["userId"] = mustRaw(result.User.ID.String())
	data["status"] = mustRaw(result.User.Status)
	data["aggregateVersion"] = mustRaw(result.User.Version.Value())
	data["assignmentCount"] = mustRaw(len(result.User.Assignments))
	data["authenticationSubject"] = mustRaw(map[string]string{"oid": "masked", "tid": "masked", "sub": "masked"})
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	return &generated.EstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(result.User.ID),
		AggregateVersion: int(result.User.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: "/api/v1/identity-access/actions/manage-users"},
		Data:             data,
	}
}

func mapIdentityError(err error, correlationID uuid.UUID) (generated.IamManageUsersRes, error) {
	switch {
	case errors.Is(err, identity.ErrAuthorizationStale):
		return iamProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID, nil)
	case errors.Is(err, identity.ErrAuthorizationExpired):
		return iamProblem(http.StatusForbidden, "POLICY_EXPIRED", "The authorization policy is no longer effective.", correlationID, nil)
	case errors.Is(err, identity.ErrAuthorizationUnavailable):
		return iamProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID, nil)
	case errors.Is(err, identity.ErrAuthorizationDenied):
		return iamProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested assignment set is outside the administering actor scope.", correlationID, nil)
	case errors.Is(err, identity.ErrVersionConflict):
		return iamProblem(http.StatusConflict, "VERSION_CONFLICT", "The user changed after it was loaded. Refresh and retry with the current version.", correlationID, nil)
	case errors.Is(err, identity.ErrIdempotencyConflict):
		return iamProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID, nil)
	case errors.Is(err, identity.ErrCommandInProgress):
		return iamProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID, nil)
	case errors.Is(err, identity.ErrDuplicateAuthenticationSubject):
		return iamProblem(http.StatusConflict, "DUPLICATE_USER", "A user for this authentication subject already exists.", correlationID, nil)
	case errors.Is(err, identity.ErrUserNotFound):
		return iamProblem(http.StatusConflict, "USER_NOT_FOUND", "The requested user does not exist.", correlationID, nil)
	case errors.Is(err, identity.ErrInvalidUserCommand):
		return iamProblem(http.StatusBadRequest, "INVALID_REQUEST", "The manage-users command is invalid.", correlationID, nil)
	case errors.Is(err, identity.ErrInvalidAssignment), errors.Is(err, identity.ErrDuplicateAssignment), errors.Is(err, identity.ErrInvalidAuthenticationSubject), errors.Is(err, identity.ErrInvalidUserTransition), errors.Is(err, identity.ErrUserTerminated), errors.Is(err, identity.ErrAuthenticationImmutable), errors.Is(err, identity.ErrDurableCommandFailed):
		return iamProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The manage-users command violates an identity rule.", correlationID, nil)
	default:
		return iamProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The identity user operation could not be completed.", correlationID, nil)
	}
}

func iamProblem(status int, code, detail string, correlationID uuid.UUID, currentVersion *int) (generated.IamManageUsersRes, error) {
	problem := generated.ProblemDetails{
		Type:          "https://tally.local/problems/" + code,
		Title:         http.StatusText(status),
		Status:        status,
		Code:          code,
		Detail:        detail,
		CorrelationId: generated.UUID(correlationID),
	}
	if currentVersion != nil {
		problem.CurrentVersion.SetTo(*currentVersion)
	}
	switch status {
	case http.StatusBadRequest:
		value := generated.IamManageUsersBadRequest(problem)
		return &value, nil
	case http.StatusForbidden:
		value := generated.IamManageUsersForbidden(problem)
		return &value, nil
	case http.StatusConflict:
		value := generated.IamManageUsersConflict(problem)
		return &value, nil
	case http.StatusUnprocessableEntity:
		value := generated.IamManageUsersUnprocessableEntity(problem)
		return &value, nil
	default:
		value := generated.IamManageUsersServiceUnavailable(problem)
		return &value, nil
	}
}

func mustRaw(value any) jx.Raw {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("identity result serialization failed: %v", err))
	}
	return jx.Raw(data)
}

func roleCorrelationFromParams(params generated.IamManageRolesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedRoleResult(result identity.RoleCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["roleId"] = mustRaw(result.Role.ID.String())
	data["name"] = mustRaw(result.Role.Name)
	data["status"] = mustRaw(result.Role.Status)
	data["aggregateVersion"] = mustRaw(result.Role.Version.Value())
	data["grantCount"] = mustRaw(len(result.Role.Grants))
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["approval"] = mustRaw(map[string]any{
		"approvalRequestId":    result.Approval.ApprovalRequestID.String(),
		"decisionId":           result.Approval.DecisionID.String(),
		"policyVersion":        result.Approval.PolicyVersion,
		"decisionVersion":      result.Approval.DecisionVersion,
		"subjectVersion":       result.Approval.SubjectVersion,
		"candidateFingerprint": result.Approval.CandidateFingerprint,
		"approverUserId":       result.Approval.ApproverUserID.String(),
	})
	return &generated.EstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(result.Role.ID),
		AggregateVersion: int(result.Role.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: "/api/v1/identity-access/actions/manage-roles"},
		Data:             data,
	}
}

func mapRoleError(err error, correlationID uuid.UUID) (generated.IamManageRolesRes, error) {
	switch {
	case errors.Is(err, identity.ErrAuthorizationStale):
		return iamRoleProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID, nil)
	case errors.Is(err, identity.ErrAuthorizationExpired):
		return iamRoleProblem(http.StatusForbidden, "POLICY_EXPIRED", "The authorization policy is no longer effective.", correlationID, nil)
	case errors.Is(err, identity.ErrAuthorizationUnavailable):
		return iamRoleProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID, nil)
	case errors.Is(err, identity.ErrRoleAuthorizationDenied):
		return iamRoleProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested role change is outside the administering actor scope.", correlationID, nil)
	case errors.Is(err, identity.ErrVersionConflict):
		return iamRoleProblem(http.StatusConflict, "VERSION_CONFLICT", "The role changed after it was loaded. Refresh and retry with the current version.", correlationID, nil)
	case errors.Is(err, identity.ErrRoleIdempotencyConflict):
		return iamRoleProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID, nil)
	case errors.Is(err, identity.ErrRoleCommandInProgress):
		return iamRoleProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID, nil)
	case errors.Is(err, identity.ErrRoleNotFound):
		return iamRoleProblem(http.StatusConflict, "ROLE_NOT_FOUND", "The requested role does not exist.", correlationID, nil)
	case errors.Is(err, identity.ErrInvalidRoleCommand):
		return iamRoleProblem(http.StatusBadRequest, "INVALID_REQUEST", "The manage-roles command is invalid.", correlationID, nil)
	case errors.Is(err, identity.ErrInvalidRole), errors.Is(err, identity.ErrInvalidPermissionGrant),
		errors.Is(err, identity.ErrDuplicatePermissionGrant), errors.Is(err, identity.ErrApprovalRequired),
		errors.Is(err, identity.ErrApprovalRejected), errors.Is(err, identity.ErrSegregationConflict),
		errors.Is(err, identity.ErrRoleRetired), errors.Is(err, identity.ErrRoleDurableCommandFailed):
		return iamRoleProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The manage-roles command violates an identity rule.", correlationID, nil)
	case errors.Is(err, identity.ErrApprovalUnavailable), errors.Is(err, identity.ErrSegregationUnavailable), errors.Is(err, identity.ErrRoleAuditUnavailable),
		errors.Is(err, identity.ErrInvalidRoleService):
		return iamRoleProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The identity role operation could not be completed.", correlationID, nil)
	default:
		return iamRoleProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The identity role operation could not be completed.", correlationID, nil)
	}
}

func iamRoleProblem(status int, code, detail string, correlationID uuid.UUID, currentVersion *int) (generated.IamManageRolesRes, error) {
	problem := generated.ProblemDetails{
		Type:          "https://tally.local/problems/" + code,
		Title:         http.StatusText(status),
		Status:        status,
		Code:          code,
		Detail:        detail,
		CorrelationId: generated.UUID(correlationID),
	}
	if currentVersion != nil {
		problem.CurrentVersion.SetTo(*currentVersion)
	}
	switch status {
	case http.StatusBadRequest:
		value := generated.IamManageRolesBadRequest(problem)
		return &value, nil
	case http.StatusForbidden:
		value := generated.IamManageRolesForbidden(problem)
		return &value, nil
	case http.StatusConflict:
		value := generated.IamManageRolesConflict(problem)
		return &value, nil
	case http.StatusUnprocessableEntity:
		value := generated.IamManageRolesUnprocessableEntity(problem)
		return &value, nil
	default:
		value := generated.IamManageRolesServiceUnavailable(problem)
		return &value, nil
	}
}

func emergencyGrantCorrelationFromParams(params generated.IamGrantEmergencyAccessParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func emergencyRevokeCorrelationFromParams(params generated.IamRevokeEmergencyAccessParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedEmergencyAccessResult(result identity.EmergencyAccessCommandResult, correlationID uuid.UUID, self string) *generated.EstablishedResult {
	grant := result.Grant
	data := generated.EstablishedResultData{}
	data["grantId"] = mustRaw(grant.ID.String())
	data["targetActorId"] = mustRaw(grant.TargetActorID.String())
	data["grantingActorId"] = mustRaw(grant.GrantingActorID.String())
	data["status"] = mustRaw(grant.Status)
	data["permissions"] = mustRaw(grant.Permissions)
	data["scopeIds"] = mustRaw(grant.ScopeIDs)
	data["reasonCode"] = mustRaw(grant.ReasonCode)
	data["startsAt"] = mustRaw(grant.StartAt)
	data["expiresAt"] = mustRaw(grant.ExpiresAt)
	data["policyVersion"] = mustRaw(grant.PolicyVersion)
	data["reviewStatus"] = mustRaw(grant.ReviewStatus)
	data["reviewOutcomeCode"] = mustRaw(grant.ReviewOutcomeCode)
	data["reviewReference"] = mustRaw(grant.ReviewReference)
	data["reviewDueAt"] = mustRaw(grant.ReviewDueAt)
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["approval"] = mustRaw(map[string]any{
		"approvalRequestId": grant.Approval.ApprovalRequestID.String(), "decisionId": grant.Approval.DecisionID.String(),
		"policyVersion": grant.Approval.PolicyVersion, "decisionVersion": grant.Approval.DecisionVersion,
		"subjectVersion": grant.Approval.SubjectVersion, "candidateFingerprint": grant.Approval.CandidateFingerprint,
		"approverUserId": grant.Approval.ApproverUserID.String(),
	})
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(grant.ID), AggregateVersion: int(grant.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: self}, Data: data}
}

func mapEmergencyGrantError(err error, correlationID uuid.UUID) (generated.IamGrantEmergencyAccessRes, error) {
	switch {
	case errors.Is(err, identity.ErrEmergencyAccessAuthorizationStale):
		return emergencyGrantProblem(http.StatusConflict, "POLICY_STALE", "The emergency-access policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessAuthorizationExpired):
		return emergencyGrantProblem(http.StatusForbidden, "POLICY_EXPIRED", "The emergency-access policy is no longer effective.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessAuthorizationUnavail), errors.Is(err, identity.ErrEmergencyAccessApprovalUnavailable), errors.Is(err, identity.ErrEmergencyAccessAuditUnavailable), errors.Is(err, identity.ErrInvalidEmergencyAccessService):
		return emergencyGrantProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The emergency access operation could not be completed.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessAuthorizationDenied):
		return emergencyGrantProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested emergency-access scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessStepUpRequired):
		return emergencyGrantProblem(http.StatusForbidden, "STEP_UP_REQUIRED", "Recent authentication assurance or an explicit step-up challenge is required.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessIdempotencyConflict):
		return emergencyGrantProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different emergency-access command data.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessVersionConflict):
		return emergencyGrantProblem(http.StatusConflict, "VERSION_CONFLICT", "The emergency-access grant changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessCommandInProgress):
		return emergencyGrantProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The emergency-access command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, identity.ErrInvalidEmergencyAccessCommand), errors.Is(err, identity.ErrInvalidEmergencyAccessGrant), errors.Is(err, identity.ErrEmergencyAccessApprovalRejected):
		return emergencyGrantProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The emergency-access grant violates an identity rule.", correlationID)
	default:
		return emergencyGrantProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The emergency access operation could not be completed.", correlationID)
	}
}

func mapEmergencyRevokeError(err error, correlationID uuid.UUID) (generated.IamRevokeEmergencyAccessRes, error) {
	switch {
	case errors.Is(err, identity.ErrEmergencyAccessAuthorizationStale):
		return emergencyRevokeProblem(http.StatusConflict, "POLICY_STALE", "The emergency-access policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessAuthorizationExpired):
		return emergencyRevokeProblem(http.StatusForbidden, "POLICY_EXPIRED", "The emergency-access policy is no longer effective.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessAuthorizationUnavail), errors.Is(err, identity.ErrEmergencyAccessAuditUnavailable), errors.Is(err, identity.ErrInvalidEmergencyAccessService):
		return emergencyRevokeProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The emergency access operation could not be completed.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessAuthorizationDenied):
		return emergencyRevokeProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested emergency-access scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessStepUpRequired):
		return emergencyRevokeProblem(http.StatusForbidden, "STEP_UP_REQUIRED", "Recent authentication assurance or an explicit step-up challenge is required.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessVersionConflict):
		return emergencyRevokeProblem(http.StatusConflict, "VERSION_CONFLICT", "The emergency-access grant changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessIdempotencyConflict):
		return emergencyRevokeProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different emergency-access command data.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessCommandInProgress):
		return emergencyRevokeProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The emergency-access command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessNotFound):
		return emergencyRevokeProblem(http.StatusConflict, "GRANT_NOT_FOUND", "The requested emergency-access grant does not exist.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessReviewCompleted):
		return emergencyRevokeProblem(http.StatusConflict, "REVIEW_COMPLETED", "The emergency-access review is already completed and cannot be changed.", correlationID)
	case errors.Is(err, identity.ErrEmergencyAccessAlreadyRevoked):
		return emergencyRevokeProblem(http.StatusConflict, "ALREADY_REVOKED", "The emergency-access grant is already revoked; submit a completed review update when review evidence is ready.", correlationID)
	case errors.Is(err, identity.ErrInvalidEmergencyAccessCommand), errors.Is(err, identity.ErrInvalidEmergencyAccessGrant), errors.Is(err, identity.ErrEmergencyAccessReviewInvalid):
		return emergencyRevokeProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The emergency-access revocation or review violates an identity rule.", correlationID)
	default:
		return emergencyRevokeProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The emergency access operation could not be completed.", correlationID)
	}
}

func emergencyGrantProblem(status int, code, detail string, correlationID uuid.UUID) (generated.IamGrantEmergencyAccessRes, error) {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.IamGrantEmergencyAccessBadRequest(problem)
		return &value, nil
	case http.StatusForbidden:
		value := generated.IamGrantEmergencyAccessForbidden(problem)
		return &value, nil
	case http.StatusConflict:
		value := generated.IamGrantEmergencyAccessConflict(problem)
		return &value, nil
	case http.StatusUnprocessableEntity:
		value := generated.IamGrantEmergencyAccessUnprocessableEntity(problem)
		return &value, nil
	default:
		value := generated.IamGrantEmergencyAccessServiceUnavailable(problem)
		return &value, nil
	}
}

func emergencyRevokeProblem(status int, code, detail string, correlationID uuid.UUID) (generated.IamRevokeEmergencyAccessRes, error) {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.IamRevokeEmergencyAccessBadRequest(problem)
		return &value, nil
	case http.StatusForbidden:
		value := generated.IamRevokeEmergencyAccessForbidden(problem)
		return &value, nil
	case http.StatusConflict:
		value := generated.IamRevokeEmergencyAccessConflict(problem)
		return &value, nil
	case http.StatusUnprocessableEntity:
		value := generated.IamRevokeEmergencyAccessUnprocessableEntity(problem)
		return &value, nil
	default:
		value := generated.IamRevokeEmergencyAccessServiceUnavailable(problem)
		return &value, nil
	}
}
