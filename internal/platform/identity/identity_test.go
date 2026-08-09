package identity

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGeneratedIDsUseUUIDV7(t *testing.T) {
	aggregate, err := NewAggregateID()
	if err != nil {
		t.Fatal(err)
	}
	if aggregate.UUID().Version() != 7 {
		t.Fatalf("generated version = %d, want 7", aggregate.UUID().Version())
	}
}

func TestFromUUIDAndParsing(t *testing.T) {
	value := uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd618")
	aggregate, err := FromAggregateUUID(value)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseAggregateID(strings.ToUpper(value.String()))
	if err != nil {
		t.Fatal(err)
	}
	if !aggregate.Equal(parsed) || aggregate.UUID() != value {
		t.Fatal("UUID construction or parsing changed the identity")
	}
}

func TestFromUUIDRejectsNil(t *testing.T) {
	constructors := []func(uuid.UUID) (any, error){
		func(value uuid.UUID) (any, error) { return FromAggregateUUID(value) },
		func(value uuid.UUID) (any, error) { return FromCorrelationUUID(value) },
		func(value uuid.UUID) (any, error) { return FromCausationUUID(value) },
	}
	for _, constructor := range constructors {
		if _, err := constructor(uuid.Nil); !errors.Is(err, ErrNilID) {
			t.Fatalf("error = %v, want ErrNilID", err)
		}
	}
}

func TestMalformedValuesReturnStableErrors(t *testing.T) {
	if _, err := ParseAggregateID(""); !errors.Is(err, ErrMalformedID) {
		t.Fatalf("empty parse error = %v", err)
	}
	if _, err := ParseCausationID("not-a-uuid"); !errors.Is(err, ErrMalformedID) {
		t.Fatalf("malformed parse error = %v", err)
	}
}

func TestJSONRoundTripsAndCanonicalizes(t *testing.T) {
	id, err := ParseCorrelationID("0195A91B-20AB-7C15-8AA8-4E111A8BD618")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `"0195a91b-20ab-7c15-8aa8-4e111a8bd618"`; got != want {
		t.Fatalf("JSON = %s, want %s", got, want)
	}
	var roundTrip CorrelationID
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if !id.Equal(roundTrip) {
		t.Fatal("JSON round-trip changed identity")
	}
}

func TestJSONInvalidValuesReturnStableErrors(t *testing.T) {
	for _, data := range []string{`null`, `123`, `"not-a-uuid"`} {
		var id AggregateID
		if err := json.Unmarshal([]byte(data), &id); !errors.Is(err, ErrInvalidJSON) && !errors.Is(err, ErrMalformedID) {
			t.Fatalf("JSON %s error = %v", data, err)
		}
	}
}

func TestZeroValueCannotSerialize(t *testing.T) {
	if _, err := json.Marshal(AggregateID{}); !errors.Is(err, ErrNilID) {
		t.Fatalf("zero-value marshal error = %v", err)
	}
}
