package redirect

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"urlshortener/internal/links"
)

type fakeLinkRepo struct {
	getCalls int
	getFn    func(code string) (links.Link, error)
}

func (f *fakeLinkRepo) Create(context.Context, string, string) (links.Link, error) {
	return links.Link{}, errors.New("not implemented")
}

func (f *fakeLinkRepo) GetByCode(_ context.Context, code string) (links.Link, error) {
	f.getCalls++
	if f.getFn == nil {
		return links.Link{}, links.ErrNotFound
	}
	return f.getFn(code)
}

func (f *fakeLinkRepo) List(context.Context, int) ([]links.Link, error) {
	return nil, errors.New("not implemented")
}

type cacheSet struct {
	code string
	url  string
	ttl  time.Duration
}

type fakeCache struct {
	getFn    func(code string) (string, bool, error)
	setCalls []cacheSet
	setErr   error
}

func (f *fakeCache) Get(_ context.Context, code string) (string, bool, error) {
	if f.getFn == nil {
		return "", false, nil
	}
	return f.getFn(code)
}

func (f *fakeCache) Set(_ context.Context, code, url string, ttl time.Duration) error {
	f.setCalls = append(f.setCalls, cacheSet{code: code, url: url, ttl: ttl})
	return f.setErr
}

type fakeRecorder struct {
	calls []string
	err   error
}

func (f *fakeRecorder) Record(_ context.Context, code string) error {
	f.calls = append(f.calls, code)
	return f.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newRedirectMux(t *testing.T, repo *fakeLinkRepo, cache Cache, rec Recorder) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	NewHandler(links.NewService(repo, "http://short"), cache, rec, testLogger()).Register(mux)
	return mux
}

func TestValidCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
		ok   bool
	}{
		{name: "generated shape", code: "Ab3dE9f", ok: true},
		{name: "digits", code: "1234567", ok: true},
		{name: "single char", code: "a", ok: true},
		{name: "max length", code: "0123456789abcdef", ok: true},
		{name: "empty", code: "", ok: false},
		{name: "too long", code: "0123456789abcdefg", ok: false},
		{name: "dash", code: "abc-def", ok: false},
		{name: "underscore", code: "abc_def", ok: false},
		{name: "space", code: "abc def", ok: false},
		{name: "unicode", code: "код", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := validCode(tt.code); got != tt.ok {
				t.Fatalf("validCode(%q) = %v, want %v", tt.code, got, tt.ok)
			}
		})
	}
}

func TestRedirectSuccess(t *testing.T) {
	t.Parallel()

	repo := &fakeLinkRepo{
		getFn: func(code string) (links.Link, error) {
			return links.Link{Code: code, URL: "https://example.com/target"}, nil
		},
	}
	rec := &fakeRecorder{}
	mux := newRedirectMux(t, repo, nil, rec)
	req := httptest.NewRequest(http.MethodGet, "/r/abc1234", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if loc := res.Header().Get("Location"); loc != "https://example.com/target" {
		t.Fatalf("Location = %q", loc)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "abc1234" {
		t.Fatalf("recorded calls = %v, want [abc1234]", rec.calls)
	}
}

func TestRedirectCacheHit(t *testing.T) {
	t.Parallel()

	repo := &fakeLinkRepo{}
	cache := &fakeCache{
		getFn: func(code string) (string, bool, error) {
			return "https://cached.example.com", true, nil
		},
	}
	rec := &fakeRecorder{}
	mux := newRedirectMux(t, repo, cache, rec)
	req := httptest.NewRequest(http.MethodGet, "/r/cached1", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", res.Code)
	}
	if loc := res.Header().Get("Location"); loc != "https://cached.example.com" {
		t.Fatalf("Location = %q", loc)
	}
	if repo.getCalls != 0 {
		t.Fatalf("repo calls = %d, want 0", repo.getCalls)
	}
	if len(cache.setCalls) != 0 {
		t.Fatalf("cache sets = %d, want 0", len(cache.setCalls))
	}
}

func TestRedirectCacheMiss(t *testing.T) {
	t.Parallel()

	repo := &fakeLinkRepo{
		getFn: func(code string) (links.Link, error) {
			return links.Link{Code: code, URL: "https://example.com/target"}, nil
		},
	}
	cache := &fakeCache{
		getFn: func(string) (string, bool, error) { return "", false, nil },
	}
	rec := &fakeRecorder{}
	mux := newRedirectMux(t, repo, cache, rec)
	req := httptest.NewRequest(http.MethodGet, "/r/miss123", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", res.Code)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("cache sets = %d, want 1", len(cache.setCalls))
	}
	set := cache.setCalls[0]
	if set.code != "miss123" || set.url != "https://example.com/target" {
		t.Fatalf("cache set = %+v", set)
	}
	if set.ttl != cacheTTL {
		t.Fatalf("ttl = %v, want %v", set.ttl, cacheTTL)
	}
}

func TestRedirectCacheFailureFallsBackToRepository(t *testing.T) {
	t.Parallel()

	repo := &fakeLinkRepo{
		getFn: func(code string) (links.Link, error) {
			return links.Link{Code: code, URL: "https://example.com/target"}, nil
		},
	}
	cache := &fakeCache{
		getFn:  func(string) (string, bool, error) { return "", false, errors.New("redis down") },
		setErr: errors.New("redis down"),
	}
	rec := &fakeRecorder{}
	mux := newRedirectMux(t, repo, cache, rec)
	req := httptest.NewRequest(http.MethodGet, "/r/fallback", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", res.Code)
	}
	if repo.getCalls != 1 {
		t.Fatalf("repo calls = %d, want 1", repo.getCalls)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("cache sets = %d, want 1 attempt", len(cache.setCalls))
	}
}

func TestRedirectNotFound(t *testing.T) {
	t.Parallel()

	repo := &fakeLinkRepo{}
	rec := &fakeRecorder{}
	mux := newRedirectMux(t, repo, nil, rec)
	req := httptest.NewRequest(http.MethodGet, "/r/missing", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("recorded calls = %v, want none", rec.calls)
	}
}

func TestRedirectInvalidCode(t *testing.T) {
	t.Parallel()

	repo := &fakeLinkRepo{}
	mux := newRedirectMux(t, repo, nil, &fakeRecorder{})
	req := httptest.NewRequest(http.MethodGet, "/r/bad-code", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.Code)
	}
	if repo.getCalls != 0 {
		t.Fatalf("repo calls = %d, want 0", repo.getCalls)
	}
}

func TestRedirectRepositoryFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeLinkRepo{
		getFn: func(string) (links.Link, error) { return links.Link{}, errors.New("db down") },
	}
	mux := newRedirectMux(t, repo, nil, &fakeRecorder{})
	req := httptest.NewRequest(http.MethodGet, "/r/abc1234", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", res.Code)
	}
}

func TestRedirectRecordErrorDoesNotBreakRedirect(t *testing.T) {
	t.Parallel()

	repo := &fakeLinkRepo{
		getFn: func(code string) (links.Link, error) {
			return links.Link{Code: code, URL: "https://example.com/target"}, nil
		},
	}
	rec := &fakeRecorder{err: errors.New("queue down")}
	mux := newRedirectMux(t, repo, nil, rec)
	req := httptest.NewRequest(http.MethodGet, "/r/abc1234", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", res.Code)
	}
	if loc := res.Header().Get("Location"); loc != "https://example.com/target" {
		t.Fatalf("Location = %q", loc)
	}
}
