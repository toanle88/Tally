package invoicing

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestInvoiceTemplateCommandValidatesExactDecimalsAndSupportedValues(t *testing.T) {
	command := testInvoiceTemplateCommand("create", uuid.Nil, nil)
	command.LineItems[0].ChargeRule.Amount = "00010.00"
	if err := command.Validate(); err != nil {
		t.Fatalf("valid command error = %v", err)
	}
	canonical := command.Canonical()
	if got := canonical.LineItems[0].ChargeRule.Amount; got != "10" {
		t.Fatalf("canonical amount = %q, want 10", got)
	}

	badAmount := command
	badAmount.LineItems = cloneLineItems(command.LineItems)
	badAmount.LineItems[0].ChargeRule.Amount = "1e2"
	if err := badAmount.Validate(); !errors.Is(err, ErrInvalidInvoiceTemplateCommand) {
		t.Fatalf("malformed decimal error = %v, want invalid command", err)
	}

	badRule := command
	badRule.LineItems = cloneLineItems(command.LineItems)
	badRule.LineItems[0].ChargeRule = ChargeRule{Kind: ChargeRulePercentage, Rate: "100.01"}
	if err := badRule.Validate(); !errors.Is(err, ErrInvalidInvoiceTemplateCommand) {
		t.Fatalf("percentage error = %v, want invalid command", err)
	}

	badApplicability := command
	badApplicability.Applicability = Applicability{Mode: ApplicabilityAll, Dimensions: map[string]string{"region": "vn"}}
	if err := badApplicability.Validate(); !errors.Is(err, ErrInvalidInvoiceTemplateCommand) {
		t.Fatalf("applicability error = %v, want invalid command", err)
	}
	duplicateDimensionKey := command
	duplicateDimensionKey.Applicability = Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{" region ": "vn", "region": "vn"}}
	if err := duplicateDimensionKey.Validate(); !errors.Is(err, ErrInvalidInvoiceTemplateCommand) {
		t.Fatalf("duplicate canonical dimension error = %v, want invalid command", err)
	}
}

func TestInvoiceTemplateApplicabilityOverlapIsExplicit(t *testing.T) {
	if !applicabilityOverlaps(
		Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{"region": "APAC"}},
		Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{"region": "APAC", "tier": "premium"}},
	) {
		t.Fatal("a broader and narrower matching dimension set should overlap")
	}
	if applicabilityOverlaps(
		Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{"region": "APAC"}},
		Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{"region": "EMEA"}},
	) {
		t.Fatal("different values for a shared dimension should not overlap")
	}
	if !applicabilityOverlaps(Applicability{Mode: ApplicabilityAll}, Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{"region": "APAC"}}) {
		t.Fatal("all applicability should overlap a dimension applicability")
	}
}

func TestInvoiceTemplateServicePreservesImmutableRevisions(t *testing.T) {
	repository := NewMemoryInvoiceTemplateRepository()
	audit := &MemoryInvoiceTemplateAuditRecorder{}
	service := newTestInvoiceTemplateService(t, repository, MemoryInvoiceTemplateAuthorizer{Decision: allowInvoiceTemplateDecision(testScope())}, AllowAllInvoiceTemplateReferenceValidator{}, audit)
	actor := testInvoiceTemplateActor()

	created, err := service.Execute(context.Background(), actor, testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil))
	if err != nil {
		t.Fatalf("create error = %v", err)
	}
	if created.Template.Version.Value() != 1 || len(created.Template.Revisions) != 1 {
		t.Fatalf("created version/revisions = %d/%d, want 1/1", created.Template.Version.Value(), len(created.Template.Revisions))
	}

	current, err := repository.GetInvoiceTemplate(context.Background(), created.Template.ID)
	if err != nil {
		t.Fatalf("get created error = %v", err)
	}
	oldName := current.Name
	updatedCommand := testInvoiceTemplateCommand(InvoiceTemplateActionUpdate, current.ID, &current.Version)
	updatedCommand.Name = "Updated service template"
	updatedCommand.IdempotencyKey = "update-key"
	updated, err := service.Execute(context.Background(), actor, updatedCommand)
	if err != nil {
		t.Fatalf("update error = %v", err)
	}
	if updated.Template.Version.Value() != 2 || updated.Template.Name != updatedCommand.Name {
		t.Fatalf("updated version/name = %d/%q", updated.Template.Version.Value(), updated.Template.Name)
	}

	revisions, err := repository.ListInvoiceTemplateRevisions(context.Background(), current.ID)
	if err != nil {
		t.Fatalf("list revisions error = %v", err)
	}
	if len(revisions) != 2 || revisions[0].Snapshot.Name != oldName || revisions[1].Snapshot.Name != updatedCommand.Name {
		t.Fatalf("revision lineage = %#v", revisions)
	}
	revisions[0].Snapshot.Name = "mutated caller copy"
	again, err := repository.GetInvoiceTemplateRevision(context.Background(), current.ID, aggregateversion.Initial())
	if err != nil {
		t.Fatalf("get revision error = %v", err)
	}
	if again.Snapshot.Name != oldName {
		t.Fatalf("revision was mutable through caller copy: %q", again.Snapshot.Name)
	}
	if len(audit.Records) != 2 || audit.Records[1].ExpectedVersion == nil || audit.Records[1].ExpectedVersion.Value() != 1 {
		t.Fatalf("audit lineage = %#v", audit.Records)
	}
}

