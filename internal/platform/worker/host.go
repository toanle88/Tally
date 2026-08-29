package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const DefaultShutdownTimeout = 10 * time.Second

var (
	ErrInvalidHost     = errors.New("invalid worker host")
	ErrWorkerExited    = errors.New("worker exited unexpectedly")
	ErrShutdownTimeout = errors.New("worker shutdown timed out")
)

// Admission bounds the number of database-consuming operations assigned to a
// worker. The host allocates one admission object per worker.
type Admission interface {
	Acquire(context.Context) error
	Release()
	Capacity() int
}

type semaphore struct {
	slots chan struct{}
}

func NewAdmission(capacity int) (Admission, error) {
	if capacity < 1 {
		return nil, fmt.Errorf("%w: admission capacity must be positive", ErrInvalidHost)
	}
	return newSemaphore(capacity), nil
}

func newSemaphore(capacity int) *semaphore {
	return &semaphore{slots: make(chan struct{}, capacity)}
}

func (s *semaphore) Acquire(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidHost
	}
	select {
	case s.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *semaphore) Release() { <-s.slots }

func (s *semaphore) Capacity() int { return cap(s.slots) }

type WorkerRuntime struct {
	DBAdmission          Admission
	ConcurrencyAdmission Admission
	MetricsNamespace     string
}

type WorkerFunc func(context.Context, WorkerRuntime) error

type WorkerSpec struct {
	Name             string
	Run              WorkerFunc
	Concurrency      int
	PoolBudget       int
	ShutdownTimeout  time.Duration
	MetricsNamespace string
}

type Host struct {
	specs    []WorkerSpec
	runtimes []WorkerRuntime
}

func NewHost(specs []WorkerSpec, dbMaxConns int) (*Host, error) {
	if len(specs) == 0 || dbMaxConns < 1 {
		return nil, ErrInvalidHost
	}

	seenNames := make(map[string]struct{}, len(specs))
	seenNamespaces := make(map[string]struct{}, len(specs))
	runtimes := make([]WorkerRuntime, len(specs))
	totalBudget := 0
	for index, spec := range specs {
		if !validText(spec.Name) || !validText(spec.MetricsNamespace) || spec.Run == nil ||
			spec.Concurrency < 1 || spec.PoolBudget < 1 || spec.ShutdownTimeout <= 0 {
			return nil, fmt.Errorf("%w: invalid worker %q", ErrInvalidHost, spec.Name)
		}
		if _, exists := seenNames[spec.Name]; exists {
			return nil, fmt.Errorf("%w: duplicate worker name %q", ErrInvalidHost, spec.Name)
		}
		if _, exists := seenNamespaces[spec.MetricsNamespace]; exists {
			return nil, fmt.Errorf("%w: duplicate metrics namespace %q", ErrInvalidHost, spec.MetricsNamespace)
		}
		seenNames[spec.Name] = struct{}{}
		seenNamespaces[spec.MetricsNamespace] = struct{}{}
		if spec.PoolBudget > dbMaxConns-totalBudget {
			return nil, fmt.Errorf("%w: worker pool budgets exceed database maximum", ErrInvalidHost)
		}
		totalBudget += spec.PoolBudget
		runtimes[index] = WorkerRuntime{
			DBAdmission:          newSemaphore(spec.PoolBudget),
			ConcurrencyAdmission: newSemaphore(spec.Concurrency),
			MetricsNamespace:     spec.MetricsNamespace,
		}
	}

	return &Host{specs: append([]WorkerSpec(nil), specs...), runtimes: runtimes}, nil
}

// Run starts all workers and returns after every worker stops or a bounded
// shutdown deadline is reached.
func (h *Host) Run(ctx context.Context) error {
	if h == nil || ctx == nil || len(h.specs) == 0 || len(h.specs) != len(h.runtimes) {
		return ErrInvalidHost
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan error, len(h.specs))
	for index, spec := range h.specs {
		index, spec := index, spec
		go func() {
			err := spec.Run(runCtx, h.runtimes[index])
			if err == nil && runCtx.Err() == nil {
				err = fmt.Errorf("%w: %s", ErrWorkerExited, spec.Name)
			} else if err != nil && runCtx.Err() == nil &&
				(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				err = fmt.Errorf("%w: %s: %v", ErrWorkerExited, spec.Name, err)
			}
			results <- err
		}()
	}

	remaining := len(h.specs)
	var firstErr error
	var shutdownTimer *time.Timer
	var shutdown <-chan time.Time
	beginShutdown := func() {
		if shutdownTimer != nil {
			return
		}
		cancel()
		maxTimeout := h.specs[0].ShutdownTimeout
		for _, spec := range h.specs[1:] {
			if spec.ShutdownTimeout > maxTimeout {
				maxTimeout = spec.ShutdownTimeout
			}
		}
		shutdownTimer = time.NewTimer(maxTimeout)
		shutdown = shutdownTimer.C
	}

	for remaining > 0 {
		select {
		case err := <-results:
			remaining--
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && firstErr == nil {
				firstErr = err
				beginShutdown()
			}
		case <-ctx.Done():
			beginShutdown()
		case <-shutdown:
			if shutdownTimer != nil {
				shutdownTimer.Stop()
			}
			return errors.Join(firstErr, ErrShutdownTimeout)
		}
	}
	if shutdownTimer != nil {
		shutdownTimer.Stop()
	}
	return firstErr
}

func validText(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\r\n\t")
}

var _ Admission = (*semaphore)(nil)
