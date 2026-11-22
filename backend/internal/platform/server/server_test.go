package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDB struct {
	err error
}

func (f fakeDB) Ping(context.Context) error { return f.err }

type stubRegistrar struct {
	path   string
	status int
}

func (s stubRegistrar) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+s.path, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(s.status)
	})
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestHandler(t *testing.T, db DB, registrars ...Registrar) http.Handler {
	t.Helper()

	return New(Config{Addr: ":0"}, testLogger(), db, registrars...).http.Handler
}

func TestHealthz(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, fakeDB{err: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "ok" {
		t.Fatalf("body = %q, want ok", body)
	}
}

func TestReadyz(t *testing.T) {
	t.Parallel()

	t.Run("ready", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(t, fakeDB{})
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if body := rec.Body.String(); body != "ready" {
			t.Fatalf("body = %q, want ready", body)
		}
	})

	t.Run("database unavailable", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(t, fakeDB{err: errors.New("db down")})
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
		if body := rec.Body.String(); body != "db unavailable" {
			t.Fatalf("body = %q", body)
		}
	})
}

func TestNotFound(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, fakeDB{})
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestMetrics(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, fakeDB{})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", ct)
	}
}

func TestRegistrarRoutesAreMounted(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, fakeDB{}, stubRegistrar{path: "/ping", status: http.StatusNoContent})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}
