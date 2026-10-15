package workflow

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ApprovalRequestAssignmentAssigned = "assigned"
	ApprovalRequestAssignmentAll      = "all"
	ApprovalRequestDueDue             = "due"
	ApprovalRequestDueAll             = "all"
	ApprovalRequestStateAll           = "all"
	ApprovalRequestStateBlocked       = "blocked"
	ApprovalRequestStateDelegated     = "delegated"
	ApprovalRequestStateEscalated     = "escalated"
	ApprovalRequestStateCompleted     = "completed"
)

type ApprovalRequestListFilter struct {
	ScopeIDs   []string
	Assignment string
	Due        string
	State      string
	PageSize   int
	PageAfter  string
}

type ApprovalRequestPage struct {
	Requests []ApprovalRequest
	Size     int
	After    string
	HasMore  bool
}

type ApprovalRequestQueryService struct {
	repository       ApprovalRequestReader
	authorizer       ApprovalRequestReadAuthorizer
	delegationReader DelegationReader
	clock            func() time.Time
}

func NewApprovalRequestQueryService(repository ApprovalRequestReader, authorizer ApprovalRequestReadAuthorizer, clock func() time.Time) (*ApprovalRequestQueryService, error) {
	if repository == nil || authorizer == nil || clock == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	return &ApprovalRequestQueryService{repository: repository, authorizer: authorizer, clock: clock}, nil
}

func (service *ApprovalRequestQueryService) BindDelegationReader(reader DelegationReader) error {
	if service == nil || reader == nil {
		return ErrInvalidApprovalPolicyService
	}
	service.delegationReader = reader
	return nil
}

func (service *ApprovalRequestQueryService) Get(ctx context.Context, actor Actor, id uuid.UUID) (ApprovalRequest, error) {
	if service == nil || service.repository == nil || service.authorizer == nil {
		return ApprovalRequest{}, ErrInvalidApprovalPolicyService
	}
	if err := actor.Validate(); err != nil || id == uuid.Nil {
		return ApprovalRequest{}, ErrInvalidApprovalPolicyCommand
	}
	request, err := service.repository.Get(ctx, id)
	if err != nil {
		return ApprovalRequest{}, err
	}
	if err := service.authorizeRead(ctx, actor, request); err != nil {
		return ApprovalRequest{}, err
	}
	if err := service.attachActiveDelegation(ctx, &request); err != nil {
		return ApprovalRequest{}, err
	}
	return request, nil
}

func (service *ApprovalRequestQueryService) List(ctx context.Context, actor Actor, filter ApprovalRequestListFilter) (ApprovalRequestPage, error) {
	if service == nil || service.repository == nil || service.authorizer == nil {
		return ApprovalRequestPage{}, ErrInvalidApprovalPolicyService
	}
	if err := actor.Validate(); err != nil {
		return ApprovalRequestPage{}, err
	}
	filter, cursor, err := normalizeApprovalRequestListFilter(filter)
	if err != nil {
		return ApprovalRequestPage{}, err
	}
	requests, err := service.repository.List(ctx)
	if err != nil {
		return ApprovalRequestPage{}, err
	}
	now := service.clock().UTC()
	visible := make([]ApprovalRequest, 0, len(requests))
	for _, request := range requests {
		if len(filter.ScopeIDs) > 0 && !request.Scope.Contains(filter.ScopeIDs) {
			continue
		}
		if err := service.authorizeRead(ctx, actor, request); err != nil {
			if errors.Is(err, ErrApprovalPolicyAuthorizationDenied) || errors.Is(err, ErrApprovalPolicyAuthorizationStale) {
				continue
			}
			return ApprovalRequestPage{}, err
		}
		if err := service.attachActiveDelegation(ctx, &request); err != nil {
			return ApprovalRequestPage{}, err
		}
		if !approvalRequestMatchesListFilter(request, filter, now) {
			continue
		}
		visible = append(visible, request)
	}
	sort.SliceStable(visible, func(left, right int) bool {
		if visible[left].CreatedAt.Equal(visible[right].CreatedAt) {
			return visible[left].ID.String() > visible[right].ID.String()
		}
		return visible[left].CreatedAt.After(visible[right].CreatedAt)
	})
	if cursor != uuid.Nil {
		start := len(visible)
		for index, request := range visible {
			if request.ID == cursor {
				start = index + 1
				break
			}
		}
		visible = visible[start:]
	}
	hasMore := len(visible) > filter.PageSize
	if hasMore {
		visible = visible[:filter.PageSize]
	}
	page := ApprovalRequestPage{Requests: visible, Size: filter.PageSize, HasMore: hasMore}
	if hasMore && len(visible) > 0 {
		page.After = visible[len(visible)-1].ID.String()
	}
	return page, nil
}

