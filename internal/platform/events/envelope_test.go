package events

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testInput(t *testing.T) EnvelopeInput {
	t.Helper()
	data := []byte(`{"b":2,"a":1}`)
	fp, _ := ComputePayloadFingerprint(data)
	return EnvelopeInput{"0195a91b-20ab-7c15-8aa8-4e111a8bd622", "JournalEntryPosted", 1, time.Date(2026, 7, 24, 8, 0, 0, 0, time.FixedZone("x", 3600)), "gl", "0195a91b-20ab-7c15-8aa8-4e111a8bd620", 8, "0195a91b-20ab-7c15-8aa8-4e111a8bd619", "0195a91b-20ab-7c15-8aa8-4e111a8bd621", "0195a91b-20ab-7c15-8aa8-4e111a8bd618", Internal, fp, data}
}
func TestEnvelopeCanonicalRoundTrip(t *testing.T) {
	e, err := NewEnvelope(testInput(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"occurredAt":"2026-07-24T07:00:00Z"`)) {
		t.Fatalf("%s", b)
	}
	var got Envelope
	if err = json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.MessageID() != e.MessageID() || got.PayloadFingerprint() != e.PayloadFingerprint() || !bytes.Equal(got.Data(), []byte(`{"a":1,"b":2}`)) {
		t.Fatal("round trip changed envelope")
	}
}
func TestEnvelopeValidationCodes(t *testing.T) {
	base := testInput(t)
	cases := []struct {
		name   string
		mutate func(*EnvelopeInput)
		code   ValidationCode
	}{{"version", func(x *EnvelopeInput) { x.EventVersion = 2 }, CodeUnsupportedEventVersion}, {"scope", func(x *EnvelopeInput) { x.AccountingScopeID = "bad" }, CodeInvalidScopeIdentifier}, {"classification", func(x *EnvelopeInput) { x.DataClassification = "secret" }, CodeInvalidClassification}, {"aggregate", func(x *EnvelopeInput) { x.AggregateVersion = 0 }, CodeInvalidAggregateVersion}, {"fingerprint", func(x *EnvelopeInput) { x.Data = []byte(`{"a":3}`) }, CodeFingerprintMismatch}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := base
			tc.mutate(&x)
			r := Envelope{input: x}.Validate()
			if r.Code() != tc.code {
				t.Fatalf("got %s want %s", r.Code(), tc.code)
			}
		})
	}
}
func TestCanonicalPayloadRejectsNonObjectAndDuplicate(t *testing.T) {
	for _, b := range [][]byte{[]byte(`[1]`), []byte(`{"a":1,"a":2}`), []byte(`{"a":1} trailing`)} {
		if _, err := CanonicalizePayload(b); err == nil {
			t.Fatalf("accepted %s", b)
		}
	}
}

func TestEnvelopeRejectsMalformedTrailingJSON(t *testing.T) {
	e, err := NewEnvelope(testInput(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{" invalid", " {"} {
		var got Envelope
		if err := json.Unmarshal(append(b, suffix...), &got); err == nil {
			t.Fatalf("accepted malformed trailing input %q", suffix)
		}
	}
}

func TestEnvelopeRejectsZeroOccurredAt(t *testing.T) {
	in := testInput(t)
	in.OccurredAt = time.Time{}
	r := (Envelope{input: in}).Validate()
	if r.Code() != CodeInvalidOccurredAt {
		t.Fatalf("got %s, want %s", r.Code(), CodeInvalidOccurredAt)
	}
}

func TestEnvelopeGoldenVectors(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "envelope_vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		Name           string          `json:"name"`
		Envelope       json.RawMessage `json:"envelope"`
		ValidationCode ValidationCode  `json:"validationCode"`
	}
	if err := json.Unmarshal(b, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			if vector.ValidationCode != "" {
				in := testInput(t)
				switch vector.ValidationCode {
				case CodeMalformedIdentity:
					in.MessageID = "bad"
				case CodeInvalidAggregateVersion:
					in.AggregateVersion = 0
				case CodeInvalidScopeIdentifier:
					in.AccountingScopeID = "bad"
				case CodeInvalidClassification:
					in.DataClassification = "secret"
				case CodeUnsupportedEventVersion:
					in.EventVersion = 2
				case CodeInvalidPayload:
					in.Data = []byte(`[1]`)
				case CodeFingerprintMismatch:
					in.Data = []byte(`{"a":3}`)
				case CodeInvalidOccurredAt:
					in.OccurredAt = time.Time{}
				}
				if got := (Envelope{input: in}).Validate().Code(); got != vector.ValidationCode {
					t.Fatalf("got %s, want %s", got, vector.ValidationCode)
				}
				return
			}
			var envelope Envelope
			if err := json.Unmarshal(vector.Envelope, &envelope); err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(vector.Envelope) {
				t.Fatalf("got %s, want %s", got, vector.Envelope)
			}
		})
	}
}
