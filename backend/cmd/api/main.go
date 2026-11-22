package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"urlshortener/internal/clicks"
	"urlshortener/internal/links"
	"urlshortener/internal/platform/config"
	"urlshortener/internal/platform/obs"
	"urlshortener/internal/platform/postgres"
	redisclient "urlshortener/internal/platform/redis"
	"urlshortener/internal/platform/server"
	"urlshortener/internal/redirect"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	log := obs.NewLogger(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := postgres.New(startCtx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	var (
		redisClient *redisclient.Client
		cache       redirect.Cache
		queue       clicks.Queue
	)
	if cfg.RedisURL != "" {
		redisClient, err = redisclient.New(startCtx, cfg.RedisURL)
		if err != nil {
			log.Error("connect redis", "err", err)
			os.Exit(1)
		}
		defer redisClient.Close()
		cache = redisClient
		queue = redisClient
	}

	linkRepo := links.NewPostgres(pool)
	linkSvc := links.NewService(linkRepo, cfg.BaseURL)
	linkHandler := links.NewHandler(linkSvc, log)

	clicksStore := clicks.NewPostgres(pool)
	clicksSvc := clicks.NewService(clicksStore, queue)
	clicksHandler := clicks.NewHandler(clicksSvc, log)

	redirectHandler := redirect.NewHandler(linkSvc, cache, clicksSvc, log)

	workerCtx, stopWorker := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	if redisClient != nil {
		worker := clicks.NewWorker(redisClient, clicksStore, log)
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.Run(workerCtx)
		}()
	}

	srv := server.New(
		server.Config{Addr: ":" + strconv.Itoa(cfg.Port)},
		log,
		pool,
		linkHandler,
		clicksHandler,
		redirectHandler,
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown", "err", err)
	}

	stopWorker()
	wg.Wait()
	log.Info("server stopped")
}
