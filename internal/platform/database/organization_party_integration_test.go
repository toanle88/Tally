//go:build integration

package database

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/organization"
)

func TestOrganizationPartyRepositoryPreservesSafeRevisionsAndRollsBackAuditFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID := uuid.New()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	party, err := organization.NewParty(
		uuid.New(), scopeID, "Integration Vendor", "vendor", "active", "TAX-PRIVATE-1234",
		[]organization.PartyContactMethod{{ID: uuid.New(), Type: "email", Value: "finance@example.test"}},
		[]organization.PartyAddress{{ID: uuid.New(), Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN"}},
		[]organization.PartyClassification{{ID: uuid.New(), Code: "segment", Value: "supplier"}},
		[]organization.PartyBankDetailReference{{ID: uuid.New(), Reference: "provider-ref-001", ProviderCode: "provider-a", ConsentReference: "consent-ref-001"}},
		organization.BankDetailControlState{Status: organization.PartyBankControlApproved}, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	auditWriter := organization.PostgresPartyAuditWriter(func(context.Context, pgx.Tx, organization.PartyAuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	repository, err := organization.NewPostgresPartyRepository(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitPartyMutation(ctx, organization.PartyMutation{After: party, Audit: organization.PartyAuditRecord{PartyID: party.ID, ActorUserID: uuid.New(), Action: organization.PartyActionCreate, ScopeID: scopeID, Permission: organization.PartyManagementPermission, DecisionReference: uuid.New(), RevisionNumber: 1}}); err != nil {
		t.Fatal(err)
	}

	stored, err := repository.Get(ctx, party.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.TaxIdentifier != "TAX-PRIVATE-1234" || len(stored.Revisions) != 1 || stored.SafeProjection().TaxIdentifierMasked != "••••1234" {
		t.Fatalf("stored party = %#v, want restricted value internally and masked projection", stored)
	}
	if strings.Contains(string(mustJSON(stored.SafeProjection())), "TAX-PRIVATE-1234") {
		t.Fatal("safe party projection leaked restricted tax identifier")
	}

	updated := stored
	if err := updated.Replace(stored, "Integration Vendor Updated", stored.PartyType, stored.Status, stored.TaxIdentifier, stored.ContactMethods, stored.Addresses, stored.Classifications, stored.BankDetailReferences, stored.BankControl, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	expected := stored.Version
	if err := repository.CommitPartyMutation(ctx, organization.PartyMutation{Before: stored, After: updated, ExpectedVersion: &expected, Audit: organization.PartyAuditRecord{PartyID: updated.ID, ActorUserID: uuid.New(), Action: organization.PartyActionMaintain, ScopeID: scopeID, Permission: organization.PartyManagementPermission, DecisionReference: uuid.New(), RevisionNumber: updated.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repository.Get(ctx, party.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version.Value() != 2 || len(reloaded.Revisions) != 2 || reloaded.Revisions[0].Snapshot.Name != "Integration Vendor" {
		t.Fatalf("reloaded party = %#v, want two immutable revisions", reloaded)
	}

	failingRepository, err := organization.NewPostgresPartyRepository(fixture.pool, func(context.Context, pgx.Tx, organization.PartyAuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failed := party
	failed.ID = uuid.New()
	failed.Name = "Rolled Back Party"
	if err := failingRepository.CommitPartyMutation(ctx, organization.PartyMutation{After: failed, Audit: organization.PartyAuditRecord{PartyID: failed.ID, ActorUserID: uuid.New(), Action: organization.PartyActionCreate, ScopeID: scopeID, Permission: organization.PartyManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failed.RevisionNumber}}); err == nil {
		t.Fatal("audit failure commit returned nil")
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM organization.party WHERE id = $1`, failed.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure left %d party rows", count)
	}
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
