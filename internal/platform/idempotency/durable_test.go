package idempotency

import (
	"errors"
	"testing"
	"time"
)

func TestIdempotencyPolicyRequiresPositiveLeaseBeforeRetention(t *testing.T) {
	valid := IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: time.Minute}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid policy error = %v", err)
	}
	for _, policy := range []IdempotencyPolicy{
		{},
		{RecordTTL: time.Minute, LeaseTTL: time.Minute},
		{RecordTTL: time.Minute, LeaseTTL: 2 * time.Minute},
		{RecordTTL: time.Minute, LeaseTTL: 0},
	} {
		if err := policy.validate(); !errors.Is(err, ErrInvalidIdempotencyPolicy) {
			t.Fatalf("policy %#v error = %v, want invalid policy", policy, err)
		}
	}
}
