package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/workflow"
)

const (
	workflowCreateApprovalRequestRoute = "/api/v1/approvals/actions/create-approval-request"
	workflowApprovalRequestListRoute   = "/api/v1/approvals/reference/requests"
)

func (handler IdentityHandler) WfaCreateApprovalRequest(ctx context.Context, request *generated.WfaCreateApprovalRequestCommandRequest, params generated.WfaCreateApprovalRequestParams) (generated.WfaCreateApprovalRequestRes, error) {
	correlationID := workflowCreateCorrelation(params)
	if handler.ApprovalRequestService == nil {
		return workflowCreateApprovalRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-request service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return workflowCreateApprovalRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return workflowCreateApprovalRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval-request command requires a command identifier.", correlationID), nil
	}
	command, err := workflowCreateApprovalRequestCommand(request, params)
	if err != nil {
		return mapWorkflowCreateApprovalRequestError(err, correlationID), nil
	}
	result, err := handler.ApprovalRequestService.CreateApprovalRequest(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapWorkflowCreateApprovalRequestError(err, correlationID), nil
	}
	return establishedWorkflowCreateApprovalRequestResult(result, correlationID), nil
}

func (handler IdentityHandler) WfaListApprovalRequests(ctx context.Context, params generated.WfaListApprovalRequestsParams) (generated.WfaListApprovalRequestsRes, error) {
	correlationID := workflowListCorrelation(params)
	if handler.ApprovalRequestQueries == nil {
		return workflowListApprovalRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-request read service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return workflowListApprovalRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	filter := workflow.ApprovalRequestListFilter{
		Assignment: string(params.Assignment.Or(generated.WfaListApprovalRequestsAssignmentAssigned)),
		Due:        string(params.Due.Or(generated.WfaListApprovalRequestsDueAll)),
		State:      string(params.State.Or(generated.WfaListApprovalRequestsStateAll)),
	}
	if params.XAccountingScopeID.Set {
		filter.ScopeIDs = []string{uuid.UUID(params.XAccountingScopeID.Value).String()}
	}
	if params.PageSize.Set {
		filter.PageSize = params.PageSize.Value
	}
	if params.PageAfter.Set {
		filter.PageAfter = params.PageAfter.Value
	}
	page, err := handler.ApprovalRequestQueries.List(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, filter)
	if err != nil {
		return mapWorkflowListApprovalRequestError(err, correlationID), nil
	}
	projections := make([]generated.WfaApprovalRequestProjection, len(page.Requests))
	for index, request := range page.Requests {
		projections[index] = generatedWorkflowApprovalRequestProjection(request)
	}
	responsePage := generated.Page{Size: page.Size, HasMore: generated.NewOptBool(page.HasMore)}
	if page.After != "" {
		responsePage.After = generated.NewOptString(page.After)
	}
	return &generated.WfaApprovalRequestListEstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(uuid.Nil),
		AggregateVersion: 0,
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: workflowApprovalRequestListRoute},
		Data:             generated.WfaApprovalRequestListResultData{Requests: projections, Page: responsePage},
	}, nil
}

func (handler IdentityHandler) WfaGetApprovalRequest(ctx context.Context, params generated.WfaGetApprovalRequestParams) (generated.WfaGetApprovalRequestRes, error) {
	correlationID := workflowGetCorrelation(params)
	if handler.ApprovalRequestQueries == nil {
		return workflowGetApprovalRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-request read service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return workflowGetApprovalRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	request, err := handler.ApprovalRequestQueries.Get(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, uuid.UUID(params.ApprovalRequestId))
	if err != nil {
		return mapWorkflowGetApprovalRequestError(err, correlationID), nil
	}
	return &generated.WfaApprovalRequestEstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(request.ID),
		AggregateVersion: int(request.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: workflowApprovalRequestListRoute + "/" + request.ID.String()},
		Data: generated.WfaApprovalRequestResultData{
			Request:          generatedWorkflowApprovalRequestProjection(request),
			ValidationStatus: generated.NewOptString("validated"),
			ApprovalStatus:   generated.NewOptString(request.Status),
			ResponsibleRoles: append([]string(nil), request.ResponsibleRoles...),
		},
	}, nil
}