func TestInvoiceTemplateServiceIdempotencyAndVersionConflict(t *testing.T) {
	repository := NewMemoryInvoiceTemplateRepository()
	service := newTestInvoiceTemplateService(t, repository, MemoryInvoiceTemplateAuthorizer{Decision: allowInvoiceTemplateDecision(testScope())}, AllowAllInvoiceTemplateReferenceValidator{}, &MemoryInvoiceTemplateAuditRecorder{})
	actor := testInvoiceTemplateActor()
	command := testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil)
	command.IdempotencyKey = "same-key"
	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatalf("first create error = %v", err)
	}
	replayed, err := service.Execute(context.Background(), actor, command)
	if err != nil || !replayed.Replayed || replayed.Template.ID != first.Template.ID {
		t.Fatalf("replay = %#v, error = %v", replayed, err)
	}
	changed := command
	changed.Name = "changed after submission"
	if _, err := service.Execute(context.Background(), actor, changed); !errors.Is(err, ErrInvoiceTemplateIdempotencyConflict) {
		t.Fatalf("changed duplicate error = %v, want idempotency conflict", err)
	}
	values, err := repository.ListInvoiceTemplates(context.Background(), &command.AccountingScopeID)
	if err != nil || len(values) != 1 {
		t.Fatalf("template count = %d, error = %v, want one", len(values), err)
	}

	current, _ := repository.GetInvoiceTemplate(context.Background(), first.Template.ID)
	staleVersion, _ := aggregateversion.FromInt64(99)
	stale := testInvoiceTemplateCommand(InvoiceTemplateActionUpdate, current.ID, &staleVersion)
	stale.IdempotencyKey = "stale-key"
	if _, err := service.Execute(context.Background(), actor, stale); !errors.Is(err, ErrInvoiceTemplateVersionConflict) {
		t.Fatalf("stale update error = %v, want version conflict", err)
	}
}

