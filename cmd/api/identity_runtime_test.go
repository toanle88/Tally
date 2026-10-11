package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/coa"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
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
