package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/coa"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type coaSegmentDefinitionCommandData struct {
	Action              string  `json:"action"`
	SegmentDefinitionID string  `json:"segmentDefinitionId"`
	SegmentType         string  `json:"segmentType"`
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	Status              string  `json:"status"`
	EffectiveDateFrom   string  `json:"effectiveDateFrom"`
	EffectiveDateTo     *string `json:"effectiveDateTo"`
}

type coaSegmentValueCommandData struct {
	Action              string  `json:"action"`
	SegmentDefinitionID string  `json:"segmentDefinitionId"`
	SegmentValueID      string  `json:"segmentValueId"`
	Value               string  `json:"value"`
	Description         string  `json:"description"`
	Status              string  `json:"status"`
	EffectiveDateFrom   string  `json:"effectiveDateFrom"`
	EffectiveDateTo     *string `json:"effectiveDateTo"`
}

type coaSegmentChangeRequestCommandData struct {
	ChangeType             string          `json:"changeType"`
	SubjectID              string          `json:"subjectId"`
	SubjectVersion         int64           `json:"subjectVersion"`
	RequestedEffectiveDate string          `json:"requestedEffectiveDate"`
	ApprovalRequestID      string          `json:"approvalRequestId"`
	ProposedChange         json.RawMessage `json:"proposedChange"`
}

type coaSegmentChangeApprovalDecisionCommandData struct {
	SegmentChangeRequestID string `json:"segmentChangeRequestId"`
	Outcome                string `json:"outcome"`
	ApprovalRequestID      string `json:"approvalRequestId"`
	DecisionID             string `json:"decisionId"`
	PolicyVersion          string `json:"policyVersion"`
	DecisionVersion        int64  `json:"decisionVersion"`
	SubjectVersion         int64  `json:"subjectVersion"`
	CandidateFingerprint   string `json:"candidateFingerprint"`
	ApproverUserID         string `json:"approverUserId"`
	DecidedAt              string `json:"decidedAt"`
}

type coaSegmentChangeProposalData struct {
	Action              string  `json:"action"`
	SegmentDefinitionID string  `json:"segmentDefinitionId"`
	SegmentValueID      string  `json:"segmentValueId"`
	SegmentType         string  `json:"segmentType"`
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	Value               string  `json:"value"`
	Description         string  `json:"description"`
	Status              string  `json:"status"`
	EffectiveDateFrom   string  `json:"effectiveDateFrom"`
	EffectiveDateTo     *string `json:"effectiveDateTo"`
}

type coaSegmentCombinationValueData struct {
	SegmentDefinitionID string `json:"segmentDefinitionId"`
	SegmentValueID      string `json:"segmentValueId"`
}

type coaSegmentCombinationValidationData struct {
	SegmentValues []coaSegmentCombinationValueData `json:"segmentValues"`
}

func (handler IdentityHandler) CoaValidateSegmentCombinations(ctx context.Context, request *generated.CommandRequest, params generated.CoaValidateSegmentCombinationsParams) (generated.CoaValidateSegmentCombinationsRes, error) {
	correlationID := correlationFromCoaValidationParams(params)
	if handler.SegmentCombinationValidationService == nil {
		return coaValidationProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-combination validation service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return coaValidationProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The validating actor is not authorized.", correlationID), nil
	}
	if request == nil || request.CommandId == (generated.UUID{}) {
		return coaValidationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-combination validation command is invalid.", correlationID), nil
	}
	if !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.BusinessDate.Set {
		return coaValidationProblem(http.StatusBadRequest, "INVALID_REQUEST", "Accounting scope and business date are required for validation.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return coaValidationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-combination validation data is invalid.", correlationID), nil
	}
	var payload coaSegmentCombinationValidationData
	if err := json.Unmarshal(data, &payload); err != nil {
		return coaValidationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-combination validation data is invalid.", correlationID), nil
	}
	command := coa.SegmentCombinationValidationCommand{
		ScopeID:        uuid.UUID(request.AccountingScopeId.Value),
		BusinessDate:   request.BusinessDate.Value,
		IdempotencyKey: params.IdempotencyKey,
		CorrelationID:  correlationID.String(),
		CausationID:    uuid.UUID(request.CommandId).String(),
		SegmentValues:  make([]coa.SegmentCombinationValueReference, 0, len(payload.SegmentValues)),
	}
	for _, item := range payload.SegmentValues {
		definitionID, parseErr := uuid.Parse(strings.TrimSpace(item.SegmentDefinitionID))
		if parseErr != nil {
			return coaValidationProblem(http.StatusBadRequest, "INVALID_REQUEST", "A segment-definition identifier is invalid.", correlationID), nil
		}
		valueID, parseErr := uuid.Parse(strings.TrimSpace(item.SegmentValueID))
		if parseErr != nil {
			return coaValidationProblem(http.StatusBadRequest, "INVALID_REQUEST", "A segment-value identifier is invalid.", correlationID), nil
		}
		command.SegmentValues = append(command.SegmentValues, coa.SegmentCombinationValueReference{SegmentDefinitionID: definitionID, SegmentValueID: valueID})
	}
	result, err := handler.SegmentCombinationValidationService.Execute(ctx, coa.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapCoaSegmentCombinationValidationError(err, correlationID), nil
	}
	return establishedCoaSegmentCombinationValidationResult(result, correlationID), nil
}

