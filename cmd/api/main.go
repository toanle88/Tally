package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/authentication"
	"github.com/toanle88/Tally/internal/platform/database"
	"github.com/toanle88/Tally/internal/platform/httpx"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

const (
	defaultHTTPAddress            = ":8080"
	shutdownTimeout               = 10 * time.Second
	defaultDatabaseMaxConnections = int32(20)
	databaseConnectTimeout        = 10 * time.Second
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

	router, closeRuntime, err := newRuntimeRouter(context.Background(), instrumentation, logger, os.Getenv, apiRuntimeDependencies{})
	if err != nil {
		return err
	}
	defer closeRuntime()

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

type apiRuntimeDependencies struct {
	IdentityAuditWriter identity.PostgresAuditWriter
}

func newRuntimeRouter(ctx context.Context, instrumentation *telemetry.Instrumentation, logger *telemetry.Logger, getenv func(string) string, dependencies apiRuntimeDependencies) (http.Handler, func(), error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if strings.TrimSpace(getenv("DATABASE_URL")) == "" {
		return newRouter(instrumentation, logger, getenv), func() {}, nil
	}
	if dependencies.IdentityAuditWriter == nil {
		return nil, func() {}, errors.New("identity audit integration is required when DATABASE_URL is configured")
	}
	pool, err := database.Open(ctx, database.Config{
		DatabaseURL:    getenv("DATABASE_URL"),
		MaxConnections: defaultDatabaseMaxConnections,
		ConnectTimeout: databaseConnectTimeout,
	})
	if err != nil {
		return nil, func() {}, fmt.Errorf("open API database: %w", err)
	}
	closeRuntime := pool.Close
	repository, err := identity.NewPostgresUserRepositoryWithAudit(pool, dependencies.IdentityAuditWriter)
	if err != nil {
		closeRuntime()
		return nil, func() {}, fmt.Errorf("construct identity repository: %w", err)
	}
	roleRepository, err := identity.NewPostgresRoleRepositoryWithAudit(pool, postgresRoleAuditWriter(dependencies.IdentityAuditWriter))
	if err != nil {
		closeRuntime()
		return nil, func() {}, fmt.Errorf("construct identity role repository: %w", err)
	}
	identityServer := newIdentityAPIServerWithPostgresRepository(getenv, pool, repository, roleRepository)
	return newRouterWithIdentityServer(instrumentation, logger, getenv, repository, identityServer), closeRuntime, nil
}

func newRouter(instrumentation *telemetry.Instrumentation, logger *telemetry.Logger, getenv func(string) string) http.Handler {
	identityRepository := identity.NewMemoryUserRepository()
	return newRouterWithIdentityServer(
		instrumentation, logger, getenv, identityRepository,
		newIdentityAPIServerWithRepository(getenv, identityRepository),
	)
}

func newRouterWithIdentityServer(instrumentation *telemetry.Instrumentation, logger *telemetry.Logger, getenv func(string) string, identityRepository identity.UserRepository, identityServer http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(telemetry.RequestTracingMiddleware(instrumentation))
	router.Use(telemetry.RequestLoggingMiddleware(logger))
	// Liveness is intentionally anonymous so orchestration can distinguish a
	// running process from an authentication-provider outage.
	router.Get("/health/live", httpx.Liveness)

	// The generated finance API is mounted under this boundary as operations
	// arrive. There is no login or /me endpoint in this story.
	protectedAPI := authentication.NewMiddlewareFromEnvironmentWithUserRepository(getenv, logger, identityRepository)
	router.Mount("/api/v1", protectedAPI(identityServer))
	return router
}
