package accountingscope

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/money"
)

func testIDs() (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	return uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd618"), uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd619"), uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd620"), uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd621")
}

func TestNewAccountingScope(t *testing.T) {
	tenant, legalEntity, ledger, book := testIDs()
	scope, err := New(tenant, legalEntity, ledger, book, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if scope.TenantID() != tenant || scope.LegalEntityID() != legalEntity || scope.LedgerID() != ledger || scope.AccountingBookID() != book || scope.FunctionalCurrency() != "USD" {
		t.Fatal("scope accessors did not preserve components")
	}
}

func TestNewAccountingScopeRejectsMissingComponents(t *testing.T) {
	tenant, legalEntity, ledger, book := testIDs()
	values := []struct {
		name                              string
		tenant, legalEntity, ledger, book uuid.UUID
	}{
		{"tenant", uuid.Nil, legalEntity, ledger, book}, {"legal entity", tenant, uuid.Nil, ledger, book},
		{"ledger", tenant, legalEntity, uuid.Nil, book}, {"book", tenant, legalEntity, ledger, uuid.Nil},
	}
	for _, test := range values {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.tenant, test.legalEntity, test.ledger, test.book, "USD"); !errors.Is(err, ErrMissingComponent) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestNewAccountingScopePropagatesCurrencyError(t *testing.T) {
	tenant, legalEntity, ledger, book := testIDs()
	if _, err := New(tenant, legalEntity, ledger, book, "usd"); !errors.Is(err, money.ErrMalformedCurrency) {
		t.Fatalf("error = %v", err)
	}
}

func TestAccountingScopeEqualityIncludesEveryComponent(t *testing.T) {
	tenant, legalEntity, ledger, book := testIDs()
	base, _ := New(tenant, legalEntity, ledger, book, "USD")
	cases := []AccountingScope{}
	otherTenant, _ := New(uuid.New(), legalEntity, ledger, book, "USD")
	cases = append(cases, otherTenant)
	otherLegal, _ := New(tenant, uuid.New(), ledger, book, "USD")
	cases = append(cases, otherLegal)
	otherLedger, _ := New(tenant, legalEntity, uuid.New(), book, "USD")
	cases = append(cases, otherLedger)
	otherBook, _ := New(tenant, legalEntity, ledger, uuid.New(), "USD")
	cases = append(cases, otherBook)
	otherCurrency, _ := New(tenant, legalEntity, ledger, book, "EUR")
	cases = append(cases, otherCurrency)
	for _, other := range cases {
		if base.Equal(other) {
			t.Fatal("distinct scope components compared equal")
		}
	}
	if !base.Equal(base) {
		t.Fatal("scope did not equal itself")
	}
}

func TestAccountingScopeSerializesAndRoundTrips(t *testing.T) {
	tenant, legalEntity, ledger, book := testIDs()
	want, _ := New(tenant, legalEntity, ledger, book, "USD")
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"tenantId":"0195a91b-20ab-7c15-8aa8-4e111a8bd618","legalEntityId":"0195a91b-20ab-7c15-8aa8-4e111a8bd619","ledgerId":"0195a91b-20ab-7c15-8aa8-4e111a8bd620","accountingBookId":"0195a91b-20ab-7c15-8aa8-4e111a8bd621","functionalCurrency":"USD"}`
	if string(data) != expected {
		t.Fatalf("JSON = %s, want %s", data, expected)
	}
	var got AccountingScope
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !want.Equal(got) {
		t.Fatal("JSON round-trip changed scope")
	}
}

func TestAccountingScopeRejectsMalformedJSONUUID(t *testing.T) {
	var scope AccountingScope
	data := []byte(`{"tenantId":"not-a-uuid","legalEntityId":"0195a91b-20ab-7c15-8aa8-4e111a8bd619","ledgerId":"0195a91b-20ab-7c15-8aa8-4e111a8bd620","accountingBookId":"0195a91b-20ab-7c15-8aa8-4e111a8bd621","functionalCurrency":"USD"}`)
	if err := json.Unmarshal(data, &scope); err == nil {
		t.Fatal("malformed UUID was accepted")
	}
}

func TestAccountingScopeRejectsZeroValueSerialization(t *testing.T) {
	if _, err := json.Marshal(AccountingScope{}); !errors.Is(err, ErrMissingComponent) {
		t.Fatalf("zero-value marshal error = %v", err)
	}
}

func TestAccountingScopeRejectsUnknownJSONFields(t *testing.T) {
	data := []byte(`{"tenantId":"0195a91b-20ab-7c15-8aa8-4e111a8bd618","legalEntityId":"0195a91b-20ab-7c15-8aa8-4e111a8bd619","ledgerId":"0195a91b-20ab-7c15-8aa8-4e111a8bd620","accountingBookId":"0195a91b-20ab-7c15-8aa8-4e111a8bd621","functionalCurrency":"USD","extra":"rejected"}`)
	var scope AccountingScope
	if err := json.Unmarshal(data, &scope); err == nil {
		t.Fatal("unknown JSON field was accepted")
	}
}

func TestAccountingScopeRejectsTrailingJSON(t *testing.T) {
	data := []byte(`{"tenantId":"0195a91b-20ab-7c15-8aa8-4e111a8bd618","legalEntityId":"0195a91b-20ab-7c15-8aa8-4e111a8bd619","ledgerId":"0195a91b-20ab-7c15-8aa8-4e111a8bd620","accountingBookId":"0195a91b-20ab-7c15-8aa8-4e111a8bd621","functionalCurrency":"USD"} {"unexpected":"value"}`)
	var scope AccountingScope
	if err := json.Unmarshal(data, &scope); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
}
