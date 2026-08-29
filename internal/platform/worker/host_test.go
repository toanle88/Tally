package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewHostValidatesBudgetsAndNamespaces(t *testing.T) {
	worker := func(context.Context, WorkerRuntime) error { return nil }
	valid := WorkerSpec{Name: "one", Run: worker, Concurrency: 2, PoolBudget: 2, ShutdownTimeout: time.Second, MetricsNamespace: "platform.one"}
	if _, err := NewHost([]WorkerSpec{valid, {Name: "two", Run: worker, Concurrency: 1, PoolBudget: 1, ShutdownTimeout: time.Second, MetricsNamespace: "platform.two"}}, 3); err != nil {
		t.Fatalf("NewHost valid configuration error = %v", err)
	}
	if _, err := NewHost([]WorkerSpec{valid, {Name: "two", Run: worker, Concurrency: 1, PoolBudget: 2, ShutdownTimeout: time.Second, MetricsNamespace: "platform.two"}}, 3); !errors.Is(err, ErrInvalidHost) {
		t.Fatalf("budget error = %v, want ErrInvalidHost", err)
	}
	if _, err := NewHost([]WorkerSpec{valid, {Name: "two", Run: worker, Concurrency: 1, PoolBudget: 1, ShutdownTimeout: time.Second, MetricsNamespace: valid.MetricsNamespace}}, 3); !errors.Is(err, ErrInvalidHost) {
		t.Fatalf("namespace error = %v, want ErrInvalidHost", err)
	}
}

func TestHostProvidesDeclaredConcurrencyAdmission(t *testing.T) {
	host, err := NewHost([]WorkerSpec{{
		Name: "bounded", Run: func(context.Context, WorkerRuntime) error { return nil },
		Concurrency: 1, PoolBudget: 2, ShutdownTimeout: time.Second, MetricsNamespace: "platform.bounded",
	}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := host.runtimes[0].ConcurrencyAdmission.Capacity(); got != 1 {
		t.Fatalf("concurrency admission capacity = %d, want 1", got)
	}
	if got := host.runtimes[0].DBAdmission.Capacity(); got != 2 {
		t.Fatalf("database admission capacity = %d, want 2", got)
	}
}

func TestHostCancelsPeersAfterWorkerFailure(t *testing.T) {
	workerErr := errors.New("worker failed")
	peerStopped := make(chan struct{})
	host, err := NewHost([]WorkerSpec{
		{Name: "failing", Run: func(context.Context, WorkerRuntime) error { return workerErr }, Concurrency: 1, PoolBudget: 1, ShutdownTimeout: time.Second, MetricsNamespace: "platform.failing"},
		{Name: "peer", Run: func(ctx context.Context, _ WorkerRuntime) error { <-ctx.Done(); close(peerStopped); return nil }, Concurrency: 1, PoolBudget: 1, ShutdownTimeout: time.Second, MetricsNamespace: "platform.peer"},
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Run(context.Background()); !errors.Is(err, workerErr) {
		t.Fatalf("host error = %v, want worker error", err)
	}
	select {
	case <-peerStopped:
	case <-time.After(time.Second):
		t.Fatal("peer was not cancelled")
	}
}

func TestHostCancellationIsGraceful(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	host, err := NewHost([]WorkerSpec{{
		Name: "blocking", Run: func(ctx context.Context, runtime WorkerRuntime) error {
			calls.Add(1)
			if err := runtime.DBAdmission.Acquire(ctx); err != nil {
				return err
			}
			runtime.DBAdmission.Release()
			<-ctx.Done()
			return ctx.Err()
		}, Concurrency: 1, PoolBudget: 1, ShutdownTimeout: time.Second, MetricsNamespace: "platform.blocking",
	}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	if err := host.Run(ctx); err != nil {
		t.Fatalf("host cancellation error = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("worker calls = %d, want 1", calls.Load())
	}
}

func TestHostReturnsBoundedShutdownTimeout(t *testing.T) {
	host, err := NewHost([]WorkerSpec{{
		Name: "stubborn", Run: func(context.Context, WorkerRuntime) error { select {} }, Concurrency: 1, PoolBudget: 1, ShutdownTimeout: 10 * time.Millisecond, MetricsNamespace: "platform.stubborn",
	}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	started := time.Now()
	err = host.Run(ctx)
	if !errors.Is(err, ErrShutdownTimeout) {
		t.Fatalf("host error = %v, want shutdown timeout", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("shutdown was not bounded")
	}
}
