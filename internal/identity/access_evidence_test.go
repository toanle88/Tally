package identity

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestAccessEvidenceRecordsOnlySafeMetadataAndUsesActorFingerprint(t *testing.T) {
	subject, err := NewAuthenticationSubject("oid-fixture", "tenant-fixture", "subject-fixture")
	if err != nil {
		t.Fatal(err)
	}
	actor := ApplicationActor{UserID: uuid.New(), Subject: subject}
	recorder := &MemoryAccessObservationRecorder{}
	service, err := NewAccessEvidenceService(recorder)
	if err != nil {
		t.Fatal(err)
	}

	observation := AccessObservation{
		ActorUserID:            uuid.New(),
		ActorAuthenticationRef: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Operation:              AccessObservationReveal,
		TargetReference:        "record-fixture-42",
		EvidenceReference:      "evidence-fixture-42",
		ScopeReference:         "scope-vietnam-statutory",
		Purpose:                "review-authorized-evidence",
		FilterReference:        "filter-fixture-1",
		DataClassification:     AccessObservationClassificationHighlyRestricted,
		Permission:             "finance.iam.evidence.reveal",
		Outcome:                AuthorizationAllowed,
		PolicyReference:        "policy-fixture-7",
		PolicyVersion:          "policy-v7",
		DecisionReference:      uuid.New(),
		CorrelationReference:   "correlation-fixture-1",
		CausationReference:     "causation-fixture-1",
		BeforeFingerprint:      "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		AfterFingerprint:       "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}

	if _, err := service.RecordAccess(context.Background(), actor, observation); err != nil {
		t.Fatal(err)
	}
	records := recorder.Records()
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].ActorUserID != actor.UserID || records[0].ActorAuthenticationRef != fingerprintSubject(actor.Subject) {
		t.Fatalf("actor metadata was not normalized: %+v", records[0])
	}
	if strings.Contains(strings.ToLower(records[0].Purpose), "secret") {
		t.Fatal("sensitive value reached observation")
	}
}

