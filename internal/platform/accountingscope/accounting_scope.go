package accountingscope

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/money"
)

var ErrMissingComponent = errors.New("accounting scope component is missing")

type AccountingScope struct {
	tenantID           uuid.UUID
	legalEntityID      uuid.UUID
	ledgerID           uuid.UUID
	accountingBookID   uuid.UUID
	functionalCurrency string
}

func New(tenantID, legalEntityID, ledgerID, accountingBookID uuid.UUID, functionalCurrency string) (AccountingScope, error) {
	for _, value := range []uuid.UUID{tenantID, legalEntityID, ledgerID, accountingBookID} {
		if value == uuid.Nil {
			return AccountingScope{}, ErrMissingComponent
		}
	}
	if err := money.ValidateCurrencyCode(functionalCurrency); err != nil {
		return AccountingScope{}, err
	}
	return AccountingScope{
		tenantID: tenantID, legalEntityID: legalEntityID, ledgerID: ledgerID,
		accountingBookID: accountingBookID, functionalCurrency: functionalCurrency,
	}, nil
}

func (s AccountingScope) TenantID() uuid.UUID         { return s.tenantID }
func (s AccountingScope) LegalEntityID() uuid.UUID    { return s.legalEntityID }
func (s AccountingScope) LedgerID() uuid.UUID         { return s.ledgerID }
func (s AccountingScope) AccountingBookID() uuid.UUID { return s.accountingBookID }
func (s AccountingScope) FunctionalCurrency() string  { return s.functionalCurrency }

func (s AccountingScope) Equal(other AccountingScope) bool { return s == other }

type jsonAccountingScope struct {
	TenantID           string `json:"tenantId"`
	LegalEntityID      string `json:"legalEntityId"`
	LedgerID           string `json:"ledgerId"`
	AccountingBookID   string `json:"accountingBookId"`
	FunctionalCurrency string `json:"functionalCurrency"`
}

func (s AccountingScope) MarshalJSON() ([]byte, error) {
	if _, err := New(s.tenantID, s.legalEntityID, s.ledgerID, s.accountingBookID, s.functionalCurrency); err != nil {
		return nil, err
	}
	return json.Marshal(jsonAccountingScope{
		TenantID: s.tenantID.String(), LegalEntityID: s.legalEntityID.String(),
		LedgerID: s.ledgerID.String(), AccountingBookID: s.accountingBookID.String(),
		FunctionalCurrency: s.functionalCurrency,
	})
}

func (s *AccountingScope) UnmarshalJSON(data []byte) error {
	var value jsonAccountingScope
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("accounting scope JSON contains trailing data")
		}
		return err
	}
	tenantID, err := uuid.Parse(value.TenantID)
	if err != nil {
		return err
	}
	legalEntityID, err := uuid.Parse(value.LegalEntityID)
	if err != nil {
		return err
	}
	ledgerID, err := uuid.Parse(value.LedgerID)
	if err != nil {
		return err
	}
	accountingBookID, err := uuid.Parse(value.AccountingBookID)
	if err != nil {
		return err
	}
	result, err := New(tenantID, legalEntityID, ledgerID, accountingBookID, value.FunctionalCurrency)
	if err != nil {
		return err
	}
	*s = result
	return nil
}
