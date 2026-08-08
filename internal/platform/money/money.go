package money

import (
	"errors"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	maxPrecision = 38
	maxScale     = 12
	maxInteger   = maxPrecision - maxScale
)

var (
	ErrMalformedCurrency = errors.New("malformed currency")
	ErrDuplicateCurrency = errors.New("duplicate currency metadata")
	ErrUnknownCurrency   = errors.New("unknown currency")
	ErrMalformedAmount   = errors.New("malformed decimal amount")
	ErrPrecisionOverflow = errors.New("money precision overflow")
	ErrScaleViolation    = errors.New("money scale violation")
	ErrCurrencyMismatch  = errors.New("currency mismatch")
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)
var amountPattern = regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?$`)

// ValidateCurrencyCode validates the canonical currency-code syntax shared by
// money values and other platform primitives. Currency master-data membership
// remains outside this package.
func ValidateCurrencyCode(code string) error {
	if !currencyCodePattern.MatchString(code) {
		return ErrMalformedCurrency
	}
	return nil
}

type CurrencyMetadata struct {
	Code  string
	Scale uint8
}

type Currency struct {
	code  string
	scale uint8
}

func (c Currency) Code() string { return c.code }
func (c Currency) Scale() uint8 { return c.scale }

type CurrencyRegistry struct {
	metadata map[string]Currency
}

func NewCurrencyRegistry(metadata []CurrencyMetadata) (CurrencyRegistry, error) {
	values := make(map[string]Currency, len(metadata))
	for _, item := range metadata {
		if err := ValidateCurrencyCode(item.Code); err != nil {
			return CurrencyRegistry{}, err
		}
		if item.Scale > maxScale {
			return CurrencyRegistry{}, ErrScaleViolation
		}
		if _, exists := values[item.Code]; exists {
			return CurrencyRegistry{}, ErrDuplicateCurrency
		}
		values[item.Code] = Currency{code: item.Code, scale: item.Scale}
	}
	return CurrencyRegistry{metadata: values}, nil
}

func (r CurrencyRegistry) Lookup(code string) (Currency, error) {
	if err := ValidateCurrencyCode(code); err != nil {
		return Currency{}, err
	}
	currency, ok := r.metadata[code]
	if !ok {
		return Currency{}, ErrUnknownCurrency
	}
	return currency, nil
}

type Money struct {
	currency Currency
	amount   decimal.Decimal
}

func NewMoney(currency Currency, amount string) (Money, error) {
	if currency.code == "" || ValidateCurrencyCode(currency.code) != nil || currency.scale > maxScale {
		return Money{}, ErrMalformedCurrency
	}
	if !amountPattern.MatchString(amount) {
		return Money{}, ErrMalformedAmount
	}

	canonical := canonicalAmount(amount)
	parsed, err := decimal.NewFromString(canonical)
	if err != nil {
		return Money{}, ErrMalformedAmount
	}
	if err := validateDecimal(parsed, currency.scale); err != nil {
		return Money{}, err
	}
	return Money{currency: currency, amount: parsed}, nil
}

func Zero(currency Currency) (Money, error) {
	if currency.code == "" || ValidateCurrencyCode(currency.code) != nil || currency.scale > maxScale {
		return Money{}, ErrMalformedCurrency
	}
	return Money{currency: currency, amount: decimal.Zero}, nil
}

func (m Money) Add(other Money) (Money, error) {
	if err := m.checkCurrency(other); err != nil {
		return Money{}, err
	}
	return m.fromResult(m.amount.Add(other.amount))
}

func (m Money) Subtract(other Money) (Money, error) {
	if err := m.checkCurrency(other); err != nil {
		return Money{}, err
	}
	return m.fromResult(m.amount.Sub(other.amount))
}

func (m Money) Negate() (Money, error) {
	return m.fromResult(m.amount.Neg())
}

func (m Money) Equal(other Money) bool {
	return m.currency == other.currency && m.amount.Equal(other.amount)
}

func (m Money) Amount() decimal.Decimal { return m.amount.Copy() }

func (m Money) AmountString() string {
	return m.amount.StringFixed(int32(m.currency.scale))
}

func (m Money) Currency() Currency { return m.currency }

func (m Money) checkCurrency(other Money) error {
	if m.currency != other.currency {
		return ErrCurrencyMismatch
	}
	return nil
}

func (m Money) fromResult(value decimal.Decimal) (Money, error) {
	if err := validateDecimal(value, m.currency.scale); err != nil {
		return Money{}, err
	}
	return Money{currency: m.currency, amount: value}, nil
}

func canonicalAmount(value string) string {
	negative := strings.HasPrefix(value, "-")
	unsigned := strings.TrimPrefix(value, "-")
	parts := strings.SplitN(unsigned, ".", 2)
	integer := strings.TrimLeft(parts[0], "0")
	if integer == "" {
		integer = "0"
	}
	if len(parts) == 1 {
		if negative && integer != "0" {
			return "-" + integer
		}
		return integer
	}
	fraction := strings.TrimRight(parts[1], "0")
	if fraction == "" {
		if negative && integer != "0" {
			return "-" + integer
		}
		return integer
	}
	result := integer + "." + fraction
	if negative && result != "0" {
		return "-" + result
	}
	return result
}

func validateDecimal(value decimal.Decimal, currencyScale uint8) error {
	text := value.String()
	unsigned := strings.TrimPrefix(text, "-")
	parts := strings.SplitN(unsigned, ".", 2)
	integer := strings.TrimLeft(parts[0], "0")
	if integer == "" {
		integer = "0"
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = strings.TrimRight(parts[1], "0")
	}
	if len(fraction) > int(currencyScale) || len(fraction) > maxScale {
		return ErrScaleViolation
	}
	if len(integer) > maxInteger || len(integer)+len(fraction) > maxPrecision {
		return ErrPrecisionOverflow
	}
	return nil
}
