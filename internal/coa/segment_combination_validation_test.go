package coa

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSegmentCombinationValidationReturnsSourceVersionsWithoutMutation(t *testing.T) {
	repository, scopeID, definition, value := validationRepositoryFixture(t, SegmentStatusActive, SegmentStatusActive)
	service, err := NewSegmentCombinationValidationService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: SegmentCombinationValidationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}})
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	command := SegmentCombinationValidationCommand{
		ScopeID: scopeID, BusinessDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		SegmentValues: []SegmentCombinationValueReference{{SegmentDefinitionID: definition.ID, SegmentValueID: value.ID}}, IdempotencyKey: "validate-1",
	}
	before, err := repository.Get(context.Background(), definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.ValidationStatus != SegmentCombinationValidationStatusValid || result.EffectiveDateResult != SegmentCombinationEffective || result.NextAction != "proceed" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.SourceVersions) != 1 || result.SourceVersions[0].SegmentDefinitionVersion != before.Version.Value() || result.SourceVersions[0].SegmentDefinitionRevision != before.RevisionNumber {
		t.Fatalf("source versions = %#v, definition = %#v", result.SourceVersions, before)
	}
	after, err := repository.Get(context.Background(), definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if FingerprintSegmentDefinition(before) != FingerprintSegmentDefinition(after) {
		t.Fatalf("validation mutated definition: before=%#v after=%#v", before, after)
	}
}

func TestSegmentCombinationValidationReportsLifecycleDateDuplicateAndMissingRules(t *testing.T) {
	repository, scopeID, definition, value := validationRepositoryFixture(t, SegmentStatusSuspended, SegmentStatusSuspended)
	service, err := NewSegmentCombinationValidationService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: SegmentCombinationValidationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "subject"}, SegmentCombinationValidationCommand{
		ScopeID: scopeID, BusinessDate: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SegmentValues: []SegmentCombinationValueReference{
			{SegmentDefinitionID: definition.ID, SegmentValueID: value.ID},
			{SegmentDefinitionID: definition.ID, SegmentValueID: uuid.New()},
		}, IdempotencyKey: "validate-invalid-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ValidationStatus != SegmentCombinationValidationStatusInvalid || result.EffectiveDateResult != SegmentCombinationNotEffective || result.NextAction != "correct-and-revalidate" {
		t.Fatalf("result = %#v", result)
	}
	for _, restriction := range []string{"duplicate", "effective-date", "lifecycle", "missing-value"} {
		if !containsString(result.Restrictions, restriction) {
			t.Fatalf("restrictions = %#v, missing %q", result.Restrictions, restriction)
		}
	}
	if len(result.RejectionReasons) < 4 || len(result.InvalidValues) < 4 {
		t.Fatalf("issues = %#v invalid values = %#v", result.RejectionReasons, result.InvalidValues)
	}
}

func TestSegmentCombinationValidationReportsEachCurrentAggregateSourceVersion(t *testing.T) {
	repository, scopeID, firstDefinition, firstValue := validationRepositoryFixture(t, SegmentStatusActive, SegmentStatusActive)
	secondDefinition, secondValue := validationRepositoryAddDefinition(t, repository, scopeID)
	validationRepositoryAddValue(t, repository, firstDefinition)
	service, err := NewSegmentCombinationValidationService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: SegmentCombinationValidationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "subject"}, SegmentCombinationValidationCommand{
		ScopeID: scopeID, BusinessDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		SegmentValues: []SegmentCombinationValueReference{
			{SegmentDefinitionID: firstDefinition.ID, SegmentValueID: firstValue.ID},
			{SegmentDefinitionID: secondDefinition.ID, SegmentValueID: secondValue.ID},
		}, IdempotencyKey: "validate-source-versions-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ValidationStatus != SegmentCombinationValidationStatusValid || len(result.SourceVersions) != 2 {
		t.Fatalf("result = %#v", result)
	}
	versions := make(map[uuid.UUID]SegmentCombinationSourceVersion, len(result.SourceVersions))
	for _, source := range result.SourceVersions {
		versions[source.SegmentDefinitionID] = source
	}
	if versions[firstDefinition.ID].SegmentDefinitionVersion != 3 || versions[firstDefinition.ID].SegmentDefinitionRevision != 3 {
		t.Fatalf("first source version = %#v", versions[firstDefinition.ID])
	}
	if versions[secondDefinition.ID].SegmentDefinitionVersion != 2 || versions[secondDefinition.ID].SegmentDefinitionRevision != 2 {
		t.Fatalf("second source version = %#v", versions[secondDefinition.ID])
	}
}

func TestSegmentCombinationValidationRejectsScopeAndAuthorization(t *testing.T) {
	repository, scopeID, definition, value := validationRepositoryFixture(t, SegmentStatusActive, SegmentStatusActive)
	denied, err := NewSegmentCombinationValidationService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{
		Allowed: false, Permission: SegmentCombinationValidationPermission, DecisionReference: uuid.New(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	command := SegmentCombinationValidationCommand{
		ScopeID: scopeID, BusinessDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		SegmentValues: []SegmentCombinationValueReference{{SegmentDefinitionID: definition.ID, SegmentValueID: value.ID}}, IdempotencyKey: "validate-denied-1",
	}
	if _, err := denied.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "subject"}, command); !errors.Is(err, ErrSegmentCombinationValidationAuthorizationDenied) {
		t.Fatalf("denial error = %v", err)
	}

	allowed, err := NewSegmentCombinationValidationService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: SegmentCombinationValidationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := allowed.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "subject"}, SegmentCombinationValidationCommand{
		ScopeID: uuid.New(), BusinessDate: command.BusinessDate, SegmentValues: command.SegmentValues, IdempotencyKey: "validate-scope-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ValidationStatus != SegmentCombinationValidationStatusInvalid || !containsString(result.Restrictions, "scope") {
		t.Fatalf("scope result = %#v", result)
	}
	if len(result.SourceVersions) != 0 || result.EffectiveDateResult != SegmentCombinationNotEffective {
		t.Fatalf("out-of-scope result disclosed source or date state = %#v", result)
	}
}

