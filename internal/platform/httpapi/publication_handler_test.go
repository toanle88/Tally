package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestMasterDataPublicationHandlerReturnsEstablishedResult(t *testing.T) {
	scopeID := uuid.New()
	aggregateID := uuid.New()
	candidate := organization.MasterDataPublicationCandidate{AggregateType: organization.MasterDataAggregateParty, AggregateID: aggregateID, ScopeID: scopeID, Status: "active", Version: aggregateversion.Initial(), RevisionNumber: 1, Fingerprint: "sha256:party", Payload: []byte(`{"name":"safe party"}`)}
	repository := organization.NewMemoryMasterDataPublicationRepository(candidate)
	audit := &organization.MemoryMasterDataPublicationAuditRecorder{}
	service, err := organization.NewMasterDataPublicationService(repository, organization.MemoryMasterDataPublicationAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.MasterDataPublicationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, func() time.Time { return time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{PublicationService: service}
	request := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), ExpectedVersion: generated.NewOptInt(1), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{"aggregateType": mustRaw("party"), "aggregateId": mustRaw(aggregateID.String())}}
	response, err := handler.OmdPublishApprovedMasterDataChanges(ctx, request, generated.OmdPublishApprovedMasterDataChangesParams{IdempotencyKey: "publication-http-1"})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := response.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("response = %T, want established result", response)
	}
	if result.AggregateId != generated.UUID(aggregateID) || result.AggregateVersion != 1 || result.Status != organization.MasterDataPublicationStatus {
		t.Fatalf("result = %#v", result)
	}
	if len(audit.Records) != 1 || audit.Records[0].AggregateType != organization.MasterDataAggregateParty {
		t.Fatalf("audit records = %#v", audit.Records)
	}
}

func TestMasterDataPublicationHandlerDoesNotAcceptClientApprovalOrSnapshot(t *testing.T) {
	scopeID := uuid.New()
	aggregateID := uuid.New()
	version := aggregateversion.Initial()
	repository := organization.NewMemoryMasterDataPublicationRepository(organization.MasterDataPublicationCandidate{AggregateType: organization.MasterDataAggregateParty, AggregateID: aggregateID, ScopeID: scopeID, Status: "active", Version: version, RevisionNumber: 1, Fingerprint: "sha256:party", Payload: []byte(`{"name":"safe party"}`)})
	service, err := organization.NewMasterDataPublicationService(repository, organization.MemoryMasterDataPublicationAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.MasterDataPublicationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, &organization.MemoryMasterDataPublicationAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	response, err := (IdentityHandler{PublicationService: service}).OmdPublishApprovedMasterDataChanges(ctx, &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), ExpectedVersion: generated.NewOptInt(1), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{"aggregateType": mustRaw("party"), "aggregateId": mustRaw(aggregateID.String()), "approval": mustRaw(map[string]any{"decisionId": uuid.NewString()}), "snapshot": mustRaw(map[string]any{"raw": "client"})}}, generated.OmdPublishApprovedMasterDataChangesParams{IdempotencyKey: "publication-http-forbidden"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.OmdPublishApprovedMasterDataChangesBadRequest); !ok {
		t.Fatalf("response = %T, want bad request", response)
	}
}
