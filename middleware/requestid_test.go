package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	var got string
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)
	if got == "" {
		t.Fatal("expected non-empty request id in context")
	}
	if rec.Header().Get("X-Request-ID") != got {
		t.Fatalf("response header X-Request-ID=%q, context id=%q", rec.Header().Get("X-Request-ID"), got)
	}
}

func TestRequestIDMiddleware_PassthroughValidHeader(t *testing.T) {
	const want = "abcdefgh-valid"
	var got string
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", want)
	h.ServeHTTP(rec, req)
	if got != want {
		t.Fatalf("context id=%q, want %q", got, want)
	}
	if rec.Header().Get("X-Request-ID") != want {
		t.Fatalf("response header=%q, want %q", rec.Header().Get("X-Request-ID"), want)
	}
}

func TestRequestIDMiddleware_InvalidHeaderGeneratesNew(t *testing.T) {
	var got string
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "short") // too short
	h.ServeHTTP(rec, req)
	if got == "" || got == "short" {
		t.Fatalf("expected generated uuid, got %q", got)
	}
	if !strings.Contains(rec.Header().Get("X-Request-ID"), "-") {
		t.Fatalf("expected uuid-like response header, got %q", rec.Header().Get("X-Request-ID"))
	}
}
