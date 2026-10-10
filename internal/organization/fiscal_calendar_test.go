package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestFiscalCalendarRejectsOverlappingOrIncompletePeriods(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := NewFiscalCalendar(uuid.New(), uuid.New(), "gregorian", "quarterly", from, []CalendarPeriod{
		{Reference: "q1", Ordinal: 1, StartDate: from, EndDate: from.AddDate(0, 0, 90)},
		{Reference: "q3", Ordinal: 3, StartDate: from.AddDate(0, 3, 0), EndDate: from.AddDate(0, 6, 0)},
	}, nil, FiscalCalendarStatusActive, from)
	if !errors.Is(err, ErrInvalidFiscalCalendar) {
		t.Fatalf("incomplete periods error = %v, want invalid fiscal calendar", err)
	}

	_, err = NewFiscalCalendar(uuid.New(), uuid.New(), "gregorian", "quarterly", from, []CalendarPeriod{
		{Reference: "q1", Ordinal: 1, StartDate: from, EndDate: from.AddDate(0, 3, 0)},
		{Reference: "q2", Ordinal: 2, StartDate: from.AddDate(0, 2, 15), EndDate: from.AddDate(0, 6, 0)},
	}, nil, FiscalCalendarStatusActive, from)
	if !errors.Is(err, ErrInvalidFiscalCalendar) {
		t.Fatalf("overlapping periods error = %v, want invalid fiscal calendar", err)
	}
}

func TestFiscalCalendarServiceIsIdempotentAndPreservesHistory(t *testing.T) {
	repository := NewMemoryFiscalCalendarRepository()
	audit := &MemoryFiscalCalendarAuditRecorder{}
	scopeID := uuid.New()
	service, err := NewFiscalCalendarService(repository, MemoryFiscalCalendarAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: FiscalCalendarManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, AllowAllFiscalCalendarApprovalValidator{}, audit, UnavailableFiscalCalendarImpactReader{}, func() time.Time { return time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "actor"}
	command := fiscalCalendarTestCommand(scopeID, FiscalCalendarActionCreate)
	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Impact.Availability != "unavailable" || len(first.FiscalCalendar.Periods) != 2 {
		t.Fatalf("create result = %#v", first)
	}
	replay, err := service.Execute(context.Background(), actor, command)
	if err != nil || !replay.Replayed {
		t.Fatalf("replay = %#v, err = %v", replay, err)
	}
	if len(audit.Records) != 1 {
		t.Fatalf("audit records after replay = %d, want 1", len(audit.Records))
	}
	reference, err := service.GetReference(context.Background(), first.FiscalCalendar.ID, scopeID)
	if err != nil || reference.ID != first.FiscalCalendar.ID || reference.ScopeID != scopeID {
		t.Fatalf("fiscal-calendar reference = %#v, err = %v", reference, err)
	}
	if _, err := service.GetReference(context.Background(), first.FiscalCalendar.ID, uuid.New()); !errors.Is(err, ErrFiscalCalendarNotFound) {
		t.Fatalf("cross-scope reference error = %v, want not found", err)
	}

	maintain := fiscalCalendarTestCommand(scopeID, FiscalCalendarActionMaintain)
	maintain.FiscalCalendarID = first.FiscalCalendar.ID
	version := aggregateversion.Initial()
	maintain.ExpectedVersion = &version
	maintain.IdempotencyKey = "maintain-1"
	maintain.Periods[1].Reference = "q2-adjusted"
	second, err := service.Execute(context.Background(), actor, maintain)
	if err != nil {
		t.Fatal(err)
	}
	if second.FiscalCalendar.Version.Value() != 2 || second.FiscalCalendar.RevisionNumber != 2 || len(second.FiscalCalendar.Revisions) != 2 {
		t.Fatalf("maintained calendar = %#v", second.FiscalCalendar)
	}
	if second.FiscalCalendar.Periods[0].ID != first.FiscalCalendar.Periods[0].ID {
		t.Fatal("unchanged period identity was not preserved")
	}

	maintain.IdempotencyKey = "maintain-stale"
	if _, err := service.Execute(context.Background(), actor, maintain); !errors.Is(err, ErrFiscalCalendarVersionConflict) {
		t.Fatalf("stale error = %v, want version conflict", err)
	}

	endDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	endDated := fiscalCalendarTestCommand(scopeID, FiscalCalendarActionMaintain)
	endDated.FiscalCalendarID = first.FiscalCalendar.ID
	endDated.ExpectedVersion = versionForTest(2)
	endDated.EffectiveTo = &endDate
	endDated.Periods[1].Reference = "q2-adjusted"
	endDated.IdempotencyKey = "maintain-end-date"
	third, err := service.Execute(context.Background(), actor, endDated)
	if err != nil {
		t.Fatal(err)
	}
	if third.FiscalCalendar.Status != FiscalCalendarStatusEndDated || third.FiscalCalendar.EffectiveTo == nil || !third.FiscalCalendar.EffectiveTo.Equal(endDate) {
		t.Fatalf("end-dated calendar = %#v", third.FiscalCalendar)
	}
}

func TestFiscalCalendarServiceRejectsScopedTypeDuplicateOnMaintain(t *testing.T) {
	repository := NewMemoryFiscalCalendarRepository()
	audit := &MemoryFiscalCalendarAuditRecorder{}
	scopeID := uuid.New()
	decision := AuthorizationDecision{Allowed: true, Permission: FiscalCalendarManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}
	service, err := NewFiscalCalendarService(repository, MemoryFiscalCalendarAuthorizer{Decision: decision}, AllowAllFiscalCalendarApprovalValidator{}, audit, UnavailableFiscalCalendarImpactReader{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "actor"}
	firstCommand := fiscalCalendarTestCommand(scopeID, FiscalCalendarActionCreate)
	first, err := service.Execute(context.Background(), actor, firstCommand)
	if err != nil {
		t.Fatal(err)
	}

	secondCommand := fiscalCalendarTestCommand(scopeID, FiscalCalendarActionCreate)
	secondCommand.CalendarType = "retail"
	secondCommand.IdempotencyKey = "create-retail"
	second, err := service.Execute(context.Background(), actor, secondCommand)
	if err != nil {
		t.Fatal(err)
	}

	maintain := fiscalCalendarTestCommand(scopeID, FiscalCalendarActionMaintain)
	maintain.FiscalCalendarID = second.FiscalCalendar.ID
	maintain.ExpectedVersion = versionForTest(1)
	maintain.CalendarType = first.FiscalCalendar.CalendarType
	maintain.IdempotencyKey = "maintain-retail-as-gregorian"
	if _, err := service.Execute(context.Background(), actor, maintain); !errors.Is(err, ErrFiscalCalendarDuplicate) {
		t.Fatalf("duplicate maintain error = %v, want duplicate", err)
	}
}

func versionForTest(value int64) *aggregateversion.AggregateVersion {
	version, err := aggregateversion.FromInt64(value)
	if err != nil {
		panic(err)
	}
	return &version
}

func fiscalCalendarTestCommand(scopeID uuid.UUID, action string) FiscalCalendarCommand {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return FiscalCalendarCommand{
		Action: action, ScopeID: scopeID, CalendarType: "gregorian", PeriodPattern: "quarterly", EffectiveFrom: from,
		Periods: []CalendarPeriod{
			{Reference: "q1", Ordinal: 1, StartDate: from, EndDate: time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)},
			{Reference: "q2", Ordinal: 2, StartDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)},
		},
		IdempotencyKey: action + "-1", CorrelationID: "correlation", CausationID: "causation",
	}
}
