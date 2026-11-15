package main

import (
	"errors"
	"flag"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"urlshortener/internal/platform/config"
	"urlshortener/migrations"
)

func main() {
	down := flag.Bool("down", false, "rollback all migrations")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("init migration source: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, pgxURL(cfg.DatabaseURL))
	if err != nil {
		log.Fatalf("init migrate: %v", err)
	}
	defer m.Close()

	if *down {
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migrate down: %v", err)
		}
		log.Println("migrations rolled back")
		return
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrate up: %v", err)
	}
	log.Println("migrations applied")
}

func pgxURL(u string) string {
	switch {
	case strings.HasPrefix(u, "postgres://"):
		return "pgx5://" + strings.TrimPrefix(u, "postgres://")
	case strings.HasPrefix(u, "postgresql://"):
		return "pgx5://" + strings.TrimPrefix(u, "postgresql://")
	default:
		return u
	}
}
