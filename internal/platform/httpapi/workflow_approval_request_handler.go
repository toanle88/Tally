package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/workflow"
)

const (
	workflowCreateApprovalRequestRoute = "/api/v1/approvals/actions/create-approval-request"
	workflowDecideApprovalRequestRoute = "/api/v1/approvals/actions/decide-approval-request"
	workflowDelegateApprovalRoute      = "/api/v1/approvals/actions/delegate-approval"
	workflowEscalateApprovalRoute      = "/api/v1/approvals/actions/escalate-approval"
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

func (handler IdentityHandler) WfaDecideApprovalRequest(ctx context.Context, request *generated.WfaDecideApprovalRequestCommandRequest, params generated.WfaDecideApprovalRequestParams) (generated.WfaDecideApprovalRequestRes, error) {
	correlationID := workflowDecideCorrelation(params)
	if handler.ApprovalRequestService == nil {
		return workflowDecideApprovalRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-request service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return workflowDecideApprovalRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The deciding actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return workflowDecideApprovalRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision command requires a command identifier.", correlationID), nil
	}
	command, err := workflowDecideApprovalRequestCommand(request, params)
	if err != nil {
		return workflowDecideApprovalRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision command is invalid.", correlationID), nil
	}
	result, err := handler.ApprovalRequestService.DecideApprovalRequest(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapWorkflowDecideApprovalRequestError(err, correlationID), nil
	}
	return establishedWorkflowApprovalRequestResult(result, correlationID), nil
}

func (handler IdentityHandler) WfaDelegateApproval(ctx context.Context, request *generated.WfaDelegateApprovalCommandRequest, params generated.WfaDelegateApprovalParams) (generated.WfaDelegateApprovalRes, error) {
	correlationID := workflowDelegateCorrelation(params)
	if handler.DelegationService == nil {
		return workflowDelegateApprovalProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The delegation service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return workflowDelegateApprovalProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The delegating actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return workflowDelegateApprovalProblem(http.StatusBadRequest, "INVALID_REQUEST", "The delegation command requires a command identifier.", correlationID), nil
	}
	data := request.Data
	command := workflow.DelegateApprovalCommand{
		DelegationID:   uuid.UUID(data.DelegationId),
		DelegateUserID: uuid.UUID(data.DelegateUserId),
		PolicyID:       uuid.UUID(data.PolicyId),
		PolicyVersion:  strings.TrimSpace(data.PolicyVersion),
		Scope:          workflowPolicyScope(data.Scope),
		EffectiveFrom:  data.EffectiveFrom,
		EffectiveTo:    data.EffectiveTo,
		IdempotencyKey: params.IdempotencyKey,
		CorrelationID:  correlationID.String(),
		CausationID:    uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.DelegationService.DelegateApproval(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapWorkflowDelegateApprovalError(err, correlationID), nil
	}
	return establishedWorkflowDelegateApprovalResult(result, correlationID), nil
}

func (handler IdentityHandler) WfaEscalateApproval(ctx context.Context, request *generated.WfaEscalateApprovalCommandRequest, params generated.WfaEscalateApprovalParams) (generated.WfaEscalateApprovalRes, error) {
	correlationID := workflowEscalateCorrelation(params)
	if handler.ApprovalRequestService == nil {
		return workflowEscalateApprovalProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-request service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return workflowEscalateApprovalProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The escalating actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return workflowEscalateApprovalProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval escalation command requires a command identifier.", correlationID), nil
	}
	command, err := workflowEscalateApprovalCommand(request, params, correlationID)
	if err != nil {
		return workflowEscalateApprovalProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval escalation command is invalid.", correlationID), nil
	}
	result, err := handler.ApprovalRequestService.EscalateApproval(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapWorkflowEscalateApprovalError(err, correlationID), nil
	}
	return establishedWorkflowEscalateApprovalResult(result, correlationID), nil
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
			Request:           generatedWorkflowApprovalRequestProjection(request),
			ValidationStatus:  generated.NewOptString("validated"),
			ApprovalStatus:    generated.NewOptString(request.Status),
			ResponsibleRoles:  append([]string(nil), request.ResponsibleRoles...),
			ApplicationStatus: generated.WfaApprovalRequestResultDataApplicationStatusNotReported,
			Replayed:          false,
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

func workflowDecideCorrelation(params generated.WfaDecideApprovalRequestParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func workflowDelegateCorrelation(params generated.WfaDelegateApprovalParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func workflowEscalateCorrelation(params generated.WfaEscalateApprovalParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func workflowDecideApprovalRequestCommand(request *generated.WfaDecideApprovalRequestCommandRequest, params generated.WfaDecideApprovalRequestParams) (workflow.DecideApprovalRequestCommand, error) {
	if request == nil || request.ExpectedVersion < 1 {
		return workflow.DecideApprovalRequestCommand{}, errors.New("expected version is required")
	}
	data := request.Data
	expectedVersion, err := aggregateversion.FromInt64(int64(request.ExpectedVersion))
	if err != nil {
		return workflow.DecideApprovalRequestCommand{}, err
	}
	command := workflow.DecideApprovalRequestCommand{
		RequestID:            uuid.UUID(data.RequestId),
		ExpectedVersion:      &expectedVersion,
		DecisionID:           uuid.UUID(data.DecisionId),
		StepID:               uuid.UUID(data.StepId),
		Outcome:              strings.TrimSpace(string(data.Outcome)),
		Reason:               strings.TrimSpace(data.Reason.Or("")),
		PolicyVersion:        strings.TrimSpace(data.PolicyVersion),
		SubjectVersion:       int64(data.SubjectVersion),
		CandidateFingerprint: strings.TrimSpace(data.CandidateFingerprint),
		IdempotencyKey:       params.IdempotencyKey,
		CorrelationID:        workflowDecideCorrelation(params).String(),
		CausationID:          uuid.UUID(request.CommandId).String(),
	}
	if data.DelegationId.Set {
		command.DelegationID = uuid.UUID(data.DelegationId.Value)
	}
	if data.DelegationReference.Set {
		command.DelegationReference = strings.TrimSpace(data.DelegationReference.Value)
	}
	return command, nil
}

func workflowEscalateApprovalCommand(request *generated.WfaEscalateApprovalCommandRequest, params generated.WfaEscalateApprovalParams, correlationID uuid.UUID) (workflow.EscalateApprovalCommand, error) {
	if request == nil || request.ExpectedVersion < 1 {
		return workflow.EscalateApprovalCommand{}, errors.New("expected version is required")
	}
	expectedVersion, err := aggregateversion.FromInt64(int64(request.ExpectedVersion))
	if err != nil {
		return workflow.EscalateApprovalCommand{}, err
	}
	data := request.Data
	return workflow.EscalateApprovalCommand{
		RequestID:            uuid.UUID(data.RequestId),
		ExpectedVersion:      &expectedVersion,
		EscalationID:         uuid.UUID(data.EscalationId),
		StepID:               uuid.UUID(data.StepId),
		Reason:               strings.TrimSpace(string(data.Reason)),
		Blocker:              strings.TrimSpace(data.Blocker.Or("")),
		PolicyVersion:        strings.TrimSpace(data.PolicyVersion),
		SubjectVersion:       int64(data.SubjectVersion),
		CandidateFingerprint: strings.TrimSpace(data.CandidateFingerprint),
		IdempotencyKey:       params.IdempotencyKey,
		CorrelationID:        correlationID.String(),
		CausationID:          uuid.UUID(request.CommandId).String(),
	}, nil
}

func workflowPolicyScope(value generated.WfaApprovalScope) workflow.ApprovalPolicyScope {
	return workflow.ApprovalPolicyScope{
		ScopeIDs:           append([]string(nil), value.ScopeIds...),
		AccountingScopeIDs: workflowUUIDs(value.AccountingScopeIds),
		LegalEntityIDs:     workflowUUIDs(value.LegalEntityIds),
		BusinessUnitIDs:    workflowUUIDs(value.BusinessUnitIds),
		SegmentIDs:         workflowUUIDs(value.SegmentIds),
		AccountIDs:         workflowUUIDs(value.AccountIds),
		FiscalPeriodIDs:    workflowUUIDs(value.FiscalPeriodIds),
	}
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
	projection.ApplicationStatus = generated.WfaApprovalRequestProjectionApplicationStatusNotReported
	projection.Decisions = generatedWorkflowApprovalDecisions(request.Decisions)
	projection.Escalations = generatedWorkflowApprovalEscalations(request.Escalations)
	if currentStep := workflowCurrentApprovalStep(request); currentStep != uuid.Nil {
		projection.CurrentStepId = generated.NewOptUUID(generated.UUID(currentStep))
	}
	if age := int(time.Since(request.CreatedAt).Seconds()); !request.CreatedAt.IsZero() && age >= 0 {
		projection.AgeSeconds = generated.NewOptInt(age)
	}
	if request.SnapshotFingerprint != "" {
		projection.SubjectSnapshot = generated.NewOptWfaApprovalSubjectSnapshot(generatedWorkflowSubjectSnapshot(request.SubjectSnapshot))
		projection.SnapshotFingerprint = generated.NewOptString(request.SnapshotFingerprint)
	}
	if request.SourceContext != "" {
		projection.SourceContext = generated.NewOptString(request.SourceContext)
	}
	if request.ActiveDelegation != nil {
		projection.ActiveDelegation = generated.NewOptWfaDelegationProjection(generatedWorkflowDelegationProjection(*request.ActiveDelegation))
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
	currentStep := workflowCurrentApprovalStepValue(request)
	if currentStep != nil {
		projection.CurrentOwnerRoles = workflowApprovalRequestStepRoles(*currentStep)
	}
	if currentEscalation := workflowCurrentApprovalEscalation(request, currentStep); currentEscalation != nil {
		projection.EscalationTargetRoles = append([]string(nil), currentEscalation.TargetRoles...)
		if currentEscalation.Blocker != "" {
			projection.Blocker = generated.NewOptString(currentEscalation.Blocker)
		}
		projection.NextAction = generated.NewOptString("review-escalated-approval")
		projection.RecoveryPath = generated.NewOptString("Review the escalation evidence, refresh the request, and decide the current step.")
	} else if request.Status == workflow.ApprovalRequestStatusPending {
		projection.NextAction = generated.NewOptString("awaiting-decision")
		projection.RecoveryPath = generated.NewOptString("Await the assigned approval owner or escalate when the configured threshold is reached.")
	}
	return projection
}

func generatedWorkflowDelegationProjection(delegation workflow.Delegation) generated.WfaDelegationProjection {
	return generated.WfaDelegationProjection{
		ID:               generated.UUID(delegation.ID),
		DelegatorUserId:  generated.UUID(delegation.DelegatorUserID),
		DelegateUserId:   generated.UUID(delegation.DelegateUserID),
		PolicyId:         generated.UUID(delegation.PolicyID),
		PolicyVersion:    delegation.PolicyVersion,
		Scope:            generatedWorkflowScope(delegation.Scope),
		EffectiveFrom:    delegation.EffectiveFrom,
		EffectiveTo:      delegation.EffectiveTo,
		Active:           delegation.ActiveAt(time.Now().UTC()),
		AggregateVersion: int(delegation.Version.Value()),
		CreatedAt:        delegation.CreatedAt,
		UpdatedAt:        delegation.UpdatedAt,
	}
}

func establishedWorkflowDelegateApprovalResult(result workflow.DelegationCommandResult, correlationID uuid.UUID) *generated.WfaDelegateApprovalEstablishedResult {
	response := &generated.WfaDelegateApprovalEstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(result.Delegation.ID),
		AggregateVersion: int(result.Delegation.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: workflowDelegateApprovalRoute},
		Data: generated.WfaDelegateApprovalResultData{
			Delegation:       generatedWorkflowDelegationProjection(result.Delegation),
			ValidationStatus: "validated",
			Replayed:         result.Replayed,
		},
	}
	if result.Delegation.AuditReference != uuid.Nil {
		response.Data.AuditReference = generated.NewOptUUID(generated.UUID(result.Delegation.AuditReference))
	}
	return response
}

func establishedWorkflowEscalateApprovalResult(result workflow.ApprovalRequestCommandResult, correlationID uuid.UUID) *generated.WfaEscalateApprovalEstablishedResult {
	request := result.Request
	var escalation workflow.ApprovalEscalation
	if result.Escalation != nil {
		escalation = *result.Escalation
	} else if len(request.Escalations) > 0 {
		escalation = request.Escalations[len(request.Escalations)-1]
	}
	return &generated.WfaEscalateApprovalEstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(request.ID),
		AggregateVersion: int(request.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: workflowEscalateApprovalRoute},
		Data: generated.WfaEscalateApprovalResultData{
			Request:          generatedWorkflowApprovalRequestProjection(request),
			Escalation:       generatedWorkflowApprovalEscalations([]workflow.ApprovalEscalation{escalation})[0],
			ValidationStatus: "validated",
			NextAction:       "review-escalated-approval",
			RecoveryPath:     "Review the escalation evidence, refresh the request, and decide the current step.",
			Replayed:         result.Replayed,
		},
	}
}

func generatedWorkflowApprovalDecisions(values []workflow.ApprovalDecision) []generated.WfaApprovalDecisionProjection {
	result := make([]generated.WfaApprovalDecisionProjection, len(values))
	for index, value := range values {
		result[index] = generated.WfaApprovalDecisionProjection{
			ID:                       generated.UUID(value.ID),
			DecisionVersion:          int(value.DecisionVersion),
			Outcome:                  generated.WfaApprovalDecisionProjectionOutcome(value.Outcome),
			ApproverUserId:           generated.UUID(value.Approver.UserID),
			ApproverSubjectReference: value.Approver.SubjectReference,
			Reason:                   value.Reason,
			PolicyVersion:            value.PolicyVersion,
			SubjectVersion:           int(value.SubjectVersion),
			CandidateFingerprint:     value.CandidateFingerprint,
			RecordedAt:               value.RecordedAt,
		}
		if value.StepID != uuid.Nil {
			result[index].StepId = generated.NewOptUUID(generated.UUID(value.StepID))
		}
		if value.DelegationReference != "" {
			result[index].DelegationReference = generated.NewOptString(value.DelegationReference)
		}
	}
	return result
}

func generatedWorkflowApprovalEscalations(values []workflow.ApprovalEscalation) []generated.WfaApprovalEscalationProjection {
	result := make([]generated.WfaApprovalEscalationProjection, len(values))
	for index, value := range values {
		result[index] = generated.WfaApprovalEscalationProjection{
			ID:                   generated.UUID(value.ID),
			StepId:               generated.UUID(value.StepID),
			Reason:               generated.WfaApprovalEscalationProjectionReason(value.Reason),
			ResponsibleRoles:     append([]string(nil), value.ResponsibleRoles...),
			TargetRoles:          append([]string(nil), value.TargetRoles...),
			PolicyId:             generated.UUID(value.PolicyID),
			PolicyVersion:        value.PolicyVersion,
			SubjectVersion:       int(value.SubjectVersion),
			CandidateFingerprint: value.CandidateFingerprint,
			OccurredAt:           value.OccurredAt,
			AuditReference:       generated.UUID(value.AuditReference),
		}
		if value.Blocker != "" {
			result[index].Blocker = generated.NewOptString(value.Blocker)
		}
		if value.ActorUserID != uuid.Nil {
			result[index].ActorUserId = generated.NewOptUUID(generated.UUID(value.ActorUserID))
		}
		if value.ActorSubjectReference != "" {
			result[index].ActorSubjectReference = generated.NewOptString(value.ActorSubjectReference)
		}
		if value.ProcessReference != "" {
			result[index].ProcessReference = generated.NewOptString(value.ProcessReference)
		}
		if value.DueAt != nil {
			result[index].DueAt = generated.NewOptDateTime(*value.DueAt)
		}
	}
	return result
}

func workflowCurrentApprovalStep(request workflow.ApprovalRequest) uuid.UUID {
	current := workflowCurrentApprovalStepValue(request)
	if current == nil {
		return uuid.Nil
	}
	return current.ID
}

func workflowCurrentApprovalStepValue(request workflow.ApprovalRequest) *workflow.ApprovalRequestStep {
	var current *workflow.ApprovalRequestStep
	for index := range request.Steps {
		step := &request.Steps[index]
		if (step.Status != workflow.ApprovalRequestStatusPending && step.Status != workflow.ApprovalRequestStepStatusEscalated) || (current != nil && step.Sequence >= current.Sequence) {
			continue
		}
		current = step
	}
	return current
}

func workflowCurrentApprovalEscalation(request workflow.ApprovalRequest, currentStep *workflow.ApprovalRequestStep) *workflow.ApprovalEscalation {
	if request.Status != workflow.ApprovalRequestStatusPending || currentStep == nil || currentStep.Status != workflow.ApprovalRequestStepStatusEscalated {
		return nil
	}
	for index := len(request.Escalations) - 1; index >= 0; index-- {
		if request.Escalations[index].StepID == currentStep.ID {
			escalation := request.Escalations[index]
			return &escalation
		}
	}
	return nil
}

func workflowApprovalRequestStepRoles(step workflow.ApprovalRequestStep) []string {
	roles := make([]string, 0, len(step.RequiredRoles))
	seen := make(map[string]struct{}, len(step.RequiredRoles))
	for _, requirement := range step.RequiredRoles {
		role := strings.TrimSpace(requirement.Role)
		if role == "" {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	return roles
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
			Request:           generatedWorkflowApprovalRequestProjection(request),
			ValidationStatus:  "validated",
			ApprovalStatus:    request.Status,
			ResponsibleRoles:  append([]string(nil), request.ResponsibleRoles...),
			ApplicationStatus: generated.WfaCreateApprovalRequestResultDataApplicationStatusNotReported,
			Replayed:          result.Replayed,
		},
	}
}

func establishedWorkflowApprovalRequestResult(result workflow.ApprovalRequestCommandResult, correlationID uuid.UUID) *generated.WfaApprovalRequestEstablishedResult {
	request := result.Request
	data := generated.WfaApprovalRequestResultData{
		Request:           generatedWorkflowApprovalRequestProjection(request),
		ValidationStatus:  generated.NewOptString("validated"),
		ApprovalStatus:    generated.NewOptString(request.Status),
		ResponsibleRoles:  append([]string(nil), request.ResponsibleRoles...),
		ApplicationStatus: generated.WfaApprovalRequestResultDataApplicationStatusNotReported,
		NextAction:        generated.NewOptString(workflowApprovalDecisionNextAction(request.Status)),
		Replayed:          result.Replayed,
	}
	if len(request.Decisions) > 0 {
		last := request.Decisions[len(request.Decisions)-1]
		data.Decision = generated.NewOptWfaApprovalDecisionProjection(generatedWorkflowApprovalDecisions([]workflow.ApprovalDecision{last})[0])
	}
	reference := result.Reference
	if reference.ApprovalRequestID == uuid.Nil && len(request.Decisions) > 0 {
		last := request.Decisions[len(request.Decisions)-1]
		reference = workflow.ApprovalDecisionReference{ApprovalRequestID: request.ID, DecisionID: last.ID, PolicyVersion: last.PolicyVersion, DecisionVersion: last.DecisionVersion, SubjectVersion: last.SubjectVersion, CandidateFingerprint: last.CandidateFingerprint, ApproverUserID: last.Approver.UserID, DecidedAt: last.RecordedAt}
	}
	if reference.ApprovalRequestID != uuid.Nil {
		mapped := generated.WfaApprovalDecisionReference{ApprovalRequestId: generated.UUID(reference.ApprovalRequestID), DecisionId: generated.UUID(reference.DecisionID), PolicyVersion: reference.PolicyVersion, DecisionVersion: int(reference.DecisionVersion), SubjectVersion: int(reference.SubjectVersion), CandidateFingerprint: reference.CandidateFingerprint, ApproverUserId: generated.UUID(reference.ApproverUserID), DecidedAt: reference.DecidedAt}
		if reference.AppliedAt != nil {
			mapped.AppliedAt = generated.NewOptDateTime(*reference.AppliedAt)
		}
		data.DecisionReference = generated.NewOptWfaApprovalDecisionReference(mapped)
	}
	return &generated.WfaApprovalRequestEstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(request.ID),
		AggregateVersion: int(request.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: workflowDecideApprovalRequestRoute},
		Data:             data,
	}
}

func workflowApprovalDecisionNextAction(status string) string {
	if status == workflow.ApprovalRequestStatusPending {
		return "awaiting-decision"
	}
	return "awaiting-application-revalidation"
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

func mapWorkflowDecideApprovalRequestError(err error, correlationID uuid.UUID) generated.WfaDecideApprovalRequestRes {
	switch {
	case errors.Is(err, workflow.ErrApprovalPolicyNotFound):
		return workflowDecideApprovalRequestProblem(http.StatusNotFound, "REQUEST_NOT_FOUND", "The approval request does not exist.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationDenied), errors.Is(err, workflow.ErrApprovalPolicySelfApproval), errors.Is(err, workflow.ErrDelegationAuthorizationDenied):
		return workflowDecideApprovalRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The deciding actor is outside the approval request authority.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationUnavailable), errors.Is(err, workflow.ErrApprovalPolicyApprovalUnavailable), errors.Is(err, workflow.ErrApprovalPolicyAuditUnavailable), errors.Is(err, workflow.ErrApprovalPolicyResolverUnavailable), errors.Is(err, workflow.ErrDelegationAuthorizationUnavailable), errors.Is(err, workflow.ErrInvalidApprovalPolicyService):
		return workflowDecideApprovalRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval decision could not reach a required dependency.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationStale), errors.Is(err, workflow.ErrApprovalPolicyVersionConflict), errors.Is(err, workflow.ErrApprovalPolicySubjectConflict), errors.Is(err, workflow.ErrApprovalPolicyIdempotencyConflict), errors.Is(err, workflow.ErrApprovalPolicyCommandInProgress), errors.Is(err, workflow.ErrApprovalPolicyExpired), errors.Is(err, workflow.ErrApprovalPolicyStepConflict), errors.Is(err, workflow.ErrApprovalPolicyApprovalRejected), errors.Is(err, workflow.ErrDelegationAuthorizationStale), errors.Is(err, workflow.ErrDelegationExpired), errors.Is(err, workflow.ErrDelegationPolicyConflict), errors.Is(err, workflow.ErrDelegationScopeConflict):
		code, detail := "VERSION_CONFLICT", "The approval request or authorization evidence changed. Refresh and retry."
		switch {
		case errors.Is(err, workflow.ErrApprovalPolicyIdempotencyConflict):
			code, detail = "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data."
		case errors.Is(err, workflow.ErrApprovalPolicyCommandInProgress):
			code, detail = "COMMAND_IN_PROGRESS", "An equivalent approval decision is already being processed."
		case errors.Is(err, workflow.ErrApprovalPolicyExpired):
			code, detail = "APPROVAL_EXPIRED", "The approval policy is no longer effective. Refresh the request."
		case errors.Is(err, workflow.ErrApprovalPolicyStepConflict):
			code, detail = "STEP_CONFLICT", "The submitted approval step is not the current actionable step."
		case errors.Is(err, workflow.ErrApprovalPolicyApprovalRejected):
			code, detail = "REQUEST_NOT_ACTIONABLE", "The approval request is already decided or otherwise not actionable."
		}
		return workflowDecideApprovalRequestProblem(http.StatusConflict, code, detail, correlationID)
	case errors.Is(err, workflow.ErrInvalidApprovalPolicyCommand):
		return workflowDecideApprovalRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision command is invalid.", correlationID)
	default:
		return workflowDecideApprovalRequestProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The approval decision violates a workflow rule.", correlationID)
	}
}

func mapWorkflowDelegateApprovalError(err error, correlationID uuid.UUID) generated.WfaDelegateApprovalRes {
	switch {
	case errors.Is(err, workflow.ErrDelegationAuthorizationDenied):
		return workflowDelegateApprovalProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The delegating actor or delegate is outside the permitted authority.", correlationID)
	case errors.Is(err, workflow.ErrDelegationAuthorizationUnavailable), errors.Is(err, workflow.ErrDelegationAuditUnavailable):
		return workflowDelegateApprovalProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The delegation operation could not reach a required dependency.", correlationID)
	case errors.Is(err, workflow.ErrDelegationAuthorizationStale):
		return workflowDelegateApprovalProblem(http.StatusConflict, "AUTHORIZATION_STALE", "Authorization changed. Refresh the delegation form and retry.", correlationID)
	case errors.Is(err, workflow.ErrDelegationIdempotencyConflict):
		return workflowDelegateApprovalProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different delegation data.", correlationID)
	case errors.Is(err, workflow.ErrDelegationCommandInProgress):
		return workflowDelegateApprovalProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "An equivalent delegation command is already being processed.", correlationID)
	case errors.Is(err, workflow.ErrDelegationDuplicate):
		return workflowDelegateApprovalProblem(http.StatusConflict, "DUPLICATE", "The delegation identity is already established.", correlationID)
	case errors.Is(err, workflow.ErrDelegationPolicyConflict), errors.Is(err, workflow.ErrDelegationScopeConflict), errors.Is(err, workflow.ErrDelegationExpired):
		return workflowDelegateApprovalProblem(http.StatusConflict, "DELEGATION_CONFLICT", "The policy, scope, or effective interval no longer permits this delegation.", correlationID)
	case errors.Is(err, workflow.ErrInvalidDelegation), errors.Is(err, workflow.ErrInvalidDelegationCommand):
		return workflowDelegateApprovalProblem(http.StatusBadRequest, "INVALID_REQUEST", "The delegation command is invalid.", correlationID)
	default:
		return workflowDelegateApprovalProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The delegation command violates a workflow rule.", correlationID)
	}
}

func mapWorkflowEscalateApprovalError(err error, correlationID uuid.UUID) generated.WfaEscalateApprovalRes {
	switch {
	case errors.Is(err, workflow.ErrApprovalPolicyNotFound):
		return workflowEscalateApprovalProblem(http.StatusNotFound, "REQUEST_NOT_FOUND", "The approval request does not exist.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationDenied):
		return workflowEscalateApprovalProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The actor is outside the approval escalation authority.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationUnavailable), errors.Is(err, workflow.ErrApprovalPolicyAuditUnavailable), errors.Is(err, workflow.ErrInvalidApprovalPolicyService):
		return workflowEscalateApprovalProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval escalation could not reach a required dependency.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationStale):
		return workflowEscalateApprovalProblem(http.StatusConflict, "AUTHORIZATION_STALE", "Authorization changed. Refresh the approval evidence and retry.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyIdempotencyConflict):
		return workflowEscalateApprovalProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different escalation data.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyCommandInProgress):
		return workflowEscalateApprovalProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "An equivalent approval escalation is already being processed.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyVersionConflict), errors.Is(err, workflow.ErrApprovalPolicySubjectConflict), errors.Is(err, workflow.ErrApprovalPolicyExpired), errors.Is(err, workflow.ErrApprovalPolicyStepConflict), errors.Is(err, workflow.ErrApprovalPolicyEscalationNotEligible), errors.Is(err, workflow.ErrApprovalPolicyAlreadyEscalated), errors.Is(err, workflow.ErrApprovalPolicyNotApplicable), errors.Is(err, workflow.ErrApprovalPolicyAmbiguous):
		return workflowEscalateApprovalProblem(http.StatusConflict, "ESCALATION_CONFLICT", "The approval request or escalation evidence changed, or the step is not eligible. Refresh and retry.", correlationID)
	case errors.Is(err, workflow.ErrInvalidApprovalPolicyCommand):
		return workflowEscalateApprovalProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval escalation command is invalid.", correlationID)
	default:
		return workflowEscalateApprovalProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The approval escalation violates a workflow rule.", correlationID)
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

func workflowDecideApprovalRequestProblem(status int, code, detail string, correlationID uuid.UUID) generated.WfaDecideApprovalRequestRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.WfaDecideApprovalRequestBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.WfaDecideApprovalRequestForbidden(problem)
		return &value
	case http.StatusNotFound:
		value := generated.WfaDecideApprovalRequestNotFound(problem)
		return &value
	case http.StatusConflict:
		value := generated.WfaDecideApprovalRequestConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.WfaDecideApprovalRequestUnprocessableEntity(problem)
		return &value
	default:
		value := generated.WfaDecideApprovalRequestServiceUnavailable(problem)
		return &value
	}
}

func workflowDelegateApprovalProblem(status int, code, detail string, correlationID uuid.UUID) generated.WfaDelegateApprovalRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.WfaDelegateApprovalBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.WfaDelegateApprovalForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.WfaDelegateApprovalConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.WfaDelegateApprovalUnprocessableEntity(problem)
		return &value
	default:
		value := generated.WfaDelegateApprovalServiceUnavailable(problem)
		return &value
	}
}

func workflowEscalateApprovalProblem(status int, code, detail string, correlationID uuid.UUID) generated.WfaEscalateApprovalRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.WfaEscalateApprovalBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.WfaEscalateApprovalForbidden(problem)
		return &value
	case http.StatusNotFound:
		value := generated.WfaEscalateApprovalNotFound(problem)
		return &value
	case http.StatusConflict:
		value := generated.WfaEscalateApprovalConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.WfaEscalateApprovalUnprocessableEntity(problem)
		return &value
	default:
		value := generated.WfaEscalateApprovalServiceUnavailable(problem)
		return &value
	}
}
