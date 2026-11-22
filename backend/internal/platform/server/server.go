package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"urlshortener/internal/platform/obs"
)

type Config struct {
	Addr string
}

type Server struct {
	http *http.Server
	log  *slog.Logger
}

type DB interface {
	Ping(ctx context.Context) error
}

type Registrar interface {
	Register(mux *http.ServeMux)
}

func New(cfg Config, log *slog.Logger, db DB, registrars ...Registrar) *Server {
	mux := http.NewServeMux()

	for _, r := range registrars {
		r.Register(mux)
	}

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("db unavailable"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	mux.Handle("GET /metrics", obs.Handler())

	handler := loggingMiddleware(log, obs.MetricsMiddleware(mux))

	return &Server{
		http: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		log: log,
	}
}

func (s *Server) Start() error {
	s.log.Info("http server listening", "addr", s.http.Addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func loggingMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
