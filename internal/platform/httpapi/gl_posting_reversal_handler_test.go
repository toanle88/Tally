package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/platform/money"
)

func TestGLJournalReversalHandlerReturnsLineageAndReplays(t *testing.T) {
	registry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{{Code: "USD", Scale: 2}})
	if err != nil {
		t.Fatal(err)
	}
	repository := gl.NewMemoryPostingRepository()
	audit := &gl.MemoryPostingAuditRecorder{}
	clock := func() time.Time { return time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC) }
	postingAuthorizer := gl.MemoryPostingAuthorizer{Decision: gl.PostingAuthorizationDecision{Allowed: true, PolicyReference: "http-policy", PolicyVersion: "http-v1"}}
	posting, err := gl.NewPostingService(repository, postingAuthorizer, gl.AllowAllPostingReferenceValidator{}, gl.MemoryPostingGateValidator{}, gl.AllowAllPostingApprovalValidator{}, audit, registry, clock)
	if err != nil {
		t.Fatal(err)
	}
	reversal, err := gl.NewJournalReversalService(repository, gl.MemoryJournalReversalAuthorizer{Decision: postingAuthorizer.Decision}, gl.AllowAllPostingReferenceValidator{}, gl.MemoryPostingGateValidator{}, registry, clock)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{PostingService: posting, JournalReversalService: reversal}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	originalRequest := newHTTPPostingRequest()
	originalResponse, err := handler.GlSubmitPostingRequest(actorContext, originalRequest, generated.GlSubmitPostingRequestParams{IdempotencyKey: "http-original"})
	if err != nil {
		t.Fatal(err)
	}
	originalEstablished, ok := originalResponse.(*generated.GlSubmitPostingRequestEstablishedResult)
	if !ok {
		t.Fatalf("original response = %#v", originalResponse)
	}
	original := repository.Journals()[0]
	if originalEstablished.AggregateId != generated.UUID(original.ID) {
		t.Fatalf("original response = %#v", originalEstablished)
	}

	authorizationID := uuid.New()
	correlationID := uuid.New()
	command := &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), ExpectedVersion: generated.NewOptInt(1), AccountingScopeId: generated.NewOptUUID(originalRequest.AccountingScopeId),
		Data: generated.CommandRequestData{
			"journalId": mustRaw(original.ID.String()), "reason": mustRaw("Correct a posted classification"), "effectivePostingDate": mustRaw("2026-08-20"),
			"fiscalPeriodId": mustRaw(original.GateEvidence.FiscalPeriodID.String()), "periodStateVersion": mustRaw(original.GateEvidence.PeriodStateVersion), "postingGateVersion": mustRaw(original.GateEvidence.PostingGateVersion),
			"postingAuthorizationId": mustRaw(authorizationID.String()), "sourceContext": mustRaw("general-ledger"), "sourceAggregateType": mustRaw("journal-correction"), "sourceAggregateId": mustRaw(uuid.NewString()), "sourceVersion": mustRaw(1),
		},
	}
	params := generated.GlReverseJournalEntryParams{IdempotencyKey: "http-reversal", XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID)), XAccountingScopeID: generated.NewOptUUID(originalRequest.AccountingScopeId)}
	response, err := handler.GlReverseJournalEntry(actorContext, command, params)
	if err != nil {
		t.Fatal(err)
	}
	established, ok := response.(*generated.EstablishedResult)
	if !ok || established.Status != "established" || established.AggregateVersion != 1 || established.CorrelationId != generated.UUID(correlationID) {
		t.Fatalf("reversal response = %#v", response)
	}
	if string(established.Data["outcome"]) != `"JournalEntryReversed"` || string(established.Data["originalJournalId"]) != `"`+original.ID.String()+`"` {
		t.Fatalf("reversal lineage = %#v", established.Data)
	}

	replay, err := handler.GlReverseJournalEntry(actorContext, command, params)
	if err != nil {
		t.Fatal(err)
	}
	replayed, ok := replay.(*generated.EstablishedResult)
	if !ok || string(replayed.Data["replayed"]) != "true" || replayed.AggregateId != established.AggregateId {
		t.Fatalf("replay response = %#v", replay)
	}

	changed := *command
	changed.Data = generated.CommandRequestData{}
	for key, value := range command.Data {
		changed.Data[key] = value
	}
	changed.Data["reason"] = mustRaw("Changed reason")
	conflict, err := handler.GlReverseJournalEntry(actorContext, &changed, params)
	if err != nil {
		t.Fatal(err)
	}
	if problem, ok := conflict.(*generated.GlReverseJournalEntryConflict); !ok || problem.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("idempotency conflict = %#v", conflict)
	}
	if got := len(repository.Journals()); got != 2 {
		t.Fatalf("journal count = %d, want original and reversal", got)
	}
}

func TestGLJournalReversalHandlerRequiresEvidenceAndActor(t *testing.T) {
	registry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{{Code: "USD", Scale: 2}})
	if err != nil {
		t.Fatal(err)
	}
	repository := gl.NewMemoryPostingRepository()
	service, err := gl.NewJournalReversalService(repository, gl.MemoryJournalReversalAuthorizer{Decision: gl.PostingAuthorizationDecision{Allowed: true}}, gl.AllowAllPostingReferenceValidator{}, gl.MemoryPostingGateValidator{}, registry, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	request := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), ExpectedVersion: generated.NewOptInt(1), AccountingScopeId: generated.NewOptUUID(generated.UUID(uuid.New())), Data: generated.CommandRequestData{"journalId": mustRaw(uuid.NewString())}}
	params := generated.GlReverseJournalEntryParams{IdempotencyKey: "http-invalid"}
	if response, err := (IdentityHandler{JournalReversalService: service}).GlReverseJournalEntry(context.Background(), request, params); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlReverseJournalEntryForbidden); !ok || problem.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("missing actor response = %#v", response)
	}

	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	if response, err := (IdentityHandler{JournalReversalService: service}).GlReverseJournalEntry(actorContext, request, params); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlReverseJournalEntryBadRequest); !ok || problem.Code != "INVALID_REQUEST" {
		t.Fatalf("missing evidence response = %#v", response)
	}
}

