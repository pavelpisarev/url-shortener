# Frontend

React SPA for the URL shortener. It creates links, lists them, and expands per-link click statistics.

The app talks to the backend through relative `/api` and `/r` paths, so both services share one origin and CORS is not needed.

## Stack

- React 19, Vite 8, TypeScript
- Pico CSS for classless styling
- oxlint for linting
- Vitest and Testing Library for tests

## Layout

```text
src/api.ts       backend client
src/App.tsx      create form and link list
src/LinkRow.tsx  link row with expandable statistics
src/i18n/        en/ru dictionaries and provider
src/test/        Vitest setup
```

## Development

```bash
npm install
npm run dev
```

The dev server proxies `/api` and `/r` to `http://localhost:8080` (see `vite.config.ts`), so the backend must be running.

## Build

```bash
npm run build
```

This runs the TypeScript check and Vite build, and writes static files to `dist/`. The container serves them with nginx; the config is in `nginx.conf`.

## Tests

```bash
npm test
npm run lint
```

Tests cover the API client (success, JSON error, non-JSON error, code encoding) and App scenarios (list, empty state, link creation, error state).

[Back to the root README](../README.md)