func TestInvoiceTemplateServiceRejectsOverlapAndDoesNotPartiallyCommit(t *testing.T) {
	scope := testScope()
	repository := NewMemoryInvoiceTemplateRepository()
	audit := &MemoryInvoiceTemplateAuditRecorder{}
	service := newTestInvoiceTemplateService(t, repository, MemoryInvoiceTemplateAuthorizer{Decision: allowInvoiceTemplateDecision(scope)}, AllowAllInvoiceTemplateReferenceValidator{}, audit)
	actor := testInvoiceTemplateActor()
	base := testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil)
	base.IdempotencyKey = "base"
	if _, err := service.Execute(context.Background(), actor, base); err != nil {
		t.Fatalf("base create error = %v", err)
	}
	overlap := testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil)
	overlap.IdempotencyKey = "overlap"
	if _, err := service.Execute(context.Background(), actor, overlap); !errors.Is(err, ErrInvoiceTemplateDuplicate) {
		t.Fatalf("overlap error = %v, want duplicate", err)
	}

	denied := newTestInvoiceTemplateService(t, NewMemoryInvoiceTemplateRepository(), MemoryInvoiceTemplateAuthorizer{Decision: AuthorizationDecision{Allowed: false, Permission: InvoiceTemplateManagementPermission, DecisionReference: uuid.New()}}, AllowAllInvoiceTemplateReferenceValidator{}, &MemoryInvoiceTemplateAuditRecorder{})
	if _, err := denied.Execute(context.Background(), actor, testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil)); !errors.Is(err, ErrInvoiceTemplateAuthorizationDenied) {
		t.Fatalf("denied error = %v, want authorization denied", err)
	}

	failingAudit := &MemoryInvoiceTemplateAuditRecorder{Err: errors.New("audit unavailable")}
	failingRepository := NewMemoryInvoiceTemplateRepository()
	failing := newTestInvoiceTemplateService(t, failingRepository, MemoryInvoiceTemplateAuthorizer{Decision: allowInvoiceTemplateDecision(scope)}, AllowAllInvoiceTemplateReferenceValidator{}, failingAudit)
	if _, err := failing.Execute(context.Background(), actor, testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil)); !errors.Is(err, failingAudit.Err) {
		t.Fatalf("audit error = %v, want audit failure", err)
	}
	if values, err := failingRepository.ListInvoiceTemplates(context.Background(), &scope); err != nil || len(values) != 0 {
		t.Fatalf("partial commit values = %d, error = %v", len(values), err)
	}

	referenceRepository := NewMemoryInvoiceTemplateRepository()
	referenceUnavailable := newTestInvoiceTemplateService(t, referenceRepository, MemoryInvoiceTemplateAuthorizer{Decision: allowInvoiceTemplateDecision(scope)}, UnavailableInvoiceTemplateReferenceValidator{}, &MemoryInvoiceTemplateAuditRecorder{})
	if _, err := referenceUnavailable.Execute(context.Background(), actor, testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil)); !errors.Is(err, ErrInvoiceTemplateReferenceUnavailable) {
		t.Fatalf("reference dependency error = %v, want unavailable", err)
	}
	if values, err := referenceRepository.ListInvoiceTemplates(context.Background(), &scope); err != nil || len(values) != 0 {
		t.Fatalf("reference dependency partial commit values = %d, error = %v", len(values), err)
	}

	dimensionRepository := NewMemoryInvoiceTemplateRepository()
	dimensionService := newTestInvoiceTemplateService(t, dimensionRepository, MemoryInvoiceTemplateAuthorizer{Decision: allowInvoiceTemplateDecision(scope)}, AllowAllInvoiceTemplateReferenceValidator{}, &MemoryInvoiceTemplateAuditRecorder{})
	dimensionBase := testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil)
	dimensionBase.IdempotencyKey = "dimension-base"
	dimensionBase.Applicability = Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{"region": "APAC"}}
	if _, err := dimensionService.Execute(context.Background(), actor, dimensionBase); err != nil {
		t.Fatalf("dimension base create error = %v", err)
	}
	dimensionOverlap := dimensionBase
	dimensionOverlap.IdempotencyKey = "dimension-overlap"
	dimensionOverlap.Applicability = Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{"region": "APAC", "tier": "premium"}}
	if _, err := dimensionService.Execute(context.Background(), actor, dimensionOverlap); !errors.Is(err, ErrInvoiceTemplateDuplicate) {
		t.Fatalf("dimension overlap error = %v, want duplicate", err)
	}
	dimensionDisjoint := dimensionBase
	dimensionDisjoint.IdempotencyKey = "dimension-disjoint"
	dimensionDisjoint.Applicability = Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{"region": "EMEA"}}
	if _, err := dimensionService.Execute(context.Background(), actor, dimensionDisjoint); err != nil {
		t.Fatalf("disjoint dimension create error = %v", err)
	}
}