func TestGLJournalReversalHandlerCarriesScopedReopenEvidence(t *testing.T) {
	registry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{{Code: "USD", Scale: 2}})
	if err != nil {
		t.Fatal(err)
	}
	repository := gl.NewMemoryPostingRepository()
	audit := &gl.MemoryPostingAuditRecorder{}
	clockTime := time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC)
	postingAuthorizer := gl.MemoryPostingAuthorizer{Decision: gl.PostingAuthorizationDecision{Allowed: true}}
	posting, err := gl.NewPostingService(repository, postingAuthorizer, gl.AllowAllPostingReferenceValidator{}, gl.MemoryPostingGateValidator{}, gl.AllowAllPostingApprovalValidator{}, audit, registry, func() time.Time { return clockTime })
	if err != nil {
		t.Fatal(err)
	}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	originalRequest := newHTTPPostingRequest()
	originalResponse, err := (IdentityHandler{PostingService: posting}).GlSubmitPostingRequest(actorContext, originalRequest, generated.GlSubmitPostingRequestParams{IdempotencyKey: "http-scoped-original"})
	if err != nil {
		t.Fatal(err)
	}
	originalEstablished, ok := originalResponse.(*generated.GlSubmitPostingRequestEstablishedResult)
	if !ok {
		t.Fatalf("original response = %#v", originalResponse)
	}

	reopenRequestID := uuid.New()
	controlAuthorityEpoch := int64(9)
	sourceAggregateType := "manual-journal"
	accountIDs := make([]uuid.UUID, 0, len(originalRequest.Data.Lines))
	for _, line := range originalRequest.Data.Lines {
		accountIDs = append(accountIDs, uuid.UUID(line.AccountId))
	}
	canonicalScope, err := (gl.ScopedReopenScope{AccountIDs: accountIDs, TransactionClasses: []string{sourceAggregateType}}).Canonical()
	if err != nil {
		t.Fatal(err)
	}
	scopeJSON, err := json.Marshal(canonicalScope)
	if err != nil {
		t.Fatal(err)
	}
	expiry := clockTime.Add(time.Hour)
	gateEvidence := gl.PostingGateEvidence{
		FiscalPeriodID: uuid.UUID(originalRequest.Data.FiscalPeriodId), PeriodStateVersion: int64(originalRequest.Data.PeriodStateVersion),
		PostingGateVersion: int64(originalRequest.Data.PostingGateVersion), GateMode: gl.PostingGateScopedReopen,
		ReopenRequestID: &reopenRequestID, AuthorizationScope: scopeJSON,
		AuthorizationExpiry: &expiry, ControlAuthorityEpoch: &controlAuthorityEpoch,
	}
	reversal, err := gl.NewJournalReversalService(repository, gl.MemoryJournalReversalAuthorizer{Decision: postingAuthorizer.Decision}, gl.AllowAllPostingReferenceValidator{}, gl.MemoryPostingGateValidator{Evidence: gateEvidence}, registry, func() time.Time { return clockTime })
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{JournalReversalService: reversal}
	command := &generated.CommandRequest{
		CommandId: generated.UUID(uuid.New()), ExpectedVersion: generated.NewOptInt(originalEstablished.AggregateVersion), AccountingScopeId: generated.NewOptUUID(originalRequest.AccountingScopeId),
		Data: generated.CommandRequestData{
			"journalId": mustRaw(uuid.UUID(originalEstablished.AggregateId).String()), "reason": mustRaw("Correct a scoped classification"), "effectivePostingDate": mustRaw("2026-08-20"),
			"fiscalPeriodId": mustRaw(uuid.UUID(originalRequest.Data.FiscalPeriodId).String()), "periodStateVersion": mustRaw(originalRequest.Data.PeriodStateVersion), "postingGateVersion": mustRaw(originalRequest.Data.PostingGateVersion),
			"postingAuthorizationId": mustRaw(uuid.NewString()), "reopenRequestId": mustRaw(reopenRequestID.String()), "controlAuthorityEpoch": mustRaw(controlAuthorityEpoch),
			"sourceContext": mustRaw("general-ledger"), "sourceAggregateType": mustRaw(sourceAggregateType), "sourceAggregateId": mustRaw(uuid.NewString()), "sourceVersion": mustRaw(1),
		},
	}
	params := generated.GlReverseJournalEntryParams{IdempotencyKey: "http-scoped-reversal", XAccountingScopeID: generated.NewOptUUID(originalRequest.AccountingScopeId)}
	response, err := handler.GlReverseJournalEntry(actorContext, command, params)
	if err != nil {
		t.Fatal(err)
	}
	established, ok := response.(*generated.EstablishedResult)
	if !ok || string(established.Data["outcome"]) != `"JournalEntryReversed"` {
		t.Fatalf("scoped reversal response = %#v", response)
	}
	if string(established.Data["gateEvidence"]) == "null" || !strings.Contains(string(established.Data["gateEvidence"]), reopenRequestID.String()) {
		t.Fatalf("scoped gate evidence = %s", established.Data["gateEvidence"])
	}
}
