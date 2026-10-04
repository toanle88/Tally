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

func TestCoaSegmentDefinitionRepositoryPreservesRevisionsAndRollsBackAuditFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	var coaTable, publicTable bool
	if err := fixture.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'coa' AND table_name = 'segment_definition'
		)`).Scan(&coaTable); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'segment_definition'
		)`).Scan(&publicTable); err != nil {
		t.Fatal(err)
	}
	if !coaTable || publicTable {
		t.Fatalf("coa segment_definition=%t, public segment_definition=%t", coaTable, publicTable)
	}

	scopeID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	definition, err := coa.NewSegmentDefinition(uuid.New(), scopeID, "department", "D-001", "Operations", coa.SegmentStatusDraft, from, nil, from)
	if err != nil {
		t.Fatal(err)
	}
	auditWriter := coa.PostgresSegmentDefinitionAuditWriter(func(context.Context, pgx.Tx, coa.AuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	repository, err := coa.NewPostgresSegmentDefinitionRepository(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitSegmentDefinitionMutation(ctx, coa.SegmentDefinitionMutation{
		After: definition,
		Audit: coa.AuditRecord{SegmentDefinitionID: definition.ID, ActorUserID: uuid.New(), Action: coa.SegmentDefinitionActionCreate, ScopeID: scopeID, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), RevisionNumber: definition.RevisionNumber},
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := repository.Get(ctx, definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	updated := stored
	if err := updated.Replace(stored, "department", "D-001", "Operations and Shared Services", coa.SegmentStatusActive, from, nil, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	expectedVersion := stored.Version
	if err := repository.CommitSegmentDefinitionMutation(ctx, coa.SegmentDefinitionMutation{
		Before: stored, After: updated, ExpectedVersion: &expectedVersion,
		Audit: coa.AuditRecord{SegmentDefinitionID: updated.ID, ActorUserID: uuid.New(), Action: coa.SegmentDefinitionActionUpdate, ScopeID: scopeID, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), RevisionNumber: updated.RevisionNumber},
	}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repository.Get(ctx, definition.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version.Value() != 2 || len(reloaded.Revisions) != 2 || reloaded.Revisions[0].Snapshot.Name != "Operations" {
		t.Fatalf("reloaded segment definition = %#v, want version 2 and preserved revision", reloaded)
	}

	duplicate, err := coa.NewSegmentDefinition(uuid.New(), scopeID, "department", "D-001", "Overlapping", coa.SegmentStatusDraft, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), nil, from)
	if err != nil {
		t.Fatal(err)
	}
	duplicateTo := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	duplicate.EffectiveDateTo = &duplicateTo
	if err := repository.CommitSegmentDefinitionMutation(ctx, coa.SegmentDefinitionMutation{
		After: duplicate,
		Audit: coa.AuditRecord{SegmentDefinitionID: duplicate.ID, ActorUserID: uuid.New(), Action: coa.SegmentDefinitionActionCreate, ScopeID: scopeID, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), RevisionNumber: duplicate.RevisionNumber},
	}); !errors.Is(err, coa.ErrSegmentDefinitionDuplicate) {
		t.Fatalf("overlapping definition error = %v, want duplicate", err)
	}

	failingRepository, err := coa.NewPostgresSegmentDefinitionRepository(fixture.pool, func(context.Context, pgx.Tx, coa.AuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := coa.NewSegmentDefinition(uuid.New(), scopeID, "department", "D-002", "Rolled Back", coa.SegmentStatusDraft, from, nil, from)
	if err != nil {
		t.Fatal(err)
	}
	if err := failingRepository.CommitSegmentDefinitionMutation(ctx, coa.SegmentDefinitionMutation{
		After: failed,
		Audit: coa.AuditRecord{SegmentDefinitionID: failed.ID, ActorUserID: uuid.New(), Action: coa.SegmentDefinitionActionCreate, ScopeID: scopeID, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failed.RevisionNumber},
	}); err == nil {
		t.Fatal("audit failure commit returned nil")
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM coa.segment_definition WHERE segment_definition_id = $1`, failed.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure left %d segment-definition rows", count)
	}

	value, err := coa.NewSegmentValue(reloaded, uuid.New(), "1000", "Operations", coa.SegmentStatusActive, from, nil, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	valueParent := reloaded
	valueParent.Values = append(valueParent.Values, value)
	valueParent.Version, err = reloaded.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	valueParent.RevisionNumber++
	valueParent.UpdatedAt = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	valueParent.Revisions = append(valueParent.Revisions, coa.SegmentDefinitionRevision{RevisionNumber: valueParent.RevisionNumber, Version: valueParent.Version, Snapshot: valueParent.Snapshot(), CreatedAt: valueParent.UpdatedAt})
	valueExpected := reloaded.Version
	if err := repository.CommitSegmentValueMutation(ctx, coa.SegmentValueMutation{
		Before: reloaded, After: valueParent, ValueAfter: value, ExpectedVersion: &valueExpected,
		Audit: coa.AuditRecord{SegmentDefinitionID: reloaded.ID, SegmentValueID: value.ID, ActorUserID: uuid.New(), Action: coa.SegmentValueActionCreate, ScopeID: scopeID, Permission: coa.SegmentValueManagementPermission, DecisionReference: uuid.New(), RevisionNumber: valueParent.RevisionNumber},
	}); err != nil {
		t.Fatal(err)
	}
	withValue, err := repository.Get(ctx, reloaded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(withValue.Values) != 1 || withValue.Values[0].Value != "1000" || withValue.Version.Value() != 3 || len(withValue.Revisions) != 3 || len(withValue.Revisions[2].Snapshot.Values) != 1 {
		t.Fatalf("stored segment value aggregate = %#v", withValue)
	}

	failingValueRepository, err := coa.NewPostgresSegmentDefinitionRepository(fixture.pool, func(context.Context, pgx.Tx, coa.AuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("value audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failedValue, err := coa.NewSegmentValue(withValue, uuid.New(), "2000", "Rolled Back", coa.SegmentStatusDraft, from, nil, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	failedValueParent := withValue
	failedValueParent.Values = append(failedValueParent.Values, failedValue)
	failedValueParent.Version, err = withValue.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	failedValueParent.RevisionNumber++
	failedValueParent.UpdatedAt = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	failedValueParent.Revisions = append(failedValueParent.Revisions, coa.SegmentDefinitionRevision{RevisionNumber: failedValueParent.RevisionNumber, Version: failedValueParent.Version, Snapshot: failedValueParent.Snapshot(), CreatedAt: failedValueParent.UpdatedAt})
	failedValueExpected := withValue.Version
	if err := failingValueRepository.CommitSegmentValueMutation(ctx, coa.SegmentValueMutation{
		Before: withValue, After: failedValueParent, ValueAfter: failedValue, ExpectedVersion: &failedValueExpected,
		Audit: coa.AuditRecord{SegmentDefinitionID: withValue.ID, SegmentValueID: failedValue.ID, ActorUserID: uuid.New(), Action: coa.SegmentValueActionCreate, ScopeID: scopeID, Permission: coa.SegmentValueManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failedValueParent.RevisionNumber},
	}); err == nil {
		t.Fatal("value audit failure returned nil")
	}
	var valueCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM coa.segment_value WHERE segment_value_id = $1`, failedValue.ID).Scan(&valueCount); err != nil {
		t.Fatal(err)
	}
	if valueCount != 0 {
		t.Fatalf("value audit failure left %d segment-value rows", valueCount)
	}
}
