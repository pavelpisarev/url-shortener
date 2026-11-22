package links

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type createCall struct {
	code string
	url  string
}

type fakeRepo struct {
	createCalls []createCall
	createErr   func(code string) error
	getFn       func(code string) (Link, error)
	listFn      func(limit int) ([]Link, error)
}

func (f *fakeRepo) Create(_ context.Context, code, url string) (Link, error) {
	f.createCalls = append(f.createCalls, createCall{code: code, url: url})
	if f.createErr != nil {
		if err := f.createErr(code); err != nil {
			return Link{}, err
		}
	}
	return Link{Code: code, URL: url, CreatedAt: time.Unix(1700000000, 0).UTC()}, nil
}

func (f *fakeRepo) GetByCode(_ context.Context, code string) (Link, error) {
	if f.getFn == nil {
		return Link{}, ErrNotFound
	}
	return f.getFn(code)
}

func (f *fakeRepo) List(_ context.Context, limit int) ([]Link, error) {
	if f.listFn == nil {
		return []Link{}, nil
	}
	return f.listFn(limit)
}

func TestValidateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		ok   bool
	}{
		{name: "https", in: "https://example.com", ok: true},
		{name: "http with path and query", in: "http://example.com/a/b?q=1#frag", ok: true},
		{name: "custom port", in: "https://example.com:8443/path", ok: true},
		{name: "localhost", in: "http://localhost:3000", ok: true},
		{name: "unicode path", in: "https://example.com/путь", ok: true},
		{name: "empty", in: "", ok: false},
		{name: "spaces", in: "   ", ok: false},
		{name: "no scheme", in: "example.com/path", ok: false},
		{name: "scheme relative", in: "//example.com", ok: false},
		{name: "ftp", in: "ftp://example.com/file", ok: false},
		{name: "javascript", in: "javascript:alert(1)", ok: false},
		{name: "mailto", in: "mailto:user@example.com", ok: false},
		{name: "empty host", in: "https://", ok: false},
		{name: "space in host", in: "https:// example.com", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateURL(tt.in)
			if tt.ok && err != nil {
				t.Fatalf("validateURL(%q) = %v, want nil", tt.in, err)
			}
			if !tt.ok && !errors.Is(err, ErrInvalidURL) {
				t.Fatalf("validateURL(%q) = %v, want ErrInvalidURL", tt.in, err)
			}
		})
	}
}

func TestGenerateCode(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		code, err := generateCode(codeLength)
		if err != nil {
			t.Fatalf("generateCode: %v", err)
		}
		if len(code) != codeLength {
			t.Fatalf("len(code) = %d, want %d", len(code), codeLength)
		}
		for _, c := range code {
			if !strings.ContainsRune(alphabet, c) {
				t.Fatalf("code %q contains %q outside alphabet", code, c)
			}
		}
		if _, ok := seen[code]; ok {
			t.Fatalf("duplicate code generated: %q", code)
		}
		seen[code] = struct{}{}
	}
}

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepo{}
		svc := NewService(repo, "http://short")

		res, err := svc.Create(context.Background(), "https://example.com/path")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if len(repo.createCalls) != 1 {
			t.Fatalf("repo called %d times, want 1", len(repo.createCalls))
		}
		call := repo.createCalls[0]
		if len(call.code) != codeLength {
			t.Fatalf("code %q length = %d, want %d", call.code, len(call.code), codeLength)
		}
		if call.url != "https://example.com/path" {
			t.Fatalf("url = %q, want original url", call.url)
		}
		if res.ShortURL != "http://short/r/"+call.code {
			t.Fatalf("ShortURL = %q, want %q", res.ShortURL, "http://short/r/"+call.code)
		}
		if res.Code != call.code || res.URL != call.url {
			t.Fatalf("result = %+v, want code %q url %q", res, call.code, call.url)
		}
	})

	t.Run("trims base url and input", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepo{}
		svc := NewService(repo, "http://short/")

		res, err := svc.Create(context.Background(), "  https://example.com  ")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if repo.createCalls[0].url != "https://example.com" {
			t.Fatalf("url = %q, want trimmed", repo.createCalls[0].url)
		}
		if strings.Contains(res.ShortURL, "short//r/") {
			t.Fatalf("ShortURL = %q, want single slash", res.ShortURL)
		}
	})

	t.Run("invalid url", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepo{}
		svc := NewService(repo, "http://short")

		_, err := svc.Create(context.Background(), "not-a-url")
		if !errors.Is(err, ErrInvalidURL) {
			t.Fatalf("err = %v, want ErrInvalidURL", err)
		}
		if len(repo.createCalls) != 0 {
			t.Fatalf("repo called %d times, want 0", len(repo.createCalls))
		}
	})

	t.Run("retries on code collision", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		repo := &fakeRepo{
			createErr: func(string) error {
				attempts++
				if attempts < 3 {
					return ErrCodeExists
				}
				return nil
			},
		}
		svc := NewService(repo, "http://short")

		res, err := svc.Create(context.Background(), "https://example.com")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if attempts != 3 {
			t.Fatalf("attempts = %d, want 3", attempts)
		}
		if res.ShortURL == "" {
			t.Fatal("ShortURL is empty")
		}
	})

	t.Run("gives up after max retries", func(t *testing.T) {
		t.Parallel()

		repo := &fakeRepo{
			createErr: func(string) error { return ErrCodeExists },
		}
		svc := NewService(repo, "http://short")

		_, err := svc.Create(context.Background(), "https://example.com")
		if err == nil {
			t.Fatal("Create returned nil error")
		}
		if len(repo.createCalls) != maxRetries {
			t.Fatalf("repo called %d times, want %d", len(repo.createCalls), maxRetries)
		}
		if !strings.Contains(err.Error(), "unique code") {
			t.Fatalf("err = %v, want unique code error", err)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		t.Parallel()

		boom := errors.New("boom")
		repo := &fakeRepo{
			createErr: func(string) error { return boom },
		}
		svc := NewService(repo, "http://short")

		_, err := svc.Create(context.Background(), "https://example.com")
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
	})
}

func TestServiceGetAndList(t *testing.T) {
	t.Parallel()

	want := Link{Code: "abc1234", URL: "https://example.com", CreatedAt: time.Unix(1700000000, 0).UTC()}
	repo := &fakeRepo{
		getFn: func(code string) (Link, error) {
			if code != want.Code {
				return Link{}, ErrNotFound
			}
			return want, nil
		},
		listFn: func(limit int) ([]Link, error) {
			if limit != 5 {
				t.Fatalf("limit = %d, want 5", limit)
			}
			return []Link{want}, nil
		},
	}
	svc := NewService(repo, "http://short")

	got, err := svc.Get(context.Background(), want.Code)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Fatalf("Get = %+v, want %+v", got, want)
	}

	if _, err := svc.Get(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing err = %v, want ErrNotFound", err)
	}

	items, err := svc.List(context.Background(), 5)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0] != want {
		t.Fatalf("List = %+v, want [%+v]", items, want)
	}
}
