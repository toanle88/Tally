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

func TestOrganizationVendorProfileRepositoryPreservesPartyReferencesAndRollsBackAuditFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID := uuid.New()
	partyID := uuid.New()
	party, err := organization.NewParty(partyID, scopeID, "Integration Vendor", "vendor", "active", "", nil, nil, nil, nil, organization.BankDetailControlState{Status: organization.PartyBankControlNotRequired}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	partyRepository, err := organization.NewPostgresPartyRepository(fixture.pool, func(context.Context, pgx.Tx, organization.PartyAuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := partyRepository.CommitPartyMutation(ctx, organization.PartyMutation{After: party, Audit: organization.PartyAuditRecord{PartyID: party.ID, ActorUserID: uuid.New(), Action: "integration-party-create", ScopeID: scopeID, Permission: organization.PartyManagementPermission, DecisionReference: uuid.New(), RevisionNumber: party.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}

	profile, err := organization.NewVendorProfile(uuid.New(), scopeID, partyID, party.Version, "net_30", "standard", "bank_transfer", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil, organization.VendorProfileStatusActive, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	profileRepository, err := organization.NewPostgresVendorProfileRepository(fixture.pool, func(context.Context, pgx.Tx, organization.VendorProfileAuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := profileRepository.CommitVendorProfileMutation(ctx, organization.VendorProfileMutation{After: profile, ExpectedPartyVersion: &party.Version, Audit: organization.VendorProfileAuditRecord{VendorProfileID: profile.ID, PartyID: partyID, ActorUserID: uuid.New(), Action: "integration-profile-create", ScopeID: scopeID, Permission: organization.VendorProfileManagementPermission, DecisionReference: uuid.New(), RevisionNumber: profile.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}

	stored, err := profileRepository.Get(ctx, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	updated := stored
	if err := updated.Replace(stored, stored.PartyVersion, "net_45", "reduced", "manual_review", stored.EffectiveFrom, nil, nil, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	expectedVersion := stored.Version
	if err := profileRepository.CommitVendorProfileMutation(ctx, organization.VendorProfileMutation{Before: stored, After: updated, ExpectedVersion: &expectedVersion, ExpectedPartyVersion: &stored.PartyVersion, Audit: organization.VendorProfileAuditRecord{VendorProfileID: updated.ID, PartyID: partyID, ActorUserID: uuid.New(), Action: "integration-profile-update", ScopeID: scopeID, Permission: organization.VendorProfileManagementPermission, DecisionReference: uuid.New(), RevisionNumber: updated.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := profileRepository.Get(ctx, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version.Value() != 2 || reloaded.PartyVersion.Value() != 1 || len(reloaded.Revisions) != 2 || reloaded.Revisions[0].Snapshot.PaymentTerms != "net_30" {
		t.Fatalf("reloaded vendor profile = %#v, want version 2, Party v1, and two preserved revisions", reloaded)
	}

	failingRepository, err := organization.NewPostgresVendorProfileRepository(fixture.pool, func(context.Context, pgx.Tx, organization.VendorProfileAuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := organization.NewVendorProfile(uuid.New(), scopeID, partyID, party.Version, "net_30", "standard", "bank_transfer", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil, organization.VendorProfileStatusActive, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := failingRepository.CommitVendorProfileMutation(ctx, organization.VendorProfileMutation{After: failed, ExpectedPartyVersion: &party.Version, Audit: organization.VendorProfileAuditRecord{VendorProfileID: failed.ID, PartyID: partyID, ActorUserID: uuid.New(), Action: "integration-profile-failure", ScopeID: scopeID, Permission: organization.VendorProfileManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failed.RevisionNumber}}); err == nil {
		t.Fatal("audit failure commit returned nil")
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM organization.vendor_profile WHERE id = $1`, failed.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure left %d vendor-profile rows", count)
	}
}