func workflowCreateCorrelation(params generated.WfaCreateApprovalRequestParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func workflowListCorrelation(params generated.WfaListApprovalRequestsParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func workflowGetCorrelation(params generated.WfaGetApprovalRequestParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func workflowCreateApprovalRequestCommand(request *generated.WfaCreateApprovalRequestCommandRequest, params generated.WfaCreateApprovalRequestParams) (workflow.CreateApprovalRequestCommand, error) {
	data := request.Data
	amount, err := decimal.NewFromString(string(data.Amount))
	if err != nil {
		return workflow.CreateApprovalRequestCommand{}, err
	}
	applicability := workflow.ApprovalApplicability{
		SourceContext:       data.Applicability.SourceContext,
		SubjectType:         data.Applicability.SubjectType,
		Operation:           data.Applicability.Operation,
		TransactionTypes:    append([]string(nil), data.Applicability.TransactionTypes...),
		DataClassifications: append([]string(nil), data.Applicability.DataClassifications...),
	}
	if data.Applicability.Dimensions.Set {
		applicability.Dimensions = generatedStringMap(map[string]string(data.Applicability.Dimensions.Value))
	}
	for _, currency := range data.Applicability.Currencies {
		applicability.Currencies = append(applicability.Currencies, string(currency))
	}
	snapshotAmount, err := workflowMoneyPointer(data.SubjectSnapshot.Amount)
	if err != nil {
		return workflow.CreateApprovalRequestCommand{}, err
	}
	snapshot := workflow.ApprovalSubjectSnapshot{
		Label:    data.SubjectSnapshot.Label,
		Summary:  data.SubjectSnapshot.Summary.Or(""),
		Amount:   snapshotAmount,
		Currency: string(data.SubjectSnapshot.Currency.Or(generated.CurrencyCode(""))),
	}
	if data.SubjectSnapshot.Reference.Set {
		snapshot.Reference = data.SubjectSnapshot.Reference.Value
	}
	scope := workflow.ApprovalPolicyScope{
		ScopeIDs:           append([]string(nil), data.Scope.ScopeIds...),
		AccountingScopeIDs: workflowUUIDs(data.Scope.AccountingScopeIds),
		LegalEntityIDs:     workflowUUIDs(data.Scope.LegalEntityIds),
		BusinessUnitIDs:    workflowUUIDs(data.Scope.BusinessUnitIds),
		SegmentIDs:         workflowUUIDs(data.Scope.SegmentIds),
		AccountIDs:         workflowUUIDs(data.Scope.AccountIds),
		FiscalPeriodIDs:    workflowUUIDs(data.Scope.FiscalPeriodIds),
	}
	return workflow.CreateApprovalRequestCommand{
		RequestID:          uuid.UUID(data.RequestId),
		SubjectType:        data.Applicability.SubjectType,
		SubjectID:          data.SubjectId,
		SubjectVersion:     int64(data.SubjectVersion),
		SubjectFingerprint: data.SubjectFingerprint,
		Scope:              scope,
		ResolvePolicy:      true,
		Applicability:      applicability,
		TransactionType:    data.TransactionType.Or(""),
		DataClassification: data.DataClassification.Or(""),
		Dimensions:         generatedCreateRequestDimensions(data.Dimensions),
		Amount:             &amount,
		Currency:           string(data.Currency),
		SubjectSnapshot:    snapshot,
		IdempotencyKey:     params.IdempotencyKey,
		CorrelationID:      workflowCreateCorrelation(params).String(),
		CausationID:        uuid.UUID(request.CommandId).String(),
	}, nil
}

func generatedStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, item := range values {
		result[key] = item
	}
	return result
}

func generatedCreateRequestDimensions(value generated.OptWfaCreateApprovalRequestCommandDataDimensions) map[string]string {
	if !value.Set {
		return nil
	}
	return generatedStringMap(map[string]string(value.Value))
}

func generatedWorkflowApprovalRequestProjection(request workflow.ApprovalRequest) generated.WfaApprovalRequestProjection {
	projection := generated.WfaApprovalRequestProjection{
		ID:                        generated.UUID(request.ID),
		SubjectType:               request.SubjectType,
		SubjectId:                 request.SubjectID,
		SubjectVersion:            int(request.SubjectVersion),
		SubjectFingerprint:        request.SubjectFingerprint,
		PolicyId:                  generated.UUID(request.PolicyID),
		PolicyVersion:             request.PolicyVersion,
		Scope:                     generatedWorkflowScope(request.Scope),
		Steps:                     generatedWorkflowApprovalRequestSteps(request.Steps),
		Status:                    generated.WfaApprovalRequestProjectionStatus(request.Status),
		AggregateVersion:          int(request.Version.Value()),
		ResponsibleRoles:          append([]string(nil), request.ResponsibleRoles...),
		CreatedAt:                 request.CreatedAt,
		UpdatedAt:                 request.UpdatedAt,
		RequesterUserId:           generated.NewOptUUID(generated.UUID(request.Requester.UserID)),
		RequesterSubjectReference: generated.NewOptString(request.Requester.SubjectReference),
	}
	if request.SnapshotFingerprint != "" {
		projection.SubjectSnapshot = generated.NewOptWfaApprovalSubjectSnapshot(generatedWorkflowSubjectSnapshot(request.SubjectSnapshot))
		projection.SnapshotFingerprint = generated.NewOptString(request.SnapshotFingerprint)
	}
	if request.SourceContext != "" {
		projection.SourceContext = generated.NewOptString(request.SourceContext)
	}
	if request.Operation != "" {
		projection.Operation = generated.NewOptString(request.Operation)
	}
	if request.TransactionType != "" {
		projection.TransactionType = generated.NewOptString(request.TransactionType)
	}
	if request.DataClassification != "" {
		projection.DataClassification = generated.NewOptString(request.DataClassification)
	}
	if len(request.Dimensions) > 0 {
		dimensions := generated.WfaApprovalRequestProjectionDimensions{}
		for key, value := range request.Dimensions {
			dimensions[key] = value
		}
		projection.Dimensions = generated.NewOptWfaApprovalRequestProjectionDimensions(dimensions)
	}
	if request.Amount != nil {
		projection.Amount = generated.NewOptMoney(generated.Money(request.Amount.String()))
	}
	if request.Currency != "" {
		projection.Currency = generated.NewOptCurrencyCode(generated.CurrencyCode(request.Currency))
	}
	if request.DueAt != nil {
		projection.DueAt = generated.NewOptDateTime(*request.DueAt)
	}
	return projection
}

func generatedWorkflowSubjectSnapshot(snapshot workflow.ApprovalSubjectSnapshot) generated.WfaApprovalSubjectSnapshot {
	result := generated.WfaApprovalSubjectSnapshot{Label: snapshot.Label}
	if snapshot.Reference != "" {
		result.Reference = generated.NewOptString(snapshot.Reference)
	}
	if snapshot.Summary != "" {
		result.Summary = generated.NewOptString(snapshot.Summary)
	}
	if snapshot.Amount != nil {
		result.Amount = generated.NewOptMoney(generated.Money(snapshot.Amount.String()))
	}
	if snapshot.Currency != "" {
		result.Currency = generated.NewOptCurrencyCode(generated.CurrencyCode(snapshot.Currency))
	}
	return result
}

func generatedWorkflowApprovalRequestSteps(values []workflow.ApprovalRequestStep) []generated.WfaApprovalRequestStep {
	result := make([]generated.WfaApprovalRequestStep, len(values))
	for index, value := range values {
		result[index] = generated.WfaApprovalRequestStep{
			ID:               generated.UUID(value.ID),
			Sequence:         value.Sequence,
			RequiredRoles:    generatedWorkflowRolesForRequirements(value.RequiredRoles),
			MinimumApprovals: value.MinimumApprovals,
			Status:           value.Status,
		}
		if value.Name != "" {
			result[index].Name = generated.NewOptString(value.Name)
		}
		result[index].Delegation = generated.NewOptWfaDelegationConfiguration(generatedWorkflowDelegation(value.Delegation))
		result[index].Escalation = generated.NewOptWfaEscalationConfiguration(generatedWorkflowEscalation(value.Escalation))
	}
	return result
}

func establishedWorkflowCreateApprovalRequestResult(result workflow.ApprovalRequestCommandResult, correlationID uuid.UUID) *generated.WfaCreateApprovalRequestEstablishedResult {
	request := result.Request
	return &generated.WfaCreateApprovalRequestEstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(request.ID),
		AggregateVersion: int(request.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: workflowCreateApprovalRequestRoute},
		Data: generated.WfaCreateApprovalRequestResultData{
			Request:          generatedWorkflowApprovalRequestProjection(request),
			ValidationStatus: "validated",
			ApprovalStatus:   request.Status,
			ResponsibleRoles: append([]string(nil), request.ResponsibleRoles...),
			Replayed:         result.Replayed,
		},
	}
}

func mapWorkflowCreateApprovalRequestError(err error, correlationID uuid.UUID) generated.WfaCreateApprovalRequestRes {
	switch {
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationDenied), errors.Is(err, workflow.ErrApprovalPolicySelfApproval):
		return workflowCreateApprovalRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The approval request is outside the requesting actor's authority.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationUnavailable), errors.Is(err, workflow.ErrApprovalPolicyAuditUnavailable), errors.Is(err, workflow.ErrApprovalPolicyApprovalUnavailable), errors.Is(err, workflow.ErrApprovalPolicyResolverUnavailable), errors.Is(err, workflow.ErrInvalidApprovalPolicyService):
		return workflowCreateApprovalRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-request operation could not reach a required dependency.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyNotApplicable):
		return workflowCreateApprovalRequestProblem(http.StatusConflict, "POLICY_NOT_APPLICABLE", "No effective approval policy matches the request context.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAmbiguous):
		return workflowCreateApprovalRequestProblem(http.StatusConflict, "POLICY_AMBIGUOUS", "More than one effective approval policy matches the request context.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicySubjectUnavailable):
		return workflowCreateApprovalRequestProblem(http.StatusConflict, "SUBJECT_EVIDENCE_UNAVAILABLE", "The approval subject evidence could not be established.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicySubjectConflict), errors.Is(err, workflow.ErrApprovalPolicyAuthorizationStale), errors.Is(err, workflow.ErrApprovalPolicyVersionConflict), errors.Is(err, workflow.ErrApprovalPolicyIdempotencyConflict), errors.Is(err, workflow.ErrApprovalPolicyCommandInProgress), errors.Is(err, workflow.ErrApprovalPolicyDuplicate):
		code, detail := "VERSION_CONFLICT", "The approval request source or policy changed. Refresh and retry."
		if errors.Is(err, workflow.ErrApprovalPolicyIdempotencyConflict) {
			code, detail = "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data."
		}
		return workflowCreateApprovalRequestProblem(http.StatusConflict, code, detail, correlationID)
	case errors.Is(err, workflow.ErrInvalidApprovalPolicyCommand):
		return workflowCreateApprovalRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval-request command is invalid.", correlationID)
	default:
		return workflowCreateApprovalRequestProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The approval-request command violates a workflow rule.", correlationID)
	}
}

func mapWorkflowListApprovalRequestError(err error, correlationID uuid.UUID) generated.WfaListApprovalRequestsRes {
	if errors.Is(err, workflow.ErrInvalidApprovalPolicyCommand) || errors.Is(err, workflow.ErrApprovalPolicyInvalidPage) {
		return workflowListApprovalRequestProblem(http.StatusBadRequest, "INVALID_PAGE", "The approval-request list filter or page cursor is invalid.", correlationID)
	}
	if errors.Is(err, workflow.ErrApprovalPolicyAuthorizationDenied) || errors.Is(err, workflow.ErrApprovalPolicyAuthorizationStale) {
		return workflowListApprovalRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized to read approval requests.", correlationID)
	}
	return workflowListApprovalRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-request read could not be completed.", correlationID)
}

func mapWorkflowGetApprovalRequestError(err error, correlationID uuid.UUID) generated.WfaGetApprovalRequestRes {
	switch {
	case errors.Is(err, workflow.ErrApprovalPolicyNotFound):
		return workflowGetApprovalRequestProblem(http.StatusNotFound, "REQUEST_NOT_FOUND", "The approval request does not exist.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationDenied), errors.Is(err, workflow.ErrApprovalPolicyAuthorizationStale):
		return workflowGetApprovalRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized to read this approval request.", correlationID)
	case errors.Is(err, workflow.ErrInvalidApprovalPolicyCommand):
		return workflowGetApprovalRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval request identifier is invalid.", correlationID)
	default:
		return workflowGetApprovalRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-request read could not be completed.", correlationID)
	}
}

func workflowCreateApprovalRequestProblem(status int, code, detail string, correlationID uuid.UUID) generated.WfaCreateApprovalRequestRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.WfaCreateApprovalRequestBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.WfaCreateApprovalRequestForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.WfaCreateApprovalRequestConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.WfaCreateApprovalRequestUnprocessableEntity(problem)
		return &value
	default:
		value := generated.WfaCreateApprovalRequestServiceUnavailable(problem)
		return &value
	}
}

func workflowListApprovalRequestProblem(status int, code, detail string, correlationID uuid.UUID) generated.WfaListApprovalRequestsRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.WfaListApprovalRequestsBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.WfaListApprovalRequestsForbidden(problem)
		return &value
	default:
		value := generated.WfaListApprovalRequestsServiceUnavailable(problem)
		return &value
	}
}

func workflowGetApprovalRequestProblem(status int, code, detail string, correlationID uuid.UUID) generated.WfaGetApprovalRequestRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.WfaGetApprovalRequestBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.WfaGetApprovalRequestForbidden(problem)
		return &value
	case http.StatusNotFound:
		value := generated.WfaGetApprovalRequestNotFound(problem)
		return &value
	default:
		value := generated.WfaGetApprovalRequestServiceUnavailable(problem)
		return &value
	}
}
