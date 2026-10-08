# Backend Bootstrap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Scaffold the Go backend skeleton: one Cobra binary (`api`, `bot`, `worker`, `migrate`), hexagonal packages, Gin server with health/metrics, Postgres and Redis adapters with graceful degradation, embedded migrations, sqlc, Dockerfile, Makefile, lint, and a project skill for adding endpoints.

**Architecture:** Hexagonal layout under `backend/internal`: `domain` (stdlib only), `app` + `ports` (import domain), `adapters/{http,postgres,redis}` (implement ports, never import each other), `telemetry` (shared slog + Prometheus), `config`, and `cmd/bragdoc` wiring. Optional dependencies (Redis) are wrapped in a degrading decorator so their failure never fails a request. Design: `docs/plans/2026-10-08-backend-bootstrap-design.md`.

**Tech Stack:** Go 1.27, Gin, Cobra, pgx/v5, sqlc, golang-migrate (iofs + pgx5 driver), go-redis/v9, prometheus/client_golang, caarlos0/env, slog, testify, testcontainers-go, golangci-lint v1.

**Conventions for every task:**
- Working directory for all commands is `backend/` unless stated.
- Module path is `github.com/xhamps/bragdocument/backend`.
- Commit after each task with the message given. Append `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>` as the last line of every commit message.
- `go test ./...` must pass at the end of every task.

---

### Task 1: Module, Makefile, lint config, directory skeleton

**Files:**
- Create: `backend/go.mod` (via `go mod init`)
- Create: `backend/Makefile`
- Create: `backend/.golangci.yml`
- Create: `backend/internal/{domain,app,ports,adapters/http,adapters/postgres,adapters/redis,telemetry,config}/` (directories, via `.gitkeep` where empty)

**Step 1: Init the module**

Run (from repo root):
```bash
mkdir -p backend && cd backend && go mod init github.com/xhamps/bragdocument/backend
```
Expected: `go: creating new go.mod: module github.com/xhamps/bragdocument/backend`

Then edit `backend/go.mod` so the `go` directive reads `go 1.27`. If the local toolchain is older, Go downloads 1.27 automatically on the first build (GOTOOLCHAIN=auto).

**Step 2: Write the Makefile**

`backend/Makefile`:
```makefile
.PHONY: run test test-integration lint migrate sqlc docker tidy

run:
	go run ./cmd/bragdoc api

test:
	go test ./...

test-integration:
	go test -tags integration ./...

lint:
	golangci-lint run ./...

migrate:
	go run ./cmd/bragdoc migrate

sqlc:
	sqlc generate

docker:
	docker build -t bragdoc .

tidy:
	go mod tidy
```

**Step 3: Write the lint config (golangci-lint v1 format)**

`backend/.golangci.yml`:
```yaml
run:
  timeout: 3m

linters:
  enable:
    - depguard
    - errcheck
    - govet
    - staticcheck
    - unused
    - gofmt
    - goimports

linters-settings:
  depguard:
    rules:
      domain:
        files:
          - "**/internal/domain/**"
        allow:
          - $gostd
      app-and-ports:
        files:
          - "**/internal/app/**"
          - "**/internal/ports/**"
        allow:
          - $gostd
          - github.com/xhamps/bragdocument/backend/internal/domain
          - github.com/xhamps/bragdocument/backend/internal/ports
      adapters:
        files:
          - "**/internal/adapters/**"
        deny:
          - pkg: github.com/xhamps/bragdocument/backend/internal/adapters
            desc: adapters must not import each other; share code through ports, domain, or telemetry
```

**Step 4: Create the directory skeleton**

```bash
mkdir -p cmd/bragdoc internal/domain internal/app internal/ports internal/adapters/http internal/adapters/postgres internal/adapters/redis internal/telemetry internal/config migrations queries
touch internal/app/.gitkeep queries/.gitkeep
```

**Step 5: Commit**

```bash
cd .. && git add backend && git commit -m "chore(backend): init module, Makefile, lint config, layout"
```

---

### Task 2: Config from environment

**Files:**
- Create: `backend/internal/config/config.go`
- Test: `backend/internal/config/config_test.go`

**Step 1: Add dependencies**

```bash
go get github.com/caarlos0/env/v11@latest github.com/stretchr/testify@latest
```

**Step 2: Write the failing test**

