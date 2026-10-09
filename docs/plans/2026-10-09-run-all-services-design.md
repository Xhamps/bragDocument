# Run all services + service-tagged logs — design

PRD: [PRD-0008](../prd/0008-run-all-services-and-service-tagged-logs.md)

## Decisions

- Command name: `bragdoc all` (`make run-all`).
- Each service keeps its own Postgres pool and Redis client under `all`; it runs exactly the code it runs alone.
- Docker Compose single-container profile: out of scope (Operations, later).
- `service` travels in the context and `telemetry.ctxHandler` adds it, like `request_id` and `tenant_id`. Rejected: a per-service `*slog.Logger` passed down (touches every package; the global logger can't carry it under `all`); child processes per service (signal forwarding and output plumbing for no gain).

## 1. Commands (`backend/cmd/bragdoc`)

- Extract each `RunE` body into `runAPI(ctx, cfg)`, `runBot(ctx, cfg)`, `runWorker(ctx, cfg)`. Each starts with `ctx = telemetry.WithService(ctx, "<name>")`.
- `api`, `bot`, `worker` become `boot()` + `runX`; behaviour unchanged (FR-2, FR-5).
- `all`: `boot()` once, then the three in an `errgroup.WithContext` (`golang.org/x/sync`, promoted from indirect). The first failure cancels the shared context; the others shut down gracefully; the command returns that error prefixed with the service name, e.g. `bot: …` (FR-3, FR-4).
- No migrations (FR-10).

## 2. Logging (`backend/internal/telemetry`)

- `WithService(ctx, name)` / `Service(ctx)` in `context.go`; `ctxHandler.Handle` adds `service` when set. Both JSON and text formats get it (FR-6, FR-9).
- API: `srv.BaseContext` returns `context.WithoutCancel(ctx)` so request contexts carry `service` next to `request_id`/`tenant_id` (FR-7) without shutdown cancelling in-flight requests.
- `adapters/telegram/bot.go`: the polling error handler captures the bot's context and uses `slog.ErrorContext`.

## 3. JSON-only output (FR-8)

- `main` installs a default JSON logger on stderr first, so a config-load failure is JSON too.
- `main` logs the exit error with `slog.Error("exit", "err", err)` instead of `fmt.Fprintln`.

## 4. Docs

- `make run-all` in `backend/Makefile`; README lists `all` among subcommands (FR-11).

## 5. Tests

- `logger_test.go`: a record logged with a service context contains `service=bot` in JSON and text.
- `all` runner: one function errors → the others see cancellation → returned error carries the service prefix.
- Manual: `make run-all 2>&1 | jq -e 'has("service")'` passes on every line.

## Out of scope

Shared pools, Compose profile, `service` on `migrate` logs.
