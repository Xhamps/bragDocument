# Run all services in one process: design

Date: 2026-10-09. Status: proposed. Extends [ADR-0012](../adr/0012-hexagonal-backend-layout.md) (single Cobra binary) and the logging baseline in [backend bootstrap](2026-10-08-backend-bootstrap-design.md). Depends on [PR #7](https://github.com/Xhamps/bragDocument/pull/7) for the `worker` subcommand.

## Problem

Running the backend locally takes three terminals: `bragdoc api`, `bragdoc bot`, `bragdoc worker`. All three write to stderr through the same logger, and no log line says which process wrote it, so once their output is merged (one terminal, `docker compose logs`, a log shipper) you cannot tell a bot error from an api error.

## Decisions

| Topic | Decision |
|---|---|
| Command | New subcommand `bragdoc all` runs api, bot and worker in one process. The existing subcommands stay unchanged, and compose keeps one container per service. |
| Migrations | `all` does not migrate. `make migrate` / the compose `migrate` service stay the only path, so the app role never needs DDL rights. |
| Lifecycle | One `errgroup.WithContext` over the signal context. A service that returns an error cancels the other two; the first error is the exit error. A clean SIGINT/SIGTERM stops all three and exits 0. |
| Config | Loaded once by `boot()` and shared. Each service keeps its own degradation rule: no `TELEGRAM_BOT_TOKEN` → bot idles, no `EXPORT_KEY` → worker idles, api still fails fast without `SUPABASE_URL`. |
| Connections | Each service opens its own Postgres pool and Redis client, as today. Sharing them is a later optimisation. |
| Service tag | Every log record carries `"service":"api" \| "bot" \| "worker"`, also in single-service mode. |
| Format | JSON stays the default (`LOG_FORMAT=json`), one object per line. `text` stays available for local use. |
| Dependency | `golang.org/x/sync/errgroup`, already in `go.mod` as an indirect dependency. No new module. |

## 1. Command

`cmd/bragdoc`:

- Move each `RunE` body into a function with the same behaviour: `runAPI(ctx, cfg) error`, `runBot(ctx, cfg) error`, `runWorker(ctx, cfg) error`.
- `api`, `bot`, `worker` become: `cfg := boot()`, then `runX(telemetry.WithService(ctx, "x"), cfg)`.
- `all` (new file `all.go`):

  ```
  cfg := boot()
  g, ctx := errgroup.WithContext(cmd.Context())
  g.Go(func() error { return runAPI(telemetry.WithService(ctx, "api"), cfg) })
  g.Go(func() error { return runBot(telemetry.WithService(ctx, "bot"), cfg) })
  g.Go(func() error { return runWorker(telemetry.WithService(ctx, "worker"), cfg) })
  return g.Wait()
  ```

- Makefile: `run-all: go run ./cmd/bragdoc all`. `run` keeps starting only the api.
- The shared shutdown path already fits: `runAPI` waits for `ctx.Done()` then calls `srv.Shutdown`; bot and worker return when `ctx` ends.

## 2. Service tag in logs

The tag rides on the context, the same way `request_id` and `tenant_id` already do, because the three services share `slog.Default()` in one process and a per-service global logger is impossible there.

`internal/telemetry`:

- `WithService(ctx, name)` and `Service(ctx)` next to `WithRequestID` / `WithTenantID`.
- `ctxHandler.Handle` adds `service` when set, before `request_id` and `tenant_id`.

Places where the context does not reach the log call today, and must:

| Where | Today | Change |
|---|---|---|
| api request handlers | `http.Server` uses `context.Background()` as the base context, so request logs lose the tag | `srv.BaseContext` returns `context.WithoutCancel(ctx)`: requests inherit the tag but not the shutdown cancellation, so graceful shutdown still lets in-flight requests finish |
| `telegram.New` errors handler | `slog.Error(...)` with no context | `New(ctx, token, r, ...)`; the handler uses `slog.ErrorContext(ctx, ...)` |
| Fatal error in `main` | `fmt.Fprintln(os.Stderr, "error:", err)`, plain text | `slog.Error("exit", slog.Any("err", err))` so the last line is JSON too. A config error before `boot()` installs the logger still goes out through the default slog handler |
| `postgres.Migrate` close warning | `slog.Warn` with no context | Left alone: `migrate` is not part of `all` and runs alone |

Example output of `bragdoc all`:

```json
{"time":"2026-10-09T10:00:00Z","level":"INFO","msg":"api listening","addr":":8080","service":"api"}
{"time":"2026-10-09T10:00:00Z","level":"WARN","msg":"TELEGRAM_BOT_TOKEN not set; bot idle","service":"bot"}
{"time":"2026-10-09T10:00:00Z","level":"INFO","msg":"worker started","service":"worker"}
{"time":"2026-10-09T10:00:05Z","level":"INFO","msg":"request","method":"GET","path":"/me","status":200,"service":"api","request_id":"9f1c…","tenant_id":"t-1"}
```

## 3. Testing

- `telemetry`: logger test asserts `"service":"worker"` from the context; a record without it has no `service` key.
- `cmd/bragdoc`: `all` with a service that fails at startup (api without `SUPABASE_URL`) returns that error and the other two stop. Cancelling the context makes `all` return nil.
- Manual: `make run-all` with compose infra up; every line parses with `jq` and carries `service`; Ctrl-C stops all three within `SHUTDOWN_TIMEOUT`.

## 4. Delivery

Branch `feat/run-all`, conventional commits (`feat(telemetry)`, `feat(cmd)`, `docs`). README "Subcommands" line gains `all` and `make run-all`. Pull request against `main` after PR #7 merges.

## Out of scope

- A `/metrics` endpoint for bot and worker (the registries stay unserved, as today).
- Sharing one Postgres pool and Redis client across the three services.
- Running migrations from `all`.
