//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/idempotency"
)

func TestIdentitySegregationRuleRepositoryPersistenceAndDurableReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	auditReferences := make([]uuid.UUID, 0, 3)
	repository, err := identity.NewPostgresSegregationRuleRepositoryWithAudit(fixture.pool, func(_ context.Context, _ pgx.Tx, _ identity.SegregationRuleAuditRecord) (uuid.UUID, error) {
		reference := uuid.New()
		auditReferences = append(auditReferences, reference)
		return reference, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	baseline, err := repository.Get(ctx, uuid.MustParse("00000000-0000-0000-0000-000000000504"))
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Version.Value() != 1 || baseline.AmountThreshold == nil || !baseline.AmountThreshold.Equal(decimalFromTest(10000)) {
		t.Fatalf("baseline rule = %#v", baseline)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	approval := identity.ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), ApproverUserID: uuid.New(), PolicyVersion: "rule-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:create"}
	rule, err := identity.NewSegregationRule(uuid.New(), "integration-segregation-rule", "Integration rule", []string{"finance.gl.submit.posting.request", "finance.gl.apply.journal.approval.decision"}, identity.SegregationEnforcementBlock, []string{"entity-1"}, nil, 0, now, approval, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitSegregationRuleMutation(ctx, identity.SegregationRuleMutation{After: rule, Audit: identity.SegregationRuleAuditRecord{RuleID: rule.ID, Action: identity.SegregationRuleActionCreate, RuleVersion: 1, AfterFingerprint: approval.CandidateFingerprint}}); err != nil {
		t.Fatal(err)
	}
	stored, err := repository.Get(ctx, rule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version.Value() != 1 || stored.AuditReference != auditReferences[0] {
		t.Fatalf("stored rule = %#v, audit=%s", stored, stored.AuditReference)
	}

	updated := stored
	updated.Name = "Integration rule v2"
	updated.UpdatedAt = now.Add(time.Minute)
	updated.Version, err = stored.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	updated.Approval = approval
	updated.Approval.CandidateFingerprint = "sha256:update"
	if err := repository.CommitSegregationRuleMutation(ctx, identity.SegregationRuleMutation{Before: stored, After: updated, ExpectedVersion: &stored.Version, Audit: identity.SegregationRuleAuditRecord{RuleID: rule.ID, Action: identity.SegregationRuleActionUpdate, RuleVersion: 2, AfterFingerprint: updated.Approval.CandidateFingerprint}}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitSegregationRuleMutation(ctx, identity.SegregationRuleMutation{Before: stored, After: updated, ExpectedVersion: &stored.Version, Audit: identity.SegregationRuleAuditRecord{RuleID: rule.ID, Action: identity.SegregationRuleActionUpdate, RuleVersion: 2}}); !errors.Is(err, identity.ErrSegregationRuleVersion) {
		t.Fatalf("stale update error = %v", err)
	}
	current, err := repository.Get(ctx, rule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Version.Value() != 2 || current.Name != "Integration rule v2" {
		t.Fatalf("current rule = %#v", current)
	}

	if _, err := fixture.pool.Exec(ctx, "UPDATE identity.segregation_rule_revision SET name = tampered WHERE rule_id = $1 AND revision_version = 1", rule.ID); err == nil {
		t.Fatal("immutable revision update unexpectedly succeeded")
	}
	if _, err := fixture.pool.Exec(ctx, "DELETE FROM identity.segregation_rule_revision WHERE rule_id = $1 AND revision_version = 1", rule.ID); err == nil {
		t.Fatal("immutable revision delete unexpectedly succeeded")
	}
	listed, err := repository.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) < 7 {
		t.Fatalf("listed rule count = %d, want seeded rules plus integration rule", len(listed))
	}

	durableCommand := identity.SegregationRuleCommand{Action: identity.SegregationRuleActionCreate, Code: "durable-segregation-rule", Name: "Durable segregation rule", ConflictingPermissions: []string{"finance.gl.submit.posting.request", "finance.gl.apply.journal.approval.decision"}, EnforcementMode: identity.SegregationEnforcementBlock, EffectiveFrom: now, Approval: identity.ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), ApproverUserID: uuid.New(), PolicyVersion: "rule-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:durable"}, IdempotencyKey: "durable-segregation-create", CorrelationID: "integration-correlation", CausationID: "integration-causation"}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "segregation-oid", TID: "tenant", Sub: "segregation-sub"}}
	authorizer := identity.MemorySegregationRuleAuthorizer{Decision: identity.AuthorizationDecision{Allowed: true, Outcome: identity.AuthorizationAllowed, Permission: identity.SegregationRuleManagementPermission, PolicyReference: "iam-policy-v1", PolicyVersion: "iam-policy-v1", DecisionReference: uuid.New()}}
	approvalPort := integrationSegregationApprovalPort{}
	policy := idempotency.IdempotencyPolicy{RecordTTL: time.Hour, LeaseTTL: time.Minute}
	newService := func() *identity.SegregationRuleService {
		service, err := identity.NewSegregationRuleServiceWithDurableIdempotency(repository, authorizer, approvalPort, &identity.MemorySegregationRuleAuditRecorder{}, time.Now, identity.DurableSegregationRuleServiceConfig{Database: fixture.pool, Coordinator: idempotency.NewPostgresCoordinator(), Policy: policy, OperationID: "identity.manage-segregation-rules.v1"})
		if err != nil {
			t.Fatal(err)
		}
		return service
	}
	first, err := newService().Execute(ctx, actor, durableCommand)
	if err != nil {
		t.Fatal(err)
	}
	second, err := newService().Execute(ctx, actor, durableCommand)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Replayed || second.Rule.ID != first.Rule.ID {
		t.Fatalf("durable replay = %#v, want rule %s", second, first.Rule.ID)
	}
	assertNoAcquiredConnections(t, fixture.pool)
}

func decimalFromTest(value int64) decimal.Decimal {
	return decimal.NewFromInt(value)
}

type integrationSegregationApprovalPort struct{}

func (integrationSegregationApprovalPort) ValidateSegregationRuleApproval(context.Context, identity.ApplicationActor, identity.SegregationRuleCommand, *identity.SegregationRule, string) error {
	return nil
}
