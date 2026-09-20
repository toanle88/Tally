package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/toanle88/Tally/internal/platform/database"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
	"github.com/toanle88/Tally/internal/platform/integration"
	"github.com/toanle88/Tally/internal/platform/telemetry"
	platformworker "github.com/toanle88/Tally/internal/platform/worker"
)

const (
	defaultDBMaxConnections = int32(20)
	defaultOutboxBatchSize  = 100
	defaultPollInterval     = time.Second
	workerLeaseDuration     = 30 * time.Second
	workerHandlerP99        = 10 * time.Second
)

var ErrNoRegisteredConsumers = errors.New("no registered integration consumers")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger, err := telemetry.NewLogger(telemetry.LoggerConfig{
		Service: "tally-worker",
		Writer:  os.Stderr,
	})
	if err != nil {
		os.Exit(1)
	}
	if err := run(ctx, logger); err != nil {
		logWorkerStoppedWithError(logger, ctx, err)
		os.Exit(1)
	}
}

func logWorkerStoppedWithError(logger *telemetry.Logger, ctx context.Context, err error) {
	logger.Emit(ctx, slog.LevelError, telemetry.Event{
		Message:   "worker_stopped",
		Module:    "platform.worker",
		Operation: "run",
		Result:    "failure",
		ErrorCode: workerErrorCode(err),
	})
}

func run(ctx context.Context, logger *telemetry.Logger) error {
	if logger == nil {
		return errors.New("logger is required")
	}
	registrations := registeredConsumers()
	if len(registrations) == 0 {
		return ErrNoRegisteredConsumers
	}

	dbMaxConnections, err := envInt32("DB_MAX_CONNS", defaultDBMaxConnections)
	if err != nil {
		return err
	}
	batchSize, err := envInt("OUTBOX_BATCH_SIZE", defaultOutboxBatchSize)
	if err != nil {
		return err
	}
	pollInterval, err := envDuration("OUTBOX_POLL_INTERVAL", defaultPollInterval)
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, database.Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		MaxConnections: dbMaxConnections,
		ConnectTimeout: 10 * time.Second,
	})
	if err != nil {
		return err
	}
	defer pool.Close()
	queries := platformdb.New(pool)

	dispatcherRegistrations := make([]integration.HandlerRegistration, len(registrations))
	copy(dispatcherRegistrations, registrations)
	concurrency := int(dbMaxConnections)
	if concurrency > 100 {
		concurrency = 100
	}
	host, err := platformworker.NewHost([]platformworker.WorkerSpec{{
		Name:             "outbox-dispatcher",
		Concurrency:      concurrency,
		PoolBudget:       int(dbMaxConnections),
		ShutdownTimeout:  platformworker.DefaultShutdownTimeout,
		MetricsNamespace: "platform.worker.outbox_dispatcher",
		Run: func(ctx context.Context, runtime platformworker.WorkerRuntime) error {
			dispatcher, err := integration.NewDispatcher(queries, dispatcherRegistrations, integration.DispatcherConfig{
				LeaseDuration:        workerLeaseDuration,
				HandlerP99Duration:   workerHandlerP99,
				BatchSize:            batchSize,
				Concurrency:          concurrency,
				Admission:            runtime.DBAdmission,
				ConcurrencyAdmission: runtime.ConcurrencyAdmission,
			})
			if err != nil {
				return err
			}
			return dispatcher.Run(ctx, pollInterval)
		},
	}}, int(dbMaxConnections))
	if err != nil {
		return err
	}
	logger.Emit(ctx, slog.LevelInfo, telemetry.Event{
		Message:   "worker_started",
		Module:    "platform.worker",
		Operation: "run",
		Result:    "started",
	})
	err = host.Run(ctx)
	if err == nil {
		logger.Emit(ctx, slog.LevelInfo, telemetry.Event{
			Message:   "worker_stopped",
			Module:    "platform.worker",
			Operation: "run",
			Result:    "success",
		})
	}
	return err
}

// The platform currently has no finance capability consumers. Capability
// packages will provide registrations when they are delivered.
func registeredConsumers() []integration.HandlerRegistration { return nil }

func workerErrorCode(err error) string {
	if errors.Is(err, ErrNoRegisteredConsumers) {
		return "no_registered_consumers"
	}
	if errors.Is(err, platformworker.ErrShutdownTimeout) {
		return "worker_shutdown_timeout"
	}
	return "worker_run_failed"
}

func envInt(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return parsed, nil
}

func envInt32(name string, fallback int32) (int32, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed < 2 || parsed > 200 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return int32(parsed), nil
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return parsed, nil
}
