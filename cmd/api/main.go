package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/toanle88/Tally/internal/platform/authentication"
	"github.com/toanle88/Tally/internal/platform/httpx"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

const (
	defaultHTTPAddress = ":8080"
	shutdownTimeout    = 10 * time.Second
)

func main() {
	logger, err := telemetry.NewLogger(telemetry.LoggerConfig{
		Service: "tally-api",
		Writer:  os.Stderr,
	})
	if err != nil {
		os.Exit(1)
	}
	if err := run(logger); err != nil {
		logAPIStoppedWithError(logger, err)
		os.Exit(1)
	}
}

func logAPIStoppedWithError(logger *telemetry.Logger, _ error) {
	logger.Emit(context.Background(), slog.LevelError, telemetry.Event{
		Message:   "api_stopped_with_error",
		Module:    "platform.http",
		Operation: "serve",
		Result:    "failure",
		ErrorCode: "api_stopped_with_error",
	})
}

func run(logger *telemetry.Logger) error {
	if logger == nil {
		return errors.New("logger is required")
	}
	instrumentation, err := telemetry.NewInstrumentation(telemetry.InstrumentationConfig{Service: "tally-api"})
	if err != nil {
		return err
	}
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := instrumentation.Shutdown(shutdownContext); err != nil {
			logger.Emit(context.Background(), slog.LevelWarn, telemetry.Event{
				Message:   "telemetry_shutdown_failed",
				Module:    "platform.telemetry",
				Operation: "shutdown",
				Result:    "failure",
				ErrorCode: "telemetry_shutdown_failed",
			})
		}
	}()
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = defaultHTTPAddress
	}

	router := newRouter(instrumentation, logger, os.Getenv)

	server := &http.Server{
		Addr:              address,
		Handler:           telemetry.Middleware(router),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	listenError := make(chan error, 1)

	go func() {
		logger.Emit(context.Background(), slog.LevelInfo, telemetry.Event{
			Message:   "api_listening",
			Module:    "platform.http",
			Operation: "serve",
			Result:    "started",
		})
		listenError <- server.ListenAndServe()
	}()

	select {
	case err := <-listenError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("listen and serve: %w", err)
	case <-shutdownSignal.Done():
		logger.Emit(context.Background(), slog.LevelInfo, telemetry.Event{
			Message:   "api_shutdown_requested",
			Module:    "platform.http",
			Operation: "shutdown",
			Result:    "started",
		})
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	err = <-listenError
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped: %w", err)
	}

	logger.Emit(context.Background(), slog.LevelInfo, telemetry.Event{
		Message:   "api_stopped_gracefully",
		Module:    "platform.http",
		Operation: "shutdown",
		Result:    "success",
	})
	return nil
}

func newRouter(instrumentation *telemetry.Instrumentation, logger *telemetry.Logger, getenv func(string) string) http.Handler {
	router := chi.NewRouter()
	router.Use(telemetry.RequestTracingMiddleware(instrumentation))
	router.Use(telemetry.RequestLoggingMiddleware(logger))
	// Liveness is intentionally anonymous so orchestration can distinguish a
	// running process from an authentication-provider outage.
	router.Get("/health/live", httpx.Liveness)

	// The generated finance API is mounted under this boundary as operations
	// arrive. There is no login or /me endpoint in this story.
	protectedAPI := authentication.NewMiddlewareFromEnvironment(getenv, logger)
	router.Mount("/api/v1", protectedAPI(http.NotFoundHandler()))
	return router
}
