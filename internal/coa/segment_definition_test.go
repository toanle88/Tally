package coa

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestSegmentDefinitionLifecycleAndInclusiveDateValidation(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	definition, err := NewSegmentDefinition(uuid.New(), uuid.New(), "department", "D-001", "Operations", SegmentStatusDraft, from, &to, now)
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version.Value() != 1 || definition.RevisionNumber != 1 {
		t.Fatalf("initial version/revision = %d/%d, want 1/1", definition.Version.Value(), definition.RevisionNumber)
	}

	active := cloneSegmentDefinition(definition)
	if err := active.Replace(definition, "department", "D-001", "Operations", SegmentStatusActive, from, &to, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if active.Status != SegmentStatusActive || active.Version.Value() != 2 || active.RevisionNumber != 2 {
		t.Fatalf("active replacement = %#v", active)
	}

	suspended := cloneSegmentDefinition(active)
	if err := suspended.Replace(active, "department", "D-001", "Operations", SegmentStatusSuspended, from, &to, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	retired := cloneSegmentDefinition(suspended)
	if err := retired.Replace(suspended, "department", "D-001", "Operations", SegmentStatusRetired, from, &to, now.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := retired.Replace(retired, "department", "D-001", "Operations", SegmentStatusActive, from, &to, now.Add(4*time.Hour)); !errors.Is(err, ErrInvalidSegmentDefinition) {
		t.Fatalf("retired transition error = %v, want invalid segment definition", err)
	}

	if _, err := NewSegmentDefinition(uuid.New(), uuid.New(), "department", "D-002", "Bad range", SegmentStatusDraft, to, &from, now); !errors.Is(err, ErrInvalidSegmentDefinition) {
		t.Fatalf("invalid range error = %v, want invalid segment definition", err)
	}
}

func TestSegmentDefinitionServiceMaintainsIdempotencyAndConcurrency(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	repository := NewMemorySegmentDefinitionRepository()
	audit := &MemoryAuditRecorder{}
	service, err := NewSegmentDefinitionService(
		repository,
		MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		audit,
		func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	command := testCreateCommand(scopeID, "create-1")
	created, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.SegmentDefinition.ID != created.SegmentDefinition.ID || len(audit.Records) != 1 {
		t.Fatalf("replay = %#v, audit records = %d", replayed, len(audit.Records))
	}

	changed := command
	changed.Name = "Changed payload"
	if _, err := service.Execute(context.Background(), actor, changed); !errors.Is(err, ErrSegmentDefinitionIdempotencyConflict) {
		t.Fatalf("changed replay error = %v, want idempotency conflict", err)
	}

	version := aggregateversion.AggregateVersion(created.SegmentDefinition.Version.Value())
	update := command
	update.Action = SegmentDefinitionActionUpdate
	update.SegmentDefinitionID = created.SegmentDefinition.ID
	update.ExpectedVersion = &version
	update.IdempotencyKey = "update-1"
	update.Status = SegmentStatusActive
	updated, err := service.Execute(context.Background(), actor, update)
	if err != nil {
		t.Fatal(err)
	}
	if updated.SegmentDefinition.Version.Value() != 2 || updated.SegmentDefinition.Status != SegmentStatusActive {
		t.Fatalf("updated result = %#v", updated)
	}

	stale := update
	stale.IdempotencyKey = "update-stale"
	stale.ExpectedVersion = &version
	if _, err := service.Execute(context.Background(), actor, stale); !errors.Is(err, ErrSegmentDefinitionVersionConflict) {
		t.Fatalf("stale update error = %v, want version conflict", err)
	}
}

func TestSegmentDefinitionServiceRejectsOverlapAndAuthorization(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	repository := NewMemorySegmentDefinitionRepository()
	service, err := NewSegmentDefinitionService(
		repository,
		MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&MemoryAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Execute(context.Background(), actor, testCreateCommand(scopeID, "first")); err != nil {
		t.Fatal(err)
	}
	overlap := testCreateCommand(scopeID, "overlap")
	if _, err := service.Execute(context.Background(), actor, overlap); !errors.Is(err, ErrSegmentDefinitionDuplicate) {
		t.Fatalf("overlap error = %v, want duplicate", err)
	}
	nonOverlap := testCreateCommand(scopeID, "non-overlap")
	nonOverlap.Code = "D-001"
	nonOverlap.EffectiveDateFrom = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	nonOverlap.EffectiveDateTo = nil
	if _, err := service.Execute(context.Background(), actor, nonOverlap); err != nil {
		t.Fatalf("non-overlap error = %v", err)
	}

	denied, err := NewSegmentDefinitionService(
		NewMemorySegmentDefinitionRepository(),
		MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: false, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New()}},
		&MemoryAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := denied.Execute(context.Background(), actor, testCreateCommand(scopeID, "denied")); !errors.Is(err, ErrSegmentDefinitionAuthorizationDenied) {
		t.Fatalf("denied error = %v, want authorization denied", err)
	}
}

func TestSegmentDefinitionServiceDoesNotPersistWhenAuditFails(t *testing.T) {
	scopeID := uuid.New()
	repository := NewMemorySegmentDefinitionRepository()
	service, err := NewSegmentDefinitionService(
		repository,
		MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		&MemoryAuditRecorder{Err: errors.New("audit unavailable")},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "subject"}, testCreateCommand(scopeID, "audit-failure")); err == nil {
		t.Fatal("audit failure returned nil")
	}
	values, err := repository.List(context.Background(), &scopeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 0 {
		t.Fatalf("audit failure persisted %d segment definitions", len(values))
	}
}

func testCreateCommand(scopeID uuid.UUID, idempotencyKey string) SegmentDefinitionCommand {
	return SegmentDefinitionCommand{
		Action:            SegmentDefinitionActionCreate,
		ScopeID:           scopeID,
		SegmentType:       "department",
		Code:              "D-001",
		Name:              "Operations",
		Status:            SegmentStatusDraft,
		EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveDateTo:   datePointer(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)),
		IdempotencyKey:    idempotencyKey,
		CorrelationID:     "correlation",
		CausationID:       "causation",
	}
}

func datePointer(value time.Time) *time.Time { return &value }
