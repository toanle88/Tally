package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/coa"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/workflow"
)

func TestPostgresCoaAuditWriterPreservesAggregateIdentityPrecedence(t *testing.T) {
	var records []identity.AuditRecord
	writer := postgresCoaAuditWriter(func(_ context.Context, _ pgx.Tx, record identity.AuditRecord) (uuid.UUID, error) {
		records = append(records, record)
		return uuid.New(), nil
	})
	definitionID := uuid.New()
	valueID := uuid.New()
	requestID := uuid.New()

	if _, err := writer(context.Background(), nil, coa.AuditRecord{SegmentDefinitionID: definitionID, SegmentValueID: valueID}); err != nil {
		t.Fatal(err)
	}
	if records[0].UserID != valueID {
		t.Fatalf("value audit user ID = %s, want %s", records[0].UserID, valueID)
	}

	if _, err := writer(context.Background(), nil, coa.AuditRecord{SegmentChangeRequestID: requestID, SegmentDefinitionID: definitionID, SegmentValueID: valueID}); err != nil {
		t.Fatal(err)
	}
	if records[1].UserID != requestID {
		t.Fatalf("change-request audit user ID = %s, want %s", records[1].UserID, requestID)
	}
}

type recordingAuthorizationEvaluator struct {
	inputs []identity.DecisionInput
}

func TestEvaluatorWorkflowRequestAuthorizerForwardsRequestAuthorizationDimensions(t *testing.T) {
	evaluator := &recordingAuthorizationEvaluator{}
	authorizer := evaluatorWorkflowRequestAuthorizer{evaluator: evaluator}
	amount := decimal.RequireFromString("125.50")
	actor := workflow.Actor{UserID: uuid.New(), SubjectReference: "requester-subject"}
	request := workflow.ApprovalRequest{
		Requester:          actor,
		Scope:              workflow.ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}},
		TransactionType:    "ordinary",
		Amount:             &amount,
		Currency:           "USD",
		DataClassification: "confidential",
	}
	if _, err := authorizer.AuthorizeApprovalDecision(context.Background(), actor, request); err != nil {
		t.Fatal(err)
	}
	if len(evaluator.inputs) != 1 {
		t.Fatalf("authorization evaluations = %d, want 1", len(evaluator.inputs))
	}
	input := evaluator.inputs[0]
	if input.DataClassification != request.DataClassification {
		t.Fatalf("data classification = %q, want %q", input.DataClassification, request.DataClassification)
	}
	if input.TransactionType == nil || *input.TransactionType != request.TransactionType {
		t.Fatalf("transaction type = %#v, want %q", input.TransactionType, request.TransactionType)
	}
	if input.Amount == nil || !input.Amount.Equal(amount) {
		t.Fatalf("amount = %#v, want %s", input.Amount, amount)
	}
	if input.Currency == nil || *input.Currency != request.Currency {
		t.Fatalf("currency = %#v, want %q", input.Currency, request.Currency)
	}
}

func TestEvaluatorWorkflowRequestAuthorizerRevalidatesStepRoleScopeDelegationAndSoD(t *testing.T) {
	actor := workflow.Actor{UserID: uuid.New(), SubjectReference: "approver"}
	roleID := uuid.New()
	userRepository := testWorkflowUserRepository{users: map[uuid.UUID]identity.User{actor.UserID: {ID: actor.UserID, Status: identity.UserStatusActive, Assignments: []identity.RoleAssignment{{RoleID: roleID, Scopes: []identity.EntityAccessScope{{ScopeID: "scope-1"}}}}}}}
	roleRepository := testWorkflowRoleRepository{roles: map[uuid.UUID]identity.Role{roleID: {ID: roleID, Name: "finance.approver", Status: identity.RoleStatusActive}}}
	evaluator := &recordingAuthorizationEvaluator{}
	authorizer := evaluatorWorkflowRequestAuthorizer{evaluator: evaluator, users: userRepository, roles: roleRepository}
	request := workflow.ApprovalRequest{Requester: workflow.Actor{UserID: uuid.New(), SubjectReference: "requester"}, Scope: workflow.ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, DataClassification: "internal"}
	step := workflow.ApprovalRequestStep{ID: uuid.New(), Sequence: 1, RequiredRoles: []workflow.RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}, MinimumApprovals: 1}
	decision, err := authorizer.AuthorizeApprovalRequestDecision(context.Background(), actor, request, step, "")
	if err != nil || !decision.Allowed || len(evaluator.inputs) != 1 || len(evaluator.inputs[0].RoleIDs) != 1 || evaluator.inputs[0].RoleIDs[0] != roleID {
		t.Fatalf("eligible role decision = %#v, err = %v, inputs = %#v", decision, err, evaluator.inputs)
	}

	delegatedStep := workflow.ApprovalRequestStep{ID: uuid.New(), Sequence: 1, RequiredRoles: []workflow.RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1, AllowDelegation: true}}, MinimumApprovals: 1, Delegation: workflow.DelegationConfiguration{Enabled: true, DelegateToRoles: []string{"finance.delegate"}, MaximumDepth: 1}}
	roleRepository.roles[roleID] = identity.Role{ID: roleID, Name: "finance.delegate", Status: identity.RoleStatusActive}
	withoutReference, err := authorizer.AuthorizeApprovalRequestDecision(context.Background(), actor, request, delegatedStep, "")
	if err != nil || withoutReference.Allowed {
		t.Fatalf("delegation without evidence = %#v, err = %v", withoutReference, err)
	}
	withReference, err := authorizer.AuthorizeApprovalRequestDecision(context.Background(), actor, request, delegatedStep, "delegation-1")
	if err != nil || !withReference.Allowed {
		t.Fatalf("delegation with evidence = %#v, err = %v", withReference, err)
	}

	segregation := testWorkflowSegregationPort{decision: identity.SegregationDecision{Allowed: false, Outcome: identity.AuthorizationDenied, DecisionReference: uuid.New()}}
	selfAuthorizer := evaluatorWorkflowRequestAuthorizer{evaluator: &recordingAuthorizationEvaluator{}, users: userRepository, roles: testWorkflowRoleRepository{roles: map[uuid.UUID]identity.Role{roleID: {ID: roleID, Name: "finance.approver", Status: identity.RoleStatusActive}}}, segregation: segregation}
	selfRequest := request
	selfRequest.Requester = actor
	selfRequest.SubjectType = "approval-policy"
	selfRequest.SubjectID = uuid.NewString()
	selfDecision, err := selfAuthorizer.AuthorizeApprovalRequestDecision(context.Background(), actor, selfRequest, step, "")
	if err != nil || selfDecision.Allowed {
		t.Fatalf("self approval SoD decision = %#v, err = %v", selfDecision, err)
	}
}

