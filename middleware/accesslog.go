package middleware

import (
	"net/http"
	"time"

	"gess-backend/internal/metrics"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rw *responseRecorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func statusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	default:
		return "other"
	}
}

// AccessLogMiddleware records Prometheus latency histograms and emits a structured access log line per request.
func AccessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		dur := time.Since(start)
		route := r.Pattern
		if route == "" {
			route = "unknown"
		}
		sc := statusClass(rw.status)
		metrics.ObserveRequest(r.Method, route, sc, dur)
		LoggerFromRequest(r).InfoContext(r.Context(), "http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"route", route,
			"status", rw.status,
			"status_class", sc,
			"duration_ms", dur.Milliseconds(),
		)
	})
}
