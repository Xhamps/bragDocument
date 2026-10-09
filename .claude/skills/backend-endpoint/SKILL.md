---
name: backend-endpoint
description: How to add or change an API endpoint, use case, repository, or migration in backend/. Use whenever touching backend/ for a feature. Enforces the hexagonal layering and graceful-degradation rules from ADR-0012.
---

# Adding a backend feature

Read first: `docs/adr/0012-hexagonal-backend-layout.md`, then the PRD for the feature.

## Two rules

1. **Dependency rule.** `domain` imports stdlib only. `app` and `ports` import `domain`. `adapters/*` implement ports and never import each other. `cmd` wires. Check: `make lint` (depguard).
2. **Degradation rule.** A feature may fail because Postgres is down. It may never fail because Redis, Gotenberg, Telegram, or Resend is down. Use `ports.Cache` through the `redis.Degrading` decorator; never call go-redis directly from a use case.
   Exception: Telegram linking needs Redis (ADR-0009); it uses the raw cache and returns `ErrUnavailable`.

## Steps, in order

1. **Migration**: `backend/migrations/NNNN_<name>.up.sql` and `.down.sql`. Every tenant-scoped table has `tenant_id` and will get an RLS policy (see ADR-0007). Run `make migrate` against the compose Postgres.
2. **Query**: add to `backend/queries/<resource>.sql` with `-- name: X :one|:many|:exec`. Run `make sqlc`. Commit the generated code.
3. **Domain**: add or extend the entity in `internal/domain/<resource>.go`. Validation lives here and returns `domain.NewValidationError(...)`. Return `domain.ErrNotFound`, `ErrForbidden`, `ErrConflict` as appropriate.
4. **Port**: add the repository interface (or method) in `internal/ports/<resource>.go`. Keep it small: only what the use case calls.
5. **Adapter**: implement the port in `internal/adapters/postgres/<resource>_repo.go`. Every tenant-scoped call runs inside `db.WithTenant(ctx, tenantID, func(ctx, tx) error { q := sqlcgen.New(tx); ... })`; `WithTenant` rejects an empty tenant id with `domain.ErrForbidden`. Pass every query error through the package's `wrap()` helper in `internal/adapters/postgres/errors.go` (it maps `pgx.ErrNoRows` to `domain.ErrNotFound` and connection failures to `domain.ErrUnavailable`); never map pgx errors by hand. Every resource insert, update, or delete goes through `write[T](ctx, db, auditEntry, fn)` in `internal/adapters/postgres/outbox.go`: it runs your queries and the audit outbox message in one transaction and returns `fn`'s result (ADR-0015). Return `errNoChange` from `fn` for a no-op. Never call `enqueue`/`audit` directly unless your write opens its own transaction. Integration test with build tag `integration`, following `db_integration_test.go`; call `drain(t, db)` before reading `audit_entries`.
6. **Use case**: one file per use case in `internal/app/<resource>_<verb>.go` (e.g. `log_create.go`). Constructor takes ports. Unit test with a hand-written fake of the port in the same package; no mock framework. Document access goes through `access(ctx, docs, docID, userID, perm)` in `internal/app/access.go`; never compare `OwnerID` yourself.
7. **Handler**: `internal/adapters/http/<resource>_handler.go`. Bind with Gin tags into a `<Verb><Resource>Request` struct, call exactly one use case, write a `<Resource>Response`. On error call `RespondError(c, err)` and nothing else. Register routes in a `Register<Resource>(e *gin.Engine, uc ...)` function and call it from `cmd/bragdoc/api.go`. Middleware order on the engine is request id → logger → metrics → recovery (innermost); handlers never log the request id themselves because `slog.*Context(ctx, ...)` attaches it.
8. **Errors**: if the feature introduces a new domain error, add a row to `RespondError` and to its test table.
9. **HTTP test**: `httptest` with `NewEngine(telemetry.NewRegistry())` and a fake use case, following `health_test.go`.
10. **Contract**: create or update `backend/api/openapi.yaml` (not yet present in the skeleton).
11. **Docs**: link the PRD or ADR in the PR description. A new technical choice gets a new ADR.

## Conventions

- Route groups per resource: `/documents`, `/documents/:id/logs`.
- DTO suffixes: `Request`, `Response`. Never expose sqlc or domain structs directly.
- Context carries request id and tenant id (`telemetry.RequestID`, `telemetry.TenantID`). Use `slog.InfoContext(ctx, ...)` so they are attached.
- Timeouts come from config (`DBTimeout`, `CacheTimeout`); do not hard-code.
- Lint runs with `make lint` from `backend/`, which uses `backend/bin/golangci-lint` (v2) when present. The depguard rules exempt `_test.go` files in `domain`, `app`, and `ports` so tests may use testify.

## Worked examples in the skeleton

- Readiness with required vs optional dependencies: `internal/adapters/http/health.go`.
- Degradation decorator and its test: `internal/adapters/redis/degrading.go`.
- Tenant-scoped transaction and integration test: `internal/adapters/postgres/db.go`, `db_integration_test.go`.
- Driver-to-domain error mapping and its test: `internal/adapters/postgres/errors.go` (`wrap` + `TestWrap`).

## Anti-patterns

- Gin, pgx, or go-redis types in `app` or `domain`.
- Business logic in a handler, or a handler calling two use cases.
- A query on a tenant-scoped table outside `WithTenant`.
- Swallowing an error without a metric or log.
- A new interface with one implementation and no test using a fake.
