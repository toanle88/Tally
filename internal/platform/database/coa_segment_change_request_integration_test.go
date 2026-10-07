//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/coa"
)

func TestCoaSegmentChangeRequestRepositoryIsAtomicAndReplaySafe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID := uuid.New()
	actor := coa.Actor{UserID: uuid.New(), SubjectReference: "integration-subject"}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	auditWriter := coa.PostgresSegmentDefinitionAuditWriter(func(context.Context, pgx.Tx, coa.AuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	definitionRepository, err := coa.NewPostgresSegmentDefinitionRepository(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := coa.NewSegmentDefinition(uuid.New(), scopeID, "department", "D-REQ", "Operations", coa.SegmentStatusDraft, from, nil, from)
	if err != nil {
		t.Fatal(err)
	}
	if err := definitionRepository.CommitSegmentDefinitionMutation(ctx, coa.SegmentDefinitionMutation{
		After: definition,
		Audit: coa.AuditRecord{SegmentDefinitionID: definition.ID, ActorUserID: actor.UserID, Action: coa.SegmentDefinitionActionCreate, ScopeID: scopeID, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), RevisionNumber: 1},
	}); err != nil {
		t.Fatal(err)
	}

	requestRepository, err := coa.NewPostgresSegmentChangeRequestRepository(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}
	requestService, err := coa.NewSegmentChangeRequestService(
		requestRepository,
		definitionRepository,
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&coa.MemoryAuditRecorder{},
		func() time.Time { return from },
	)
	if err != nil {
		t.Fatal(err)
	}
	command := coa.SegmentChangeRequestCommand{
		ChangeType: coa.SegmentChangeTypeDefinition, SubjectID: definition.ID, SubjectVersion: definition.Version, ScopeID: scopeID,
		RequestedEffectiveDate: from, ApprovalRequestID: uuid.New(), IdempotencyKey: "integration-request-1", CorrelationID: "integration-correlation",
		ProposedChange: coa.SegmentChangeProposal{Action: coa.SegmentChangeRequestAction, SegmentDefinitionID: definition.ID, SegmentType: "department", Code: "D-REQ", Name: "Operations and Shared Services", Status: coa.SegmentStatusActive, EffectiveDateFrom: from},
	}
	created, err := requestService.Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if created.SegmentChangeRequest.ID == uuid.Nil || created.SegmentChangeRequest.ApprovalStatus != coa.SegmentChangeRequestApprovalPending {
		t.Fatalf("created request = %#v", created)
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM coa.segment_change_request WHERE segment_change_request_id=$1`, created.SegmentChangeRequest.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("stored request count = %d, want 1", count)
	}
	storedSubject, err := definitionRepository.Get(ctx, definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedSubject.Version.Value() != 1 || storedSubject.Name != "Operations" {
		t.Fatalf("subject changed during request creation = %#v", storedSubject)
	}
	replayed, err := requestService.Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.SegmentChangeRequest.ID != created.SegmentChangeRequest.ID {
		t.Fatalf("replayed request = %#v", replayed)
	}

	failingRepository, err := coa.NewPostgresSegmentChangeRequestRepository(fixture.pool, func(context.Context, pgx.Tx, coa.AuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failingService, err := coa.NewSegmentChangeRequestService(
		failingRepository,
		definitionRepository,
		coa.MemoryAuthorizer{Decision: coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&coa.MemoryAuditRecorder{},
		func() time.Time { return from },
	)
	if err != nil {
		t.Fatal(err)
	}
	failingCommand := command
	failingCommand.IdempotencyKey = "integration-request-audit-failure"
	failingCommand.ProposedChange.Name = "Rolled Back Request"
	if _, err := failingService.Execute(ctx, actor, failingCommand); err == nil {
		t.Fatal("audit failure returned nil")
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM coa.segment_change_request WHERE proposed_change->>'name' = 'Rolled Back Request'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure left %d request rows", count)
	}
}
