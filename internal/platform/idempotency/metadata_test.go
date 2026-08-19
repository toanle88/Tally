package idempotency

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/accountingscope"
)

func metadataScope(t *testing.T, seed string) accountingscope.AccountingScope {
	t.Helper()
	ids := []uuid.UUID{
		uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd618"),
		uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd619"),
		uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd620"),
		uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd621"),
	}
	if seed == "other" {
		ids[3] = uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd622")
	}
	scope, err := accountingscope.New(ids[0], ids[1], ids[2], ids[3], "USD")
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestIdentityEqualityIncludesScopeAndKey(t *testing.T) {
	a, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	differentScope, err := NewIdentity(metadataScope(t, "other"), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	differentKey, err := NewIdentity(metadataScope(t, ""), "payment-124")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Equal(b) || a.Equal(differentScope) || a.Equal(differentKey) {
		t.Fatal("identity equality did not include scope and key")
	}
}

func TestIdentityValidation(t *testing.T) {
	cases := []string{"", "   ", " payment", "payment ", "payment\x00", string([]byte{0xff}), string(make([]byte, maxIdentityTextBytes+1))}
	for _, key := range cases {
		if _, err := NewIdentity(metadataScope(t, ""), key); !errors.Is(err, ErrInvalidIdempotencyKey) {
			t.Errorf("key %q error = %v, want ErrInvalidIdempotencyKey", key, err)
		}
	}
	identity, err := NewIdentity(metadataScope(t, ""), "内部 payment ①")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Key() != "内部 payment ①" {
		t.Fatal("valid key was not preserved")
	}
}

func TestScopeKeyGolden(t *testing.T) {
	identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"tenantId":"0195a91b-20ab-7c15-8aa8-4e111a8bd618","legalEntityId":"0195a91b-20ab-7c15-8aa8-4e111a8bd619","ledgerId":"0195a91b-20ab-7c15-8aa8-4e111a8bd620","accountingBookId":"0195a91b-20ab-7c15-8aa8-4e111a8bd621","functionalCurrency":"USD"}`
	got, err := identity.ScopeKey()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("scope key = %s, want %s", got, want)
	}
}

func TestResultStates(t *testing.T) {
	for _, value := range []ResultState{StateInProgress, StateEstablished, StateFailed} {
		if got, err := ParseResultState(string(value)); err != nil || got != value {
			t.Errorf("state %q = %q, %v", value, got, err)
		}
	}
	if _, err := ParseResultState("unknown"); !errors.Is(err, ErrInvalidState) {
		t.Fatal(err)
	}
}

func TestCommandResultMetadataValidatesAndCopies(t *testing.T) {
	identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	status := 202
	body := []byte(`{"reference":"process-123"}`)
	aggregateID := uuid.New()
	processID := uuid.New()
	metadata, err := NewCommandResultMetadata(identity, "sha256:abc", "payments.submit", StateInProgress, &status, body, &aggregateID, &processID)
	if err != nil {
		t.Fatal(err)
	}
	body[0] = '['
	status = 500
	if !reflect.DeepEqual(metadata.ResultBody(), []byte(`{"reference":"process-123"}`)) {
		t.Fatal("result body was not copied")
	}
	if *metadata.ResultStatus() != 202 {
		t.Fatal("result status was not copied")
	}
	if metadata.OperationID() != "payments.submit" || metadata.State() != StateInProgress {
		t.Fatal("metadata accessors returned incorrect values")
	}
}

func TestCommandResultMetadataValidation(t *testing.T) {
	identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name        string
		fingerprint Fingerprint
		operation   string
		state       ResultState
		body        []byte
		want        error
	}{
		{"missing fingerprint", "", "payments.submit", StateEstablished, nil, ErrInvalidResultMetadata},
		{"invalid operation", "sha256:abc", " ", StateEstablished, nil, ErrInvalidOperationID},
		{"invalid body", "sha256:abc", "payments.submit", StateEstablished, []byte(`{"broken"`), ErrInvalidResultBody},
		{"nil aggregate", "sha256:abc", "payments.submit", StateEstablished, nil, ErrInvalidResultMetadata},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var aggregate *uuid.UUID
			if tc.name == "nil aggregate" {
				value := uuid.Nil
				aggregate = &value
			}
			errWant := tc.want
			if tc.name == "invalid body" {
				metadata, err := NewCommandResultMetadata(identity, tc.fingerprint, tc.operation, tc.state, nil, tc.body, aggregate, nil)
				if !errors.Is(err, ErrInvalidResultBody) || !errors.Is(err, ErrInvalidJSON) {
					t.Fatalf("error = %v, want ErrInvalidResultBody and ErrInvalidJSON", err)
				}
				if !reflect.DeepEqual(metadata, CommandResultMetadata{}) {
					t.Fatal("invalid result body returned metadata")
				}
				return
			}
			if _, err := NewCommandResultMetadata(identity, tc.fingerprint, tc.operation, tc.state, nil, tc.body, aggregate, nil); !errors.Is(err, errWant) {
				t.Fatalf("error = %v, want %v", err, errWant)
			}
		})
	}
}