`backend/internal/config/config_test.go`:
```go
package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, ":8080", cfg.HTTPAddr)
	require.Equal(t, "json", cfg.LogFormat)
	require.Equal(t, 5*time.Second, cfg.DBTimeout)
	require.Equal(t, 200*time.Millisecond, cfg.CacheTimeout)
	require.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	require.Error(t, err)
}

func TestLoadRejectsBadLogFormat(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("LOG_FORMAT", "xml")

	_, err := Load()
	require.ErrorContains(t, err, "LOG_FORMAT")
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL, `undefined: Load`

**Step 4: Write minimal implementation**

`backend/internal/config/config.go`:
```go
// Package config loads the process configuration from environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config is the full configuration for every subcommand. Unused fields for a
// given subcommand are harmless.
type Config struct {
	Env       string `env:"APP_ENV" envDefault:"development"`
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"json"`
	HTTPAddr  string `env:"HTTP_ADDR" envDefault:":8080"`

	DatabaseURL  string `env:"DATABASE_URL,required"`
	RedisURL     string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
	GotenbergURL string `env:"GOTENBERG_URL" envDefault:"http://localhost:3000"`

	TelegramToken string `env:"TELEGRAM_BOT_TOKEN"`
	TelegramMode  string `env:"TELEGRAM_MODE" envDefault:"polling"`

	DBTimeout       time.Duration `env:"DB_TIMEOUT" envDefault:"5s"`
	CacheTimeout    time.Duration `env:"CACHE_TIMEOUT" envDefault:"200ms"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`
}

// Load reads and validates the configuration. It is called once per process.
func Load() (Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if c.LogFormat != "json" && c.LogFormat != "text" {
		return Config{}, fmt.Errorf("config: LOG_FORMAT must be json or text, got %q", c.LogFormat)
	}
	return c, nil
}
```

**Step 5: Run tests**

Run: `go test ./internal/config/ -v`
Expected: PASS (3 tests)

**Step 6: Commit**

```bash
git add go.mod go.sum internal/config && git commit -m "feat(backend): env-based config with validation"
```

---

### Task 3: Domain errors

**Files:**
- Create: `backend/internal/domain/errors.go`
- Test: `backend/internal/domain/errors_test.go`

**Step 1: Write the failing test**

`backend/internal/domain/errors_test.go`:
```go
package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSentinelErrorsWrap(t *testing.T) {
	err := fmt.Errorf("document 42: %w", ErrNotFound)
	require.ErrorIs(t, err, ErrNotFound)
	require.NotErrorIs(t, err, ErrForbidden)
}

func TestValidationErrorCarriesFields(t *testing.T) {
	err := NewValidationError(map[string]string{"name": "required"})
	wrapped := fmt.Errorf("create log: %w", err)

	var ve *ValidationError
	require.True(t, errors.As(wrapped, &ve))
	require.Equal(t, "required", ve.Fields["name"])
	require.Equal(t, "validation failed", ve.Error())
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/ -v`
Expected: FAIL, `undefined: ErrNotFound`

**Step 3: Write minimal implementation**

`backend/internal/domain/errors.go`:
```go
// Package domain holds entities, value objects, and errors. It imports only
// the standard library.
package domain

import "errors"

// Sentinel errors. Use cases return these (wrapped with context); the HTTP
// adapter maps them to status codes.
var (
	ErrNotFound    = errors.New("not found")
	ErrForbidden   = errors.New("forbidden")
	ErrConflict    = errors.New("conflict")
	ErrUnavailable = errors.New("dependency unavailable") // required dependency (Postgres) down
)

// ValidationError reports per-field problems with an input.
type ValidationError struct {
	Fields map[string]string
}

// NewValidationError builds a ValidationError from field → message.
func NewValidationError(fields map[string]string) *ValidationError {
	return &ValidationError{Fields: fields}
}

func (e *ValidationError) Error() string { return "validation failed" }
```

**Step 4: Run tests**

Run: `go test ./internal/domain/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain && git commit -m "feat(backend): domain error types"
```

---

### Task 4: Ports

**Files:**
- Create: `backend/internal/ports/ports.go`

No test: interfaces only.

**Step 1: Write the interfaces**

`backend/internal/ports/ports.go`:
```go
// Package ports declares the interfaces the application layer depends on.
// Adapters implement them; use cases consume them.
package ports

import (
	"context"
	"time"
)

// Cache is a byte-oriented key/value cache with TTL. Implementations must
// never make a request fail: see adapters/redis.Degrading.
type Cache interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Pinger reports whether a dependency is reachable. Used by readiness checks.
type Pinger interface {
	Ping(ctx context.Context) error
}
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no output

**Step 3: Commit**

```bash
git add internal/ports && git commit -m "feat(backend): cache and pinger ports"
```

---

### Task 5: Telemetry: context-aware slog and Prometheus registry

**Files:**
- Create: `backend/internal/telemetry/context.go`
- Create: `backend/internal/telemetry/logger.go`
- Create: `backend/internal/telemetry/metrics.go`
- Test: `backend/internal/telemetry/logger_test.go`

**Step 1: Add dependency**

```bash
go get github.com/prometheus/client_golang@latest
```

**Step 2: Write the failing test**

`backend/internal/telemetry/logger_test.go`:
```go
package telemetry

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoggerAttachesContextIDs(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger("info", "json", &buf)

	ctx := WithRequestID(context.Background(), "req-1")
	ctx = WithTenantID(ctx, "tenant-9")
	log.InfoContext(ctx, "hello")

	require.Contains(t, buf.String(), `"request_id":"req-1"`)
	require.Contains(t, buf.String(), `"tenant_id":"tenant-9"`)
}

func TestLoggerWithoutContextIDs(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger("info", "json", &buf)

	log.Info("plain")

	require.NotContains(t, buf.String(), "request_id")
}

func TestLoggerLevel(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger("warn", "text", &buf)

	log.Info("hidden")
	log.Warn("shown")

	require.NotContains(t, buf.String(), "hidden")
	require.Contains(t, buf.String(), "shown")
}

func TestRequestIDRoundTrip(t *testing.T) {
	require.Equal(t, "", RequestID(context.Background()))
	require.Equal(t, "abc", RequestID(WithRequestID(context.Background(), "abc")))
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/telemetry/ -v`
Expected: FAIL, `undefined: NewLogger`

**Step 4: Write implementation**

`backend/internal/telemetry/context.go`:
```go
// Package telemetry provides logging and metrics plumbing shared by cmd and
// adapters.
package telemetry

import "context"

type ctxKey int

const (
	requestIDKey ctxKey = iota
	tenantIDKey
)

// WithRequestID stores the request id in ctx.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID returns the request id stored in ctx, or "".
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// WithTenantID stores the tenant id in ctx.
func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, tenantIDKey, id)
}

// TenantID returns the tenant id stored in ctx, or "".
func TenantID(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}
```

`backend/internal/telemetry/logger.go`:
```go
package telemetry

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// NewLogger builds a slog.Logger that attaches request_id and tenant_id from
// the context to every record. format is "json" or "text".
func NewLogger(level, format string, w io.Writer) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}

	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(ctxHandler{h})
}

type ctxHandler struct{ slog.Handler }