func TestAccessEvidenceRejectsSensitiveReferencesAndFailsClosed(t *testing.T) {
	base := AccessObservation{
		ActorUserID:            uuid.New(),
		ActorAuthenticationRef: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Operation:              AccessObservationExport,
		TargetReference:        "record-fixture-42",
		EvidenceReference:      "evidence-fixture-42",
		ScopeReference:         "scope-fixture",
		Purpose:                "review-authorized-evidence",
		DataClassification:     AccessObservationClassificationConfidential,
		Permission:             "finance.iam.evidence.export",
		Outcome:                AuthorizationDenied,
		DecisionReference:      uuid.New(),
	}
	for _, field := range []struct {
		name  string
		apply func(*AccessObservation)
	}{
		{name: "target", apply: func(value *AccessObservation) { value.TargetReference = "bank-account-value" }},
		{name: "purpose", apply: func(value *AccessObservation) { value.Purpose = "export-secret" }},
		{name: "policy payload", apply: func(value *AccessObservation) { value.PolicyReference = "raw-policy-payload" }},
	} {
		candidate := base
		field.apply(&candidate)
		if err := candidate.Validate(); !errors.Is(err, ErrInvalidAccessObservation) {
			t.Errorf("%s error = %v, want invalid observation", field.name, err)
		}
	}

	actorSubject, _ := NewAuthenticationSubject("oid-fixture", "tenant-fixture", "subject-fixture")
	service, err := NewAccessEvidenceService(&MemoryAccessObservationRecorder{Err: errors.New("audit dependency down")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.RecordAccess(context.Background(), ApplicationActor{UserID: uuid.New(), Subject: actorSubject}, base)
	if !errors.Is(err, ErrAccessEvidenceUnavailable) {
		t.Fatalf("error = %v, want fail-closed evidence error", err)
	}
	service, err = NewAccessEvidenceService(nilAccessObservationRecorder{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.RecordAccess(context.Background(), ApplicationActor{UserID: uuid.New(), Subject: actorSubject}, base)
	if !errors.Is(err, ErrAccessEvidenceUnavailable) {
		t.Fatalf("nil audit reference error = %v, want fail-closed evidence error", err)
	}
}

type nilAccessObservationRecorder struct{}

func (nilAccessObservationRecorder) RecordAccessObservation(context.Context, AccessObservation) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func TestSafeDecisionExplanationsDoNotExposeHumanReasons(t *testing.T) {
	decision := AuthorizationDecision{
		Outcome: AuthorizationStale, Permission: "finance.iam.evidence.view", PolicyReference: "policy-fixture",
		PolicyVersion: "v7", DecisionReference: uuid.New(), ReasonCode: AuthorizationReasonPolicyVersionStale,
		Reason: "raw policy payload must never be projected", ApplicableDimensions: []string{AuthorizationDimensionScope},
	}
	explanation := ExplainAuthorizationDecision(decision)
	if explanation.Outcome != AuthorizationStale || explanation.NextAction != "Refresh policy state and retry." {
		t.Fatalf("unexpected authorization explanation: %+v", explanation)
	}
	if explanation.DecisionReference != decision.DecisionReference || len(explanation.ApplicableDimensions) != 1 {
		t.Fatalf("safe references missing: %+v", explanation)
	}
	if strings.Contains(explanation.NextAction, "raw policy") || strings.Contains(explanation.NextAction, decision.Reason) {
		t.Fatal("human reason was projected")
	}
	explanation.ApplicableDimensions[0] = "changed"
	if decision.ApplicableDimensions[0] == "changed" {
		t.Fatal("explanation shares mutable dimension slice")
	}
	unsafe := ExplainAuthorizationDecision(AuthorizationDecision{
		Outcome: AuthorizationDenied, Permission: "secret-value", PolicyReference: "raw-policy-payload",
		PolicyVersion: "policy v7", ReasonCode: "private-reason", ApplicableDimensions: []string{"secret-dimension"}, DecisionReference: uuid.New(),
	})
	if unsafe.Permission != "" || unsafe.PolicyReference != "" || unsafe.PolicyVersion != "" || unsafe.ReasonCode != "" || len(unsafe.ApplicableDimensions) != 0 {
		t.Fatalf("unsafe decision data was projected: %+v", unsafe)
	}

	segregation := ExplainSegregationDecision(SegregationDecision{
		Allowed: false, Outcome: AuthorizationDenied, RuleReference: uuid.New(), RuleCode: SegregationRulePaymentBatch,
		RuleVersion: aggregateversion.Initial(), PolicyVersion: "policy-v7", DecisionReference: uuid.New(),
		ReasonCode: SegregationReasonConflict, Reason: "private conflict details", Resolution: "private resolution details",
	})
	if segregation.NextAction != "Use an independent actor or request an approved exception." || segregation.RuleCode != SegregationRulePaymentBatch {
		t.Fatalf("unexpected segregation explanation: %+v", segregation)
	}
	unsafeSegregation := ExplainSegregationDecision(SegregationDecision{Outcome: AuthorizationDenied, RuleCode: "raw-policy-payload", PolicyVersion: "policy v7", ReasonCode: "private-reason", DecisionReference: uuid.New()})
	if unsafeSegregation.RuleCode != "" || unsafeSegregation.PolicyVersion != "" || unsafeSegregation.ReasonCode != "" {
		t.Fatalf("unsafe segregation data was projected: %+v", unsafeSegregation)
	}
	for _, outcome := range []AuthorizationOutcome{AuthorizationAllowed, AuthorizationDenied, AuthorizationExpired, AuthorizationStale, AuthorizationUnavailable} {
		projected := ExplainAuthorizationDecision(AuthorizationDecision{Outcome: outcome, ReasonCode: AuthorizationReasonDefaultDeny, DecisionReference: uuid.New()})
		if projected.Outcome != outcome || projected.NextAction == "" {
			t.Fatalf("outcome %q was not safely projected: %+v", outcome, projected)
		}
	}
}