func (service *ApprovalRequestQueryService) attachActiveDelegation(ctx context.Context, request *ApprovalRequest) error {
	if service == nil || request == nil || service.delegationReader == nil {
		return nil
	}
	if request.Status != ApprovalRequestStatusPending || !approvalRequestHasPendingStep(*request) {
		request.ActiveDelegation = nil
		return nil
	}
	delegations, err := service.delegationReader.List(ctx)
	if err != nil {
		return ErrDelegationAuthorizationUnavailable
	}
	now := service.clock().UTC()
	var selected *Delegation
	for index := range delegations {
		delegation := &delegations[index]
		if delegation.PolicyID != request.PolicyID || delegation.PolicyVersion != request.PolicyVersion || !delegation.ActiveAt(now) || !delegation.Scope.Contains(request.Scope.CanonicalScopeIDs()) {
			continue
		}
		if selected == nil || delegation.CreatedAt.After(selected.CreatedAt) {
			copy := *delegation
			copy.Scope = cloneScope(delegation.Scope)
			selected = &copy
		}
	}
	request.ActiveDelegation = selected
	return nil
}

func approvalRequestHasPendingStep(request ApprovalRequest) bool {
	for _, step := range request.Steps {
		if step.Status == ApprovalRequestStatusPending {
			return true
		}
	}
	return false
}

func (service *ApprovalRequestQueryService) authorizeRead(ctx context.Context, actor Actor, request ApprovalRequest) error {
	decision, err := service.authorizer.AuthorizeApprovalRequestRead(ctx, actor, request)
	if err != nil {
		return ErrApprovalPolicyAuthorizationUnavailable
	}
	return validateAuthorization(decision, ReadApprovalRequestsPermission, request.Scope.CanonicalScopeIDs())
}

func normalizeApprovalRequestListFilter(filter ApprovalRequestListFilter) (ApprovalRequestListFilter, uuid.UUID, error) {
	filter.Assignment = strings.TrimSpace(filter.Assignment)
	filter.Due = strings.TrimSpace(filter.Due)
	filter.State = strings.TrimSpace(filter.State)
	if filter.Assignment == "" {
		filter.Assignment = ApprovalRequestAssignmentAssigned
	}
	if filter.Due == "" {
		filter.Due = ApprovalRequestDueAll
	}
	if filter.State == "" {
		filter.State = ApprovalRequestStateAll
	}
	if filter.Assignment != ApprovalRequestAssignmentAssigned && filter.Assignment != ApprovalRequestAssignmentAll {
		return ApprovalRequestListFilter{}, uuid.Nil, ErrInvalidApprovalPolicyCommand
	}
	if filter.Due != ApprovalRequestDueDue && filter.Due != ApprovalRequestDueAll {
		return ApprovalRequestListFilter{}, uuid.Nil, ErrInvalidApprovalPolicyCommand
	}
	switch filter.State {
	case ApprovalRequestStateAll, ApprovalRequestStateBlocked, ApprovalRequestStateDelegated, ApprovalRequestStateEscalated, ApprovalRequestStateCompleted:
	default:
		return ApprovalRequestListFilter{}, uuid.Nil, ErrInvalidApprovalPolicyCommand
	}
	if filter.PageSize == 0 {
		filter.PageSize = 50
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		return ApprovalRequestListFilter{}, uuid.Nil, ErrInvalidApprovalPolicyCommand
	}
	if strings.TrimSpace(filter.PageAfter) == "" {
		return filter, uuid.Nil, nil
	}
	cursor, err := uuid.Parse(strings.TrimSpace(filter.PageAfter))
	if err != nil || cursor == uuid.Nil {
		return ApprovalRequestListFilter{}, uuid.Nil, ErrApprovalPolicyInvalidPage
	}
	return filter, cursor, nil
}

func approvalRequestMatchesListFilter(request ApprovalRequest, filter ApprovalRequestListFilter, now time.Time) bool {
	if filter.Assignment == ApprovalRequestAssignmentAssigned && (request.Status != ApprovalRequestStatusPending || (len(request.ResponsibleRoles) == 0 && request.ActiveDelegation == nil)) {
		return false
	}
	if filter.Due == ApprovalRequestDueDue && (request.DueAt == nil || request.DueAt.After(now)) {
		return false
	}
	switch filter.State {
	case ApprovalRequestStateBlocked:
		return request.Status == ApprovalRequestStatusPending && approvalRequestHasBlockedEscalation(request)
	case ApprovalRequestStateDelegated:
		return request.ActiveDelegation != nil || approvalRequestHasStepStatus(request, "delegated")
	case ApprovalRequestStateEscalated:
		return approvalRequestHasStepStatus(request, "escalated")
	case ApprovalRequestStateCompleted:
		return request.Status == ApprovalRequestStatusApproved || request.Status == ApprovalRequestStatusRejected
	default:
		return true
	}
}

func approvalRequestHasBlockedEscalation(request ApprovalRequest) bool {
	for _, escalation := range request.Escalations {
		if escalation.Reason == ApprovalEscalationReasonBlocked && strings.TrimSpace(escalation.Blocker) != "" {
			return true
		}
	}
	return false
}

func approvalRequestHasStepStatus(request ApprovalRequest, expected string) bool {
	for _, step := range request.Steps {
		if strings.EqualFold(strings.TrimSpace(step.Status), expected) {
			return true
		}
	}
	return false
}
