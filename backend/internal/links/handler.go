package links

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"urlshortener/internal/platform/httpx"
)

const maxBodyBytes = 1 << 20

type Handler struct {
	svc *Service
	log *slog.Logger
}

func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/links", h.create)
	mux.HandleFunc("GET /api/links", h.list)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	res, err := h.svc.Create(r.Context(), in.URL)
	if err != nil {
		if errors.Is(err, ErrInvalidURL) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid url")
			return
		}
		h.log.Error("create link", "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	items, err := h.svc.List(r.Context(), limit)
	if err != nil {
		h.log.Error("list links", "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
