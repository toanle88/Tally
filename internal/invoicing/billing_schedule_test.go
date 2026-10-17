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

func TestBillingScheduleCommandValidatesExactDecimalsAndIANAZone(t *testing.T) {
	command := testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil)
	command.Charges[0].Quantity = "0002.00"
	command.Charges[0].Amount = "00010.00"
	if err := command.Validate(); err != nil {
		t.Fatalf("valid command error = %v", err)
	}
	canonical := command.Canonical()
	if canonical.Charges[0].Quantity != "2" || canonical.Charges[0].Amount != "10" {
		t.Fatalf("canonical charge = %#v", canonical.Charges[0])
	}

	badDecimal := command
	badDecimal.Charges = cloneScheduledCharges(command.Charges)
	badDecimal.Charges[0].Amount = "1e2"
	if err := badDecimal.Validate(); !errors.Is(err, ErrInvalidBillingScheduleCommand) {
		t.Fatalf("malformed amount error = %v, want invalid command", err)
	}

	badZone := command
	badZone.TimeZone = "Mars/Olympus"
	if err := badZone.Validate(); !errors.Is(err, ErrInvalidBillingScheduleCommand) {
		t.Fatalf("invalid zone error = %v, want invalid command", err)
	}

	badQuantity := command
	badQuantity.Charges = cloneScheduledCharges(command.Charges)
	badQuantity.Charges[0].Quantity = "0"
	if err := badQuantity.Validate(); !errors.Is(err, ErrInvalidBillingScheduleCommand) {
		t.Fatalf("zero quantity error = %v, want invalid command", err)
	}
}

func TestBillingScheduleServicePreservesRevisionsAndIdempotency(t *testing.T) {
	repository := NewMemoryBillingScheduleRepository()
	audit := &MemoryBillingScheduleAuditRecorder{}
	service := newTestBillingScheduleService(t, repository, MemoryBillingScheduleAuthorizer{Decision: allowBillingScheduleDecision(testScope())}, AllowAllBillingScheduleReferenceValidator{}, audit)
	actor := testBillingScheduleActor()

	created, err := service.Execute(context.Background(), actor, testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil))
	if err != nil {
		t.Fatalf("create error = %v", err)
	}
	if created.Schedule.Version.Value() != 1 || len(created.Schedule.Revisions) != 1 || created.Schedule.Status != ScheduleStatusConfigured {
		t.Fatalf("created projection = %#v", created.Schedule)
	}

	replayed, err := service.Execute(context.Background(), actor, testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil))
	if err != nil || !replayed.Replayed || replayed.Schedule.ID != created.Schedule.ID {
		t.Fatalf("replay = %#v, error = %v", replayed, err)
	}
	changed := testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil)
	changed.ContractReference = "contract-changed"
	if _, err := service.Execute(context.Background(), actor, changed); !errors.Is(err, ErrBillingScheduleIdempotencyConflict) {
		t.Fatalf("changed duplicate error = %v, want idempotency conflict", err)
	}

	current, err := repository.GetBillingSchedule(context.Background(), created.Schedule.ID)
	if err != nil {
		t.Fatalf("get created error = %v", err)
	}
	oldContract := current.ContractReference
	update := testBillingScheduleCommand(BillingScheduleActionUpdate, current.ID, &current.Version)
	update.ContractReference = "contract-v2"
	update.IdempotencyKey = "billing-update"
	updated, err := service.Execute(context.Background(), actor, update)
	if err != nil {
		t.Fatalf("update error = %v", err)
	}
	if updated.Schedule.Version.Value() != 2 || len(updated.Schedule.Revisions) != 2 || updated.Schedule.ContractReference != update.ContractReference {
		t.Fatalf("updated projection = %#v", updated.Schedule)
	}
	revisions, err := repository.GetBillingSchedule(context.Background(), current.ID)
	if err != nil {
		t.Fatalf("get updated error = %v", err)
	}
	if revisions.Revisions[0].Snapshot.ContractReference != oldContract || revisions.Revisions[1].Snapshot.ContractReference != update.ContractReference {
		t.Fatalf("revision lineage = %#v", revisions.Revisions)
	}

	staleVersion, _ := aggregateversion.FromInt64(1)
	stale := testBillingScheduleCommand(BillingScheduleActionUpdate, current.ID, &staleVersion)
	stale.IdempotencyKey = "billing-stale"
	if _, err := service.Execute(context.Background(), actor, stale); !errors.Is(err, ErrBillingScheduleVersionConflict) {
		t.Fatalf("stale update error = %v, want version conflict", err)
	}
	if len(audit.Records) != 2 || audit.Records[1].ExpectedVersion == nil || audit.Records[1].ExpectedVersion.Value() != 1 {
		t.Fatalf("audit lineage = %#v", audit.Records)
	}
}