func (evaluator *recordingAuthorizationEvaluator) Evaluate(_ context.Context, input identity.DecisionInput) (identity.AuthorizationDecision, error) {
	evaluator.inputs = append(evaluator.inputs, input)
	return identity.AuthorizationDecision{
		Allowed:           true,
		Outcome:           identity.AuthorizationAllowed,
		Permission:        input.Permission,
		ApprovedScopeIDs:  input.RequestedScopeIDs,
		DecisionReference: uuid.New(),
	}, nil
}

func (recordingAuthorizationEvaluator) EvaluateField(context.Context, identity.FieldAccessInput) (identity.FieldAccessDecision, error) {
	return identity.FieldAccessDecision{}, nil
}

type testWorkflowUserRepository struct {
	users map[uuid.UUID]identity.User
}

func (repository testWorkflowUserRepository) Get(_ context.Context, id uuid.UUID) (identity.User, error) {
	user, ok := repository.users[id]
	if !ok {
		return identity.User{}, identity.ErrUserNotFound
	}
	return user, nil
}

func (repository testWorkflowUserRepository) FindByAuthenticationSubject(context.Context, identity.AuthenticationSubject) (identity.User, error) {
	return identity.User{}, identity.ErrUserNotFound
}

func (repository testWorkflowUserRepository) List(context.Context) ([]identity.User, error) {
	return nil, nil
}

type testWorkflowRoleRepository struct {
	roles map[uuid.UUID]identity.Role
}

func (repository testWorkflowRoleRepository) Get(_ context.Context, id uuid.UUID) (identity.Role, error) {
	role, ok := repository.roles[id]
	if !ok {
		return identity.Role{}, identity.ErrRoleNotFound
	}
	return role, nil
}

func (repository testWorkflowRoleRepository) List(context.Context) ([]identity.Role, error) {
	return nil, nil
}

func (repository testWorkflowRoleRepository) ValidateRoleAssignments(context.Context, []identity.RoleAssignment) error {
	return nil
}

type testWorkflowSegregationPort struct {
	decision identity.SegregationDecision
}

func (port testWorkflowSegregationPort) Evaluate(context.Context, identity.SegregationDecisionInput) (identity.SegregationDecision, error) {
	return port.decision, nil
}

func TestEvaluatorGLSoftCloseGateAuthorizerUsesOperationPermissions(t *testing.T) {
	evaluator := &recordingAuthorizationEvaluator{}
	authorizer := evaluatorGLSoftCloseGateAuthorizer{evaluator: evaluator}
	actor := gl.Actor{UserID: uuid.New()}
	scopeID := uuid.New()
	tests := []struct {
		operation  string
		permission string
	}{
		{operation: gl.SoftCloseGateEnterOperation, permission: gl.SoftCloseGateEnterPermission},
		{operation: gl.SoftCloseGateExitOperation, permission: gl.SoftCloseGateExitPermission},
		{operation: gl.ScopedReopenGateOpenOperation, permission: gl.ScopedReopenGateOpenPermission},
		{operation: gl.ScopedReopenGateCloseOperation, permission: gl.ScopedReopenGateClosePermission},
		{operation: gl.PostingBarrierAcquireOperation, permission: gl.PostingBarrierAcquirePermission},
		{operation: gl.PostingBarrierReleaseOperation, permission: gl.PostingBarrierReleasePermission},
		{operation: gl.PostingBarrierFinalizeOperation, permission: gl.PostingBarrierFinalizePermission},
	}

	for _, test := range tests {
		t.Run(test.operation, func(t *testing.T) {
			if _, err := authorizer.AuthorizeSoftCloseGate(context.Background(), actor, test.operation, scopeID); err != nil {
				t.Fatal(err)
			}
			input := evaluator.inputs[len(evaluator.inputs)-1]
			if input.Permission != test.permission {
				t.Fatalf("permission = %q, want %q", input.Permission, test.permission)
			}
			if len(input.RequestedScopeIDs) != 1 || input.RequestedScopeIDs[0] != scopeID.String() {
				t.Fatalf("requested scopes = %#v, want %q", input.RequestedScopeIDs, scopeID)
			}
		})
	}
}
