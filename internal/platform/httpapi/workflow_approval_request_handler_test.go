package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/workflow"
)

func TestApprovalRequestHandlerCreatesReplaysAndReadsAuthorizedEvidence(t *testing.T) {
	now := time.Now().UTC()
	policy := httpApprovalRequestPolicy(now)
	authorizer := workflow.MemoryApprovalRequestAuthorizer{
		Decision:     workflow.AuthorizationDecision{Allowed: true, Outcome: workflow.AuthorizationAllowed, Permission: workflow.CreateApprovalRequestsPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()},
		ReadDecision: &workflow.AuthorizationDecision{Allowed: true, Outcome: workflow.AuthorizationAllowed, Permission: workflow.ReadApprovalRequestsPermission, ApprovedScopeIDs: []string{"*"}, DecisionReference: uuid.New()},
	}
	approvalWorkflow, err := workflow.NewMemoryApprovalWorkflowWithAuthorizer(authorizer)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := workflow.NewEffectiveApprovalPolicyResolver(staticApprovalPolicyRepository{policies: []workflow.ApprovalPolicy{policy}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := approvalWorkflow.Service.BindPolicyResolver(resolver); err != nil {
		t.Fatal(err)
	}
	queries, err := workflow.NewApprovalRequestQueryService(approvalWorkflow.Requests, authorizer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{ApprovalRequestService: approvalWorkflow.Service, ApprovalRequestQueries: queries}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	request := newHTTPApprovalRequestCreateRequest()
	params := generated.WfaCreateApprovalRequestParams{IdempotencyKey: "http-create-approval", XCorrelationID: generated.NewOptUUID(generated.UUID(uuid.New()))}
	_, commandErr := workflowCreateApprovalRequestCommand(request, params)
	if commandErr != nil {
		t.Fatalf("command conversion failed: %v", commandErr)
	}
	response, err := handler.WfaCreateApprovalRequest(actorContext, request, params)
	if err != nil {
		t.Fatal(err)
	}
	created, ok := response.(*generated.WfaCreateApprovalRequestEstablishedResult)
	if !ok || created.Data.Request.PolicyVersion != policy.PolicyVersion || created.Data.Request.SubjectSnapshot.Value.Label != "Posting 1" || created.Data.Request.SnapshotFingerprint.Value == "" {
		t.Fatalf("created response = %#v", response)
	}

	replayedResponse, err := handler.WfaCreateApprovalRequest(actorContext, request, params)
	if err != nil {
		t.Fatal(err)
	}
	replayed, ok := replayedResponse.(*generated.WfaCreateApprovalRequestEstablishedResult)
	if !ok || !replayed.Data.Replayed {
		t.Fatalf("replayed response = %#v", replayedResponse)
	}

	changed := *request
	changed.Data.SubjectFingerprint = "sha256:changed"
	conflictResponse, err := handler.WfaCreateApprovalRequest(actorContext, &changed, params)
	if err != nil {
		t.Fatal(err)
	}
	if conflict, ok := conflictResponse.(*generated.WfaCreateApprovalRequestConflict); !ok || conflict.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("conflict response = %#v", conflictResponse)
	}

	listResponse, err := handler.WfaListApprovalRequests(actorContext, generated.WfaListApprovalRequestsParams{Assignment: generated.NewOptWfaListApprovalRequestsAssignment(generated.WfaListApprovalRequestsAssignmentAll)})
	if err != nil {
		t.Fatal(err)
	}
	list, ok := listResponse.(*generated.WfaApprovalRequestListEstablishedResult)
	if !ok || len(list.Data.Requests) != 1 || list.Data.Requests[0].ID != created.AggregateId {
		t.Fatalf("list response = %#v", listResponse)
	}

	detailResponse, err := handler.WfaGetApprovalRequest(actorContext, generated.WfaGetApprovalRequestParams{ApprovalRequestId: created.AggregateId})
	if err != nil {
		t.Fatal(err)
	}
	detail, ok := detailResponse.(*generated.WfaApprovalRequestEstablishedResult)
	if !ok || detail.Data.Request.ID != created.AggregateId || detail.Data.Request.PolicyVersion != policy.PolicyVersion {
		t.Fatalf("detail response = %#v", detailResponse)
	}
}

func TestApprovalRequestHandlerFailsClosedForAuthorizationAndPolicyConflicts(t *testing.T) {
	now := time.Now().UTC()
	policy := httpApprovalRequestPolicy(now)
	denied := workflow.MemoryApprovalRequestAuthorizer{Decision: workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: workflow.CreateApprovalRequestsPermission, DecisionReference: uuid.New()}}
	approvalWorkflow, err := workflow.NewMemoryApprovalWorkflowWithAuthorizer(denied)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := workflow.NewEffectiveApprovalPolicyResolver(staticApprovalPolicyRepository{policies: []workflow.ApprovalPolicy{policy}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := approvalWorkflow.Service.BindPolicyResolver(resolver); err != nil {
		t.Fatal(err)
	}
	queries, err := workflow.NewApprovalRequestQueryService(approvalWorkflow.Requests, denied, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{ApprovalRequestService: approvalWorkflow.Service, ApprovalRequestQueries: queries}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := handler.WfaCreateApprovalRequest(actorContext, newHTTPApprovalRequestCreateRequest(), generated.WfaCreateApprovalRequestParams{IdempotencyKey: "http-denied"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.WfaCreateApprovalRequestForbidden); !ok {
		t.Fatalf("denied response = %#v", response)
	}

	missingPolicy := *handler.ApprovalRequestService
	missingResolver, err := workflow.NewEffectiveApprovalPolicyResolver(staticApprovalPolicyRepository{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := missingPolicy.BindPolicyResolver(missingResolver); err != nil {
		t.Fatal(err)
	}
	missingResponse, err := (IdentityHandler{ApprovalRequestService: &missingPolicy}).WfaCreateApprovalRequest(actorContext, newHTTPApprovalRequestCreateRequest(), generated.WfaCreateApprovalRequestParams{IdempotencyKey: "http-missing-policy"})
	if err != nil {
		t.Fatal(err)
	}
	if conflict, ok := missingResponse.(*generated.WfaCreateApprovalRequestConflict); !ok || conflict.Code != "POLICY_NOT_APPLICABLE" {
		t.Fatalf("missing-policy response = %#v", missingResponse)
	}
}

func TestApprovalDecisionHandlerRecordsAndExposesDecisionHistory(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	policy := httpApprovalRequestPolicy(now)
	decision := workflow.AuthorizationDecision{Allowed: true, Outcome: workflow.AuthorizationAllowed, Permission: workflow.DecideApprovalRequestsPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()}
	authorizer := workflow.MemoryApprovalRequestAuthorizer{
		Decision:       workflow.AuthorizationDecision{Allowed: true, Outcome: workflow.AuthorizationAllowed, Permission: workflow.CreateApprovalRequestsPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()},
		DecideDecision: &decision,
	}
	approvalWorkflow, err := workflow.NewMemoryApprovalWorkflowWithAuthorizer(authorizer)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := workflow.NewEffectiveApprovalPolicyResolver(staticApprovalPolicyRepository{policies: []workflow.ApprovalPolicy{policy}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := approvalWorkflow.Service.BindPolicyResolver(resolver); err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{ApprovalRequestService: approvalWorkflow.Service}
	requesterContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "requester"}})
	if err != nil {
		t.Fatal(err)
	}
	createParams := generated.WfaCreateApprovalRequestParams{IdempotencyKey: "handler-create-decision"}
	createdResponse, err := handler.WfaCreateApprovalRequest(requesterContext, newHTTPApprovalRequestCreateRequest(), createParams)
	if err != nil {
		t.Fatal(err)
	}
	created, ok := createdResponse.(*generated.WfaCreateApprovalRequestEstablishedResult)
	if !ok {
		t.Fatalf("create response = %#v", createdResponse)
	}
	stepID := created.Data.Request.CurrentStepId.Value
	approverContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "approver"}})
	if err != nil {
		t.Fatal(err)
	}
	decisionRequest := &generated.CommandRequest{
		CommandId:       generated.UUID(uuid.New()),
		ExpectedVersion: generated.NewOptInt(created.AggregateVersion),
		Data: generated.CommandRequestData{
			"requestId":            mustRaw(uuid.UUID(created.AggregateId).String()),
			"decisionId":           mustRaw(uuid.NewString()),
			"stepId":               mustRaw(uuid.UUID(stepID).String()),
			"outcome":              mustRaw("approved"),
			"reason":               mustRaw("Independent review complete"),
			"policyVersion":        mustRaw(created.Data.Request.PolicyVersion),
			"subjectVersion":       mustRaw(created.Data.Request.SubjectVersion),
			"candidateFingerprint": mustRaw(created.Data.Request.SubjectFingerprint),
		},
	}
	params := generated.WfaDecideApprovalRequestParams{IdempotencyKey: "handler-decision", XCorrelationID: generated.NewOptUUID(generated.UUID(uuid.New()))}
	response, err := handler.WfaDecideApprovalRequest(approverContext, decisionRequest, params)
	if err != nil {
		t.Fatal(err)
	}
	established, ok := response.(*generated.WfaApprovalRequestEstablishedResult)
	if !ok || established.Data.Request.Status != generated.WfaApprovalRequestProjectionStatusApproved || len(established.Data.Request.Decisions) != 1 || !established.Data.DecisionReference.Set || established.Data.ApplicationStatus != generated.WfaApprovalRequestResultDataApplicationStatusNotReported {
		t.Fatalf("decision response = %#v", response)
	}

	replayed, err := handler.WfaDecideApprovalRequest(approverContext, decisionRequest, params)
	if err != nil {
		t.Fatal(err)
	}
	if replay, ok := replayed.(*generated.WfaApprovalRequestEstablishedResult); !ok || !replay.Data.Replayed {
		t.Fatalf("replayed decision response = %#v", replayed)
	}

	unknown := *decisionRequest
	unknown.Data = generated.CommandRequestData{"requestId": mustRaw(uuid.UUID(created.AggregateId).String()), "decisionId": mustRaw(uuid.NewString()), "stepId": mustRaw(uuid.UUID(stepID).String()), "outcome": mustRaw("approved"), "policyVersion": mustRaw(created.Data.Request.PolicyVersion), "subjectVersion": mustRaw(created.Data.Request.SubjectVersion), "candidateFingerprint": mustRaw(created.Data.Request.SubjectFingerprint), "unexpected": mustRaw("reject")}
	invalid, err := handler.WfaDecideApprovalRequest(approverContext, &unknown, generated.WfaDecideApprovalRequestParams{IdempotencyKey: "handler-unknown-field"})
	if err != nil {
		t.Fatal(err)
	}
	if problem, ok := invalid.(*generated.WfaDecideApprovalRequestBadRequest); !ok || problem.Code != "INVALID_REQUEST" {
		t.Fatalf("unknown-field response = %#v", invalid)
	}
}

type staticApprovalPolicyRepository struct {
	policies []workflow.ApprovalPolicy
}

func (repository staticApprovalPolicyRepository) Get(_ context.Context, id uuid.UUID) (workflow.ApprovalPolicy, error) {
	for _, policy := range repository.policies {
		if policy.ID == id {
			return policy, nil
		}
	}
	return workflow.ApprovalPolicy{}, workflow.ErrApprovalPolicyNotFound
}

func (repository staticApprovalPolicyRepository) List(context.Context) ([]workflow.ApprovalPolicy, error) {
	return append([]workflow.ApprovalPolicy(nil), repository.policies...), nil
}

func (repository staticApprovalPolicyRepository) ListSelectable(context.Context) ([]workflow.ApprovalPolicy, error) {
	return append([]workflow.ApprovalPolicy(nil), repository.policies...), nil
}

func (staticApprovalPolicyRepository) Commit(context.Context, workflow.ApprovalPolicyMutation) error {
	return nil
}

func httpApprovalRequestPolicy(now time.Time) workflow.ApprovalPolicy {
	minimum := decimal.RequireFromString("0.00")
	maximum := decimal.RequireFromString("1000.00")
	policy, err := workflow.NewApprovalPolicy(uuid.New(), "http-policy", "HTTP approval policy", "v1", workflow.ApprovalApplicability{SourceContext: "general-ledger", SubjectType: "posting-request", Operation: "submit", Currencies: []string{"USD"}, DataClassifications: []string{"internal"}}, workflow.ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, []workflow.ApprovalRule{{ID: uuid.New(), Threshold: workflow.AmountThreshold{Minimum: &minimum, Maximum: &maximum, Currency: "USD"}, RequiredRoles: []workflow.RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}}}, []workflow.ApprovalStepDefinition{{ID: uuid.New(), Sequence: 1, Name: "Review", RequiredRoles: []workflow.RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}, MinimumApprovals: 1}}, now.Add(-time.Hour), nil, now)
	if err != nil {
		panic(err)
	}
	policy.Status = workflow.ApprovalPolicyStatusActive
	policy.Approval = &workflow.ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: policy.PolicyVersion, DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: policy.CandidateFingerprint(), ApproverUserID: uuid.New()}
	return policy
}

func newHTTPApprovalRequestCreateRequest() *generated.WfaCreateApprovalRequestCommandRequest {
	return &generated.WfaCreateApprovalRequestCommandRequest{CommandId: generated.UUID(uuid.New()), Data: generated.WfaCreateApprovalRequestCommandData{RequestId: generated.UUID(uuid.New()), SubjectId: "posting-1", SubjectVersion: 1, SubjectFingerprint: "sha256:subject-v1", SubjectSnapshot: generated.WfaApprovalSubjectSnapshot{Label: "Posting 1", Reference: generated.NewOptString("safe-reference"), Summary: generated.NewOptString("Safe projection"), Amount: generated.NewOptMoney(generated.Money("100.00")), Currency: generated.NewOptCurrencyCode(generated.CurrencyCode("USD"))}, Applicability: generated.WfaApprovalApplicability{SourceContext: "general-ledger", SubjectType: "posting-request", Operation: "submit", Currencies: []generated.CurrencyCode{"USD"}, DataClassifications: []string{"internal"}}, Scope: generated.WfaApprovalScope{ScopeIds: []string{"scope-1"}}, DataClassification: generated.NewOptString("internal"), Amount: generated.Money("100.00"), Currency: generated.CurrencyCode("USD")}}
}
