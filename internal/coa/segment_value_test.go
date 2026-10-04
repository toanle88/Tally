package coa

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestSegmentValueServiceMaintainsParentVersionLifecycleAndIdempotency(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	repository := NewMemorySegmentDefinitionRepository()
	audit := &MemoryAuditRecorder{}
	authorizer := MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentValueManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}
	definitionService, err := NewSegmentDefinitionService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	definitionResult, err := definitionService.Execute(context.Background(), actor, testCreateCommand(scopeID, "definition-create"))
	if err != nil {
		t.Fatal(err)
	}

	valueService, err := NewSegmentValueService(repository, authorizer, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	parentVersion := aggregateversion.AggregateVersion(definitionResult.SegmentDefinition.Version.Value())
	create := SegmentValueCommand{
		Action:              SegmentValueActionCreate,
		SegmentDefinitionID: definitionResult.SegmentDefinition.ID,
		ScopeID:             scopeID,
		Value:               " 1000 ",
		Description:         "Operations",
		Status:              SegmentStatusDraft,
		EffectiveDateFrom:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectiveDateTo:     datePointer(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)),
		ExpectedVersion:     &parentVersion,
		IdempotencyKey:      "value-create",
	}
	created, err := valueService.Execute(context.Background(), actor, create)
	if err != nil {
		t.Fatal(err)
	}
	if created.SegmentValue.Value != "1000" || created.SegmentValue.Version.Value() != 2 || created.SegmentValue.SegmentDefinitionID != definitionResult.SegmentDefinition.ID {
		t.Fatalf("created value = %#v", created)
	}
	replayed, err := valueService.Execute(context.Background(), actor, create)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.SegmentValue.ID != created.SegmentValue.ID {
		t.Fatalf("replayed value = %#v", replayed)
	}

	updatedParentVersion := aggregateversion.AggregateVersion(created.SegmentValue.Version.Value())
	update := create
	update.Action = SegmentValueActionUpdate
	update.SegmentValueID = created.SegmentValue.ID
	update.ExpectedVersion = &updatedParentVersion
	update.IdempotencyKey = "value-update"
	update.Status = SegmentStatusActive
	updated, err := valueService.Execute(context.Background(), actor, update)
	if err != nil {
		t.Fatal(err)
	}
	if updated.SegmentValue.Status != SegmentStatusActive || updated.SegmentValue.Version.Value() != 3 {
		t.Fatalf("updated value = %#v", updated)
	}

	stale := update
	stale.IdempotencyKey = "value-stale"
	stale.ExpectedVersion = &updatedParentVersion
	if _, err := valueService.Execute(context.Background(), actor, stale); !errors.Is(err, ErrSegmentValueVersionConflict) {
		t.Fatalf("stale value update error = %v, want version conflict", err)
	}
	stored, err := repository.Get(context.Background(), definitionResult.SegmentDefinition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Values) != 1 || len(stored.Revisions) != 3 || stored.Values[0].Status != SegmentStatusActive {
		t.Fatalf("stored parent = %#v", stored)
	}
}

func TestSegmentValueServiceEnforcesContainmentOverlapAndRetirement(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	repository := NewMemorySegmentDefinitionRepository()
	audit := &MemoryAuditRecorder{}
	definitionService, err := NewSegmentDefinitionService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	parentCommand := testCreateCommand(scopeID, "definition-open")
	parentCommand.EffectiveDateTo = nil
	parent, err := definitionService.Execute(context.Background(), actor, parentCommand)
	if err != nil {
		t.Fatal(err)
	}
	valueService, err := NewSegmentValueService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: SegmentValueManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, audit, fixedCoaClock())
	if err != nil {
		t.Fatal(err)
	}
	version := aggregateversion.AggregateVersion(parent.SegmentDefinition.Version.Value())
	base := SegmentValueCommand{Action: SegmentValueActionCreate, SegmentDefinitionID: parent.SegmentDefinition.ID, ScopeID: scopeID, Value: "A", Description: "A", Status: SegmentStatusDraft, EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ExpectedVersion: &version, IdempotencyKey: "value-a"}
	created, err := valueService.Execute(context.Background(), actor, base)
	if err != nil {
		t.Fatal(err)
	}

	duplicate := base
	duplicate.IdempotencyKey = "value-a-duplicate"
	duplicate.Value = " A "
	duplicate.ExpectedVersion = aggregateVersionPointer(created.SegmentValue.Version.Value())
	duplicate.EffectiveDateTo = datePointer(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
	if _, err := valueService.Execute(context.Background(), actor, duplicate); !errors.Is(err, ErrSegmentValueDuplicate) {
		t.Fatalf("overlap error = %v, want duplicate", err)
	}

	outside := base
	outside.IdempotencyKey = "value-outside"
	outside.Value = "B"
	outside.EffectiveDateFrom = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	outside.ExpectedVersion = aggregateVersionPointer(created.SegmentValue.Version.Value())
	if _, err := valueService.Execute(context.Background(), actor, outside); !errors.Is(err, ErrInvalidSegmentValue) {
		t.Fatalf("outside range error = %v, want invalid value", err)
	}

	retire := base
	retire.Action = SegmentValueActionUpdate
	retire.SegmentValueID = created.SegmentValue.ID
	retire.Status = SegmentStatusRetired
	retire.ExpectedVersion = aggregateVersionPointer(created.SegmentValue.Version.Value())
	retire.IdempotencyKey = "value-retire"
	retired, err := valueService.Execute(context.Background(), actor, retire)
	if err != nil {
		t.Fatal(err)
	}
	if retired.SegmentValue.Status != SegmentStatusRetired {
		t.Fatalf("retired value = %#v", retired)
	}
	maintainRetired := retire
	maintainRetired.Status = SegmentStatusActive
	maintainRetired.ExpectedVersion = aggregateVersionPointer(retired.SegmentValue.Version.Value())
	maintainRetired.IdempotencyKey = "value-maintain-retired"
	if _, err := valueService.Execute(context.Background(), actor, maintainRetired); !errors.Is(err, ErrInvalidSegmentValue) {
		t.Fatalf("retired maintenance error = %v, want invalid value", err)
	}
}

func TestSegmentValueValidationRejectsNonCanonicalStoredValue(t *testing.T) {
	scopeID := uuid.New()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	parent, err := NewSegmentDefinition(uuid.New(), scopeID, "department", "D-001", "Operations", SegmentStatusDraft, now, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	value := SegmentValue{
		ID:                  uuid.New(),
		SegmentDefinitionID: parent.ID,
		Value:               " 1000 ",
		Status:              SegmentStatusDraft,
		EffectiveDateFrom:   now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := value.validate(parent); !errors.Is(err, ErrInvalidSegmentValue) {
		t.Fatalf("non-canonical value validation error = %v, want invalid value", err)
	}
}

func fixedCoaClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
}

func aggregateVersionPointer(value int64) *aggregateversion.AggregateVersion {
	version := aggregateversion.AggregateVersion(value)
	return &version
}
