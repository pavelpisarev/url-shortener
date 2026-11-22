package clicks

import (
	"log/slog"
	"net/http"

	"urlshortener/internal/platform/httpx"
)

type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/links/{code}/stats", h.stats)
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	stats, err := h.svc.Stats(r.Context(), code)
	if err != nil {
		h.log.Error("link stats", "code", code, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, stats)
}