func (handler IdentityHandler) CoaMaintainSegmentDefinitions(ctx context.Context, request *generated.CommandRequest, params generated.CoaMaintainSegmentDefinitionsParams) (generated.CoaMaintainSegmentDefinitionsRes, error) {
	correlationID := correlationFromCoaParams(params)
	if handler.SegmentDefinitionService == nil {
		return coaProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-definition service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return coaProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || request.CommandId == (generated.UUID{}) {
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command is invalid.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command data is invalid.", correlationID), nil
	}
	var payload coaSegmentDefinitionCommandData
	if err := json.Unmarshal(data, &payload); err != nil {
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command data is invalid.", correlationID), nil
	}
	command, err := coaSegmentDefinitionCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command data is invalid.", correlationID), nil
	}
	result, err := handler.SegmentDefinitionService.Execute(ctx, coa.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapCoaSegmentDefinitionError(err, correlationID), nil
	}
	return establishedCoaSegmentDefinitionResult(result, correlationID), nil
}

func (handler IdentityHandler) CoaMaintainSegmentValues(ctx context.Context, request *generated.CommandRequest, params generated.CoaMaintainSegmentValuesParams) (generated.CoaMaintainSegmentValuesRes, error) {
	correlationID := correlationFromCoaValueParams(params)
	if handler.SegmentValueService == nil {
		return coaValueProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-value service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return coaValueProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || request.CommandId == (generated.UUID{}) {
		return coaValueProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-value command is invalid.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return coaValueProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-value command data is invalid.", correlationID), nil
	}
	var payload coaSegmentValueCommandData
	if err := json.Unmarshal(data, &payload); err != nil {
		return coaValueProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-value command data is invalid.", correlationID), nil
	}
	command, err := coaSegmentValueCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		return coaValueProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-value command data is invalid.", correlationID), nil
	}
	result, err := handler.SegmentValueService.Execute(ctx, coa.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapCoaSegmentValueError(err, correlationID), nil
	}
	return establishedCoaSegmentValueResult(result, correlationID), nil
}

func (handler IdentityHandler) CoaRequestSegmentChanges(ctx context.Context, request *generated.CommandRequest, params generated.CoaRequestSegmentChangesParams) (generated.CoaRequestSegmentChangesRes, error) {
	correlationID := correlationFromCoaChangeRequestParams(params)
	if handler.SegmentChangeRequestService == nil {
		return coaChangeRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-change request service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return coaChangeRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	if request == nil || request.CommandId == (generated.UUID{}) || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil {
		return coaChangeRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-change request command is invalid and requires an accounting scope.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return coaChangeRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-change request data is invalid.", correlationID), nil
	}
	var payload coaSegmentChangeRequestCommandData
	if err := json.Unmarshal(data, &payload); err != nil {
		return coaChangeRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-change request data is invalid.", correlationID), nil
	}
	command, err := coaSegmentChangeRequestCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		return coaChangeRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-change request data is invalid.", correlationID), nil
	}
	result, err := handler.SegmentChangeRequestService.Execute(ctx, coa.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapCoaSegmentChangeRequestError(err, correlationID), nil
	}
	return establishedCoaSegmentChangeRequestResult(result, correlationID), nil
}

