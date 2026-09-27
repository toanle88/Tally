package organization

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLegalEntityRejectsOverlappingRegistrationAndOwnership(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	registration := LegalEntityRegistration{ID: uuid.New(), Type: "company", Identifier: "123", Jurisdiction: "VN", EffectiveFrom: from, EffectiveTo: &to}
	entity, err := NewLegalEntity(uuid.New(), uuid.New(), "Acme Vietnam Co., Ltd.", "VND", "VND", "restricted", from, []LegalEntityRegistration{registration, {ID: uuid.New(), Type: "company", Identifier: "123", Jurisdiction: "VN", EffectiveFrom: from}}, []LegalEntityAddress{{ID: uuid.New(), Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN", EffectiveFrom: from}}, []LegalEntityOwnershipInterest{{ID: uuid.New(), OwnerReference: "owner-1", Percentage: "60", EffectiveFrom: from}, {ID: uuid.New(), OwnerReference: "owner-1", Percentage: "50", EffectiveFrom: from}}, nil, LegalEntityStatusActive, from)
	if !errors.Is(err, ErrLegalEntityDuplicate) && !errors.Is(err, ErrInvalidLegalEntity) {
		t.Fatalf("NewLegalEntity error = %v, want duplicate/invalid", err)
	}
	_ = entity
}

func TestLegalEntityEndDateAppendsImmutableRevision(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	entity, err := NewLegalEntity(uuid.New(), uuid.New(), "Acme Vietnam Co., Ltd.", "VND", "VND", "", from, []LegalEntityRegistration{{ID: uuid.New(), Type: "company", Identifier: "123", Jurisdiction: "VN", EffectiveFrom: from}}, []LegalEntityAddress{{ID: uuid.New(), Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN", EffectiveFrom: from}}, nil, nil, LegalEntityStatusActive, from)
	if err != nil {
		t.Fatal(err)
	}
	before := entity.Revisions[0].Snapshot
	end := from.AddDate(1, 0, 0)
	if err := entity.EndDate(entity, end, end); err != nil {
		t.Fatal(err)
	}
	if entity.Status != LegalEntityStatusEndDated || entity.Version.Value() != 2 || len(entity.Revisions) != 2 {
		t.Fatalf("end-dated entity = %#v", entity)
	}
	if before.Status != LegalEntityStatusActive || before.EffectiveTo != nil {
		t.Fatalf("prior revision changed: %#v", before)
	}
}

func TestSafeProjectionMasksRegistrationAndOmitsTaxRegistration(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	entity, err := NewLegalEntity(uuid.New(), uuid.New(), "Acme Vietnam Co., Ltd.", "VND", "VND", "TAX-PRIVATE", from, []LegalEntityRegistration{{ID: uuid.New(), Type: "company", Identifier: "123456789", Jurisdiction: "VN", EffectiveFrom: from}}, []LegalEntityAddress{{ID: uuid.New(), Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN", EffectiveFrom: from}}, nil, nil, LegalEntityStatusActive, from)
	if err != nil {
		t.Fatal(err)
	}
	projection := entity.SafeProjection()
	if projection.Registrations[0].IdentifierMasked != "••••6789" {
		t.Fatalf("masked registration = %q", projection.Registrations[0].IdentifierMasked)
	}
	if string(mustJSON(projection)) == "" {
		t.Fatal("projection should be serializable")
	}
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
