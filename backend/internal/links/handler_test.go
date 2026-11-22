package links

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newLinksMux(t *testing.T, repo Repository) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	NewHandler(NewService(repo, "http://short"), testLogger()).Register(mux)
	return mux
}

func TestHandlerCreate(t *testing.T) {
	t.Parallel()

	t.Run("created", func(t *testing.T) {
		t.Parallel()

		mux := newLinksMux(t, &fakeRepo{})
		req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"url":"https://example.com"}`))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", ct)
		}

		var res CreateResult
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(res.Code) != codeLength {
			t.Fatalf("code = %q, want length %d", res.Code, codeLength)
		}
		if res.URL != "https://example.com" {
			t.Fatalf("url = %q", res.URL)
		}
		if res.ShortURL != "http://short/r/"+res.Code {
			t.Fatalf("short_url = %q", res.ShortURL)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		mux := newLinksMux(t, &fakeRepo{})
		req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"url":`))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assertErrorStatus(t, rec, http.StatusBadRequest, "invalid json")
	})

	t.Run("invalid url", func(t *testing.T) {
		t.Parallel()

		mux := newLinksMux(t, &fakeRepo{})
		req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"url":"nope"}`))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assertErrorStatus(t, rec, http.StatusBadRequest, "invalid url")
	})

	t.Run("repository failure", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepo{createErr: func(string) error { return errors.New("boom") }}
		mux := newLinksMux(t, repo)
		req := httptest.NewRequest(http.MethodPost, "/api/links", strings.NewReader(`{"url":"https://example.com"}`))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assertErrorStatus(t, rec, http.StatusInternalServerError, "internal error")
	})
}

func TestHandlerList(t *testing.T) {
	t.Parallel()

	t.Run("default limit", func(t *testing.T) {
		t.Parallel()

		gotLimit := 0
		repo := &fakeRepo{
			listFn: func(limit int) ([]Link, error) {
				gotLimit = limit
				return []Link{}, nil
			},
		}
		mux := newLinksMux(t, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if gotLimit != 100 {
			t.Fatalf("limit = %d, want 100", gotLimit)
		}
		if body := strings.TrimSpace(rec.Body.String()); body != `{"items":[]}` {
			t.Fatalf("body = %q, want empty items array", body)
		}
	})

	t.Run("explicit limit", func(t *testing.T) {
		t.Parallel()

		gotLimit := 0
		repo := &fakeRepo{
			listFn: func(limit int) ([]Link, error) {
				gotLimit = limit
				return []Link{}, nil
			},
		}
		mux := newLinksMux(t, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/links?limit=5", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if gotLimit != 5 {
			t.Fatalf("limit = %d, want 5", gotLimit)
		}
	})

	t.Run("invalid limit falls back to default", func(t *testing.T) {
		t.Parallel()

		gotLimit := 0
		repo := &fakeRepo{
			listFn: func(limit int) ([]Link, error) {
				gotLimit = limit
				return []Link{}, nil
			},
		}
		mux := newLinksMux(t, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/links?limit=abc", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if gotLimit != 100 {
			t.Fatalf("limit = %d, want 100", gotLimit)
		}
	})

	t.Run("repository failure", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepo{
			listFn: func(int) ([]Link, error) { return nil, errors.New("boom") },
		}
		mux := newLinksMux(t, repo)
		req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assertErrorStatus(t, rec, http.StatusInternalServerError, "internal error")
	})
}

func assertErrorStatus(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()

	if rec.Code != status {
		t.Fatalf("status = %d, want %d", rec.Code, status)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != message {
		t.Fatalf("error = %q, want %q", body["error"], message)
	}
}
