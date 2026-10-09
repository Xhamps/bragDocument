# Run All Services + Service-Tagged Logs Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** `bragdoc all` runs api, bot and worker in one process, and every log line is JSON carrying `service` (PRD-0008).

**Architecture:** `service` rides in the context like `request_id`/`tenant_id`; `telemetry.ctxHandler` adds it. Each command's body becomes `runX(ctx, cfg)`, which tags its context first. `all` runs the three in an errgroup: the first error cancels the rest and is returned prefixed with the service name. `main` logs in JSON from the first line and logs the exit error through slog.

**Tech Stack:** Go 1.27, log/slog, cobra, golang.org/x/sync/errgroup, testify.

Design: `docs/plans/2026-10-09-run-all-services-design.md`. Branch: `feat/run-all-services`. All commands run from `backend/`.

---

### Task 1: `service` in context and logs

**Files:**
- Modify: `backend/internal/telemetry/context.go`
- Modify: `backend/internal/telemetry/logger.go` (`ctxHandler.Handle`)
- Test: `backend/internal/telemetry/logger_test.go`

**Step 1: Write the failing test** (append to `logger_test.go`)

```go
func TestLoggerAttachesService(t *testing.T) {
	for _, tc := range []struct{ format, want string }{
		{"json", `"service":"bot"`},
		{"text", `service=bot`},
	} {
		var buf bytes.Buffer
		NewLogger("info", tc.format, &buf).InfoContext(WithService(context.Background(), "bot"), "hi")
		require.Contains(t, buf.String(), tc.want, tc.format)
	}
}
```

**Step 2: Run to verify it fails**

Run: `go test ./internal/telemetry/ -run TestLoggerAttachesService`
Expected: FAIL, `undefined: WithService`.

**Step 3: Implement**

`context.go`: add `serviceKey` to the const block, then:

```go
// WithService stores the service name (api, bot, worker) in ctx.
func WithService(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, serviceKey, name)
}

// Service returns the service name stored in ctx, or "".
func Service(ctx context.Context) string {
	v, _ := ctx.Value(serviceKey).(string)
	return v
}
```

`logger.go` `Handle`, before the request_id check:

```go
	if s := Service(ctx); s != "" {
		r.AddAttrs(slog.String("service", s))
	}
```

Update the `NewLogger` and `ctxHandler` doc comments to mention `service`.

**Step 4: Run tests**

