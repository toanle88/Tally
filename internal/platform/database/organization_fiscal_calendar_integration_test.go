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

func TestOrganizationFiscalCalendarRepositoryPreservesPeriodsAndRollsBackAuditFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	calendar, err := organization.NewFiscalCalendar(uuid.New(), scopeID, "gregorian", "quarterly", from, []organization.CalendarPeriod{
		{Reference: "q1", Ordinal: 1, StartDate: from, EndDate: time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)},
		{Reference: "q2", Ordinal: 2, StartDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)},
	}, nil, organization.FiscalCalendarStatusActive, from)
	if err != nil {
		t.Fatal(err)
	}
	auditWriter := organization.PostgresFiscalCalendarAuditWriter(func(context.Context, pgx.Tx, organization.FiscalCalendarAuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	repository, err := organization.NewPostgresFiscalCalendarRepository(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitFiscalCalendarMutation(ctx, organization.FiscalCalendarMutation{After: calendar, Audit: organization.FiscalCalendarAuditRecord{FiscalCalendarID: calendar.ID, ActorUserID: uuid.New(), Action: "integration-create", ScopeID: scopeID, Permission: organization.FiscalCalendarManagementPermission, DecisionReference: uuid.New(), RevisionNumber: calendar.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}

	stored, err := repository.Get(ctx, calendar.ID)
	if err != nil {
		t.Fatal(err)
	}
	updated := stored
	updatedPeriods := append([]organization.CalendarPeriod(nil), stored.Periods...)
	updatedPeriods[1].Reference = "q2-adjusted"
	if err := updated.Replace(stored, "gregorian", "quarterly-adjusted", stored.EffectiveFrom, nil, updatedPeriods, nil, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	expected := stored.Version
	if err := repository.CommitFiscalCalendarMutation(ctx, organization.FiscalCalendarMutation{Before: stored, After: updated, ExpectedVersion: &expected, Audit: organization.FiscalCalendarAuditRecord{FiscalCalendarID: updated.ID, ActorUserID: uuid.New(), Action: "integration-update", ScopeID: scopeID, Permission: organization.FiscalCalendarManagementPermission, DecisionReference: uuid.New(), RevisionNumber: updated.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repository.Get(ctx, calendar.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version.Value() != 2 || len(reloaded.Periods) != 2 || reloaded.Periods[0].ID != stored.Periods[0].ID || len(reloaded.Revisions) != 2 || reloaded.Revisions[0].Snapshot.Periods[1].Reference != "q2" {
		t.Fatalf("reloaded fiscal calendar = %#v, want two periods and preserved revision", reloaded)
	}

	failingRepository, err := organization.NewPostgresFiscalCalendarRepository(fixture.pool, func(context.Context, pgx.Tx, organization.FiscalCalendarAuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failed := calendar
	failed.ID = uuid.New()
	failed.ScopeID = uuid.New()
	if err := failingRepository.CommitFiscalCalendarMutation(ctx, organization.FiscalCalendarMutation{After: failed, Audit: organization.FiscalCalendarAuditRecord{FiscalCalendarID: failed.ID, ActorUserID: uuid.New(), Action: "integration-failure", ScopeID: failed.ScopeID, Permission: organization.FiscalCalendarManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failed.RevisionNumber}}); err == nil {
		t.Fatal("audit failure commit returned nil")
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM organization.fiscal_calendar WHERE id = $1`, failed.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure left %d fiscal-calendar rows", count)
	}
}
