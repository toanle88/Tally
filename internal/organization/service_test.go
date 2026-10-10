package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestLegalEntityServiceIsIdempotentAndChecksVersion(t *testing.T) {
	repository := NewMemoryLegalEntityRepository()
	audit := &MemoryAuditRecorder{}
	scopeID := uuid.New()
	service, err := NewLegalEntityService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: LegalEntityManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, AllowAllApprovalValidator{}, audit, func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	command := legalEntityTestCommand(scopeID, LegalEntityActionCreate)
	actor := Actor{UserID: uuid.New(), SubjectReference: "actor"}
	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.Execute(context.Background(), actor, command)
	if err != nil || !replay.Replayed {
		t.Fatalf("replay = %#v, err = %v", replay, err)
	}
	if len(audit.Records) != 1 {
		t.Fatalf("audit records = %d, want 1", len(audit.Records))
	}
	reference, err := service.GetReference(context.Background(), first.LegalEntity.ID, scopeID)
	if err != nil || reference.ID != first.LegalEntity.ID || reference.ScopeID != scopeID {
		t.Fatalf("legal-entity reference = %#v, err = %v", reference, err)
	}
	if _, err := service.GetReference(context.Background(), first.LegalEntity.ID, uuid.New()); !errors.Is(err, ErrLegalEntityNotFound) {
		t.Fatalf("cross-scope reference error = %v, want not found", err)
	}

	maintain := legalEntityTestCommand(scopeID, LegalEntityActionMaintain)
	maintain.LegalEntityID = first.LegalEntity.ID
	version := aggregateversion.AggregateVersion(1)
	maintain.ExpectedVersion = &version
	maintain.IdempotencyKey = "maintain-1"
	if _, err := service.Execute(context.Background(), actor, maintain); err != nil {
		t.Fatal(err)
	}
	maintain.IdempotencyKey = "maintain-stale"
	if _, err := service.Execute(context.Background(), actor, maintain); !errors.Is(err, ErrLegalEntityVersionConflict) {
		t.Fatalf("stale error = %v, want version conflict", err)
	}
}

func TestLegalEntityServiceRejectsChangedIdempotencyPayload(t *testing.T) {
	repository := NewMemoryLegalEntityRepository()
	service, err := NewLegalEntityService(repository, MemoryAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: LegalEntityManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}}, AllowAllApprovalValidator{}, &MemoryAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	command := legalEntityTestCommand(uuid.New(), LegalEntityActionCreate)
	actor := Actor{UserID: uuid.New(), SubjectReference: "actor"}
	if _, err := service.Execute(context.Background(), actor, command); err != nil {
		t.Fatal(err)
	}
	command.LegalName = "Changed name"
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrLegalEntityIdempotencyConflict) {
		t.Fatalf("changed payload error = %v, want idempotency conflict", err)
	}
}

func legalEntityTestCommand(scopeID uuid.UUID, action string) LegalEntityCommand {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return LegalEntityCommand{Action: action, ScopeID: scopeID, LegalName: "Acme Vietnam Co., Ltd.", FunctionalCurrency: "VND", PresentationCurrency: "VND", EffectiveFrom: from, Registrations: []LegalEntityRegistration{{Type: "company", Identifier: "123456789", Jurisdiction: "VN", EffectiveFrom: from}}, Addresses: []LegalEntityAddress{{Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN", EffectiveFrom: from}}, OwnershipInterests: []LegalEntityOwnershipInterest{{OwnerReference: "owner-1", Percentage: "100", EffectiveFrom: from}}, IdempotencyKey: action + "-1", CorrelationID: "correlation", CausationID: "causation"}
}
