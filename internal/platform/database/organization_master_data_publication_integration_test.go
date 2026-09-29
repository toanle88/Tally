//go:build integration

package database

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

func TestOrganizationMasterDataPublicationCommitsPublicationAndOutboxAtomically(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID := uuid.New()
	entity, err := organization.NewLegalEntity(uuid.New(), scopeID, "Publication Holdings", "VND", "USD", "raw-tax-value", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), []organization.LegalEntityRegistration{{ID: uuid.New(), Type: "company", Identifier: "123456789", Jurisdiction: "VN", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, []organization.LegalEntityAddress{{ID: uuid.New(), Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, []organization.LegalEntityOwnershipInterest{{ID: uuid.New(), OwnerReference: "owner-1", Percentage: "100", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, nil, organization.LegalEntityStatusActive, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	legalRepository, err := organization.NewPostgresLegalEntityRepository(fixture.pool, func(context.Context, pgx.Tx, organization.AuditRecord) (uuid.UUID, error) { return uuid.New(), nil })
	if err != nil {
		t.Fatal(err)
	}
	if err := legalRepository.CommitLegalEntityMutation(ctx, organization.LegalEntityMutation{After: entity, Audit: organization.AuditRecord{LegalEntityID: entity.ID, ActorUserID: uuid.New(), Action: "publication-source-create", ScopeID: scopeID, Permission: organization.LegalEntityManagementPermission, DecisionReference: uuid.New(), RevisionNumber: entity.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}

	publicationRepository, err := organization.NewPostgresMasterDataPublicationRepository(fixture.pool, func(context.Context, pgx.Tx, organization.MasterDataPublicationAuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := organization.NewMasterDataPublicationServiceWithDurableIdempotency(publicationRepository, organization.MemoryMasterDataPublicationAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.MasterDataPublicationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, &organization.MemoryMasterDataPublicationAuditRecorder{}, time.Now, organization.DurableMasterDataPublicationServiceConfig{Database: fixture.pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: "organization.publish-approved-master-data-changes.v1"})
	if err != nil {
		t.Fatal(err)
	}
	version := aggregateversion.Initial()
	actor := organization.Actor{UserID: uuid.New(), SubjectReference: "publication-actor"}
	correlationID, causationID := uuid.New(), uuid.New()
	command := organization.MasterDataPublicationCommand{AggregateType: organization.MasterDataAggregateLegalEntity, AggregateID: entity.ID, ScopeID: scopeID, ExpectedVersion: &version, IdempotencyKey: "publication-integration-1", CorrelationID: correlationID.String(), CausationID: causationID.String()}
	result, err := service.Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.EventType != organization.LegalEntityPublishedEvent {
		t.Fatalf("event type = %q", result.EventType)
	}
	replayed, err := service.Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.PublicationID != result.PublicationID {
		t.Fatalf("replayed result = %#v", replayed)
	}
	var publicationCount, outboxCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM organization.master_data_publication WHERE aggregate_id=$1 AND aggregate_version=1`, entity.ID).Scan(&publicationCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM integration.outbox WHERE aggregate_id=$1 AND aggregate_version=1 AND event_type=$2`, entity.ID, organization.LegalEntityPublishedEvent).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if publicationCount != 1 || outboxCount != 1 {
		t.Fatalf("publication/outbox counts = %d/%d, want 1/1", publicationCount, outboxCount)
	}
	var storedCorrelationID, storedCausationID uuid.UUID
	if err := fixture.pool.QueryRow(ctx, `SELECT correlation_id, causation_id FROM integration.outbox WHERE outbox_id=$1`, result.MessageID).Scan(&storedCorrelationID, &storedCausationID); err != nil {
		t.Fatal(err)
	}
	if storedCorrelationID != correlationID || storedCausationID != causationID {
		t.Fatalf("outbox correlation/causation = %s/%s, want %s/%s", storedCorrelationID, storedCausationID, correlationID, causationID)
	}
	var payload []byte
	if err := fixture.pool.QueryRow(ctx, `SELECT payload FROM integration.outbox WHERE outbox_id=$1`, result.MessageID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 || containsPublicationSecret(payload) {
		t.Fatalf("outbox payload contains restricted source value: %s", payload)
	}

	duplicateVersion := version
	_, err = service.Execute(ctx, organization.Actor{UserID: uuid.New(), SubjectReference: "second-actor"}, organization.MasterDataPublicationCommand{AggregateType: organization.MasterDataAggregateLegalEntity, AggregateID: entity.ID, ScopeID: scopeID, ExpectedVersion: &duplicateVersion, IdempotencyKey: "publication-integration-duplicate", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if !errors.Is(err, organization.ErrMasterDataPublicationAlreadyPublished) {
		t.Fatalf("duplicate error = %v, want already published", err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM integration.outbox WHERE aggregate_id=$1 AND aggregate_version=1 AND event_type=$2`, entity.ID, organization.LegalEntityPublishedEvent).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 1 {
		t.Fatalf("duplicate outbox count = %d, want 1", outboxCount)
	}

	failedEntity, err := organization.NewLegalEntity(uuid.New(), scopeID, "Publication Failure Holdings", "VND", "USD", "raw-tax-value-2", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), []organization.LegalEntityRegistration{{ID: uuid.New(), Type: "company", Identifier: "987654321", Jurisdiction: "VN", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, []organization.LegalEntityAddress{{ID: uuid.New(), Type: "registered", Line1: "2 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, []organization.LegalEntityOwnershipInterest{{ID: uuid.New(), OwnerReference: "owner-2", Percentage: "100", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}, nil, organization.LegalEntityStatusActive, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := legalRepository.CommitLegalEntityMutation(ctx, organization.LegalEntityMutation{After: failedEntity, Audit: organization.AuditRecord{LegalEntityID: failedEntity.ID, ActorUserID: uuid.New(), Action: "publication-source-create", ScopeID: scopeID, Permission: organization.LegalEntityManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failedEntity.RevisionNumber}}); err != nil {
		t.Fatal(err)
	}
	failingRepository, err := organization.NewPostgresMasterDataPublicationRepository(fixture.pool, func(context.Context, pgx.Tx, organization.MasterDataPublicationAuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit unavailable")
	})
	if err != nil {
		t.Fatal(err)
	}
	failingService, err := organization.NewMasterDataPublicationService(failingRepository, organization.MemoryMasterDataPublicationAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.MasterDataPublicationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, &organization.MemoryMasterDataPublicationAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := failingService.Execute(ctx, organization.Actor{UserID: uuid.New(), SubjectReference: "publication-actor"}, organization.MasterDataPublicationCommand{AggregateType: organization.MasterDataAggregateLegalEntity, AggregateID: failedEntity.ID, ScopeID: scopeID, ExpectedVersion: &version, IdempotencyKey: "publication-integration-audit-failure", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}); err == nil {
		t.Fatal("audit failure returned nil")
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM organization.master_data_publication WHERE aggregate_id=$1`, failedEntity.ID).Scan(&publicationCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM integration.outbox WHERE aggregate_id=$1`, failedEntity.ID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if publicationCount != 0 || outboxCount != 0 {
		t.Fatalf("failed publication left publication/outbox rows = %d/%d", publicationCount, outboxCount)
	}
}

func containsPublicationSecret(payload []byte) bool {
	return bytes.Contains(payload, []byte("raw-tax-value")) || bytes.Contains(payload, []byte("123456789")) || bytes.Contains(payload, []byte("987654321"))
}