Run: `go test ./internal/telemetry/`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/telemetry
git commit -m "feat(telemetry): service field from context on every log record (PRD-0008 FR-6)"
```

---

### Task 2: Telegram polling errors log with the bot context

`tg.WithErrorsHandler` gives no context, so `slog.Error` there would miss `service`. Give `New` the context.

**Files:**
- Modify: `backend/internal/adapters/telegram/bot.go:40,65`
- Modify: `backend/internal/adapters/telegram/bot_test.go:89`
- Modify: `backend/cmd/bragdoc/bot.go:63`

**Step 1: Change the signature**

```go
// New builds the bot. ctx carries log attributes (service) for errors the library reports without one.
func New(ctx context.Context, token string, r Replier, opts ...Option) (*Bot, error) {
```

and line 65:

```go
		tg.WithErrorsHandler(func(err error) { slog.ErrorContext(ctx, "telegram polling failed", slog.Any("err", err)) }),
```

Callers: `bot_test.go:89` → `New(context.Background(), "TOKEN", rep, WithServerURL(srv.URL))` (add the `context` import if missing); `cmd/bragdoc/bot.go:63` → `telegram.New(ctx, cfg.TelegramToken, uc)`.

**Step 2: Run tests**

Run: `go build ./... && go test ./internal/adapters/telegram/`
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/adapters/telegram cmd/bragdoc/bot.go
git commit -m "fix(telegram): polling errors log with the bot context"
```

---

### Task 3: Extract `runAPI` / `runBot` / `runWorker`, tag each context

Pure move: each `RunE` body (everything after `boot()`) moves into a function. No behaviour change apart from the tag and the API's `BaseContext`.

**Files:**
- Modify: `backend/cmd/bragdoc/api.go`, `bot.go`, `worker.go`

**Step 1: api.go**

```go
func apiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "Run the HTTP API",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := boot()
			if err != nil {
				return err
			}
			return runAPI(cmd.Context(), cfg)
		},
	}
}

// runAPI serves HTTP until ctx is done, then shuts down within cfg.ShutdownTimeout.
func runAPI(ctx context.Context, cfg config.Config) error {
	ctx = telemetry.WithService(ctx, "api")
	// ... the old body from postgres.Connect onwards, unchanged ...
}
```

In the `http.Server` literal add:

```go
		// Requests inherit service (FR-7) but not cancellation: Shutdown drains them.
		BaseContext: func(net.Listener) context.Context { return context.WithoutCancel(ctx) },
```

Imports: add `net` and `internal/config`.

**Step 2: bot.go and worker.go:** same shape: `runBot(ctx, cfg)` starting with `ctx = telemetry.WithService(ctx, "bot")`, `runWorker(ctx, cfg)` starting with `ctx = telemetry.WithService(ctx, "worker")`. The tag goes first, so the "idle" warnings carry it too.

**Step 3: Verify**

Run: `go build ./... && go test ./...`
Expected: PASS.

**Step 4: Commit**

```bash
git add cmd/bragdoc
git commit -m "refactor(cmd): service bodies as runAPI/runBot/runWorker, tagged with service"
```

---

### Task 4: `bragdoc all`

**Files:**
- Create: `backend/cmd/bragdoc/all.go`
- Test: `backend/cmd/bragdoc/all_test.go`
- Modify: `backend/cmd/bragdoc/main.go` (`root.AddCommand`), `backend/go.mod`

**Step 1: Write the failing test**

```go
package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunServicesFirstErrorStopsTheRest(t *testing.T) {
	boom := errors.New("boom")
	waitStop := func(ctx context.Context) error { <-ctx.Done(); return nil }

	err := runServices(context.Background(),
		service{"api", waitStop},
		service{"bot", func(context.Context) error { return boom }},
		service{"worker", waitStop},
	)

	require.ErrorIs(t, err, boom)
	require.EqualError(t, err, "bot: boom")
}

func TestRunServicesStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	waitStop := func(ctx context.Context) error { <-ctx.Done(); return nil }

	require.NoError(t, runServices(ctx, service{"api", waitStop}, service{"bot", waitStop}))
}
```

**Step 2: Run to verify it fails**

Run: `go test ./cmd/bragdoc/ -run TestRunServices`
Expected: FAIL, `undefined: runServices`.

**Step 3: Implement** `all.go`

```go
package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// allCmd runs api, bot and worker in one process (PRD-0008). No migrations.
func allCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Run the api, bot and worker together",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := boot()
			if err != nil {
				return err
			}
			return runServices(cmd.Context(),
				service{"api", func(ctx context.Context) error { return runAPI(ctx, cfg) }},
				service{"bot", func(ctx context.Context) error { return runBot(ctx, cfg) }},
				service{"worker", func(ctx context.Context) error { return runWorker(ctx, cfg) }},
			)
		},
	}
}

type service struct {
	name string
	run  func(context.Context) error
}

// runServices runs each service until ctx is done. The first error cancels the
// others, which shut down as they do alone; it is returned prefixed with the
// failing service's name.
func runServices(ctx context.Context, svcs ...service) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, s := range svcs {
		g.Go(func() error {
			if err := s.run(ctx); err != nil {
				return fmt.Errorf("%s: %w", s.name, err)
			}
			return nil
		})
	}
	return g.Wait()
}
```

`main.go`: `root.AddCommand(apiCmd(), migrateCmd(), botCmd(), workerCmd(), allCmd())`.

Run `go mod tidy`. `golang.org/x/sync` should move to the direct `require` block.

**Step 4: Run tests**

Run: `go test ./cmd/bragdoc/`
Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/bragdoc go.mod go.sum
git commit -m "feat(cmd): bragdoc all runs api, bot and worker in one process (PRD-0008 FR-1..4)"
```

---

### Task 5: JSON from the first line to the exit error

**Files:**
- Modify: `backend/cmd/bragdoc/main.go`

**Step 1: Implement**

First line of `main()`:

```go
	// JSON before config loads, so a config error is JSON too (FR-8); boot replaces it.
	slog.SetDefault(telemetry.NewLogger("info", "json", os.Stderr))
```

Replace the exit print:

```go
	if err := root.ExecuteContext(ctx); err != nil {
		slog.Error("exit", slog.Any("err", err))
		os.Exit(1)
	}
```

Drop the `fmt` import.

**Step 2: Verify**

Run: `go run ./cmd/bragdoc nope 2>&1 | jq -e .`
Expected: one JSON object, `"msg":"exit"`, `err` mentions the unknown command; jq exits 0.

Run: `env -i PATH="$PATH" HOME="$HOME" go run ./cmd/bragdoc api 2>&1 | jq -e .`
Expected: a JSON exit line with the config/settings error.

**Step 3: Commit**

```bash
git add cmd/bragdoc/main.go
git commit -m "feat(cmd): exit error and pre-config logs are JSON (PRD-0008 FR-8)"
```

---

### Task 6: Makefile and README

**Files:**
- Modify: `backend/Makefile`, `README.md:31,39`

**Step 1:** Makefile: add `run-all` to `.PHONY` and:

```make
run-all:
	go run ./cmd/bragdoc all
```

**Step 2:** README: under `make run` add `make run-all   # api + bot + worker in one process; no migrations`. Change the subcommands line to `bragdoc api | bot | worker | all | migrate`, and add one sentence: every log line is JSON with `service` (`api`, `bot`, `worker`); filter it with `jq 'select(.service=="bot")'`.

**Step 3: Commit**

```bash
git add Makefile ../README.md
git commit -m "docs: make run-all and service-tagged logs (PRD-0008 FR-11)"
```

---

### Task 7: Verify and open the PR

**Step 1:** `go test ./... && make lint`. Expected: all pass, no lint findings.

**Step 2: Manual run** (needs `docker compose up -d` for db/redis, migrations applied, `.env` sourced):

```bash
timeout -s INT 10 make run-all 2>&1 | grep -v '^go run' | tee /tmp/all.log | jq -e 'has("service")'
```

Expected: every line is `true`. You see `api listening`, `bot polling` or `bot idle`, and `worker started` or `worker idle`. After SIGINT you see `shutting down`, `bot stopped` and `worker stopped`, and the process exits 0. Hit `/healthz` during the run: the request log line has `service":"api"` and `request_id`.

**Step 3:** Failure path: unset `SUPABASE_URL` and run `make run-all`. Expected: a JSON `exit` line containing `api: SUPABASE_URL is required for api`, the bot and worker stop, and the exit code is non-zero.

**Step 4:** Mark PRD-0008 `status: accepted` and record the decisions in §13: name `all`, one pool per service, Compose profile deferred. Commit.

**Step 5:** `git push -u origin feat/run-all-services` and `gh pr create --base main`, with a summary that maps each change to its FR.