func (handler IdentityHandler) CoaApplySegmentChangeApprovalDecision(ctx context.Context, request *generated.CommandRequest, params generated.CoaApplySegmentChangeApprovalDecisionParams) (generated.CoaApplySegmentChangeApprovalDecisionRes, error) {
	correlationID := correlationFromCoaApprovalParams(params)
	if handler.SegmentChangeApprovalDecisionService == nil {
		return coaApprovalDecisionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-change approval application service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return coaApprovalDecisionProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The applying actor is not authorized.", correlationID), nil
	}
	if request == nil || request.CommandId == (generated.UUID{}) || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil {
		return coaApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision command is invalid and requires an accounting scope.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return coaApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision data is invalid.", correlationID), nil
	}
	var payload coaSegmentChangeApprovalDecisionCommandData
	if err := json.Unmarshal(data, &payload); err != nil {
		return coaApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision data is invalid.", correlationID), nil
	}
	command, err := coaSegmentChangeApprovalDecisionCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		return coaApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision data is invalid.", correlationID), nil
	}
	result, err := handler.SegmentChangeApprovalDecisionService.Execute(ctx, coa.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapCoaSegmentChangeApprovalDecisionError(err, correlationID), nil
	}
	return establishedCoaSegmentChangeApprovalDecisionResult(result, correlationID), nil
}

func coaSegmentDefinitionCommandFromTransport(request *generated.CommandRequest, params generated.CoaMaintainSegmentDefinitionsParams, payload coaSegmentDefinitionCommandData, correlationID uuid.UUID) (coa.SegmentDefinitionCommand, error) {
	command := coa.SegmentDefinitionCommand{
		Action:         payload.Action,
		SegmentType:    payload.SegmentType,
		Code:           payload.Code,
		Name:           payload.Name,
		Status:         coa.SegmentStatus(payload.Status),
		IdempotencyKey: params.IdempotencyKey,
		CorrelationID:  correlationID.String(),
		CausationID:    uuid.UUID(request.CommandId).String(),
	}
	if request.AccountingScopeId.Set {
		command.ScopeID = uuid.UUID(request.AccountingScopeId.Value)
	}
	if payload.SegmentDefinitionID != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(payload.SegmentDefinitionID))
		if err != nil {
			return command, err
		}
		command.SegmentDefinitionID = parsed
	}
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return command, err
		}
		command.ExpectedVersion = &version
	}
	if params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return command, err
		}
		if command.ExpectedVersion != nil && command.ExpectedVersion.Value() != headerVersion.Value() {
			return command, errors.New("If-Match and expectedVersion do not agree")
		}
		command.ExpectedVersion = &headerVersion
	}
	from, err := parseCoaDate(payload.EffectiveDateFrom)
	if err != nil {
		return command, err
	}
	command.EffectiveDateFrom = from
	if payload.EffectiveDateTo != nil && strings.TrimSpace(*payload.EffectiveDateTo) != "" {
		to, err := parseCoaDate(*payload.EffectiveDateTo)
		if err != nil {
			return command, err
		}
		command.EffectiveDateTo = &to
	}
	return command, nil
}

func coaSegmentValueCommandFromTransport(request *generated.CommandRequest, params generated.CoaMaintainSegmentValuesParams, payload coaSegmentValueCommandData, correlationID uuid.UUID) (coa.SegmentValueCommand, error) {
	command := coa.SegmentValueCommand{
		Action:         payload.Action,
		Value:          payload.Value,
		Description:    payload.Description,
		Status:         coa.SegmentStatus(payload.Status),
		IdempotencyKey: params.IdempotencyKey,
		CorrelationID:  correlationID.String(),
		CausationID:    uuid.UUID(request.CommandId).String(),
	}
	if request.AccountingScopeId.Set {
		command.ScopeID = uuid.UUID(request.AccountingScopeId.Value)
	}
	if payload.SegmentDefinitionID != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(payload.SegmentDefinitionID))
		if err != nil {
			return command, err
		}
		command.SegmentDefinitionID = parsed
	}
	if payload.SegmentValueID != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(payload.SegmentValueID))
		if err != nil {
			return command, err
		}
		command.SegmentValueID = parsed
	}
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return command, err
		}
		command.ExpectedVersion = &version
	}
	if params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return command, err
		}
		if command.ExpectedVersion != nil && command.ExpectedVersion.Value() != headerVersion.Value() {
			return command, errors.New("If-Match and expectedVersion do not agree")
		}
		command.ExpectedVersion = &headerVersion
	}
	from, err := parseCoaDate(payload.EffectiveDateFrom)
	if err != nil {
		return command, err
	}
	command.EffectiveDateFrom = from
	if payload.EffectiveDateTo != nil && strings.TrimSpace(*payload.EffectiveDateTo) != "" {
		to, err := parseCoaDate(*payload.EffectiveDateTo)
		if err != nil {
			return command, err
		}
		command.EffectiveDateTo = &to
	}
	return command, nil
}

