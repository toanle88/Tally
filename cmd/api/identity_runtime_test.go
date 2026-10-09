package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/coa"
	"github.com/toanle88/Tally/internal/identity"
)

func TestPostgresCoaAuditWriterPreservesAggregateIdentityPrecedence(t *testing.T) {
	var records []identity.AuditRecord
	writer := postgresCoaAuditWriter(func(_ context.Context, _ pgx.Tx, record identity.AuditRecord) (uuid.UUID, error) {
		records = append(records, record)
		return uuid.New(), nil
	})
	definitionID := uuid.New()
	valueID := uuid.New()
	requestID := uuid.New()

	if _, err := writer(context.Background(), nil, coa.AuditRecord{SegmentDefinitionID: definitionID, SegmentValueID: valueID}); err != nil {
		t.Fatal(err)
	}
	if records[0].UserID != valueID {
		t.Fatalf("value audit user ID = %s, want %s", records[0].UserID, valueID)
	}

	if _, err := writer(context.Background(), nil, coa.AuditRecord{SegmentChangeRequestID: requestID, SegmentDefinitionID: definitionID, SegmentValueID: valueID}); err != nil {
		t.Fatal(err)
	}
	if records[1].UserID != requestID {
		t.Fatalf("change-request audit user ID = %s, want %s", records[1].UserID, requestID)
	}
}