func TestSegmentCombinationValidationReplaysDeterministicallyAndConflictsOnChangedContent(t *testing.T) {
	repository, scopeID, definition, value := validationRepositoryFixture(t, SegmentStatusActive, SegmentStatusActive)
	service, err := NewSegmentCombinationValidationService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: SegmentCombinationValidationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}})
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	command := SegmentCombinationValidationCommand{
		ScopeID: scopeID, BusinessDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		SegmentValues: []SegmentCombinationValueReference{{SegmentDefinitionID: definition.ID, SegmentValueID: value.ID}}, IdempotencyKey: "validate-replay-1",
	}
	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || first.ValidationStatus != second.ValidationStatus || len(first.SourceVersions) != len(second.SourceVersions) {
		t.Fatalf("replay first=%#v second=%#v", first, second)
	}
	command.BusinessDate = command.BusinessDate.AddDate(0, 0, 1)
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrSegmentCombinationValidationIdempotencyConflict) {
		t.Fatalf("changed-content error = %v", err)
	}
}

func validationRepositoryFixture(t *testing.T, definitionStatus, valueStatus SegmentStatus) (*MemorySegmentDefinitionRepository, uuid.UUID, SegmentDefinition, SegmentValue) {
	t.Helper()
	repository := NewMemorySegmentDefinitionRepository()
	repository.BindAuditRecorder(&MemoryAuditRecorder{})
	scopeID := uuid.New()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	definition, err := NewSegmentDefinition(uuid.New(), scopeID, "department", "D-001", "Operations", definitionStatus, now, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitSegmentDefinitionMutation(context.Background(), SegmentDefinitionMutation{After: definition, Audit: AuditRecord{SegmentDefinitionID: definition.ID, ScopeID: scopeID}}); err != nil {
		t.Fatal(err)
	}
	value, err := NewSegmentValue(definition, uuid.New(), "1000", "Operations", valueStatus, now, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	after := cloneSegmentDefinition(definition)
	afterVersion, err := definition.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	after.Version = afterVersion
	after.RevisionNumber++
	after.UpdatedAt = now.Add(time.Hour)
	after.Values = append(after.Values, value)
	after.Revisions = append(cloneRevisions(definition.Revisions), SegmentDefinitionRevision{RevisionNumber: after.RevisionNumber, Version: after.Version, Snapshot: after.Snapshot(), CreatedAt: after.UpdatedAt})
	expected := definition.Version
	if err := repository.CommitSegmentValueMutation(context.Background(), SegmentValueMutation{Before: definition, After: after, ValueAfter: value, ExpectedVersion: &expected, Audit: AuditRecord{SegmentDefinitionID: definition.ID, SegmentValueID: value.ID, ScopeID: scopeID}}); err != nil {
		t.Fatal(err)
	}
	return repository, scopeID, after, value
}

func validationRepositoryAddDefinition(t *testing.T, repository *MemorySegmentDefinitionRepository, scopeID uuid.UUID) (SegmentDefinition, SegmentValue) {
	t.Helper()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	definition, err := NewSegmentDefinition(uuid.New(), scopeID, "cost-center", "C-001", "Corporate", SegmentStatusActive, now, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitSegmentDefinitionMutation(context.Background(), SegmentDefinitionMutation{After: definition, Audit: AuditRecord{SegmentDefinitionID: definition.ID, ScopeID: scopeID}}); err != nil {
		t.Fatal(err)
	}
	value, err := NewSegmentValue(definition, uuid.New(), "2000", "Corporate", SegmentStatusActive, now, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	return validationRepositoryCommitValue(t, repository, definition, value)
}

func validationRepositoryAddValue(t *testing.T, repository *MemorySegmentDefinitionRepository, definition SegmentDefinition) (SegmentDefinition, SegmentValue) {
	t.Helper()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(2 * time.Hour)
	value, err := NewSegmentValue(definition, uuid.New(), "1001", "Operations East", SegmentStatusActive, now, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	return validationRepositoryCommitValue(t, repository, definition, value)
}

func validationRepositoryCommitValue(t *testing.T, repository *MemorySegmentDefinitionRepository, definition SegmentDefinition, value SegmentValue) (SegmentDefinition, SegmentValue) {
	t.Helper()
	after := cloneSegmentDefinition(definition)
	afterVersion, err := definition.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	after.Version = afterVersion
	after.RevisionNumber++
	after.UpdatedAt = value.UpdatedAt
	after.Values = append(after.Values, value)
	after.Revisions = append(cloneRevisions(definition.Revisions), SegmentDefinitionRevision{RevisionNumber: after.RevisionNumber, Version: after.Version, Snapshot: after.Snapshot(), CreatedAt: after.UpdatedAt})
	expected := definition.Version
	if err := repository.CommitSegmentValueMutation(context.Background(), SegmentValueMutation{Before: definition, After: after, ValueAfter: value, ExpectedVersion: &expected, Audit: AuditRecord{SegmentDefinitionID: definition.ID, SegmentValueID: value.ID, ScopeID: definition.ScopeID}}); err != nil {
		t.Fatal(err)
	}
	return after, value
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
