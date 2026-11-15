package links

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) Create(ctx context.Context, code, url string) (Link, error) {
	const q = `INSERT INTO links (code, original_url) VALUES ($1, $2) RETURNING code, original_url, created_at`

	var l Link
	err := p.pool.QueryRow(ctx, q, code, url).Scan(&l.Code, &l.URL, &l.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Link{}, ErrCodeExists
		}
		return Link{}, fmt.Errorf("create link: %w", err)
	}
	return l, nil
}

func (p *Postgres) GetByCode(ctx context.Context, code string) (Link, error) {
	const q = `SELECT code, original_url, created_at FROM links WHERE code = $1`

	var l Link
	err := p.pool.QueryRow(ctx, q, code).Scan(&l.Code, &l.URL, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	if err != nil {
		return Link{}, fmt.Errorf("get link: %w", err)
	}
	return l, nil
}

func (p *Postgres) List(ctx context.Context, limit int) ([]Link, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `SELECT code, original_url, created_at FROM links ORDER BY created_at DESC LIMIT $1`

	rows, err := p.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	defer rows.Close()

	result := make([]Link, 0, limit)
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.Code, &l.URL, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		result = append(result, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate links: %w", err)
	}
	return result, nil
}