func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if id := TenantID(ctx); id != "" {
		r.AddAttrs(slog.String("tenant_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ctxHandler{h.Handler.WithAttrs(attrs)}
}

func (h ctxHandler) WithGroup(name string) slog.Handler {
	return ctxHandler{h.Handler.WithGroup(name)}
}
```

`backend/internal/telemetry/metrics.go`:
```go
package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// NewRegistry returns a Prometheus registry with Go runtime and process
// collectors already registered. Adapters register their own metrics on it.
func NewRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return reg
}
```

**Step 5: Run tests**

Run: `go test ./internal/telemetry/ -v`
Expected: PASS (4 tests)

**Step 6: Commit**

```bash
git add go.mod go.sum internal/telemetry && git commit -m "feat(backend): context-aware slog logger and prometheus registry"
```

---

### Task 6: HTTP adapter: error mapping

**Files:**
- Create: `backend/internal/adapters/http/errors.go`
- Test: `backend/internal/adapters/http/errors_test.go`

**Step 1: Add dependencies**

```bash
go get github.com/gin-gonic/gin@latest github.com/google/uuid@latest
```

**Step 2: Write the failing test**

`backend/internal/adapters/http/errors_test.go`:
```go
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestRespondErrorMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"not found", domain.ErrNotFound, 404, "not_found"},
		{"forbidden", domain.ErrForbidden, 403, "forbidden"},
		{"conflict", domain.ErrConflict, 409, "conflict"},
		{"validation", domain.NewValidationError(map[string]string{"name": "required"}), 422, "validation"},
		{"timeout", context.DeadlineExceeded, 504, "timeout"},
		{"unavailable", domain.ErrUnavailable, 503, "unavailable"},
		{"unknown", errors.New("boom"), 500, "internal"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil).
				WithContext(telemetry.WithRequestID(context.Background(), "req-7"))

			RespondError(c, tc.err)

			require.Equal(t, tc.status, rec.Code)
			var body ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tc.code, body.Error)
			require.Equal(t, "req-7", body.RequestID)
			if tc.code == "validation" {
				require.Equal(t, "required", body.Fields["name"])
			}
			if tc.code == "unavailable" {
				require.Equal(t, "5", rec.Header().Get("Retry-After"))
			}
		})
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/http/ -v`
Expected: FAIL, `undefined: RespondError`

**Step 4: Write implementation**

`backend/internal/adapters/http/errors.go`:
```go
// Package http is the Gin adapter: engine, middleware, health, metrics, and
// the mapping from domain errors to HTTP responses.
package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// ErrorResponse is the body of every error response.
type ErrorResponse struct {
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id"`
	Fields    map[string]string `json:"fields,omitempty"`
}

func errorBody(c *gin.Context, code, msg string, fields map[string]string) ErrorResponse {
	return ErrorResponse{
		Error:     code,
		Message:   msg,
		RequestID: telemetry.RequestID(c.Request.Context()),
		Fields:    fields,
	}
}

// RespondError writes err as an HTTP response. Handlers call it for every
// error returned by a use case; no handler maps errors itself.
func RespondError(c *gin.Context, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, errorBody(c, "validation", "validation failed", ve.Fields))
	case errors.Is(err, domain.ErrNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, errorBody(c, "not_found", "resource not found", nil))
	case errors.Is(err, domain.ErrForbidden):
		c.AbortWithStatusJSON(http.StatusForbidden, errorBody(c, "forbidden", "not allowed", nil))
	case errors.Is(err, domain.ErrConflict):
		c.AbortWithStatusJSON(http.StatusConflict, errorBody(c, "conflict", "conflict with current state", nil))
	case errors.Is(err, context.DeadlineExceeded):
		c.AbortWithStatusJSON(http.StatusGatewayTimeout, errorBody(c, "timeout", "upstream timed out", nil))
	case errors.Is(err, domain.ErrUnavailable):
		c.Header("Retry-After", "5")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, errorBody(c, "unavailable", "service temporarily unavailable", nil))
	default:
		slog.ErrorContext(c.Request.Context(), "unhandled error", slog.Any("err", err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody(c, "internal", "internal server error", nil))
	}
}
```

**Step 5: Run tests**

Run: `go test ./internal/adapters/http/ -v`
Expected: PASS (7 subtests)

**Step 6: Commit**

```bash
git add go.mod go.sum internal/adapters/http && git commit -m "feat(backend): map domain errors to HTTP responses"
```

---

### Task 7: HTTP adapter: engine and middleware (request id, logging, recovery, metrics)

**Files:**
- Create: `backend/internal/adapters/http/middleware.go`
- Create: `backend/internal/adapters/http/metrics.go`
- Create: `backend/internal/adapters/http/engine.go`
- Test: `backend/internal/adapters/http/engine_test.go`

**Step 1: Write the failing test**

`backend/internal/adapters/http/engine_test.go`:
```go
package http

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func newTestEngine(t *testing.T) (*gin.Engine, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(telemetry.NewLogger("info", "json", &buf))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return NewEngine(telemetry.NewRegistry()), &buf
}

func TestRequestIDIsEchoedAndGenerated(t *testing.T) {
	e, _ := newTestEngine(t)
	e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(HeaderRequestID, "given-id")
	e.ServeHTTP(rec, req)
	require.Equal(t, "given-id", rec.Header().Get(HeaderRequestID))

	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.NotEmpty(t, rec.Header().Get(HeaderRequestID))
}

func TestRequestIsLoggedWithRequestID(t *testing.T) {
	e, buf := newTestEngine(t)
	e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(HeaderRequestID, "log-me")
	e.ServeHTTP(httptest.NewRecorder(), req)

	require.Contains(t, buf.String(), `"request_id":"log-me"`)
	require.Contains(t, buf.String(), `"status":200`)
}

func TestPanicBecomes500WithRequestID(t *testing.T) {
	e, _ := newTestEngine(t)
	e.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set(HeaderRequestID, "p-1")
	e.ServeHTTP(rec, req)

	require.Equal(t, 500, rec.Code)
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "internal", body.Error)
	require.Equal(t, "p-1", body.RequestID)
}

func TestMetricsAreExposed(t *testing.T) {
	reg := telemetry.NewRegistry()
	e := NewEngine(reg)
	e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })

	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	n, err := testutil.GatherAndCount(reg, "http_requests_total")
	require.NoError(t, err)
	require.Equal(t, 1, n)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, 200, rec.Code)
	require.Contains(t, rec.Body.String(), `http_requests_total{method="GET",route="/ping",status="200"} 1`)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/http/ -run 'RequestID|Logged|Panic|Metrics' -v`
Expected: FAIL, `undefined: NewEngine`

**Step 3: Write implementation**

`backend/internal/adapters/http/middleware.go`:
```go
package http

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// HeaderRequestID is read from the request and always set on the response.
const HeaderRequestID = "X-Request-ID"

