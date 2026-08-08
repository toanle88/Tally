package money

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestCurrencyRegistry(t *testing.T) {
	registry, err := NewCurrencyRegistry([]CurrencyMetadata{{Code: "USD", Scale: 2}, {Code: "JPY", Scale: 0}})
	if err != nil {
		t.Fatal(err)
	}
	currency, err := registry.Lookup("USD")
	if err != nil || currency.Code() != "USD" || currency.Scale() != 2 {
		t.Fatalf("unexpected currency: %#v, %v", currency, err)
	}
	for _, code := range []string{"usd", "US", "US1"} {
		if _, err := registry.Lookup(code); !errors.Is(err, ErrMalformedCurrency) {
			t.Errorf("Lookup(%q) error = %v", code, err)
		}
	}
	if _, err := registry.Lookup("EUR"); !errors.Is(err, ErrUnknownCurrency) {
		t.Errorf("unknown currency error = %v", err)
	}
}

func TestCurrencyRegistryRejectsInvalidMetadata(t *testing.T) {
	for _, metadata := range [][]CurrencyMetadata{
		{{Code: "USD", Scale: 2}, {Code: "USD", Scale: 2}},
		{{Code: "usd", Scale: 2}},
		{{Code: "USD", Scale: 13}},
	} {
		if _, err := NewCurrencyRegistry(metadata); err == nil {
			t.Errorf("metadata %#v was accepted", metadata)
		}
	}
}

func TestMoneyNormalizesAndSerializes(t *testing.T) {
	currency := Currency{code: "USD", scale: 2}
	money, err := NewMoney(currency, "001.2300")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := money.AmountString(), "1.23"; got != want {
		t.Fatalf("AmountString() = %q, want %q", got, want)
	}
	zero, err := NewMoney(currency, "-0.00")
	if err != nil || zero.AmountString() != "0.00" {
		t.Fatalf("negative zero = %q, %v", zero.AmountString(), err)
	}
}

func TestMoneyRejectsMalformedAndOutOfScaleAmounts(t *testing.T) {
	currency := Currency{code: "USD", scale: 2}
	for _, amount := range []string{"", "1e2", "1.234", ".5", "1."} {
		if _, err := NewMoney(currency, amount); !errors.Is(err, ErrMalformedAmount) && !errors.Is(err, ErrScaleViolation) {
			t.Errorf("NewMoney(%q) error = %v", amount, err)
		}
	}
	if _, err := NewMoney(Currency{code: "JPY", scale: 0}, "1.1"); !errors.Is(err, ErrScaleViolation) {
		t.Errorf("JPY precision error = %v", err)
	}
}

func TestMoneyBoundsAndArithmetic(t *testing.T) {
	currency := Currency{code: "USD", scale: 2}
	max, err := NewMoney(currency, "99999999999999999999999999.99")
	if err != nil {
		t.Fatal(err)
	}
	zero, err := Zero(currency)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := max.Add(zero); err != nil {
		t.Fatalf("max amount addition failed: %v", err)
	}
	cent, err := NewMoney(currency, "0.01")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := max.Add(cent); !errors.Is(err, ErrPrecisionOverflow) {
		t.Errorf("overflow error = %v", err)
	}

	maxScaleCurrency := Currency{code: "XAU", scale: 12}
	maxPrecisionMoney, err := NewMoney(maxScaleCurrency, "99999999999999999999999999.999999999999")
	if err != nil {
		t.Fatal(err)
	}
	unit, err := NewMoney(maxScaleCurrency, "0.000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := maxPrecisionMoney.Add(unit); !errors.Is(err, ErrPrecisionOverflow) {
		t.Errorf("38-digit overflow error = %v", err)
	}

	other, _ := NewMoney(Currency{code: "EUR", scale: 2}, "1.00")
	one, _ := NewMoney(currency, "1.00")
	if _, err := one.Add(other); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("currency mismatch error = %v", err)
	}
	two, err := one.Add(one)
	if err != nil || two.AmountString() != "2.00" {
		t.Fatalf("addition = %q, %v", two.AmountString(), err)
	}
	result, err := two.Subtract(one)
	if err != nil || result.AmountString() != "1.00" {
		t.Fatalf("subtraction = %q, %v", result.AmountString(), err)
	}
	result, err = one.Negate()
	if err != nil || result.AmountString() != "-1.00" {
		t.Fatalf("negation = %q, %v", result.AmountString(), err)
	}
}

func TestZeroRejectsInvalidCurrency(t *testing.T) {
	if _, err := Zero(Currency{}); !errors.Is(err, ErrMalformedCurrency) {
		t.Fatalf("invalid zero currency error = %v", err)
	}
}

func TestMoneyAmountIsDefensiveCopy(t *testing.T) {
	money, err := NewMoney(Currency{code: "USD", scale: 2}, "1.00")
	if err != nil {
		t.Fatal(err)
	}
	returned := money.Amount().Add(decimal.NewFromInt(5))
	if returned.String() != "6" || money.AmountString() != "1.00" {
		t.Fatalf("Amount() exposed mutable state: returned=%s original=%s", returned, money.AmountString())
	}
}
