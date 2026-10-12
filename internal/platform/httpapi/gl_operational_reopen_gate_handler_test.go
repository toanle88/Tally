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

func TestGLOperationalReopenGateHandlersCloseAndBeginReclose(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scopeID, periodID, requestID, recloseRunID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	actorID := uuid.New()
	repository := gl.NewMemorySoftCloseGateRepository()
	gate := gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 4)
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
	ctx, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: actorID, Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SoftCloseGateService: service}
	approval := gl.ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "reopen-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:approved", ApproverUserID: uuid.New()}
	openRequest := softCloseGateRequest(scopeID, uuid.New(), 6, map[string]any{
		"fiscalPeriodId": periodID.String(), "reopenRequestId": requestID.String(),
		"authorizationScope":  map[string]any{"actorIds": []string{actorID.String()}, "transactionClasses": []string{"operational-recovery"}},
		"authorizationExpiry": now.Add(time.Hour).Format(time.RFC3339Nano), "controlAuthorityEpoch": 12,
		"approval": approval, "noChangeClosePermitted": true,
	})
	openResponse, err := handler.GlOpenOperationalReopenGate(ctx, openRequest, generated.GlOpenOperationalReopenGateParams{IdempotencyKey: "http-open-operational-reopen"})
	if err != nil {
		t.Fatal(err)
	}
	opened, ok := openResponse.(*generated.EstablishedResult)
	if !ok || opened.AggregateVersion != 7 {
		t.Fatalf("open response = %#v", openResponse)
	}
	var returnedScope gl.OperationalReopenScope
	if err := json.Unmarshal([]byte(opened.Data["authorizationScope"]), &returnedScope); err != nil || len(returnedScope.ActorIDs) != 1 || returnedScope.ActorIDs[0] != actorID || len(returnedScope.TransactionClasses) != 1 {
		t.Fatalf("open scope = %#v, err = %v", returnedScope, err)
	}

	active, err := repository.GetSoftCloseGate(context.Background(), scopeID, periodID)
	if err != nil {
		t.Fatal(err)
	}
	active.ActiveAdmissionCounters.AdmittedPostingCount = 1
	active.ActiveAdmissionCounters.ReopenPostingAdmitted = true
	active.NextLedgerPosition = 5
	if err := repository.Seed(active); err != nil {
		t.Fatal(err)
	}
	closeRequest := softCloseGateRequest(scopeID, uuid.New(), 7, map[string]any{"fiscalPeriodId": periodID.String(), "reopenRequestId": requestID.String()})
	closeResponse, err := handler.GlCloseOperationalReopenGate(ctx, closeRequest, generated.GlCloseOperationalReopenGateParams{IdempotencyKey: "http-close-operational-reopen"})
	if err != nil {
		t.Fatal(err)
	}
	closed, ok := closeResponse.(*generated.EstablishedResult)
	if !ok || closed.AggregateVersion != 8 {
		t.Fatalf("close response = %#v", closeResponse)
	}
	var mode string
	if err := json.Unmarshal([]byte(closed.Data["gateMode"]), &mode); err != nil || mode != gl.PostingGateCloseOnly {
		t.Fatalf("close mode = %q, err = %v", mode, err)
	}

	beginRequest := softCloseGateRequest(scopeID, uuid.New(), 8, map[string]any{"fiscalPeriodId": periodID.String(), "reopenRequestId": requestID.String(), "recloseRunId": recloseRunID.String()})
	beginResponse, err := handler.GlBeginRecloseGate(ctx, beginRequest, generated.GlBeginRecloseGateParams{IdempotencyKey: "http-begin-reclose"})
	if err != nil {
		t.Fatal(err)
	}
	started, ok := beginResponse.(*generated.EstablishedResult)
	if !ok || started.AggregateVersion != 9 {
		t.Fatalf("begin response = %#v", beginResponse)
	}
	var closeType string
	if err := json.Unmarshal([]byte(started.Data["closeType"]), &closeType); err != nil || closeType != gl.CloseTypeReclose {
		t.Fatalf("close type = %q, err = %v", closeType, err)
	}
}