func TestBillingScheduleServiceRejectsOverlapAndDoesNotPartiallyCommit(t *testing.T) {
	scope := testScope()
	repository := NewMemoryBillingScheduleRepository()
	service := newTestBillingScheduleService(t, repository, MemoryBillingScheduleAuthorizer{Decision: allowBillingScheduleDecision(scope)}, AllowAllBillingScheduleReferenceValidator{}, &MemoryBillingScheduleAuditRecorder{})
	actor := testBillingScheduleActor()
	base := testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil)
	if _, err := service.Execute(context.Background(), actor, base); err != nil {
		t.Fatalf("base create error = %v", err)
	}
	overlap := testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil)
	overlap.IdempotencyKey = "overlap"
	if _, err := service.Execute(context.Background(), actor, overlap); !errors.Is(err, ErrBillingScheduleDuplicate) {
		t.Fatalf("overlap error = %v, want duplicate", err)
	}

	failingRepository := NewMemoryBillingScheduleRepository()
	failingAudit := &MemoryBillingScheduleAuditRecorder{Err: errors.New("audit unavailable")}
	failing := newTestBillingScheduleService(t, failingRepository, MemoryBillingScheduleAuthorizer{Decision: allowBillingScheduleDecision(scope)}, AllowAllBillingScheduleReferenceValidator{}, failingAudit)
	if _, err := failing.Execute(context.Background(), actor, testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil)); !errors.Is(err, failingAudit.Err) {
		t.Fatalf("audit error = %v, want audit failure", err)
	}
	if values, err := failingRepository.ListBillingSchedules(context.Background(), &scope); err != nil || len(values) != 0 {
		t.Fatalf("partial commit values = %d, error = %v", len(values), err)
	}

	referenceRepository := NewMemoryBillingScheduleRepository()
	referenceUnavailable := newTestBillingScheduleService(t, referenceRepository, MemoryBillingScheduleAuthorizer{Decision: allowBillingScheduleDecision(scope)}, UnavailableBillingScheduleReferenceValidator{}, &MemoryBillingScheduleAuditRecorder{})
	if _, err := referenceUnavailable.Execute(context.Background(), actor, testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil)); !errors.Is(err, ErrBillingScheduleReferenceUnavailable) {
		t.Fatalf("reference dependency error = %v, want unavailable", err)
	}
	if values, err := referenceRepository.ListBillingSchedules(context.Background(), &scope); err != nil || len(values) != 0 {
		t.Fatalf("reference dependency partial commit values = %d, error = %v", len(values), err)
	}
}

func TestBillingScheduleServiceOptimisticConcurrencyRace(t *testing.T) {
	repository := NewMemoryBillingScheduleRepository()
	service := newTestBillingScheduleService(t, repository, MemoryBillingScheduleAuthorizer{Decision: allowBillingScheduleDecision(testScope())}, AllowAllBillingScheduleReferenceValidator{}, &MemoryBillingScheduleAuditRecorder{})
	actor := testBillingScheduleActor()
	created, err := service.Execute(context.Background(), actor, testBillingScheduleCommand(BillingScheduleActionCreate, uuid.Nil, nil))
	if err != nil {
		t.Fatalf("create error = %v", err)
	}
	version := aggregateversion.Initial()
	first := testBillingScheduleCommand(BillingScheduleActionUpdate, created.Schedule.ID, &version)
	first.IdempotencyKey = "race-one"
	second := first
	second.IdempotencyKey = "race-two"
	second.ContractReference = "race-two-contract"
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
		case errors.Is(err, ErrBillingScheduleVersionConflict):
			conflicts++
		default:
			t.Fatalf("race error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("race results = successes %d conflicts %d, want one each", successes, conflicts)
	}
}

