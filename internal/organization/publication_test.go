package organization

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestMasterDataPublicationPublishesAggregateSpecificEvents(t *testing.T) {
	scopeID := uuid.New()
	actorID := uuid.New()
	for _, testCase := range []struct {
		name      string
		aggregate MasterDataAggregateType
		eventType string
		effective bool
	}{
		{name: "legal entity", aggregate: MasterDataAggregateLegalEntity, eventType: LegalEntityPublishedEvent, effective: true},
		{name: "party", aggregate: MasterDataAggregateParty, eventType: PartyPublishedEvent},
		{name: "customer profile", aggregate: MasterDataAggregateCustomerProfile, eventType: CustomerProfilePublishedEvent, effective: true},
		{name: "vendor profile", aggregate: MasterDataAggregateVendorProfile, eventType: VendorProfilePublishedEvent, effective: true},
		{name: "fiscal calendar", aggregate: MasterDataAggregateFiscalCalendar, eventType: FiscalCalendarPublishedEvent, effective: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			aggregateID := uuid.New()
			candidate := publicationTestCandidate(testCase.aggregate, aggregateID, scopeID, false)
			repository := NewMemoryMasterDataPublicationRepository(candidate)
			audit := &MemoryMasterDataPublicationAuditRecorder{}
			service, err := NewMasterDataPublicationService(repository, MemoryMasterDataPublicationAuthorizer{Decision: publicationTestDecision(scopeID)}, audit, func() time.Time { return time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC) })
			if err != nil {
				t.Fatal(err)
			}
			version := aggregateversion.Initial()
			result, err := service.Execute(context.Background(), Actor{UserID: actorID, SubjectReference: actorID.String()}, MasterDataPublicationCommand{AggregateType: testCase.aggregate, AggregateID: aggregateID, ScopeID: scopeID, ExpectedVersion: &version, IdempotencyKey: "publish-" + string(testCase.aggregate), CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
			if err != nil {
				t.Fatal(err)
			}
			if result.EventType != testCase.eventType || result.Status != MasterDataPublicationStatus || result.DependentAvailability != MasterDataDependentPending {
				t.Fatalf("result = %#v", result)
			}
			if testCase.effective != (result.EffectiveFrom != nil) {
				t.Fatalf("effective date presence = %v, want %v", result.EffectiveFrom != nil, testCase.effective)
			}
			events := repository.PublishedEvents()
			if len(events) != 1 || events[0].EventType() != testCase.eventType || events[0].AggregateVersion() != 1 {
				t.Fatalf("published events = %#v", events)
			}
			var payload struct {
				AggregateType    MasterDataAggregateType `json:"aggregateType"`
				AggregateID      uuid.UUID               `json:"aggregateId"`
				ScopeID          uuid.UUID               `json:"scopeId"`
				AggregateVersion int                     `json:"aggregateVersion"`
				EffectiveFrom    *time.Time              `json:"effectiveFrom"`
				Snapshot         json.RawMessage         `json:"snapshot"`
			}
			if err := json.Unmarshal(events[0].Data(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.AggregateType != testCase.aggregate || payload.AggregateID != aggregateID || payload.ScopeID != scopeID || payload.AggregateVersion != 1 || (testCase.effective != (payload.EffectiveFrom != nil)) || len(payload.Snapshot) == 0 {
				t.Fatalf("publication payload = %#v", payload)
			}
			if events[0].DataClassification() != "internal" {
				t.Fatalf("data classification = %q, want internal", events[0].DataClassification())
			}
			if strings.Contains(string(events[0].Data()), "secret") || strings.Contains(string(events[0].Data()), "taxIdentifier\":\"raw") {
				t.Fatalf("publication payload contains restricted data: %s", events[0].Data())
			}
			if len(audit.Records) != 1 || audit.Records[0].Action != MasterDataPublicationAction {
				t.Fatalf("audit records = %#v", audit.Records)
			}
		})
	}
}

func TestMasterDataPublicationRequiresCurrentApprovalEvidence(t *testing.T) {
	scopeID := uuid.New()
	aggregateID := uuid.New()
	version := aggregateversion.Initial()
	serviceCandidate := publicationTestCandidate(MasterDataAggregateLegalEntity, aggregateID, scopeID, true)
	repository := NewMemoryMasterDataPublicationRepository(serviceCandidate)
	service, err := NewMasterDataPublicationService(repository, MemoryMasterDataPublicationAuthorizer{Decision: publicationTestDecision(scopeID)}, &MemoryMasterDataPublicationAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, MasterDataPublicationCommand{AggregateType: MasterDataAggregateLegalEntity, AggregateID: aggregateID, ScopeID: scopeID, ExpectedVersion: &version, IdempotencyKey: "approval-required", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if !errors.Is(err, ErrMasterDataPublicationApprovalRequired) {
		t.Fatalf("error = %v, want approval required", err)
	}

	stale := serviceCandidate
	stale.Approval = &ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "policy-v1", DecisionVersion: 1, SubjectVersion: 99, CandidateFingerprint: stale.Fingerprint, ApproverUserID: uuid.New()}
	if err := repository.Put(stale); err != nil {
		t.Fatal(err)
	}
	_, err = service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, MasterDataPublicationCommand{AggregateType: MasterDataAggregateLegalEntity, AggregateID: aggregateID, ScopeID: scopeID, ExpectedVersion: &version, IdempotencyKey: "approval-stale", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if !errors.Is(err, ErrMasterDataPublicationApprovalStale) {
		t.Fatalf("error = %v, want stale approval", err)
	}
}

func TestMasterDataPublicationIsIdempotentAndDoesNotDuplicateEvents(t *testing.T) {
	scopeID := uuid.New()
	aggregateID := uuid.New()
	candidate := publicationTestCandidate(MasterDataAggregateParty, aggregateID, scopeID, false)
	repository := NewMemoryMasterDataPublicationRepository(candidate)
	service, err := NewMasterDataPublicationService(repository, MemoryMasterDataPublicationAuthorizer{Decision: publicationTestDecision(scopeID)}, &MemoryMasterDataPublicationAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	version := aggregateversion.Initial()
	command := MasterDataPublicationCommand{AggregateType: MasterDataAggregateParty, AggregateID: aggregateID, ScopeID: scopeID, ExpectedVersion: &version, IdempotencyKey: "replay-safe", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	actor := Actor{UserID: uuid.New(), SubjectReference: "actor"}
	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if second.PublicationID != first.PublicationID || !second.Replayed || len(repository.PublishedEvents()) != 1 {
		t.Fatalf("first=%#v second=%#v events=%d", first, second, len(repository.PublishedEvents()))
	}
}

func TestMasterDataPublicationRejectsSupersededAndUnavailableOutcomes(t *testing.T) {
	scopeID := uuid.New()
	aggregateID := uuid.New()
	candidate := publicationTestCandidate(MasterDataAggregateParty, aggregateID, scopeID, false)
	version := aggregateversion.Initial()
	deniedByUnavailablePolicy, err := NewMasterDataPublicationService(NewMemoryMasterDataPublicationRepository(candidate), MemoryMasterDataPublicationAuthorizer{Decision: AuthorizationDecision{Outcome: "unavailable"}}, &MemoryMasterDataPublicationAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = deniedByUnavailablePolicy.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, MasterDataPublicationCommand{AggregateType: MasterDataAggregateParty, AggregateID: aggregateID, ScopeID: scopeID, ExpectedVersion: &version, IdempotencyKey: "policy-unavailable", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if !errors.Is(err, ErrMasterDataPublicationAuthorizationUnavailable) {
		t.Fatalf("unavailable policy error = %v", err)
	}

	superseded, err := NewMasterDataPublicationService(NewMemoryMasterDataPublicationRepository(candidate), MemoryMasterDataPublicationAuthorizer{Decision: publicationTestDecision(scopeID)}, &MemoryMasterDataPublicationAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	newerVersion, err := aggregateversion.FromInt64(2)
	if err != nil {
		t.Fatal(err)
	}
	_, err = superseded.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, MasterDataPublicationCommand{AggregateType: MasterDataAggregateParty, AggregateID: aggregateID, ScopeID: scopeID, ExpectedVersion: &newerVersion, IdempotencyKey: "version-conflict", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if !errors.Is(err, ErrMasterDataPublicationVersionConflict) {
		t.Fatalf("superseded version error = %v", err)
	}
}

func publicationTestCandidate(aggregateType MasterDataAggregateType, aggregateID, scopeID uuid.UUID, draft bool) MasterDataPublicationCandidate {
	status := "active"
	if draft {
		status = "draft"
	}
	candidate := MasterDataPublicationCandidate{AggregateType: aggregateType, AggregateID: aggregateID, ScopeID: scopeID, Status: status, Version: aggregateversion.Initial(), RevisionNumber: 1, Fingerprint: "sha256:candidate", Payload: []byte(`{"safe":"value"}`)}
	if aggregateType != MasterDataAggregateParty {
		effectiveFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		candidate.EffectiveFrom = &effectiveFrom
	}
	if draft {
		candidate.Approval = nil
	}
	return candidate
}

func publicationTestDecision(scopeID uuid.UUID) AuthorizationDecision {
	return AuthorizationDecision{Allowed: true, Permission: MasterDataPublicationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}
}
