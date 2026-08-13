package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fingerprintVector struct{ Name, InputHex, CanonicalHex, HashInputHex, Fingerprint string }

func TestFingerprintGoldenVectors(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "fingerprint_vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vectors []fingerprintVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			input, err := hex.DecodeString(v.InputHex)
			if err != nil {
				t.Fatalf("invalid inputHex: %v", err)
			}
			wantCanonical, err := hex.DecodeString(v.CanonicalHex)
			if err != nil {
				t.Fatalf("invalid canonicalHex: %v", err)
			}
			wantHash, err := hex.DecodeString(v.HashInputHex)
			if err != nil {
				t.Fatalf("invalid hashInputHex: %v", err)
			}
			canonical, err := Canonicalize(input)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(canonical, wantCanonical) {
				t.Fatalf("canonical %x, want %x", canonical, wantCanonical)
			}
			got, err := ComputeFingerprint(input)
			if err != nil || got != Fingerprint(v.Fingerprint) {
				t.Fatalf("fingerprint %q, %v; want %q", got, err, v.Fingerprint)
			}
			hashInput := append(append([]byte(fingerprintPrefix), 0), canonical...)
			if !bytes.Equal(hashInput, wantHash) {
				t.Fatalf("hash input %x, want %x", hashInput, wantHash)
			}
			sum := sha256.Sum256(wantHash)
			if "sha256:"+hex.EncodeToString(sum[:]) != v.Fingerprint {
				t.Fatal("golden hash mismatch")
			}
		})
	}
}

func TestFingerprintRejectedVectors(t *testing.T) {
	tests := []struct {
		name, input string
		want        error
	}{
		{"duplicate decoded key", `{"a":1,"\u0061":2}`, ErrDuplicateObjectKey},
		{"unsupported fraction", `{"a":1.0}`, ErrUnsupportedNumber},
		{"lone surrogate", `{"a":"\uD800"}`, ErrInvalidJSON},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Canonicalize([]byte(tc.input)); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCanonicalizeEquivalentInputs(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"whitespace", `{ "b" : 2, "a" : 1 }`, `{"a":1,"b":2}`},
		{"escaped key", `{"\u0061":1}`, `{"a":1}`},
		{"unicode byte ordering", `{"é":1,"a":2}`, `{"a":2,"é":1}`},
		{"surrogate pair", `{"x":"\uD83D\uDE00"}`, `{"x":"😀"}`},
		{"escaped string", `{"x":"\u0061"}`, `{"x":"a"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Canonicalize([]byte(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestComputeFingerprintStableAndMaterialChangesDiffer(t *testing.T) {
	a, err := ComputeFingerprint([]byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := ComputeFingerprint([]byte("{\n \"a\": 1, \"b\": 2\n}"))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("equivalent inputs differ: %s != %s", a, b)
	}
	c, err := ComputeFingerprint([]byte(`{"a":1,"b":3}`))
	if err != nil {
		t.Fatal(err)
	}
	if a == c {
		t.Fatal("material change produced the same fingerprint")
	}
	if len(a) != len("sha256:")+64 {
		t.Fatalf("unexpected fingerprint format: %s", a)
	}
	if a != "sha256:a53814e80c186602b65d3e2b1d651d94330b3375de4226338c270a9f6e5045ad" {
		t.Fatalf("golden fingerprint changed: %s", a)
	}
}

func TestCanonicalizeRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name, input string
		want        error
	}{
		{"root", `[]`, ErrInvalidRoot},
		{"duplicate", `{"a":1,"\u0061":2}`, ErrDuplicateObjectKey},
		{"fraction", `{"a":1.0}`, ErrUnsupportedNumber},
		{"exponent", `{"a":1e2}`, ErrUnsupportedNumber},
		{"negative zero", `{"a":-0}`, ErrUnsupportedNumber},
		{"lone surrogate", `{"a":"\uD800"}`, ErrInvalidJSON},
		{"malformed", `{"a":}`, ErrInvalidJSON},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Canonicalize([]byte(tc.input))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCanonicalizeDepthLimit(t *testing.T) {
	for _, tc := range []struct{ name, open, close string }{{"arrays", "[", "]"}, {"objects", `{"x":`, "}"}} {
		t.Run(tc.name, func(t *testing.T) {
			build := func(count int) []byte {
				input := []byte(`{"x":`)
				if tc.name == "objects" {
					count--
				}
				for i := 0; i < count; i++ {
					input = append(input, tc.open...)
				}
				input = append(input, '1')
				for i := 0; i < count; i++ {
					input = append(input, tc.close...)
				}
				return append(input, '}')
			}
			validCount := maxDepth
			if tc.name == "arrays" {
				validCount--
			}
			valid := build(validCount)
			if _, err := Canonicalize(valid); err != nil {
				t.Fatalf("boundary rejected: %v", err)
			}
			tooDeep := build(maxDepth + 1)
			if _, err := Canonicalize(tooDeep); !errors.Is(err, ErrMaxDepth) {
				t.Fatalf("got %v, want %v", err, ErrMaxDepth)
			}
		})
	}
}
