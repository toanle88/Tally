package authentication

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

type MiddlewareConfig struct {
	Validator *Validator
	Resolver  identity.UserResolver
	Logger    *telemetry.Logger
}

func NewMiddleware(config MiddlewareConfig) func(http.Handler) http.Handler {
	if config.Validator == nil {
		panic("authentication: validator is required")
	}
	if config.Resolver == nil {
		panic("authentication: resolver is required")
	}
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("authentication: nil HTTP handler")
		}
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			token, ok := bearerToken(request.Header.Get("Authorization"))
			if !ok {
				writeProblem(writer, request.Context(), http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication is required.")
				logFailure(config.Logger, request.Context(), "authentication_required", false)
				return
			}

			subject, err := config.Validator.ValidateAccessToken(request.Context(), token)
			if err != nil {
				if errors.Is(err, ErrAuthenticationDependency) {
					writeProblem(writer, request.Context(), http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "Authentication is temporarily unavailable.")
					logFailure(config.Logger, request.Context(), "authentication_dependency_unavailable", true)
					return
				}
				writeProblem(writer, request.Context(), http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication is required.")
				logFailure(config.Logger, request.Context(), "authentication_required", false)
				return
			}

			actor, err := config.Resolver.ResolveApplicationActor(request.Context(), subject)
			if err != nil {
				if errors.Is(err, identity.ErrActorResolverUnavailable) {
					writeProblem(writer, request.Context(), http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "Authentication is temporarily unavailable.")
					logFailure(config.Logger, request.Context(), "actor_resolution_unavailable", true)
					return
				}
				writeProblem(writer, request.Context(), http.StatusForbidden, "AUTHORIZATION_DENIED", "The request is not authorized.")
				logFailure(config.Logger, request.Context(), "application_actor_denied", false)
				return
			}

			ctx, err := identity.WithActor(request.Context(), actor)
			if err != nil {
				writeProblem(writer, request.Context(), http.StatusForbidden, "AUTHORIZATION_DENIED", "The request is not authorized.")
				logFailure(config.Logger, request.Context(), "application_actor_denied", false)
				return
			}
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

type problemDetails struct {
	Type          string    `json:"type"`
	Title         string    `json:"title"`
	Status        int       `json:"status"`
	Code          string    `json:"code"`
	Detail        string    `json:"detail"`
	CorrelationID uuid.UUID `json:"correlationId"`
}

func writeProblem(writer http.ResponseWriter, ctx context.Context, status int, code, detail string) {
	correlationID := uuid.New()
	if telemetryContext, ok := telemetry.FromContext(ctx); ok && telemetryContext.CorrelationID != uuid.Nil {
		correlationID = telemetryContext.CorrelationID
	}
	if status == http.StatusUnauthorized {
		writer.Header().Set("WWW-Authenticate", "Bearer")
	}
	writer.Header().Set("Content-Type", "application/problem+json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(problemDetails{
		Type:          "https://tally.example/problems/" + strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		Title:         detail,
		Status:        status,
		Code:          code,
		Detail:        detail,
		CorrelationID: correlationID,
	})
}

func logFailure(logger *telemetry.Logger, ctx context.Context, code string, retryable bool) {
	if logger == nil {
		return
	}
	logger.Emit(ctx, levelForAuthenticationFailure(retryable), telemetry.Event{
		Message:   "authentication_boundary_rejected",
		Module:    "platform.authentication",
		Operation: "authenticate_request",
		Result:    "rejected",
		ErrorCode: code,
		Retryable: retryable,
	})
}

func levelForAuthenticationFailure(retryable bool) slog.Level {
	if retryable {
		return slog.LevelError
	}
	return slog.LevelWarn
}
