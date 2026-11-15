package clicks

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) AddClick(ctx context.Context, code string, day time.Time, n int64) error {
	const q = `INSERT INTO clicks (code, day, count) VALUES ($1, $2, $3)
	           ON CONFLICT (code, day) DO UPDATE SET count = clicks.count + EXCLUDED.count`

	if _, err := p.pool.Exec(ctx, q, code, day, n); err != nil {
		return fmt.Errorf("add click: %w", err)
	}
	return nil
}

type DayCount struct {
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

type Stats struct {
	Code  string     `json:"code"`
	Total int64      `json:"total"`
	Days  []DayCount `json:"days"`
}

func (p *Postgres) Stats(ctx context.Context, code string) (Stats, error) {
	const q = `SELECT day, count FROM clicks WHERE code = $1 ORDER BY day`

	rows, err := p.pool.Query(ctx, q, code)
	if err != nil {
		return Stats{}, fmt.Errorf("query stats: %w", err)
	}
	defer rows.Close()

	result := Stats{Code: code, Days: make([]DayCount, 0)}
	for rows.Next() {
		var day time.Time
		var count int64
		if err := rows.Scan(&day, &count); err != nil {
			return Stats{}, fmt.Errorf("scan stats: %w", err)
		}
		result.Days = append(result.Days, DayCount{Day: day.Format("2006-01-02"), Count: count})
		result.Total += count
	}
	if err := rows.Err(); err != nil {
		return Stats{}, fmt.Errorf("iterate stats: %w", err)
	}
	return result, nil
}