// RequestID reads or generates a request id and stores it in the request
// context so logs and error bodies can carry it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		c.Header(HeaderRequestID, id)
		c.Request = c.Request.WithContext(telemetry.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}

// Logger writes one structured line per request.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.LogAttrs(c.Request.Context(), slog.LevelInfo, "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
		)
	}
}

// Recovery turns a panic into a 500 with the request id, and logs the stack.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.LogAttrs(c.Request.Context(), slog.LevelError, "panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					errorBody(c, "internal", "internal server error", nil))
			}
		}()
		c.Next()
	}
}
```

`backend/internal/adapters/http/metrics.go`:
```go
package http

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template, and status.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method, route template, and status.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
	}
	reg.MustRegister(m.requests, m.duration)
	return m
}

// Middleware records one observation per request, labelled by the route
// template (not the raw path) so cardinality stays bounded.
func (m *metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		m.requests.WithLabelValues(c.Request.Method, route, status).Inc()
		m.duration.WithLabelValues(c.Request.Method, route, status).Observe(time.Since(start).Seconds())
	}
}
```

`backend/internal/adapters/http/engine.go`:
```go
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewEngine builds the Gin engine with the standard middleware chain and the
// /metrics endpoint. Feature handlers register their routes on it afterwards.
//
// Middleware order: recovery → request id → logger → metrics. Recovery runs
// outermost so it catches panics from everything below; the request id is
// already in the context by the time a handler can panic.
func NewEngine(reg *prometheus.Registry) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	m := newMetrics(reg)
	e.Use(Recovery(), RequestID(), Logger(), m.Middleware())
	e.GET("/metrics", gin.WrapH(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))
	return e
}
```

**Step 4: Run tests**

Run: `go test ./internal/adapters/http/ -v`
Expected: PASS (all)

**Step 5: Commit**

```bash
git add internal/adapters/http && git commit -m "feat(backend): gin engine with request id, logging, recovery, metrics"
```

---

### Task 8: HTTP adapter: health and readiness

**Files:**
- Create: `backend/internal/adapters/http/health.go`
- Test: `backend/internal/adapters/http/health_test.go`

**Step 1: Write the failing test**

`backend/internal/adapters/http/health_test.go`:
```go
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func ping(err error) func(context.Context) error {
	return func(context.Context) error { return err }
}

func readyz(t *testing.T, checks []Check) (int, map[string]string) {
	t.Helper()
	e, _ := newTestEngine(t)
	RegisterHealth(e, checks)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return rec.Code, body
}

func TestHealthz(t *testing.T) {
	e, _ := newTestEngine(t)
	RegisterHealth(e, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, 200, rec.Code)
}

func TestReadyzAllHealthy(t *testing.T) {
	code, body := readyz(t, []Check{
		{Name: "postgres", Required: true, Ping: ping(nil)},
		{Name: "cache", Ping: ping(nil)},
	})
	require.Equal(t, 200, code)
	require.Equal(t, map[string]string{"postgres": "ok", "cache": "ok"}, body)
}

func TestReadyzOptionalDownIsDegradedBut200(t *testing.T) {
	code, body := readyz(t, []Check{
		{Name: "postgres", Required: true, Ping: ping(nil)},
		{Name: "cache", Ping: ping(errors.New("conn refused"))},
	})
	require.Equal(t, 200, code)
	require.Equal(t, "degraded", body["cache"])
}

func TestReadyzRequiredDownIs503(t *testing.T) {
	code, body := readyz(t, []Check{
		{Name: "postgres", Required: true, Ping: ping(errors.New("conn refused"))},
		{Name: "cache", Ping: ping(nil)},
	})
	require.Equal(t, 503, code)
	require.Equal(t, "down", body["postgres"])
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/http/ -run 'Healthz|Readyz' -v`
Expected: FAIL, `undefined: Check`

**Step 3: Write implementation**

`backend/internal/adapters/http/health.go`:
```go
package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Check is one dependency probed by /readyz. Required dependencies make
// readiness fail with 503; optional ones are reported as "degraded" while the
// service keeps answering 200.
type Check struct {
	Name     string
	Required bool
	Ping     func(context.Context) error
}

// RegisterHealth adds /healthz (liveness) and /readyz (readiness).
func RegisterHealth(e *gin.Engine, checks []Check) {
	e.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	e.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		status := http.StatusOK
		body := make(map[string]string, len(checks))
		for _, ch := range checks {
			switch err := ch.Ping(ctx); {
			case err == nil:
				body[ch.Name] = "ok"
			case ch.Required:
				body[ch.Name] = "down"
				status = http.StatusServiceUnavailable
			default:
				body[ch.Name] = "degraded"
			}
		}
		c.JSON(status, body)
	})
}
```

**Step 4: Run tests**

Run: `go test ./internal/adapters/http/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters/http && git commit -m "feat(backend): healthz and readyz with required/optional checks"
```

---

### Task 9: Migrations package and initial migration

**Files:**
- Create: `backend/migrations/migrations.go`
- Create: `backend/migrations/0001_init.up.sql`
- Create: `backend/migrations/0001_init.down.sql`

No unit test: verified by the integration test in Task 10.

**Step 1: Write the files**

`backend/migrations/migrations.go`:
```go
// Package migrations embeds the SQL migration files so the binary can apply
// them without a separate CLI. Files are NNNN_name.up.sql / .down.sql.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

`backend/migrations/0001_init.up.sql`:
```sql
-- Proves the migration pipeline. Feature tables come with their PRDs.
CREATE TABLE app_meta (
    key   text PRIMARY KEY,
    value text NOT NULL
);

INSERT INTO app_meta (key, value) VALUES ('schema', '1');
```

