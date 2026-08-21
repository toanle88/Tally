package idempotency

import (
	"errors"
	"sync"
	"testing"
)

func coordinatorResult(t *testing.T, identity IdempotencyIdentity, fingerprint Fingerprint, state ResultState) CommandResultMetadata {
	t.Helper()
	status := 200
	result, err := NewCommandResultMetadata(identity, fingerprint, "payments.submit", state, &status, []byte(`{"reference":"payment-123"}`), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestMemoryCoordinatorReturnsEstablishedResultsForIdenticalRetries(t *testing.T) {
	for _, state := range []ResultState{StateInProgress, StateEstablished, StateFailed} {
		t.Run(string(state), func(t *testing.T) {
			coordinator := NewMemoryCoordinator()
			identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
			if err != nil {
				t.Fatal(err)
			}
			first, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
			if err != nil || first.Decision() != DecisionExecute || first.Token() == "" {
				t.Fatalf("first acquisition = %#v, %v", first, err)
			}
			if state != StateInProgress {
				final := coordinatorResult(t, identity, "sha256:payment", state)
				if err := coordinator.Finalize(first, func(CommandResultMetadata) (CommandResultMetadata, error) { return final, nil }); err != nil {
					t.Fatal(err)
				}
			}
			retry, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
			if err != nil || retry.Decision() != DecisionReturn || retry.Result().State() != state {
				t.Fatalf("retry = %#v, %v; want return %s", retry, err, state)
			}
		})
	}
}

func TestMemoryCoordinatorFinalizationReturnsTerminalResult(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	final := coordinatorResult(t, identity, "sha256:payment", StateEstablished)
	var callbackCalls int
	if err := coordinator.Finalize(acquisition, func(input CommandResultMetadata) (CommandResultMetadata, error) {
		callbackCalls++
		if input.State() != StateInProgress {
			t.Fatalf("callback input state = %s", input.State())
		}
		return final, nil
	}); err != nil {
		t.Fatal(err)
	}
	if callbackCalls != 1 {
		t.Fatalf("callback calls = %d, want 1", callbackCalls)
	}
	retry, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
	if err != nil || retry.Decision() != DecisionReturn || retry.Result().State() != StateEstablished {
		t.Fatalf("retry = %#v, %v", retry, err)
	}
	if err := coordinator.Finalize(acquisition, func(CommandResultMetadata) (CommandResultMetadata, error) { return final, nil }); !errors.Is(err, ErrAlreadyFinalized) {
		t.Fatalf("duplicate finalization error = %v", err)
	}
}

func TestMemoryCoordinatorCallbackFailureAndPanicRemainInProgress(t *testing.T) {
	for _, test := range []struct {
		name     string
		callback func(CommandResultMetadata) (CommandResultMetadata, error)
		want     error
	}{
		{name: "error", callback: func(CommandResultMetadata) (CommandResultMetadata, error) {
			return CommandResultMetadata{}, errors.New("rollback")
		}, want: nil},
		{name: "panic", callback: func(CommandResultMetadata) (CommandResultMetadata, error) { panic("boom") }, want: ErrCommitPanic},
	} {
		t.Run(test.name, func(t *testing.T) {
			coordinator := NewMemoryCoordinator()
			identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
			if err != nil {
				t.Fatal(err)
			}
			acquisition, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
			if err != nil {
				t.Fatal(err)
			}
			err = coordinator.Finalize(acquisition, test.callback)
			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if test.want == nil && err == nil {
				t.Fatal("callback error was lost")
			}
			retry, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
			if err != nil || retry.Decision() != DecisionReturn || retry.Result().State() != StateInProgress {
				t.Fatalf("retry = %#v, %v", retry, err)
			}
		})
	}
}

func TestMemoryCoordinatorRetriesSeeInProgressDuringFinalization(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	final := coordinatorResult(t, identity, "sha256:payment", StateEstablished)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := coordinator.Finalize(acquisition, func(CommandResultMetadata) (CommandResultMetadata, error) {
			close(started)
			<-release
			return final, nil
		}); err != nil {
			t.Errorf("finalize: %v", err)
		}
	}()
	<-started
	retry, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
	if err != nil || retry.Decision() != DecisionReturn || retry.Result().State() != StateInProgress {
		t.Fatalf("retry = %#v, %v", retry, err)
	}
	close(release)
	wg.Wait()
}

func TestMemoryCoordinatorRejectsMismatchAndInvalidFinalization(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.Acquire(identity, "sha256:other", "payments.submit"); !errors.Is(err, ErrFingerprintMismatch) {
		t.Fatalf("mismatch error = %v", err)
	}
	final := coordinatorResult(t, identity, "sha256:payment", StateEstablished)
	if err := coordinator.Finalize(acquisition, func(CommandResultMetadata) (CommandResultMetadata, error) {
		return coordinatorResult(t, identity, "sha256:payment", StateInProgress), nil
	}); !errors.Is(err, ErrInvalidTerminalState) {
		t.Fatalf("invalid terminal error = %v", err)
	}
	otherID, err := NewIdentity(metadataScope(t, ""), "payment-other")
	if err != nil {
		t.Fatal(err)
	}
	other, err := coordinator.Acquire(otherID, "sha256:other", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Finalize(other, func(CommandResultMetadata) (CommandResultMetadata, error) { return final, nil }); !errors.Is(err, ErrInvalidTerminalState) {
		t.Fatalf("foreign result error = %v", err)
	}
	if err := coordinator.Finalize(Acquisition{}, func(CommandResultMetadata) (CommandResultMetadata, error) { return final, nil }); !errors.Is(err, ErrInvalidAcquisition) {
		t.Fatalf("invalid acquisition error = %v", err)
	}
}

func TestMemoryCoordinatorConcurrentAcquisitionHasOneOwner(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	const callers = 32
	decisions := make(chan AcquisitionDecision, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acquisition, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			decisions <- acquisition.Decision()
		}()
	}
	wg.Wait()
	close(decisions)
	var owners int
	for decision := range decisions {
		if decision == DecisionExecute {
			owners++
		}
	}
	if owners != 1 {
		t.Fatalf("execution owners = %d, want 1", owners)
	}
}

func TestAcquisitionResultIsDefensivelyCopied(t *testing.T) {
	coordinator := NewMemoryCoordinator()
	identity, err := NewIdentity(metadataScope(t, ""), "payment-123")
	if err != nil {
		t.Fatal(err)
	}
	first, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	if first.Result().ResultBody() != nil {
		t.Fatal("new acquisition unexpectedly contained a result body")
	}
	final := coordinatorResult(t, identity, "sha256:payment", StateEstablished)
	if err := coordinator.Finalize(first, func(CommandResultMetadata) (CommandResultMetadata, error) { return final, nil }); err != nil {
		t.Fatal(err)
	}
	retry, err := coordinator.Acquire(identity, "sha256:payment", "payments.submit")
	if err != nil {
		t.Fatal(err)
	}
	body := retry.Result().ResultBody()
	body[0] = '['
	if string(retry.Result().ResultBody()) != `{"reference":"payment-123"}` {
		t.Fatal("result body was not defensively copied")
	}
}
