package redirect

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"urlshortener/internal/links"
	"urlshortener/internal/platform/obs"
)

const cacheTTL = 24 * time.Hour

type Cache interface {
	Get(ctx context.Context, code string) (string, bool, error)
	Set(ctx context.Context, code, url string, ttl time.Duration) error
}

type Recorder interface {
	Record(ctx context.Context, code string) error
}

type Handler struct {
	links    *links.Service
	cache    Cache
	recorder Recorder
	log      *slog.Logger
}

func NewHandler(linkSvc *links.Service, cache Cache, recorder Recorder, log *slog.Logger) *Handler {
	return &Handler{links: linkSvc, cache: cache, recorder: recorder, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /r/{code}", h.redirect)
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if !validCode(code) {
		http.NotFound(w, r)
		return
	}

	target, err := h.resolve(r.Context(), code)
	if err != nil {
		if errors.Is(err, links.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		h.log.Error("resolve link", "code", code, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.recorder.Record(r.Context(), code); err != nil {
		h.log.Error("record click", "code", code, "err", err)
	}

	obs.IncRedirect()
	http.Redirect(w, r, target, http.StatusFound)
}

func (h *Handler) resolve(ctx context.Context, code string) (string, error) {
	if h.cache != nil {
		if target, ok, err := h.cache.Get(ctx, code); err == nil && ok {
			obs.IncCacheHit()
			return target, nil
		}
		obs.IncCacheMiss()
	}

	link, err := h.links.Get(ctx, code)
	if err != nil {
		return "", err
	}

	if h.cache != nil {
		_ = h.cache.Set(ctx, code, link.URL, cacheTTL)
	}
	return link.URL, nil
}

func validCode(code string) bool {
	if len(code) == 0 || len(code) > 16 {
		return false
	}
	for _, c := range code {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		default:
			return false
		}
	}
	return true
}