func coaSegmentChangeRequestCommandFromTransport(request *generated.CommandRequest, params generated.CoaRequestSegmentChangesParams, payload coaSegmentChangeRequestCommandData, correlationID uuid.UUID) (coa.SegmentChangeRequestCommand, error) {
	command := coa.SegmentChangeRequestCommand{
		ChangeType:             payload.ChangeType,
		ScopeID:                uuid.UUID(request.AccountingScopeId.Value),
		RequestedEffectiveDate: time.Time{},
		IdempotencyKey:         params.IdempotencyKey,
		CorrelationID:          correlationID.String(),
		CausationID:            uuid.UUID(request.CommandId).String(),
	}
	var err error
	command.SubjectID, err = uuid.Parse(strings.TrimSpace(payload.SubjectID))
	if err != nil {
		return command, err
	}
	command.SubjectVersion, err = aggregateversion.FromInt64(payload.SubjectVersion)
	if err != nil {
		return command, err
	}
	command.RequestedEffectiveDate, err = parseCoaDate(payload.RequestedEffectiveDate)
	if err != nil {
		return command, err
	}
	command.ApprovalRequestID, err = uuid.Parse(strings.TrimSpace(payload.ApprovalRequestID))
	if err != nil {
		return command, err
	}
	var proposal coaSegmentChangeProposalData
	if len(payload.ProposedChange) == 0 || string(payload.ProposedChange) == "null" {
		return command, errors.New("proposedChange is required")
	}
	if err := json.Unmarshal(payload.ProposedChange, &proposal); err != nil {
		return command, err
	}
	command.ProposedChange = coa.SegmentChangeProposal{
		Action: proposal.Action, SegmentType: proposal.SegmentType, Code: proposal.Code, Name: proposal.Name,
		Value: proposal.Value, Description: proposal.Description, Status: coa.SegmentStatus(proposal.Status),
	}
	if proposal.SegmentDefinitionID != "" {
		command.ProposedChange.SegmentDefinitionID, err = uuid.Parse(strings.TrimSpace(proposal.SegmentDefinitionID))
		if err != nil {
			return command, err
		}
	}
	if proposal.SegmentValueID != "" {
		command.ProposedChange.SegmentValueID, err = uuid.Parse(strings.TrimSpace(proposal.SegmentValueID))
		if err != nil {
			return command, err
		}
	}
	command.ProposedChange.EffectiveDateFrom, err = parseCoaDate(proposal.EffectiveDateFrom)
	if err != nil {
		return command, err
	}
	if proposal.EffectiveDateTo != nil && strings.TrimSpace(*proposal.EffectiveDateTo) != "" {
		parsed, parseErr := parseCoaDate(*proposal.EffectiveDateTo)
		if parseErr != nil {
			return command, parseErr
		}
		command.ProposedChange.EffectiveDateTo = &parsed
	}
	return command, nil
}

func coaSegmentChangeApprovalDecisionCommandFromTransport(request *generated.CommandRequest, params generated.CoaApplySegmentChangeApprovalDecisionParams, payload coaSegmentChangeApprovalDecisionCommandData, correlationID uuid.UUID) (coa.SegmentChangeApprovalDecisionCommand, error) {
	command := coa.SegmentChangeApprovalDecisionCommand{
		ScopeID: uuid.UUID(request.AccountingScopeId.Value), Outcome: payload.Outcome,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
		Approval: coa.SegmentChangeApprovalDecisionReference{PolicyVersion: payload.PolicyVersion, DecisionVersion: payload.DecisionVersion, CandidateFingerprint: payload.CandidateFingerprint},
	}
	var err error
	command.SegmentChangeRequestID, err = uuid.Parse(strings.TrimSpace(payload.SegmentChangeRequestID))
	if err != nil {
		return command, err
	}
	command.Approval.ApprovalRequestID, err = uuid.Parse(strings.TrimSpace(payload.ApprovalRequestID))
	if err != nil {
		return command, err
	}
	command.Approval.DecisionID, err = uuid.Parse(strings.TrimSpace(payload.DecisionID))
	if err != nil {
		return command, err
	}
	command.Approval.SubjectVersion, err = aggregateversion.FromInt64(payload.SubjectVersion)
	if err != nil {
		return command, err
	}
	command.Approval.ApproverUserID, err = uuid.Parse(strings.TrimSpace(payload.ApproverUserID))
	if err != nil {
		return command, err
	}
	command.Approval.DecidedAt, err = time.Parse(time.RFC3339, strings.TrimSpace(payload.DecidedAt))
	if err != nil {
		return command, err
	}
	return command, nil
}

func parseCoaDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func correlationFromCoaParams(params generated.CoaMaintainSegmentDefinitionsParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func correlationFromCoaValueParams(params generated.CoaMaintainSegmentValuesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func correlationFromCoaValidationParams(params generated.CoaValidateSegmentCombinationsParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func correlationFromCoaChangeRequestParams(params generated.CoaRequestSegmentChangesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func correlationFromCoaApprovalParams(params generated.CoaApplySegmentChangeApprovalDecisionParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedCoaSegmentDefinitionResult(result coa.SegmentDefinitionCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["segmentDefinition"] = mustRaw(result.SegmentDefinition)
	data["status"] = mustRaw(result.SegmentDefinition.Status)
	data["aggregateVersion"] = mustRaw(result.SegmentDefinition.Version.Value())
	data["effectiveDateFrom"] = mustRaw(result.SegmentDefinition.EffectiveDateFrom)
	data["effectiveDateTo"] = mustRaw(result.SegmentDefinition.EffectiveDateTo)
	data["approvalStatus"] = mustRaw(result.SegmentDefinition.ApprovalStatus)
	data["validationOutcome"] = mustRaw(result.ValidationOutcome)
	data["nextAction"] = mustRaw(result.SegmentDefinition.NextAction)
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["policyReference"] = mustRaw(result.PolicyReference)
	return &generated.EstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(result.SegmentDefinition.ID),
		AggregateVersion: int(result.SegmentDefinition.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: "/api/v1/coa-segments/configuration/maintain-segment-definitions"},
		Data:             data,
	}
}

func establishedCoaSegmentValueResult(result coa.SegmentValueCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["segmentValue"] = mustRaw(result.SegmentValue)
	data["status"] = mustRaw(result.SegmentValue.Status)
	data["aggregateVersion"] = mustRaw(result.SegmentValue.Version.Value())
	data["effectiveDateFrom"] = mustRaw(result.SegmentValue.EffectiveDateFrom)
	data["effectiveDateTo"] = mustRaw(result.SegmentValue.EffectiveDateTo)
	data["approvalStatus"] = mustRaw(result.SegmentValue.ApprovalStatus)
	data["validationOutcome"] = mustRaw(result.ValidationOutcome)
	data["nextAction"] = mustRaw(result.SegmentValue.NextAction)
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["policyReference"] = mustRaw(result.PolicyReference)
	return &generated.EstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(result.SegmentValue.SegmentDefinitionID),
		AggregateVersion: int(result.SegmentValue.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: "/api/v1/coa-segments/configuration/maintain-segment-values"},
		Data:             data,
	}
}

func establishedCoaSegmentCombinationValidationResult(result coa.SegmentCombinationValidationResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{
		"validationStatus":    mustRaw(result.ValidationStatus),
		"effectiveDateResult": mustRaw(result.EffectiveDateResult),
		"sourceVersions":      mustRaw(result.SourceVersions),
		"invalidValues":       mustRaw(result.InvalidValues),
		"restrictions":        mustRaw(result.Restrictions),
		"rejectionReasons":    mustRaw(result.RejectionReasons),
		"nextAction":          mustRaw(result.NextAction),
		"replayed":            mustRaw(result.Replayed),
	}
	return &generated.EstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(uuid.Nil),
		AggregateVersion: 0,
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: "/api/v1/coa-segments/actions/validate-segment-combinations"},
		Data:             data,
	}
}

func establishedCoaSegmentChangeRequestResult(result coa.SegmentChangeRequestCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["segmentChangeRequest"] = mustRaw(result.SegmentChangeRequest)
	data["changeType"] = mustRaw(result.SegmentChangeRequest.ChangeType)
	data["subjectId"] = mustRaw(result.SegmentChangeRequest.SubjectID.String())
	data["subjectVersion"] = mustRaw(result.SegmentChangeRequest.SubjectVersion)
	data["requestedEffectiveDate"] = mustRaw(result.SegmentChangeRequest.RequestedEffectiveDate)
	data["approvalRequestId"] = mustRaw(result.SegmentChangeRequest.ApprovalRequestID.String())
	data["approvalStatus"] = mustRaw(result.SegmentChangeRequest.ApprovalStatus)
	data["applicationStatus"] = mustRaw(result.SegmentChangeRequest.ApplicationStatus)
	data["validationOutcome"] = mustRaw(result.ValidationOutcome)
	data["nextAction"] = mustRaw(result.SegmentChangeRequest.NextAction)
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["policyReference"] = mustRaw(result.PolicyReference)
	data["replayed"] = mustRaw(result.Replayed)
	return &generated.EstablishedResult{
		Status: "established", AggregateId: generated.UUID(result.SegmentChangeRequest.ID), AggregateVersion: int(result.SegmentChangeRequest.Version.Value()),
		CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/coa-segments/actions/request-segment-changes"}, Data: data,
	}
}

func establishedCoaSegmentChangeApprovalDecisionResult(result coa.SegmentChangeApprovalDecisionResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["segmentChangeRequest"] = mustRaw(result.SegmentChangeRequest)
	data["outcome"] = mustRaw(result.Outcome)
	data["applicationStatus"] = mustRaw(result.ApplicationStatus)
	data["resultingSubjectVersion"] = mustRaw(result.ResultingSubjectVersion)
	data["appliedSubjectVersion"] = mustRaw(result.AppliedSubjectVersion)
	data["effectiveDateResult"] = mustRaw(result.EffectiveDateResult)
	data["conflictCode"] = mustRaw(result.ConflictCode)
	data["rejectionReason"] = mustRaw(result.RejectionReason)
	data["approval"] = mustRaw(result.Approval)
	data["approvalRequestId"] = mustRaw(result.Approval.ApprovalRequestID.String())
	data["approvalDecisionId"] = mustRaw(result.Approval.DecisionID.String())
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["policyReference"] = mustRaw(result.PolicyReference)
	data["validationOutcome"] = mustRaw(result.ValidationOutcome)
	data["replayed"] = mustRaw(result.Replayed)
	return &generated.EstablishedResult{
		Status: "established", AggregateId: generated.UUID(result.SegmentChangeRequest.ID), AggregateVersion: int(result.SegmentChangeRequest.Version.Value()),
		CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/coa-segments/actions/apply-segment-change-approval-decision"}, Data: data,
	}
}

func mapCoaSegmentCombinationValidationError(err error, correlationID uuid.UUID) generated.CoaValidateSegmentCombinationsRes {
	switch {
	case errors.Is(err, coa.ErrSegmentCombinationValidationAuthorizationUnavailable):
		return coaValidationProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, coa.ErrSegmentCombinationValidationAuthorizationStale):
		return coaValidationProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, coa.ErrSegmentCombinationValidationAuthorizationDenied):
		return coaValidationProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested segment-combination scope is outside the validating actor scope.", correlationID)
	case errors.Is(err, coa.ErrSegmentCombinationValidationIdempotencyConflict):
		return coaValidationProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different validation data.", correlationID)
	case errors.Is(err, coa.ErrSegmentCombinationValidationInProgress):
		return coaValidationProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The validation is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, coa.ErrSegmentCombinationValidationPreviouslyFailed):
		return coaValidationProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The previous validation attempt failed. Use a new idempotency key after correcting the request or dependency.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentCombinationValidationCommand):
		return coaValidationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-combination validation command is invalid.", correlationID)
	default:
		return coaValidationProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-combination validation could not be completed.", correlationID)
	}
}

