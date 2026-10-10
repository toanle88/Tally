package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/organization"
)

type fakeGLLegalEntityReferenceReader struct {
	err       error
	reference *organization.LegalEntityReference
}

func (reader fakeGLLegalEntityReferenceReader) GetReference(_ context.Context, id, scopeID uuid.UUID) (organization.LegalEntityReference, error) {
	if reader.err != nil {
		return organization.LegalEntityReference{}, reader.err
	}
	if reader.reference != nil {
		return *reader.reference, nil
	}
	return organization.LegalEntityReference{ID: id, ScopeID: scopeID}, nil
}

type fakeGLFiscalCalendarReferenceReader struct {
	err       error
	reference *organization.FiscalCalendarReference
}

func (reader fakeGLFiscalCalendarReferenceReader) GetReference(_ context.Context, id, scopeID uuid.UUID) (organization.FiscalCalendarReference, error) {
	if reader.err != nil {
		return organization.FiscalCalendarReference{}, reader.err
	}
	if reader.reference != nil {
		return *reader.reference, nil
	}
	return organization.FiscalCalendarReference{ID: id, ScopeID: scopeID}, nil
}

func TestOMDGLReferenceValidatorChecksBothPublishedReferences(t *testing.T) {
	validator := omdGLReferenceValidator{legalEntities: fakeGLLegalEntityReferenceReader{}, fiscalCalendars: fakeGLFiscalCalendarReferenceReader{}}
	err := validator.ValidateLedgerReferences(context.Background(), gl.Actor{UserID: uuid.New(), SubjectReference: "actor"}, gl.LedgerCommand{AccountingScopeID: uuid.New(), LegalEntityID: uuid.New(), FiscalCalendarID: uuid.New()})
	if err != nil {
		t.Fatalf("reference validation error = %v", err)
	}
}

