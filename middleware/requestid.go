package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// RequestIDMiddleware ensures X-Request-ID on the response and attaches a request-scoped slog.Logger to the context.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID(id) {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)

		args := []any{"request_id", id}
		if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
			args = append(args, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
		}
		logger := slog.Default().With(args...)
		ctx := context.WithValue(r.Context(), loggerContextKey, logger)
		ctx = context.WithValue(ctx, requestIDContextKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validRequestID(s string) bool {
	if len(s) < 8 || len(s) > 128 {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < 0x20 || b > 0x7e {
			return false
		}
	}
	return true
}

// LoggerFromRequest returns the contextual logger or slog.Default().
func LoggerFromRequest(r *http.Request) *slog.Logger {
	if r == nil {
		return slog.Default()
	}
	if v := r.Context().Value(loggerContextKey); v != nil {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}

// RequestIDFromContext returns the request ID if present.
func RequestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(requestIDContextKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