func coaValidationProblem(status int, code, detail string, correlationID uuid.UUID) generated.CoaValidateSegmentCombinationsRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.CoaValidateSegmentCombinationsBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.CoaValidateSegmentCombinationsForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.CoaValidateSegmentCombinationsConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.CoaValidateSegmentCombinationsUnprocessableEntity(problem)
		return &value
	default:
		value := generated.CoaValidateSegmentCombinationsServiceUnavailable(problem)
		return &value
	}
}

func mapCoaSegmentDefinitionError(err error, correlationID uuid.UUID) generated.CoaMaintainSegmentDefinitionsRes {
	switch {
	case errors.Is(err, coa.ErrSegmentDefinitionAuthorizationUnavailable):
		return coaProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionAuthorizationStale):
		return coaProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionAuthorizationDenied):
		return coaProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested segment-definition scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionVersionConflict):
		return coaProblem(http.StatusConflict, "VERSION_CONFLICT", "The segment definition changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionIdempotencyConflict):
		return coaProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionCommandInProgress):
		return coaProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionNotFound):
		return coaProblem(http.StatusConflict, "SEGMENT_DEFINITION_NOT_FOUND", "The requested segment definition does not exist.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentDefinitionCommand):
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command is invalid.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentDefinition), errors.Is(err, coa.ErrSegmentDefinitionDuplicate), errors.Is(err, coa.ErrSegmentDefinitionDurableCommandFailed):
		return coaProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The segment-definition command violates a COA rule.", correlationID)
	default:
		return coaProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-definition operation could not be completed.", correlationID)
	}
}