func TestOMDGLReferenceValidatorClassifiesMissingAndUnavailableReferences(t *testing.T) {
	tests := []struct {
		name      string
		validator omdGLReferenceValidator
		want      error
	}{
		{
			name:      "missing legal entity",
			validator: omdGLReferenceValidator{legalEntities: fakeGLLegalEntityReferenceReader{err: organization.ErrLegalEntityNotFound}, fiscalCalendars: fakeGLFiscalCalendarReferenceReader{}},
			want:      gl.ErrLedgerReferenceInvalid,
		},
		{
			name:      "missing calendar",
			validator: omdGLReferenceValidator{legalEntities: fakeGLLegalEntityReferenceReader{}, fiscalCalendars: fakeGLFiscalCalendarReferenceReader{err: organization.ErrFiscalCalendarNotFound}},
			want:      gl.ErrLedgerReferenceInvalid,
		},
		{
			name:      "OMD unavailable",
			validator: omdGLReferenceValidator{legalEntities: fakeGLLegalEntityReferenceReader{err: errors.New("database unavailable")}, fiscalCalendars: fakeGLFiscalCalendarReferenceReader{}},
			want:      gl.ErrLedgerReferenceUnavailable,
		},
		{
			name: "malformed legal-entity reference",
			validator: omdGLReferenceValidator{
				legalEntities:   fakeGLLegalEntityReferenceReader{reference: &organization.LegalEntityReference{ID: uuid.New(), ScopeID: uuid.New()}},
				fiscalCalendars: fakeGLFiscalCalendarReferenceReader{},
			},
			want: gl.ErrLedgerReferenceInvalid,
		},
		{
			name: "malformed fiscal-calendar reference",
			validator: omdGLReferenceValidator{
				legalEntities:   fakeGLLegalEntityReferenceReader{},
				fiscalCalendars: fakeGLFiscalCalendarReferenceReader{reference: &organization.FiscalCalendarReference{ID: uuid.New(), ScopeID: uuid.New()}},
			},
			want: gl.ErrLedgerReferenceInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.validator.ValidateLedgerReferences(context.Background(), gl.Actor{UserID: uuid.New(), SubjectReference: "actor"}, gl.LedgerCommand{AccountingScopeID: uuid.New(), LegalEntityID: uuid.New(), FiscalCalendarID: uuid.New()})
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestOMDGLReferenceValidatorFailsClosedWhenReaderIsMissing(t *testing.T) {
	validator := omdGLReferenceValidator{}
	if err := validator.ValidateLedgerReferences(context.Background(), gl.Actor{UserID: uuid.New(), SubjectReference: "actor"}, gl.LedgerCommand{}); !errors.Is(err, gl.ErrLedgerReferenceUnavailable) {
		t.Fatalf("ledger error = %v, want reference unavailable", err)
	}
	if err := validator.ValidateAccountingBookReferences(context.Background(), gl.Actor{UserID: uuid.New(), SubjectReference: "actor"}, gl.AccountingBookCommand{}); !errors.Is(err, gl.ErrAccountingBookReferenceUnavailable) {
		t.Fatalf("book error = %v, want reference unavailable", err)
	}
}

func TestOMDGLReferenceValidatorUsesOMDApplicationServices(t *testing.T) {
	scopeID := uuid.New()
	decision := organization.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}
	legalEntityService, err := organization.NewLegalEntityService(
		organization.NewMemoryLegalEntityRepository(),
		organization.MemoryAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.LegalEntityManagementPermission, DecisionReference: decision.DecisionReference, ApprovedScopeIDs: decision.ApprovedScopeIDs}},
		organization.AllowAllApprovalValidator{},
		&organization.MemoryAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	legalEntity, err := legalEntityService.Execute(context.Background(), organization.Actor{UserID: uuid.New(), SubjectReference: "omd-actor"}, organization.LegalEntityCommand{
		Action:               organization.LegalEntityActionCreate,
		ScopeID:              scopeID,
		LegalName:            "Reference Test Co.",
		FunctionalCurrency:   "USD",
		PresentationCurrency: "USD",
		EffectiveFrom:        from,
		Registrations:        []organization.LegalEntityRegistration{{Type: "company", Identifier: "REF-1", Jurisdiction: "US", EffectiveFrom: from}},
		Addresses:            []organization.LegalEntityAddress{{Type: "registered", Line1: "1 Reference Way", Locality: "Austin", PostalCode: "78701", CountryCode: "US", EffectiveFrom: from}},
		OwnershipInterests:   []organization.LegalEntityOwnershipInterest{{OwnerReference: "owner", Percentage: "100", EffectiveFrom: from}},
		IdempotencyKey:       "reference-legal-entity",
	})
	if err != nil {
		t.Fatal(err)
	}

	fiscalCalendarService, err := organization.NewFiscalCalendarService(
		organization.NewMemoryFiscalCalendarRepository(),
		organization.MemoryFiscalCalendarAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.FiscalCalendarManagementPermission, DecisionReference: decision.DecisionReference, ApprovedScopeIDs: decision.ApprovedScopeIDs}},
		organization.AllowAllFiscalCalendarApprovalValidator{},
		&organization.MemoryFiscalCalendarAuditRecorder{},
		organization.UnavailableFiscalCalendarImpactReader{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	fiscalCalendar, err := fiscalCalendarService.Execute(context.Background(), organization.Actor{UserID: uuid.New(), SubjectReference: "omd-actor"}, organization.FiscalCalendarCommand{
		Action:         organization.FiscalCalendarActionCreate,
		ScopeID:        scopeID,
		CalendarType:   "gregorian",
		PeriodPattern:  "annual",
		EffectiveFrom:  from,
		Periods:        []organization.CalendarPeriod{{Reference: "fy2026", Ordinal: 1, StartDate: from, EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)}},
		IdempotencyKey: "reference-fiscal-calendar",
	})
	if err != nil {
		t.Fatal(err)
	}

	validator := omdGLReferenceValidator{legalEntities: legalEntityService, fiscalCalendars: fiscalCalendarService}
	command := gl.LedgerCommand{AccountingScopeID: scopeID, LegalEntityID: legalEntity.LegalEntity.ID, FiscalCalendarID: fiscalCalendar.FiscalCalendar.ID}
	if err := validator.ValidateLedgerReferences(context.Background(), gl.Actor{UserID: uuid.New(), SubjectReference: "gl-actor"}, command); err != nil {
		t.Fatalf("application reference validation error = %v", err)
	}
	command.AccountingScopeID = uuid.New()
	if err := validator.ValidateLedgerReferences(context.Background(), gl.Actor{UserID: uuid.New(), SubjectReference: "gl-actor"}, command); !errors.Is(err, gl.ErrLedgerReferenceInvalid) {
		t.Fatalf("cross-scope application reference error = %v, want invalid reference", err)
	}
}
