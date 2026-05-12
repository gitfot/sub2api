# Repository Guidelines

## Project Structure & Module Organization
This repo is split into two main apps:
- `backend/`: Go API server, with `cmd/server` for the executable, `internal/` for application code, `ent/` for generated models, and `migrations/` for schema changes.
- `frontend/`: Vue 3 + Vite app. Source lives in `src/`, with tests under `src/**/__tests__`.
- `deploy/`: Docker Compose files, `.env.example`, and deployment docs.

## Build, Test, and Development Commands
Use the repo root for cross-stack tasks:
- `make build` builds backend and frontend.
- `make test` runs backend tests, frontend lint/type checks, and the core Vitest suite.
- `make secret-scan` runs the repo secret scanner.

Backend commands live in `backend/`:
- `make build` produces `bin/server`.
- `make build-embed` builds the backend with embedded frontend assets.
- `make test`, `make test-unit`, `make test-integration`, and `make test-e2e` run the matching Go test sets.

Frontend commands live in `frontend/`:
- `pnpm install` installs dependencies.
- `pnpm dev` starts the Vite dev server.
- `pnpm build` type-checks and builds production assets.
- `pnpm lint:check`, `pnpm typecheck`, and `pnpm test:run` cover linting, type checking, and Vitest.

## Running From Source on Linux
For local source runs on Linux, prefer matching the checked-in toolchain and config expectations:
Install `Go 1.26.3` (matches `backend/go.mod`), `Node.js 18+`, `pnpm`, `PostgreSQL 15+`, and `Redis 7+`.

```bash
# Build the frontend first so static assets are emitted to `backend/internal/web/dist`:
cd frontend
pnpm install && pnpm build
pnpm dev
# For an embedded single-binary run, create `backend/config.yaml` from `deploy/config.example.yaml`, fill in PostgreSQL, Redis, and `jwt.secret`, then start from `backend/` with:
cd backend
go build -tags embed -o sub2api ./cmd/server
# For day-to-day development, run the backend separately with 
go run ./cmd/server

```
The Vite dev server defaults to port `3000` and proxies `/api`, `/v1`, and `/setup` to `http://localhost:8080`; keep the backend on port `8080` unless you also update the frontend proxy settings.

## Coding Style & Naming Conventions
Backend Go code should follow `gofmt` formatting and pass `golangci-lint` (`backend/.golangci.yml`). Keep package and file names short, lower-case, and domain-focused. Frontend code uses ESLint with TypeScript and Vue rules; prefer `PascalCase` for components, `camelCase` for functions/variables, and `__tests__` for colocated test folders.

## Testing Guidelines
Go tests use `*_test.go` with unit, integration, and e2e tags where needed. Frontend tests use Vitest and `*.spec.ts` files. Add or update tests alongside behavior changes, especially for handlers, services, and user-facing views.

## Commit & Pull Request Guidelines
History shows a conventional style: `feat:`, `fix:`, `chore:`, `docs:`, and `ci:`. Keep commits focused and imperative. Pull requests should summarize the change, note any config or migration impact, and include screenshots for UI updates when relevant.

## Security & Configuration Tips
Copy `deploy/.env.example` instead of committing secrets. Review migration files carefully before applying them, and run `make secret-scan` before opening a PR that touches credentials, auth, or deployment settings.
