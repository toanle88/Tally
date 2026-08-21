package idempotency

import (
	"errors"
	"sync"
	"testing"
)

// transactionHarness is a test-only commit boundary. It models staged effects
// and result finalization without pretending to provide PostgreSQL semantics.
type transactionHarness struct {
	mu               sync.Mutex
	committedEffects int
	committedResults map[IdempotencyIdentity]CommandResultMetadata
}

type transactionHarnessTx struct {
	parent       *transactionHarness
	stagedEffect func()
	finalize     func() error
	result       CommandResultMetadata
	closed       bool
}

func newTransactionHarness() *transactionHarness {
	return &transactionHarness{
		committedResults: make(map[IdempotencyIdentity]CommandResultMetadata),
	}
}

func (h *transactionHarness) Begin() *transactionHarnessTx {
	return &transactionHarnessTx{parent: h}
}

func (h *transactionHarness) committedEffectCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.committedEffects
}

func (h *transactionHarness) committedResult(identity IdempotencyIdentity) (CommandResultMetadata, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	result, ok := h.committedResults[identity]
	return cloneMetadata(result), ok
}

func (tx *transactionHarnessTx) StageEffect(effect func()) {
	tx.stagedEffect = effect
}

func (tx *transactionHarnessTx) FinalizeResult(result CommandResultMetadata, finalize func() error) {
	tx.result = cloneMetadata(result)
	tx.finalize = finalize
}

func (tx *transactionHarnessTx) Commit() error {
	if tx.closed {
		return ErrAlreadyFinalized
	}
	tx.closed = true
	if tx.finalize != nil {
		if err := tx.finalize(); err != nil {
			return err
		}
	}

	tx.parent.mu.Lock()
	defer tx.parent.mu.Unlock()
	if tx.stagedEffect != nil {
		tx.stagedEffect()
		tx.parent.committedEffects++
	}
	if tx.finalize != nil {
		tx.parent.committedResults[tx.result.Identity()] = cloneMetadata(tx.result)
	}
	return nil
}

func (tx *transactionHarnessTx) Rollback() {
	tx.closed = true
	tx.stagedEffect = nil
	tx.finalize = nil
}

func TestTransactionHarnessRollbackLeavesCoordinatorInProgress(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-rollback")
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := coordinator.Acquire(identity, "sha256:rollback", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	final := coordinatorResult(t, identity, "sha256:rollback", StateEstablished)
	harness := newTransactionHarness()
	tx := harness.Begin()
	effectCommitted := false
	tx.StageEffect(func() { effectCommitted = true })
	tx.FinalizeResult(final, func() error {
		return coordinator.Finalize(acquisition, func(CommandResultMetadata) (CommandResultMetadata, error) {
			return final, nil
		})
	})
	tx.Rollback()

	if effectCommitted {
		t.Fatal("rolled-back effect became visible")
	}
	if harness.committedEffectCount() != 0 {
		t.Fatalf("committed effects = %d, want 0", harness.committedEffectCount())
	}
	retry, err := coordinator.Acquire(identity, "sha256:rollback", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	if retry.Decision() != DecisionReturn || retry.Result().State() != StateInProgress {
		t.Fatalf("retry = %#v, want in-progress established result", retry)
	}
}

func TestTransactionHarnessCommitPublishesEffectAndTerminalResult(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-commit")
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := coordinator.Acquire(identity, "sha256:commit", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	final := coordinatorResult(t, identity, "sha256:commit", StateEstablished)
	harness := newTransactionHarness()
	tx := harness.Begin()
	tx.StageEffect(func() {})
	tx.FinalizeResult(final, func() error {
		return coordinator.Finalize(acquisition, func(CommandResultMetadata) (CommandResultMetadata, error) {
			return final, nil
		})
	})
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if harness.committedEffectCount() != 1 {
		t.Fatalf("committed effects = %d, want 1", harness.committedEffectCount())
	}
	if result, ok := harness.committedResult(identity); !ok || result.State() != StateEstablished {
		t.Fatalf("committed result = %#v, present = %t", result, ok)
	}
	retry, err := coordinator.Acquire(identity, "sha256:commit", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	if retry.Decision() != DecisionReturn || retry.Result().State() != StateEstablished {
		t.Fatalf("retry = %#v, want established result", retry)
	}
}

func TestMemoryCoordinatorConcurrentAcquisitionAndFinalizationHasOneEffect(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-concurrent-finalize")
	if err != nil {
		t.Fatal(err)
	}
	const callers = 32
	start := make(chan struct{})
	acquisitions := make(chan Acquisition, callers)
	errors := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			acquisition, acquireErr := coordinator.Acquire(identity, "sha256:concurrent-finalize", "payments.submit")
			if acquireErr != nil {
				errors <- acquireErr
				return
			}
			acquisitions <- acquisition
		}()
	}
	close(start)
	wg.Wait()
	close(acquisitions)
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}

	var owner Acquisition
	owners := 0
	for acquisition := range acquisitions {
		if acquisition.Decision() == DecisionExecute {
			owner = acquisition
			owners++
		}
	}
	if owners != 1 {
		t.Fatalf("execution owners = %d, want 1", owners)
	}
	final := coordinatorResult(t, identity, "sha256:concurrent-finalize", StateEstablished)
	effects := 0
	if err := coordinator.Finalize(owner, func(CommandResultMetadata) (CommandResultMetadata, error) {
		effects++
		return final, nil
	}); err != nil {
		t.Fatal(err)
	}
	if effects != 1 {
		t.Fatalf("effects = %d, want 1", effects)
	}
}

func TestMemoryCoordinatorConcurrentFinalizationHasOneCommitter(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-finalizers")
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := coordinator.Acquire(identity, "sha256:finalizers", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	final := coordinatorResult(t, identity, "sha256:finalizers", StateEstablished)
	const finalizers = 16
	start := make(chan struct{})
	finalizationErrors := make(chan error, finalizers)
	var wg sync.WaitGroup
	for range finalizers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			finalizationErrors <- coordinator.Finalize(acquisition, func(CommandResultMetadata) (CommandResultMetadata, error) {
				return final, nil
			})
		}()
	}
	close(start)
	wg.Wait()
	close(finalizationErrors)

	successes := 0
	for finalizeErr := range finalizationErrors {
		if finalizeErr == nil {
			successes++
			continue
		}
		if !errors.Is(finalizeErr, ErrFinalizationProgress) && !errors.Is(finalizeErr, ErrAlreadyFinalized) {
			t.Fatalf("unexpected finalization error: %v", finalizeErr)
		}
	}
	if successes != 1 {
		t.Fatalf("successful finalizations = %d, want 1", successes)
	}
}
