# Backend bootstrap design

Date: 2026-10-08. Status: approved. Scope: skeleton only, no business features.
Implements [ADR-0001](../adr/0001-go-and-gin-for-the-backend.md), [ADR-0005](../adr/0005-postgresql-as-primary-database.md), [ADR-0006](../adr/0006-redis-as-cache.md), [ADR-0008](../adr/0008-docker-compose-for-local-provisioning.md), [ADR-0012](../adr/0012-hexagonal-backend-layout.md).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Scope | Skeleton: config, Cobra commands, Gin server, health and metrics, Postgres and Redis adapters, migrations, Dockerfile, Makefile, lint. No auth, no domain features. |
| DB access | pgx/v5 + sqlc. |
| Migrations | golang-migrate library, SQL files embedded in the binary, `bragdoc migrate` subcommand. |
| Toolchain | Go 1.27 in go.mod, Makefile, golangci-lint. |
| CLI | Cobra. One binary `bragdoc` with `api`, `bot`, `worker`, `migrate`. |
| Ops baseline | `/healthz`, `/readyz`, `/metrics` (Prometheus), slog JSON, request id. |
| Layout | Hexagonal: domain, app, ports, adapters. |
| Skill | `.claude/skills/backend-endpoint/SKILL.md` documents how to add a feature. |

## 1. Architecture and layout

```
backend/
  cmd/bragdoc/main.go              # cobra root: api | bot | worker | migrate
  internal/
    domain/                        # entities, value objects, typed errors. Stdlib only.
    app/                           # use cases. Imports domain + ports.
    ports/                         # interfaces: repositories, Cache, Clock, IDs.
    adapters/
      http/                        # Gin engine, middleware, health, metrics, handlers, error mapping
      postgres/                    # pgx pool, sqlc output, tenant-scoped Tx, repositories
      redis/                       # Cache implementation + degrading decorator
    telemetry/                     # slog + prometheus setup (shared infra, imported by adapters and cmd)
    config/                        # env → Config, validated at startup
  migrations/                      # NNNN_name.up.sql / .down.sql, embedded
  queries/                         # sqlc input
  sqlc.yaml  Makefile  Dockerfile  .golangci.yml
.claude/skills/backend-endpoint/SKILL.md
```

Dependency rule, enforced by golangci-lint depguard:

- `domain` imports only stdlib.
- `app` and `ports` import `domain` and stdlib.
- `adapters/*` may import anything external, never each other.
- `cmd` wires adapters into app services.

Skeleton deliverables: config loading; Cobra commands; Gin server with recovery, request id, slog request log, Prometheus middleware; `/healthz`, `/readyz` (pings Postgres and Redis, JSON body per dependency), `/metrics`; pgx pool with `WithTenant(ctx, tenantID, fn)` doing `SET LOCAL app.tenant_id`; Redis client; `migrate` subcommand; initial migration creating `app_meta`; sqlc config; multi-stage Dockerfile; Makefile targets `run`, `test`, `test-integration`, `lint`, `migrate`, `sqlc`, `docker`. `bot` and `worker` start, log, and block on signal.

### Graceful degradation and shutdown

| Dependency | Tier | Unavailable at startup | Lost at runtime |
|---|---|---|---|
| PostgreSQL | Required | `api` exits non-zero after 5 retries with backoff | `/readyz` 503; handlers 503 + `Retry-After`; pool reconnects |
| Redis | Optional | Warn, start with no-op cache | Decorator returns miss, logs once per minute, increments `cache_errors_total`; `/readyz` shows `cache: degraded`, status 200 |
| Gotenberg | Optional, worker | Worker starts; export jobs fail retriable | Same |
| Telegram API | Optional, bot | Retry with backoff forever; readiness shows degraded | Same |

- `ports.Cache` has the real Redis implementation and a `degrading` decorator with `Healthy()`; `cmd` always wires the decorator.
- `/readyz` body: `{"postgres":"ok","cache":"degraded"}`. HTTP 503 only when a required tier is down.
- Outbound calls carry context timeouts from config: `DB_TIMEOUT` 5 s, `CACHE_TIMEOUT` 200 ms.
- Shutdown on SIGTERM/SIGINT: stop accepting, drain up to `SHUTDOWN_TIMEOUT` 15 s, close Redis then pgx pool. `bot` and `worker` finish the current item, then exit.
- Handler panics are recovered to 500 with the request id.

Rule for features: a feature may fail because Postgres is down. It may never fail because Redis, Gotenberg, or Telegram is down.

## 2. Request lifecycle, errors, config

Middleware order: request id (`X-Request-ID` in/out, UUID fallback) → slog request log (method, path, status, latency, request id, tenant id) → Prometheus (`http_requests_total`, `http_request_duration_seconds` by route template and status) → recovery (innermost, so panics are logged and counted as 500) → auth slot (next pass).

Flow: handler binds and validates with Gin tags → converts to a plain input struct → calls one `app` use case → use case touches only `domain` and `ports` → adapters implement ports, Postgres repositories run inside the tenant-scoped Tx when a tenant is in context → handler maps result to a response DTO and error to HTTP via `adapters/http/errors.go`.

| Domain error | HTTP |
|---|---|
| `domain.ErrNotFound` | 404 |
| `domain.ErrForbidden` | 403 |
| `domain.ErrValidation` (field errors) | 422 `{"error":"validation","fields":{...}}` |
| `domain.ErrConflict` | 409 |
| `context.DeadlineExceeded` | 504 |
| pgx connection errors | 503 + `Retry-After` |
| other | 500, logged with stack, body has only the request id |

Error body shape: `{"error":"<code>","message":"<text>","request_id":"<id>","fields":{...}}`.

Config: `config.Config` from environment via `caarlos0/env`, validated once in `cmd`. `.env.example` is the documentation.

Logging: `slog`, JSON by default, `LOG_FORMAT=text` for local, `LOG_LEVEL`. A handler wrapper attaches request id and tenant id from context.

Dependencies: `gin`, `cobra`, `pgx/v5`, `go-redis/v9`, `golang-migrate` (pgx driver, iofs source), `prometheus/client_golang`, `caarlos0/env`, `google/uuid`, `testify`, `testcontainers-go` (test only).

## 3. Testing

- Unit: `domain` and `app` with hand-written fakes of `ports`. No mock framework.
- Adapter integration: `adapters/postgres` against testcontainers Postgres, build tag `integration`, `make test-integration`. Skeleton ships: migrate up, set tenant, select from `app_meta`.
- HTTP: `httptest` + Gin + fake services. Skeleton tests `/healthz`, `/readyz` healthy and degraded, every row of the error table, panic → 500 with request id.
- Degradation: decorator over a failing fake returns miss, no error, counter increments.
- CI: `make lint test` always; `make test-integration` in the compose-backed job.

## 4. Project skill: `.claude/skills/backend-endpoint/SKILL.md`

1. Trigger: adding or changing an endpoint, use case, repository, or migration in `backend/`.
2. Dependency rule and degradation rule, one sentence each, with the lint command.
3. Ordered steps: migration → sqlc query → `make sqlc` → domain type and error → port method → adapter implementation → use case + unit test with fake → handler, DTOs, route → error mapping → `httptest` test → `backend/api/openapi.yaml` → link PRD/ADR in the PR.
4. Conventions: one file per use case in `app/`; handler file per resource; route groups per resource; DTO suffixes `Request`/`Response`.
5. Pointers to the worked examples: `/readyz` and the cache decorator.
6. Anti-patterns: Gin or pgx types in `app`/`domain`; logic in handlers; a query without tenant scope; swallowing an error without a metric.
