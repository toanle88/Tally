//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/organization"
)

func TestOrganizationLegalEntityRepositoryPreservesRevisionsAndRollsBackAuditFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	t.Run("organization owns legal-entity tables", func(t *testing.T) {
		var organizationTable, platformTable bool
		if err := fixture.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'organization' AND table_name = 'legal_entity'
			)`).Scan(&organizationTable); err != nil {
			t.Fatal(err)
		}
		if err := fixture.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'platform' AND table_name = 'legal_entity'
			)`).Scan(&platformTable); err != nil {
			t.Fatal(err)
		}
		if !organizationTable || platformTable {
			t.Fatalf("organization legal_entity=%t, platform legal_entity=%t", organizationTable, platformTable)
		}
	})

	entity, err := organization.NewLegalEntity(
		uuid.New(), uuid.New(), "Integration Holdings", "VND", "USD", "private-tax-value",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		[]organization.LegalEntityRegistration{{ID: uuid.New(), Type: "company", Identifier: "123456789", Jurisdiction: "VN", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
		[]organization.LegalEntityAddress{{ID: uuid.New(), Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
		[]organization.LegalEntityOwnershipInterest{{ID: uuid.New(), OwnerReference: "owner-1", Percentage: "100", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
		nil, organization.LegalEntityStatusActive, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	auditWriter := organization.PostgresAuditWriter(func(context.Context, pgx.Tx, organization.AuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	repository, err := organization.NewPostgresLegalEntityRepository(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitLegalEntityMutation(ctx, organization.LegalEntityMutation{After: entity, Audit: organization.AuditRecord{LegalEntityID: entity.ID, ActorUserID: uuid.New(), Action: "integration-test", ScopeID: entity.ScopeID, Permission: organization.LegalEntityManagementPermission, DecisionReference: uuid.New(), RevisionNumber: entity.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}

	stored, err := repository.Get(ctx, entity.ID)
	if err != nil {
		t.Fatal(err)
	}
	updated := stored
	if err := updated.Replace(stored, "Integration Holdings Updated", "VND", "USD", stored.TaxRegistrationID, stored.EffectiveFrom, nil, stored.Registrations, stored.Addresses, stored.OwnershipInterests, nil, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	expected := stored.Version
	if err := repository.CommitLegalEntityMutation(ctx, organization.LegalEntityMutation{Before: stored, After: updated, ExpectedVersion: &expected, Audit: organization.AuditRecord{LegalEntityID: updated.ID, ActorUserID: uuid.New(), Action: "integration-update", ScopeID: updated.ScopeID, Permission: organization.LegalEntityManagementPermission, DecisionReference: uuid.New(), RevisionNumber: updated.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repository.Get(ctx, entity.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version.Value() != 2 || len(reloaded.Revisions) != 2 || reloaded.Revisions[0].Snapshot.LegalName != "Integration Holdings" {
		t.Fatalf("reloaded legal entity = %#v, want version 2 and two preserved revisions", reloaded)
	}

	failingRepository, err := organization.NewPostgresLegalEntityRepository(fixture.pool, func(context.Context, pgx.Tx, organization.AuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failed := entity
	failed.ID = uuid.New()
	failed.LegalName = "Rolled Back Holdings"
	if err := failingRepository.CommitLegalEntityMutation(ctx, organization.LegalEntityMutation{After: failed, Audit: organization.AuditRecord{LegalEntityID: failed.ID, ActorUserID: uuid.New(), Action: "integration-failure", ScopeID: failed.ScopeID, Permission: organization.LegalEntityManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failed.RevisionNumber}}); err == nil {
		t.Fatal("audit failure commit returned nil")
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM organization.legal_entity WHERE id = $1`, failed.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure left %d legal-entity rows", count)
	}
}