func TestInvoiceTemplateServiceRejectsCanonicalApplicabilityKeyCollision(t *testing.T) {
	scope := testScope()
	repository := NewMemoryInvoiceTemplateRepository()
	service := newTestInvoiceTemplateService(t, repository, MemoryInvoiceTemplateAuthorizer{Decision: allowInvoiceTemplateDecision(scope)}, AllowAllInvoiceTemplateReferenceValidator{}, &MemoryInvoiceTemplateAuditRecorder{})
	command := testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil)
	command.Applicability = Applicability{Mode: ApplicabilityDimensions, Dimensions: map[string]string{" region ": "APAC", "region": "EMEA"}}

	if _, err := service.Execute(context.Background(), testInvoiceTemplateActor(), command); !errors.Is(err, ErrInvalidInvoiceTemplateCommand) {
		t.Fatalf("canonical applicability key collision error = %v, want invalid command", err)
	}
	if values, err := repository.ListInvoiceTemplates(context.Background(), &scope); err != nil || len(values) != 0 {
		t.Fatalf("canonical applicability collision partial commit values = %d, error = %v", len(values), err)
	}
}

func TestInvoiceTemplateServiceOptimisticConcurrencyRace(t *testing.T) {
	repository := NewMemoryInvoiceTemplateRepository()
	service := newTestInvoiceTemplateService(t, repository, MemoryInvoiceTemplateAuthorizer{Decision: allowInvoiceTemplateDecision(testScope())}, AllowAllInvoiceTemplateReferenceValidator{}, &MemoryInvoiceTemplateAuditRecorder{})
	actor := testInvoiceTemplateActor()
	created, err := service.Execute(context.Background(), actor, testInvoiceTemplateCommand(InvoiceTemplateActionCreate, uuid.Nil, nil))
	if err != nil {
		t.Fatalf("create error = %v", err)
	}
	version := aggregateversion.Initial()
	first := testInvoiceTemplateCommand(InvoiceTemplateActionUpdate, created.Template.ID, &version)
	first.IdempotencyKey = "race-one"
	second := first
	second.IdempotencyKey = "race-two"
	second.Name = "second race winner"
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, executeErr := service.Execute(context.Background(), actor, first)
		errs <- executeErr
	}()
	go func() {
		defer wg.Done()
		_, executeErr := service.Execute(context.Background(), actor, second)
		errs <- executeErr
	}()
	wg.Wait()
	close(errs)
	var successes, conflicts int
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInvoiceTemplateVersionConflict):
			conflicts++
		default:
			t.Fatalf("race error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("race results = successes %d conflicts %d, want one each", successes, conflicts)
	}
}

func newTestInvoiceTemplateService(t *testing.T, repository InvoiceTemplateRepository, authorizer InvoiceTemplateAuthorizer, references InvoiceTemplateReferenceValidator, audit InvoiceTemplateAuditRecorder) *InvoiceTemplateService {
	t.Helper()
	service, err := NewInvoiceTemplateService(repository, authorizer, references, audit, func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatalf("new service error = %v", err)
	}
	return service
}

func testInvoiceTemplateActor() Actor {
	return Actor{UserID: uuid.New(), SubjectReference: "subject-1"}
}

func testScope() uuid.UUID { return uuid.MustParse("11111111-1111-4111-8111-111111111111") }

func allowInvoiceTemplateDecision(scope uuid.UUID) AuthorizationDecision {
	return AuthorizationDecision{Allowed: true, Permission: InvoiceTemplateManagementPermission, ApprovedScopeIDs: []uuid.UUID{scope}, PolicyReference: "test-policy", PolicyVersion: "v1", DecisionReference: uuid.New()}
}

func testInvoiceTemplateCommand(action string, id uuid.UUID, version *aggregateversion.AggregateVersion) InvoiceTemplateCommand {
	return InvoiceTemplateCommand{
		Action: action, TemplateID: id, AccountingScopeID: testScope(), Code: "SAAS-BASE", Name: "Base service",
		BillingFrequency: BillingFrequencyMonthly,
		LineItems:        []TemplateLineItem{{ID: uuid.MustParse("22222222-2222-4222-8222-222222222222"), Code: "BASE", Description: "Base subscription", ChargeRule: ChargeRule{Kind: ChargeRuleFixed, Amount: "10.00", Currency: "USD"}, TaxCategory: TaxCategory{Code: "standard"}}},
		PaymentMethod:    PaymentMethod{Method: PaymentMethodBankTransfer, TermsDays: 30},
		Applicability:    Applicability{Mode: ApplicabilityAll}, EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpectedVersion: version, IdempotencyKey: "test-key", CorrelationID: "correlation", CausationID: "causation",
	}
}
