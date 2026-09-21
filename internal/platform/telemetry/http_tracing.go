package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// RequestTracingMiddleware creates the HTTP request span and records the
// approved request-duration metric. It records only the canonical route
// pattern exposed by net/http/chi, never the raw URL or query string.
func RequestTracingMiddleware(instrumentation *Instrumentation) func(http.Handler) http.Handler {
	if instrumentation == nil {
		panic("telemetry: nil instrumentation")
	}
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("telemetry: nil HTTP handler")
		}
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx, span := instrumentation.StartSpan(request.Context(), "http.request", SpanAttributes{
				Module:    "platform.http",
				Operation: "http_request",
				Method:    request.Method,
			})
			observed := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
			started := time.Now()
			completed := false
			defer func() {
				result := "success"
				errorCode := ""
				if !completed {
					result = "failure"
					errorCode = "http_panic"
				} else if observed.status >= http.StatusInternalServerError {
					result = "failure"
					errorCode = "http_server_error"
				} else if observed.status >= http.StatusBadRequest {
					result = "rejected"
					errorCode = "http_client_error"
				}
				fields := SpanAttributes{
					Module:      "platform.http",
					Operation:   "http_request",
					Result:      result,
					ErrorCode:   errorCode,
					Route:       requestRoute(request),
					Method:      request.Method,
					StatusClass: httpStatusClass(observed.status),
				}
				span.SetAttributes(instrumentation.spanAttributes(fields)...)
				instrumentation.ObserveHTTPRequest(ctx, time.Since(started), requestRoute(request), request.Method, httpStatusClass(observed.status))
				span.End()
			}()

			next.ServeHTTP(observed, request.WithContext(ctx))
			completed = true
		})
	}
}

func requestRoute(request *http.Request) string {
	if request == nil {
		return ""
	}
	if request.Pattern != "" {
		return request.Pattern
	}
	if routeContext := chi.RouteContext(request.Context()); routeContext != nil {
		return routeContext.RoutePattern()
	}
	return ""
}

func httpStatusClass(status int) string {
	if status < 100 || status > 599 {
		return "other"
	}
	return strconv.Itoa(status/100) + "xx"
}
