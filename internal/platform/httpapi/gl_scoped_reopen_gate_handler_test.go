package httpapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestGLScopedReopenGateHandlersReturnScopeExpiryOwnerAndCloseSummary(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scopeID, periodID, requestID, accountID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := gl.NewMemorySoftCloseGateRepository()
	gate := gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 3)
	gate.GateMode = gl.PostingGateHardClosed
	gate.GateVersion = 6
	gate.AggregateVersion = 6
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateService(
		repository,
		gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "iam-policy", PolicyVersion: "iam-policy-v1"}},
		gl.AllowAllSoftClosePolicyEvaluator{},
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SoftCloseGateService: service}
	approval := gl.ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "reopen-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:approved", ApproverUserID: uuid.New()}
	openRequest := softCloseGateRequest(scopeID, uuid.New(), 6, map[string]any{
		"fiscalPeriodId": periodID.String(), "reopenRequestId": requestID.String(),
		"authorizationScope":  map[string]any{"accountIds": []string{accountID.String()}, "transactionClasses": []string{"manual-journal"}},
		"authorizationExpiry": now.Add(time.Hour).Format(time.RFC3339Nano), "controlAuthorityEpoch": 12,
		"approval": approval, "noChangeClosePermitted": true,
	})
	openResponse, err := handler.GlOpenScopedReopenGate(ctx, openRequest, generated.GlOpenScopedReopenGateParams{IdempotencyKey: "http-open-scoped-reopen"})
	if err != nil {
		t.Fatal(err)
	}
	opened, ok := openResponse.(*generated.EstablishedResult)
	if !ok || opened.AggregateVersion != 7 {
		t.Fatalf("open response = %#v", openResponse)
	}
	var mode string
	if err := json.Unmarshal([]byte(opened.Data["gateMode"]), &mode); err != nil || mode != gl.PostingGateScopedReopen {
		t.Fatalf("open mode = %q, err = %v", mode, err)
	}
	var returnedScope gl.ScopedReopenScope
	if err := json.Unmarshal([]byte(opened.Data["authorizationScope"]), &returnedScope); err != nil || len(returnedScope.AccountIDs) != 1 || returnedScope.AccountIDs[0] != accountID {
		t.Fatalf("open scope = %#v, err = %v", returnedScope, err)
	}
	var returnedExpiry time.Time
	if err := json.Unmarshal([]byte(opened.Data["authorizationExpiry"]), &returnedExpiry); err != nil || !returnedExpiry.Equal(now.Add(time.Hour)) {
		t.Fatalf("open expiry = %v, err = %v", returnedExpiry, err)
	}

	closeRequest := softCloseGateRequest(scopeID, uuid.New(), 7, map[string]any{"fiscalPeriodId": periodID.String(), "reopenRequestId": requestID.String()})
	closeResponse, err := handler.GlCloseScopedReopenGate(ctx, closeRequest, generated.GlCloseScopedReopenGateParams{IdempotencyKey: "http-close-scoped-reopen"})
	if err != nil {
		t.Fatal(err)
	}
	closed, ok := closeResponse.(*generated.EstablishedResult)
	if !ok || closed.AggregateVersion != 8 {
		t.Fatalf("close response = %#v", closeResponse)
	}
	if err := json.Unmarshal([]byte(closed.Data["gateMode"]), &mode); err != nil || mode != gl.PostingGateHardClosed {
		t.Fatalf("close mode = %q, err = %v", mode, err)
	}
	var summary gl.FrozenGateAdmissionSummary
	if err := json.Unmarshal([]byte(closed.Data["lastFrozenAdmissionSummary"]), &summary); err != nil || summary.ControlOwnerID != requestID || summary.AdmittedPostingCount != 0 {
		t.Fatalf("close summary = %#v, err = %v", summary, err)
	}
}