func coaProblem(status int, code, detail string, correlationID uuid.UUID) generated.CoaMaintainSegmentDefinitionsRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.CoaMaintainSegmentDefinitionsBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.CoaMaintainSegmentDefinitionsForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.CoaMaintainSegmentDefinitionsConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.CoaMaintainSegmentDefinitionsUnprocessableEntity(problem)
		return &value
	default:
		value := generated.CoaMaintainSegmentDefinitionsServiceUnavailable(problem)
		return &value
	}
}

func mapCoaSegmentValueError(err error, correlationID uuid.UUID) generated.CoaMaintainSegmentValuesRes {
	switch {
	case errors.Is(err, coa.ErrSegmentValueAuthorizationUnavailable):
		return coaValueProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, coa.ErrSegmentValueAuthorizationStale):
		return coaValueProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, coa.ErrSegmentValueAuthorizationDenied):
		return coaValueProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested segment-value scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, coa.ErrSegmentValueVersionConflict):
		return coaValueProblem(http.StatusConflict, "VERSION_CONFLICT", "The segment definition changed after it was loaded. Refresh and retry with the current parent version.", correlationID)
	case errors.Is(err, coa.ErrSegmentValueIdempotencyConflict):
		return coaValueProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, coa.ErrSegmentValueCommandInProgress):
		return coaValueProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionNotFound):
		return coaValueProblem(http.StatusConflict, "SEGMENT_DEFINITION_NOT_FOUND", "The requested segment definition does not exist.", correlationID)
	case errors.Is(err, coa.ErrSegmentValueNotFound):
		return coaValueProblem(http.StatusConflict, "SEGMENT_VALUE_NOT_FOUND", "The requested segment value does not exist.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentValueCommand):
		return coaValueProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-value command is invalid.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentValue), errors.Is(err, coa.ErrSegmentValueDuplicate), errors.Is(err, coa.ErrSegmentValueDurableCommandFailed):
		return coaValueProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The segment-value command violates a COA rule.", correlationID)
	default:
		return coaValueProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-value operation could not be completed.", correlationID)
	}
}

func coaValueProblem(status int, code, detail string, correlationID uuid.UUID) generated.CoaMaintainSegmentValuesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.CoaMaintainSegmentValuesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.CoaMaintainSegmentValuesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.CoaMaintainSegmentValuesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.CoaMaintainSegmentValuesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.CoaMaintainSegmentValuesServiceUnavailable(problem)
		return &value
	}
}

