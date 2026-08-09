package aggregateversion

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func TestInitial(t *testing.T) {
	if got := Initial(); got.Value() != 1 {
		t.Fatalf("initial version = %d, want 1", got.Value())
	}
}

func TestFromInt64(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  AggregateVersion
		err   error
	}{
		{name: "initial", input: 1, want: Initial()},
		{name: "interior", input: 7, want: AggregateVersion(7)},
		{name: "maximum", input: math.MaxInt64, want: AggregateVersion(math.MaxInt64)},
		{name: "zero", input: 0, err: ErrInvalidVersion},
		{name: "negative", input: -1, err: ErrInvalidVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromInt64(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("error = %v, want %v", err, tt.err)
			}
			if err == nil && got != tt.want {
				t.Fatalf("version = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	version := AggregateVersion(7)
	if !version.Matches(AggregateVersion(7)) {
		t.Fatal("matching version reported stale")
	}
	if version.Matches(AggregateVersion(6)) {
		t.Fatal("stale version reported matching")
	}
	if AggregateVersion(0).Matches(version) {
		t.Fatal("invalid version reported matching")
	}
}

func TestAdvance(t *testing.T) {
	got, err := AggregateVersion(7).Advance()
	if err != nil || got != AggregateVersion(8) {
		t.Fatalf("advance = %d, %v; want 8, nil", got, err)
	}
	if _, err := AggregateVersion(0).Advance(); !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("invalid advance error = %v, want %v", err, ErrInvalidVersion)
	}
	if _, err := AggregateVersion(math.MaxInt64).Advance(); !errors.Is(err, ErrVersionOverflow) {
		t.Fatalf("overflow error = %v, want %v", err, ErrVersionOverflow)
	}
}

func TestInt64RoundTrip(t *testing.T) {
	for _, input := range []int64{1, 7, math.MaxInt64} {
		version, err := FromInt64(input)
		if err != nil {
			t.Fatalf("FromInt64(%d): %v", input, err)
		}
		if got := version.Value(); got != input {
			t.Fatalf("Value() = %d, want %d", got, input)
		}
		again, err := FromInt64(version.Value())
		if err != nil || again != version {
			t.Fatalf("explicit conversion round-trip = %d, %v; want %d, nil", again, err, version)
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	for _, input := range []AggregateVersion{1, 7, AggregateVersion(math.MaxInt64)} {
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("marshal %d: %v", input, err)
		}
		var got AggregateVersion
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", data, err)
		}
		if got != input {
			t.Fatalf("round-trip = %d, want %d", got, input)
		}
	}
}

func TestJSONRejectsInvalidValues(t *testing.T) {
	var version AggregateVersion
	for _, input := range []string{"0", "-1", "1.5", "\"1\"", "null", "9223372036854775808"} {
		if err := json.Unmarshal([]byte(input), &version); !errors.Is(err, ErrInvalidJSON) {
			t.Errorf("unmarshal %q error = %v, want %v", input, err, ErrInvalidJSON)
		}
	}
	if err := json.Unmarshal([]byte("1 2"), &version); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
	if err := json.Unmarshal(nil, &version); err == nil {
		t.Fatal("empty JSON was accepted")
	}
	for _, input := range []string{"", "1 2"} {
		if err := version.UnmarshalJSON([]byte(input)); !errors.Is(err, ErrInvalidJSON) {
			t.Errorf("direct unmarshal %q error = %v, want %v", input, err, ErrInvalidJSON)
		}
	}
	if _, err := json.Marshal(AggregateVersion(0)); !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("marshal zero error = %v, want %v", err, ErrInvalidVersion)
	}
}

func TestAggregateVersionIsDistinctFromInt64(t *testing.T) {
	var version AggregateVersion = 7
	var raw int64 = version.Value()
	if AggregateVersion(raw) != version {
		t.Fatal("explicit conversion changed the version")
	}
}
