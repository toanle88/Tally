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

func TestOrganizationCustomerProfileRepositoryPreservesPartyReferenceRevisionsAndRollsBackAuditFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID := uuid.New()
	partyID := uuid.New()
	party, err := organization.NewParty(partyID, scopeID, "Integration Customer", "customer", "active", "", nil, nil, nil, nil, organization.BankDetailControlState{Status: organization.PartyBankControlNotRequired}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
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

	profile, err := organization.NewCustomerProfile(uuid.New(), scopeID, partyID, party.Version, "net_30", organization.CreditLimit{Amount: "1000.00", Currency: "USD"}, "invoice", "standard", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil, organization.CustomerProfileStatusActive, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	profileRepository, err := organization.NewPostgresCustomerProfileRepository(fixture.pool, func(context.Context, pgx.Tx, organization.CustomerProfileAuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := profileRepository.CommitCustomerProfileMutation(ctx, organization.CustomerProfileMutation{After: profile, ExpectedPartyVersion: &party.Version, Audit: organization.CustomerProfileAuditRecord{CustomerProfileID: profile.ID, PartyID: partyID, ActorUserID: uuid.New(), Action: "integration-profile-create", ScopeID: scopeID, Permission: organization.CustomerProfileManagementPermission, DecisionReference: uuid.New(), RevisionNumber: profile.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}

	stored, err := profileRepository.Get(ctx, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	updated := stored
	if err := updated.Replace(stored, stored.PartyVersion, "net_45", organization.CreditLimit{Amount: "1250.00", Currency: "USD"}, "invoice", "standard", stored.EffectiveFrom, nil, nil, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	expectedVersion := stored.Version
	if err := profileRepository.CommitCustomerProfileMutation(ctx, organization.CustomerProfileMutation{Before: stored, After: updated, ExpectedVersion: &expectedVersion, ExpectedPartyVersion: &stored.PartyVersion, Audit: organization.CustomerProfileAuditRecord{CustomerProfileID: updated.ID, PartyID: partyID, ActorUserID: uuid.New(), Action: "integration-profile-update", ScopeID: scopeID, Permission: organization.CustomerProfileManagementPermission, DecisionReference: uuid.New(), RevisionNumber: updated.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := profileRepository.Get(ctx, profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version.Value() != 2 || reloaded.PartyVersion.Value() != 1 || len(reloaded.Revisions) != 2 || reloaded.Revisions[0].Snapshot.CreditTerms != "net_30" {
		t.Fatalf("reloaded customer profile = %#v, want version 2, Party v1, and two preserved revisions", reloaded)
	}

	failingRepository, err := organization.NewPostgresCustomerProfileRepository(fixture.pool, func(context.Context, pgx.Tx, organization.CustomerProfileAuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := organization.NewCustomerProfile(uuid.New(), scopeID, partyID, party.Version, "net_30", organization.CreditLimit{Amount: "1", Currency: "USD"}, "invoice", "standard", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil, organization.CustomerProfileStatusActive, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := failingRepository.CommitCustomerProfileMutation(ctx, organization.CustomerProfileMutation{After: failed, ExpectedPartyVersion: &party.Version, Audit: organization.CustomerProfileAuditRecord{CustomerProfileID: failed.ID, PartyID: partyID, ActorUserID: uuid.New(), Action: "integration-profile-failure", ScopeID: scopeID, Permission: organization.CustomerProfileManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failed.RevisionNumber}}); err == nil {
		t.Fatal("audit failure commit returned nil")
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM organization.customer_profile WHERE id = $1`, failed.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure left %d customer-profile rows", count)
	}
}