`backend/migrations/0001_init.down.sql`:
```sql
DROP TABLE app_meta;
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no output

**Step 3: Commit**

```bash
git add migrations && git commit -m "feat(backend): embedded migrations with initial app_meta table"
```

---

### Task 10: sqlc config and first generated query

**Files:**
- Create: `backend/sqlc.yaml`
- Create: `backend/queries/meta.sql`
- Delete: `backend/queries/.gitkeep`
- Generated: `backend/internal/adapters/postgres/sqlcgen/*.go` (committed)

**Step 1: Add the pgx dependency**

```bash
go get github.com/jackc/pgx/v5@latest
```

**Step 2: Write sqlc config and query**

`backend/sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations"
    queries: "queries"
    gen:
      go:
        package: "sqlcgen"
        out: "internal/adapters/postgres/sqlcgen"
        sql_package: "pgx/v5"
        emit_json_tags: false
        emit_interface: true
```

`backend/queries/meta.sql`:
```sql
-- name: GetMeta :one
SELECT value FROM app_meta WHERE key = $1;
```

```bash
rm queries/.gitkeep
```

**Step 3: Generate**

Run: `make sqlc`
Expected: files `db.go`, `models.go`, `querier.go`, `meta.sql.go` under `internal/adapters/postgres/sqlcgen/`.

Run: `go build ./...`
Expected: no output

**Step 4: Commit**

```bash
git add go.mod go.sum sqlc.yaml queries internal/adapters/postgres/sqlcgen && git commit -m "feat(backend): sqlc config and generated GetMeta query"
```

---

### Task 11: Postgres adapter: pool, tenant-scoped transaction, migrate

**Files:**
- Create: `backend/internal/adapters/postgres/db.go`
- Create: `backend/internal/adapters/postgres/migrate.go`
- Test: `backend/internal/adapters/postgres/db_integration_test.go` (build tag `integration`)
- Test: `backend/internal/adapters/postgres/migrate_test.go`

**Step 1: Add dependencies**

```bash
go get github.com/golang-migrate/migrate/v4@latest github.com/golang-migrate/migrate/v4/database/pgx/v5@latest github.com/golang-migrate/migrate/v4/source/iofs@latest
go get github.com/testcontainers/testcontainers-go@latest github.com/testcontainers/testcontainers-go/modules/postgres@latest
```

**Step 2: Write the failing unit test (URL rewriting)**

`backend/internal/adapters/postgres/migrate_test.go`:
```go
package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrateURL(t *testing.T) {
	require.Equal(t, "pgx5://u:p@h:5432/db?sslmode=disable", migrateURL("postgres://u:p@h:5432/db?sslmode=disable"))
	require.Equal(t, "pgx5://u:p@h:5432/db", migrateURL("postgresql://u:p@h:5432/db"))
	require.Equal(t, "pgx5://u:p@h:5432/db", migrateURL("pgx5://u:p@h:5432/db"))
}
```

Run: `go test ./internal/adapters/postgres/ -v`
Expected: FAIL, `undefined: migrateURL`

**Step 3: Write the integration test**

`backend/internal/adapters/postgres/db_integration_test.go`:
```go
//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
)

func startPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("brag"),
		tcpostgres.WithUsername("brag"),
		tcpostgres.WithPassword("brag"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	url, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	return url
}

func TestMigrateAndTenantScopedTx(t *testing.T) {
	url := startPostgres(t)
	ctx := context.Background()

	require.NoError(t, Migrate(url))
	require.NoError(t, Migrate(url), "second run is a no-op")

	db, err := Connect(ctx, url, 5*time.Second)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Ping(ctx))

	err = db.WithTenant(ctx, "tenant-a", func(ctx context.Context, tx pgx.Tx) error {
		var tenant string
		if err := tx.QueryRow(ctx, "SELECT current_setting('app.tenant_id', true)").Scan(&tenant); err != nil {
			return err
		}
		require.Equal(t, "tenant-a", tenant)

		v, err := sqlcgen.New(tx).GetMeta(ctx, "schema")
		require.NoError(t, err)
		require.Equal(t, "1", v)
		return nil
	})
	require.NoError(t, err)

	// Outside the transaction the setting is gone.
	var after string
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT current_setting('app.tenant_id', true)").Scan(&after))
	require.Equal(t, "", after)
}

func TestConnectFailsFastWhenDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err := Connect(ctx, "postgres://u:p@127.0.0.1:1/db?sslmode=disable", time.Second)
	require.Error(t, err)
}
```

**Step 4: Write implementation**

`backend/internal/adapters/postgres/db.go`:
```go
// Package postgres is the pgx adapter: connection pool, tenant-scoped
// transactions, migrations, and repository implementations.
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

const connectAttempts = 5

// DB wraps the pool and the per-call timeout.
type DB struct {
	Pool    *pgxpool.Pool
	timeout time.Duration
}

// Connect opens the pool and verifies it with a bounded number of retries.
// Postgres is a required dependency: callers exit when this fails.
func Connect(ctx context.Context, url string, timeout time.Duration) (*DB, error) {
	var lastErr error
	for attempt := 1; attempt <= connectAttempts; attempt++ {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, timeout)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				return &DB{Pool: pool, timeout: timeout}, nil
			}
			pool.Close()
		}
		lastErr = err
		backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond // ponytail: quadratic backoff, 0.5s..12.5s; jitter if herds appear
		slog.WarnContext(ctx, "postgres not ready", slog.Int("attempt", attempt), slog.Any("err", err))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, fmt.Errorf("postgres: connect after %d attempts: %w", connectAttempts, lastErr)
}

// Ping reports pool health; used by /readyz.
func (d *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	return d.Pool.Ping(ctx)
}

// Close releases the pool.
func (d *DB) Close() { d.Pool.Close() }

// WithTenant runs fn inside a transaction whose app.tenant_id setting is set
// for the duration of the transaction. Row-level-security policies read that
// setting. Every tenant-scoped repository call goes through here.
func (d *DB) WithTenant(ctx context.Context, tenantID string, fn func(ctx context.Context, tx pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: begin: %v", domain.ErrUnavailable, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SET LOCAL cannot take bind parameters; set_config with is_local=true is the equivalent.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("postgres: set tenant: %w", err)
	}
	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}
```

`backend/internal/adapters/postgres/migrate.go`:
```go
package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the pgx5:// driver
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/xhamps/bragdocument/backend/migrations"
)

// Migrate applies every pending migration embedded in the binary.
func Migrate(databaseURL string) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migrate: source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL(databaseURL))
	if err != nil {
		return fmt.Errorf("migrate: init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}
	return nil
}

// migrateURL rewrites a postgres:// URL to the pgx5:// scheme golang-migrate expects.
func migrateURL(u string) string {
	for _, p := range []string{"postgresql://", "postgres://"} {
		if strings.HasPrefix(u, p) {
			return "pgx5://" + strings.TrimPrefix(u, p)
		}
	}
	return u
}
```

**Step 5: Run unit tests, then integration tests (Docker must be running)**

Run: `go test ./internal/adapters/postgres/ -v`
Expected: PASS (`TestMigrateURL`)

Run: `go test -tags integration ./internal/adapters/postgres/ -v`
Expected: PASS (`TestMigrateAndTenantScopedTx`, `TestConnectFailsFastWhenDown`). First run pulls `postgres:16-alpine`.

**Step 6: Commit**

```bash
go mod tidy
git add go.mod go.sum internal/adapters/postgres && git commit -m "feat(backend): postgres pool, tenant-scoped tx, embedded migrations"
```

---

### Task 12: Redis adapter with degrading decorator

**Files:**
- Create: `backend/internal/adapters/redis/cache.go`
- Create: `backend/internal/adapters/redis/degrading.go`
- Test: `backend/internal/adapters/redis/degrading_test.go`

**Step 1: Add dependency**

```bash
go get github.com/redis/go-redis/v9@latest
```

**Step 2: Write the failing test**

`backend/internal/adapters/redis/degrading_test.go`:
```go
package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	err  error
	data map[string][]byte
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	v, ok := f.data[key]
	return v, ok, nil
}

func (f *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.data[key] = value
	return nil
}

func (f *fakeCache) Delete(_ context.Context, key string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.data, key)
	return nil
}

func TestDegradingSwallowsErrorsAndCounts(t *testing.T) {
	reg := prometheus.NewRegistry()
	inner := &fakeCache{err: errors.New("connection refused")}
	d := NewDegrading(inner, reg)
	ctx := context.Background()

	v, found, err := d.Get(ctx, "k")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, v)

	require.NoError(t, d.Set(ctx, "k", []byte("v"), time.Minute))
	require.NoError(t, d.Delete(ctx, "k"))

	require.False(t, d.Healthy())
	require.Equal(t, float64(3), testutil.ToFloat64(d.errors))
}

func TestDegradingPassesThroughWhenHealthy(t *testing.T) {
	inner := &fakeCache{data: map[string][]byte{}}
	d := NewDegrading(inner, prometheus.NewRegistry())
	ctx := context.Background()

	require.NoError(t, d.Set(ctx, "k", []byte("v"), time.Minute))
	v, found, err := d.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []byte("v"), v)
	require.True(t, d.Healthy())
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/redis/ -v`
Expected: FAIL, `undefined: NewDegrading`

**Step 4: Write implementation**

`backend/internal/adapters/redis/cache.go`:
```go
// Package redis is the go-redis adapter implementing ports.Cache, plus the
// Degrading decorator that makes cache failures non-fatal.
package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache talks to Redis. Redis is optional: Connect never fails hard, it only
// warns, because go-redis reconnects on its own once the server is back.
type Cache struct {
	client  *redis.Client
	timeout time.Duration
}

// Connect parses url, opens the client, and pings once. A failed ping is
// logged, not returned: wrap the result in Degrading and keep going.
func Connect(ctx context.Context, url string, timeout time.Duration) (*Cache, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	c := &Cache{client: redis.NewClient(opt), timeout: timeout}
	if err := c.Ping(ctx); err != nil {
		slog.WarnContext(ctx, "redis not reachable at startup; running degraded", slog.Any("err", err))
	}
	return c, nil
}

func (c *Cache) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.timeout)
}

// Ping reports connectivity; used by /readyz.
func (c *Cache) Ping(ctx context.Context) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.client.Ping(ctx).Err()
}

// Close releases the client.
func (c *Cache) Close() error { return c.client.Close() }

// Get implements ports.Cache.
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	v, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

// Set implements ports.Cache.
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete implements ports.Cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.client.Del(ctx, key).Err()
}
```

`backend/internal/adapters/redis/degrading.go`:
```go
package redis

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/xhamps/bragdocument/backend/internal/ports"
)

const degradedLogInterval = time.Minute

// Degrading wraps a ports.Cache so that every error becomes a cache miss.
// It counts errors in cache_errors_total and logs at most once per minute.
// cmd always wires this around the real cache: a feature may fail because
// Postgres is down, never because Redis is.
type Degrading struct {
	inner   ports.Cache
	errors  prometheus.Counter
	healthy atomic.Bool
	lastLog atomic.Int64
}

// NewDegrading registers the error counter on reg and returns the decorator.
func NewDegrading(inner ports.Cache, reg prometheus.Registerer) *Degrading {
	d := &Degrading{
		inner: inner,
		errors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "cache_errors_total",
			Help: "Cache operations that failed and were treated as misses.",
		}),
	}
	d.healthy.Store(true)
	reg.MustRegister(d.errors)
	return d
}

// Healthy is false after the last operation failed, true after one succeeds.
func (d *Degrading) Healthy() bool { return d.healthy.Load() }

func (d *Degrading) fail(ctx context.Context, op string, err error) {
	d.errors.Inc()
	d.healthy.Store(false)
	now := time.Now().Unix()
	last := d.lastLog.Load()
	if now-last >= int64(degradedLogInterval.Seconds()) && d.lastLog.CompareAndSwap(last, now) {
		slog.WarnContext(ctx, "cache degraded, treating as miss", slog.String("op", op), slog.Any("err", err))
	}
}

// Get implements ports.Cache; errors become misses.
func (d *Degrading) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, found, err := d.inner.Get(ctx, key)
	if err != nil {
		d.fail(ctx, "get", err)
		return nil, false, nil
	}
	d.healthy.Store(true)
	return v, found, nil
}

// Set implements ports.Cache; errors are dropped.
func (d *Degrading) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := d.inner.Set(ctx, key, value, ttl); err != nil {
		d.fail(ctx, "set", err)
		return nil
	}
	d.healthy.Store(true)
	return nil
}

// Delete implements ports.Cache; errors are dropped.
func (d *Degrading) Delete(ctx context.Context, key string) error {
	if err := d.inner.Delete(ctx, key); err != nil {
		d.fail(ctx, "delete", err)
		return nil
	}
	d.healthy.Store(true)
	return nil
}
```

**Step 5: Run tests**

Run: `go test ./internal/adapters/redis/ -v`
Expected: PASS (2 tests)

**Step 6: Commit**

```bash
git add go.mod go.sum internal/adapters/redis && git commit -m "feat(backend): redis cache adapter with degrading decorator"
```

---

### Task 13: Cobra binary: api, bot, worker, migrate

**Files:**
- Create: `backend/cmd/bragdoc/main.go`
- Create: `backend/cmd/bragdoc/api.go`
- Create: `backend/cmd/bragdoc/migrate.go`
- Create: `backend/cmd/bragdoc/daemon.go` (bot and worker stubs)

**Step 1: Add dependency**

```bash
go get github.com/spf13/cobra@latest
```

**Step 2: Write the commands**

`backend/cmd/bragdoc/main.go`:
```go
// bragdoc is the single backend binary. Subcommands select the mode.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/config"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func main() {
	root := &cobra.Command{
		Use:           "bragdoc",
		Short:         "Brag Document backend",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(apiCmd(), migrateCmd(), daemonCmd("bot"), daemonCmd("worker"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// boot loads config and installs the process logger. Every subcommand starts here.
func boot() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, err
	}
	slog.SetDefault(telemetry.NewLogger(cfg.LogLevel, cfg.LogFormat, os.Stderr))
	return cfg, nil
}
```

`backend/cmd/bragdoc/api.go`:
```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	httpadapter "github.com/xhamps/bragdocument/backend/internal/adapters/http"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func apiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "Run the HTTP API",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := boot()
			if err != nil {
				return err
			}

			db, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DBTimeout)
			if err != nil {
				return err
			}
			defer db.Close()

			rc, err := redis.Connect(ctx, cfg.RedisURL, cfg.CacheTimeout)
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }() // deferred after db.Close, so runs first: Redis closes before the pool

			reg := telemetry.NewRegistry()
			_ = redis.NewDegrading(rc, reg) // ponytail: wired now so the counter exists; use cases take it in the next pass

			engine := httpadapter.NewEngine(reg)
			httpadapter.RegisterHealth(engine, []httpadapter.Check{
				{Name: "postgres", Required: true, Ping: db.Ping},
				{Name: "cache", Ping: rc.Ping},
			})

			srv := &http.Server{
				Addr:              cfg.HTTPAddr,
				Handler:           engine,
				ReadHeaderTimeout: 5 * time.Second,
			}
			errCh := make(chan error, 1)
			go func() { errCh <- srv.ListenAndServe() }()
			slog.InfoContext(ctx, "api listening", slog.String("addr", cfg.HTTPAddr))

			select {
			case <-ctx.Done():
			case err := <-errCh:
				if !errors.Is(err, http.ErrServerClosed) {
					return err
				}
			}

			slog.InfoContext(ctx, "shutting down", slog.Duration("timeout", cfg.ShutdownTimeout))
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		},
	}
}
```

`backend/cmd/bragdoc/migrate.go`:
```go
package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
)

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending database migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := boot()
			if err != nil {
				return err
			}
			if err := postgres.Migrate(cfg.DatabaseURL); err != nil {
				return err
			}
			slog.InfoContext(cmd.Context(), "migrations applied")
			return nil
		},
	}
}
```

`backend/cmd/bragdoc/daemon.go`:
```go
package main

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// daemonCmd builds the bot and worker commands. Both are placeholders that
// start, log, and wait for a signal; their loops arrive with PRD-0003 and
// PRD-0006.
func daemonCmd(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: "Run the " + name + " process",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if _, err := boot(); err != nil {
				return err
			}
			slog.InfoContext(ctx, name+" started")
			<-ctx.Done()
			slog.InfoContext(ctx, name+" stopped")
			return nil
		},
	}
}
```

**Step 3: Build and smoke-test against the compose infra**

From the repo root:
```bash
docker compose up -d postgres redis
cd backend
go build ./... && go vet ./...
DATABASE_URL=postgres://brag:brag@localhost:5432/brag?sslmode=disable go run ./cmd/bragdoc migrate
```
Expected: a JSON log line `"msg":"migrations applied"`.

```bash
DATABASE_URL=postgres://brag:brag@localhost:5432/brag?sslmode=disable go run ./cmd/bragdoc api &
sleep 2
curl -s localhost:8080/readyz; echo
curl -s -o /dev/null -w '%{http_code}\n' localhost:8080/metrics
docker compose -f ../docker-compose.yml stop redis
curl -s localhost:8080/readyz; echo
docker compose -f ../docker-compose.yml start redis
kill %1
```
Expected: `{"cache":"ok","postgres":"ok"}`, then `200`, then `{"cache":"degraded","postgres":"ok"}`, and the api logs `shutting down` on kill.

**Step 4: Commit**

```bash
go mod tidy
git add go.mod go.sum cmd && git commit -m "feat(backend): bragdoc binary with api, migrate, bot, worker commands"
```

---

### Task 14: Dockerfile and compose check

**Files:**
- Create: `backend/Dockerfile`
- Create: `backend/.dockerignore`

**Step 1: Write the files**

`backend/Dockerfile`:
```dockerfile
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bragdoc ./cmd/bragdoc

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bragdoc /bragdoc
EXPOSE 8080
ENTRYPOINT ["/bragdoc"]
CMD ["api"]
```

`backend/.dockerignore`:
```
bin/
*_test.go
.golangci.yml
Makefile
```

**Step 2: Build and run through compose**

From the repo root:
```bash
cp -n .env.example .env
docker compose --profile app build api
docker compose --profile app up -d api
sleep 3
curl -s localhost:8080/readyz; echo
docker compose --profile app logs api | tail -5
docker compose --profile app down
```
Expected: `{"cache":"ok","postgres":"ok"}` and JSON logs from the container. `bot`, `worker`, and `frontend` build contexts: bot and worker reuse the backend image; `frontend/` does not exist yet, so start only `api` as shown.

**Step 3: Commit**

```bash
git add backend/Dockerfile backend/.dockerignore && git commit -m "build(backend): multi-stage Dockerfile"
```

---

### Task 15: Lint gate

**Step 1: Run lint**

Run (in `backend/`): `make lint`
Expected: no findings. If `goimports` complains about grouping, run `goimports -w -local github.com/xhamps/bragdocument .` and re-run.

**Step 2: Prove depguard works**

Temporarily add `import _ "github.com/gin-gonic/gin"` to `internal/domain/errors.go`, run `make lint`, expect a depguard error naming the domain rule, then revert the change.

**Step 3: Commit (only if lint required fixes)**

```bash
git add -A backend && git commit -m "chore(backend): lint fixes"
```

---

### Task 16: Project skill for adding endpoints

**Files:**
- Create: `.claude/skills/backend-endpoint/SKILL.md` (repo root)

**Step 1: Write the skill**

```markdown
---
name: backend-endpoint
description: How to add or change an API endpoint, use case, repository, or migration in backend/. Use whenever touching backend/ for a feature. Enforces the hexagonal layering and graceful-degradation rules from ADR-0012.
---

# Adding a backend feature

Read first: `docs/adr/0012-hexagonal-backend-layout.md`, then the PRD for the feature.

## Two rules

1. **Dependency rule.** `domain` imports stdlib only. `app` and `ports` import `domain`. `adapters/*` implement ports and never import each other. `cmd` wires. Check: `make lint` (depguard).
2. **Degradation rule.** A feature may fail because Postgres is down. It may never fail because Redis, Gotenberg, or Telegram is down. Use `ports.Cache` through the `redis.Degrading` decorator; never call go-redis directly from a use case.

## Steps, in order

1. **Migration**: `backend/migrations/NNNN_<name>.up.sql` and `.down.sql`. Every tenant-scoped table has `tenant_id` and will get an RLS policy (see ADR-0007). Run `make migrate` against the compose Postgres.
2. **Query**: add to `backend/queries/<resource>.sql` with `-- name: X :one|:many|:exec`. Run `make sqlc`. Commit the generated code.
3. **Domain**: add or extend the entity in `internal/domain/<resource>.go`. Validation lives here and returns `domain.NewValidationError(...)`. Return `domain.ErrNotFound`, `ErrForbidden`, `ErrConflict` as appropriate.
4. **Port**: add the repository interface (or method) in `internal/ports/<resource>.go`. Keep it small: only what the use case calls.
5. **Adapter**: implement the port in `internal/adapters/postgres/<resource>_repo.go`. Every tenant-scoped call runs inside `db.WithTenant(ctx, tenantID, func(ctx, tx) error { q := sqlcgen.New(tx); ... })`. Map `pgx.ErrNoRows` to `domain.ErrNotFound`. Integration test with build tag `integration`, following `db_integration_test.go`.
6. **Use case**: one file per use case in `internal/app/<resource>_<verb>.go` (e.g. `log_create.go`). Constructor takes ports. Unit test with a hand-written fake of the port in the same package; no mock framework.
7. **Handler**: `internal/adapters/http/<resource>_handler.go`. Bind with Gin tags into a `<Verb><Resource>Request` struct, call exactly one use case, write a `<Resource>Response`. On error call `RespondError(c, err)` and nothing else. Register routes in a `Register<Resource>(e *gin.Engine, uc ...)` function and call it from `cmd/bragdoc/api.go`.
8. **Errors**: if the feature introduces a new domain error, add a row to `RespondError` and to its test table.
9. **HTTP test**: `httptest` with `NewEngine(telemetry.NewRegistry())` and a fake use case, following `health_test.go`.
10. **Contract**: update `backend/api/openapi.yaml`.
11. **Docs**: link the PRD or ADR in the PR description. A new technical choice gets a new ADR.

## Conventions

- Route groups per resource: `/documents`, `/documents/:id/logs`.
- DTO suffixes: `Request`, `Response`. Never expose sqlc or domain structs directly.
- Context carries request id and tenant id (`telemetry.RequestID`, `telemetry.TenantID`). Use `slog.InfoContext(ctx, ...)` so they are attached.
- Timeouts come from config (`DBTimeout`, `CacheTimeout`); do not hard-code.

## Worked examples in the skeleton

- Readiness with required vs optional dependencies: `internal/adapters/http/health.go`.
- Degradation decorator and its test: `internal/adapters/redis/degrading.go`.
- Tenant-scoped transaction and integration test: `internal/adapters/postgres/db.go`, `db_integration_test.go`.

## Anti-patterns

- Gin, pgx, or go-redis types in `app` or `domain`.
- Business logic in a handler, or a handler calling two use cases.
- A query on a tenant-scoped table outside `WithTenant`.
- Swallowing an error without a metric or log.
- A new interface with one implementation and no test using a fake.
```

**Step 2: Commit**

```bash
git add .claude/skills/backend-endpoint/SKILL.md && git commit -m "docs: project skill for adding backend endpoints"
```

---

### Task 17: README and compose wiring check

**Files:**
- Modify: `README.md` (repo root)

**Step 1: Add a backend section to the root README**

Append:
```markdown
## Backend

```sh
cd backend
make migrate   # apply migrations to the compose Postgres
make run       # api on :8080 → /healthz /readyz /metrics
make test      # unit tests
make test-integration   # needs Docker (testcontainers)
make lint
```

Subcommands: `bragdoc api | bot | worker | migrate`. Layout and rules: `docs/adr/0012-hexagonal-backend-layout.md`. Adding a feature: `.claude/skills/backend-endpoint/SKILL.md`.
```

**Step 2: Final verification**

```bash
cd backend && make lint && make test && go build ./... && cd .. && docker compose config -q
```
Expected: all pass, no output from `compose config`.

**Step 3: Commit**

```bash
git add README.md && git commit -m "docs: backend run instructions"
```