func TestBillingScheduleDurableFailureReplaysOriginalRejection(t *testing.T) {
	tests := []struct {
		code string
		want error
	}{
		{code: "INVALID_REQUEST", want: ErrInvalidBillingScheduleCommand},
		{code: "AUTHORIZATION_DENIED", want: ErrBillingScheduleAuthorizationDenied},
		{code: "POLICY_STALE", want: ErrBillingScheduleAuthorizationStale},
		{code: "AUTHORIZATION_UNAVAILABLE", want: ErrBillingScheduleAuthorizationUnavailable},
		{code: "REFERENCE_UNAVAILABLE", want: ErrBillingScheduleReferenceUnavailable},
		{code: "AUDIT_UNAVAILABLE", want: ErrBillingScheduleAuditUnavailable},
		{code: "REFERENCE_INVALID", want: ErrBillingScheduleReferenceInvalid},
		{code: "VERSION_CONFLICT", want: ErrBillingScheduleVersionConflict},
		{code: "BILLING_SCHEDULE_NOT_FOUND", want: ErrBillingScheduleNotFound},
		{code: "DUPLICATE_BILLING_SCHEDULE", want: ErrBillingScheduleDuplicate},
		{code: "IDEMPOTENCY_CONFLICT", want: ErrBillingScheduleIdempotencyConflict},
		{code: "VALIDATION_FAILED", want: ErrInvalidBillingSchedule},
	}
	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			body := []byte(`{"code":"` + test.code + `"}`)
			if got := billingScheduleDurableFailureError(body); !errors.Is(got, test.want) {
				t.Fatalf("error = %v, want %v", got, test.want)
			}
		})
	}
	if got := billingScheduleDurableFailureError([]byte(`{"code":"UNKNOWN"}`)); !errors.Is(got, ErrBillingScheduleDurableCommandFailed) {
		t.Fatalf("unknown failure error = %v", got)
	}
}

func newTestBillingScheduleService(t *testing.T, repository BillingScheduleRepository, authorizer BillingScheduleAuthorizer, references BillingScheduleReferenceValidator, audit BillingScheduleAuditRecorder) *BillingScheduleService {
	t.Helper()
	service, err := NewBillingScheduleService(repository, authorizer, references, audit, func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatalf("new service error = %v", err)
	}
	return service
}

func testBillingScheduleActor() Actor {
	return Actor{UserID: uuid.New(), SubjectReference: "billing-subject"}
}

func allowBillingScheduleDecision(scope uuid.UUID) AuthorizationDecision {
	return AuthorizationDecision{Allowed: true, Permission: BillingScheduleManagementPermission, ApprovedScopeIDs: []uuid.UUID{scope}, PolicyReference: "billing-policy", PolicyVersion: "v1", DecisionReference: uuid.New()}
}

func testBillingScheduleCommand(action string, id uuid.UUID, version *aggregateversion.AggregateVersion) BillingScheduleCommand {
	return BillingScheduleCommand{
		Action: action, BillingScheduleID: id, AccountingScopeID: testScope(), CustomerID: uuid.MustParse("33333333-3333-4333-8333-333333333333"), ContractReference: "contract-1",
		Charges:         []ScheduledCharge{{ID: uuid.MustParse("44444444-4444-4444-8444-444444444444"), TemplateID: uuid.MustParse("55555555-5555-4555-8555-555555555555"), TemplateVersion: 1, TemplateLineItemID: uuid.MustParse("66666666-6666-4666-8666-666666666666"), Quantity: "2", Amount: "10.00", Currency: "USD"}},
		NextBillingDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), TimeZone: "Asia/Ho_Chi_Minh", Status: ScheduleStatusConfigured,
		EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ExpectedVersion: version, IdempotencyKey: "billing-create", CorrelationID: "billing-correlation", CausationID: "billing-causation",
	}
}
