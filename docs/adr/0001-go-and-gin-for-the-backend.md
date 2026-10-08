---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# Go and Gin for the backend

## Context and Problem Statement

The backend serves a JSON API for the web frontend, runs the Telegram bot, enforces authorization, and generates reports. Which language and HTTP framework do we use?

## Decision Drivers

* Project requirement: Go with Gin
* Single static binary, simple Docker image
* Good concurrency for the bot poller and report jobs
* Mature middleware ecosystem (auth, CORS, logging, recovery)

## Considered Options

* Go with Gin
* Go with the standard library `net/http` and `chi`
* Go with Echo or Fiber

## Decision Outcome

Chosen option: "Go with Gin", because it is the stated requirement and it meets every driver: routing groups map cleanly onto `/tenants/{id}/documents/{id}/logs`, middleware covers JWT verification and request logging, and binding/validation of request bodies is built in.

### Consequences

* Good, because one binary runs the API, the bot worker, and the export worker, selected by a command flag.
* Good, because Gin is widely documented; hiring and onboarding are easy.
* Neutral, because Gin's context differs from `net/http`; handlers stay thin and business logic lives in plain Go packages that do not import Gin.
* Bad, because Gin's default JSON rendering and validation tags need discipline to keep errors consistent; a shared error-response helper is required.

### Confirmation

`backend/` compiles to one module; `go vet` and `golangci-lint` run in CI; business packages under `backend/internal/` must not import `github.com/gin-gonic/gin` (checked with a lint rule).

## Pros and Cons of the Options

### Go with Gin

* Good, because required by the project.
* Good, because fast router, large ecosystem, built-in validation.
* Bad, because framework-specific context leaks into handlers if not contained.

### Go with `net/http` and `chi`

* Good, because closest to the standard library, `http.Handler` everywhere.
* Bad, because not the stated requirement; binding and validation must be hand-rolled.

### Go with Echo or Fiber

* Good, because similar ergonomics to Gin.
* Bad, because Fiber uses fasthttp and is incompatible with `net/http` middleware; not the requirement.

## More Information

Layout: `backend/cmd/api`, `backend/cmd/bot`, `backend/cmd/worker`, `backend/internal/{auth,tenant,document,log,share,report,telegram}`. See [ADR-0003](0003-monorepo-layout.md).