func mapCoaSegmentChangeRequestError(err error, correlationID uuid.UUID) generated.CoaRequestSegmentChangesRes {
	switch {
	case errors.Is(err, coa.ErrSegmentChangeRequestAuthorizationUnavailable):
		return coaChangeRequestProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeRequestAuthorizationStale):
		return coaChangeRequestProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeRequestAuthorizationDenied):
		return coaChangeRequestProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested segment-change scope is outside the requesting actor scope.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeRequestVersionConflict):
		return coaChangeRequestProblem(http.StatusConflict, "VERSION_CONFLICT", "The subject changed after it was loaded. Refresh and request the current version.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeRequestIdempotencyConflict):
		return coaChangeRequestProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different request data.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeRequestCommandInProgress):
		return coaChangeRequestProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The request is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeRequestSubjectNotFound):
		return coaChangeRequestProblem(http.StatusConflict, "SUBJECT_NOT_FOUND", "The requested COA subject does not exist.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeRequestUnsupportedSubject):
		return coaChangeRequestProblem(http.StatusUnprocessableEntity, "UNSUPPORTED_SUBJECT", "This COA request slice supports only existing segment definitions and values.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentChangeRequestCommand):
		return coaChangeRequestProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-change request command is invalid.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentChangeRequest), errors.Is(err, coa.ErrSegmentChangeRequestDurableCommandFailed), errors.Is(err, coa.ErrSegmentChangeRequestDuplicate):
		return coaChangeRequestProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The segment-change request violates a COA rule.", correlationID)
	default:
		return coaChangeRequestProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-change request could not be completed.", correlationID)
	}
}

func coaChangeRequestProblem(status int, code, detail string, correlationID uuid.UUID) generated.CoaRequestSegmentChangesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.CoaRequestSegmentChangesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.CoaRequestSegmentChangesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.CoaRequestSegmentChangesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.CoaRequestSegmentChangesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.CoaRequestSegmentChangesServiceUnavailable(problem)
		return &value
	}
}

func mapCoaSegmentChangeApprovalDecisionError(err error, correlationID uuid.UUID) generated.CoaApplySegmentChangeApprovalDecisionRes {
	switch {
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionAuthorizationUnavailable):
		return coaApprovalDecisionProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionAuthorizationStale):
		return coaApprovalDecisionProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionAuthorizationDenied):
		return coaApprovalDecisionProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The applying actor is outside the approved COA scope.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionIdempotencyConflict):
		return coaApprovalDecisionProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different approval-decision data.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionCommandInProgress):
		return coaApprovalDecisionProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The approval decision is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionNotFound):
		return coaApprovalDecisionProblem(http.StatusConflict, "REQUEST_NOT_FOUND", "The segment-change request does not exist.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionDuplicate):
		return coaApprovalDecisionProblem(http.StatusConflict, "DUPLICATE_DECISION", "The segment-change request already has a different terminal approval decision.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionReferenceInvalid):
		return coaApprovalDecisionProblem(http.StatusUnprocessableEntity, "INVALID_APPROVAL_REFERENCE", "The Workflow approval decision does not match the stored segment-change proposal.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionSubjectConflict):
		return coaApprovalDecisionProblem(http.StatusConflict, "SUBJECT_CONFLICT", "The current COA subject changed while the approval decision was being applied.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionDuplicate), errors.Is(err, coa.ErrSegmentValueDuplicate):
		return coaApprovalDecisionProblem(http.StatusConflict, "POLICY_CONFLICT", "The approved change conflicts with another effective COA definition or value.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeRequestUnsupportedSubject):
		return coaApprovalDecisionProblem(http.StatusUnprocessableEntity, "UNSUPPORTED_SUBJECT", "This COA application slice supports only existing segment definitions and values.", correlationID)
	case errors.Is(err, coa.ErrSegmentChangeApprovalDecisionInvalid):
		return coaApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision command is invalid.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentChangeRequest), errors.Is(err, coa.ErrSegmentChangeApprovalDecisionDurableCommandFailed):
		return coaApprovalDecisionProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The approval decision violates a COA rule.", correlationID)
	default:
		return coaApprovalDecisionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-change approval decision could not be completed.", correlationID)
	}
}

func coaApprovalDecisionProblem(status int, code, detail string, correlationID uuid.UUID) generated.CoaApplySegmentChangeApprovalDecisionRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.CoaApplySegmentChangeApprovalDecisionBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.CoaApplySegmentChangeApprovalDecisionForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.CoaApplySegmentChangeApprovalDecisionConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.CoaApplySegmentChangeApprovalDecisionUnprocessableEntity(problem)
		return &value
	default:
		value := generated.CoaApplySegmentChangeApprovalDecisionServiceUnavailable(problem)
		return &value
	}
}
