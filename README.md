# URL Shortener

Portfolio pet project: a URL shortener with click analytics.

## Stack

- Backend: Go (stdlib `net/http`, `pgx`, `go-redis`)
- Frontend: Vite + React + TypeScript
- Data: PostgreSQL (links and click aggregates), Redis (redirect cache and click queue)
- Containers: Docker (multi-stage), docker compose

## Architecture

```text
edge nginx (ingress)
  /api/*  -> backend  -> PostgreSQL
  /r/*    -> backend  -> Redis (cache + queue) -> worker -> PostgreSQL
  /*      -> frontend (nginx + React static files)
```

Frontend and backend are separate services with their own Deployment/Service and independent scaling. A short link is `BASE_URL/r/{code}`.

## Repository layout

- [backend/](backend/README.md) - Go API: links, redirects, statistics
- [frontend/](frontend/README.md) - React SPA
- [docker/](docker/) - Dockerfiles, compose file, edge nginx
- [scripts/](scripts/) - repository utilities (smoke check)

## Quick start

```bash
cp .env.example .env
make compose-up
```

The app is available at <http://localhost:8080>. Stop it with `make compose-down`.

Local development without application containers:

```bash
make migrate
make run
make frontend-dev
```

## Tests

| Command | What it checks |
| --- | --- |
| `make test` | `go vet` and backend unit tests with coverage |
| `make frontend-test` | Vitest: API client and App scenarios |
| `make smoke` | End-to-end check of the running stack |

Backend unit tests run without Docker or external services: repository, cache, and queue interfaces are replaced with hand-written fakes. They cover URL validation, code generation and collision retries, Redis fallback, click aggregation, DSN building, HTTP statuses, and `/healthz`/`/readyz`.

`make smoke` requires a running stack (`make compose-up`) and checks what unit tests cannot: image builds, migrations, ENV wiring, edge nginx routing, and work with real PostgreSQL and Redis. It creates a link, follows the `302` redirect, and waits for the click to appear in statistics.