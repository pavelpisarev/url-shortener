# Backend

Stateless Go API for the URL shortener. It creates short links, resolves redirects, records clicks, and serves click statistics.

## Stack

- Go 1.27, stdlib `net/http`
- PostgreSQL via `pgx`, migrations via `golang-migrate`
- Redis via `go-redis`: redirect cache and click queue
- Prometheus metrics and structured logs (`log/slog`)

## Layout

```text
cmd/api             HTTP API and click worker
cmd/migrate         database migrations
internal/links      link creation, code generation, repository
internal/clicks     click recording and aggregation worker
internal/redirect   redirect resolution with cache
internal/platform   config, server, postgres, redis, obs, httpx
migrations          embedded SQL migrations
```

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| POST | `/api/links` | Create a link, returns `201` with `code` and `short_url` |
| GET | `/api/links?limit=100` | List the latest links |
| GET | `/api/links/{code}/stats` | Total clicks and per-day breakdown |
| GET | `/r/{code}` | `302` redirect or `404` |
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe with a database ping |
| GET | `/metrics` | Prometheus metrics |

Errors are returned as `{"error":"..."}`.

## Configuration

All settings come from ENV. The canonical list is in [`../.env.example`](../.env.example).

- `POSTGRES_*` - connection settings; the DSN is built at startup
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_DB`, `REDIS_USER`, `REDIS_PASSWORD` - optional; without `REDIS_HOST` the service writes clicks directly to PostgreSQL
- `PORT` (default `8080`), `BASE_URL` (used to build `short_url`), `LOG_LEVEL`

## Run

Start the whole stack from the repository root with `make compose-up` (see [`../README.md`](../README.md)).

Run the backend on its own:

```bash
go run ./cmd/migrate
go run ./cmd/api
```

It requires a reachable PostgreSQL instance and, optionally, Redis.

## Tests

```bash
go vet ./...
go test ./... -cover
```

Unit tests use hand-written fakes and need no external services. Integration tests against real PostgreSQL and Redis are deferred to Phases 6 and 7; the repo-level `make smoke` covers the running system today.

[Back to the root README](../README.md)
