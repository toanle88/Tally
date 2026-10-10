package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/organization"
)

type omdLegalEntityReferenceReader interface {
	GetReference(context.Context, uuid.UUID, uuid.UUID) (organization.LegalEntityReference, error)
}

type omdFiscalCalendarReferenceReader interface {
	GetReference(context.Context, uuid.UUID, uuid.UUID) (organization.FiscalCalendarReference, error)
}

// omdGLReferenceValidator is the GL-owned reference port adapter. It calls
// OMD application services only; it never reads organization tables or
// repositories directly.
type omdGLReferenceValidator struct {
	legalEntities   omdLegalEntityReferenceReader
	fiscalCalendars omdFiscalCalendarReferenceReader
}

func (validator omdGLReferenceValidator) ValidateLedgerReferences(ctx context.Context, _ gl.Actor, command gl.LedgerCommand) error {
	if validator.legalEntities == nil || validator.fiscalCalendars == nil {
		return gl.ErrLedgerReferenceUnavailable
	}
	legalEntity, err := validator.legalEntities.GetReference(ctx, command.LegalEntityID, command.AccountingScopeID)
	if err != nil {
		return mapOMDReferenceError(gl.ErrLedgerReferenceInvalid, gl.ErrLedgerReferenceUnavailable, err)
	}
	if legalEntity.ID != command.LegalEntityID || legalEntity.ScopeID != command.AccountingScopeID {
		return gl.ErrLedgerReferenceInvalid
	}
	fiscalCalendar, err := validator.fiscalCalendars.GetReference(ctx, command.FiscalCalendarID, command.AccountingScopeID)
	if err != nil {
		return mapOMDReferenceError(gl.ErrLedgerReferenceInvalid, gl.ErrLedgerReferenceUnavailable, err)
	}
	if fiscalCalendar.ID != command.FiscalCalendarID || fiscalCalendar.ScopeID != command.AccountingScopeID {
		return gl.ErrLedgerReferenceInvalid
	}
	return nil
}

func (validator omdGLReferenceValidator) ValidateAccountingBookReferences(context.Context, gl.Actor, gl.AccountingBookCommand) error {
	if validator.legalEntities == nil || validator.fiscalCalendars == nil {
		return gl.ErrAccountingBookReferenceUnavailable
	}
	// The referenced ledger is owned by GL and is validated atomically by the
	// GL repository in the same transaction as the accounting-book mutation.
	return nil
}

func mapOMDReferenceError(invalid, unavailable, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, organization.ErrLegalEntityNotFound) || errors.Is(err, organization.ErrFiscalCalendarNotFound) {
		return fmt.Errorf("%w: %v", invalid, err)
	}
	return fmt.Errorf("%w: %v", unavailable, err)
}
