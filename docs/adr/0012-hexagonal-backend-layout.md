---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# Hexagonal backend layout with a single Cobra binary

## Context and Problem Statement

[ADR-0001](0001-go-and-gin-for-the-backend.md) fixes Go and Gin and asks that business code stay free of Gin, sketching `cmd/api`, `cmd/bot`, `cmd/worker` and flat `internal/<domain>` packages. Before scaffolding we need to fix the package structure every feature will copy, and how the one binary selects its mode.

## Decision Drivers

* Business rules testable without HTTP, Postgres, or Redis
* The "no Gin in business code" rule enforceable by lint at package level
* Graceful degradation when optional dependencies fail
* One binary, one Docker image, modes selected by subcommand

## Considered Options

* Flat domain packages, handler and service together
* Flat domain packages plus a separate `internal/api` for handlers
* Hexagonal: `domain`, `app`, `ports`, `adapters/{http,postgres,redis,telemetry}`

## Decision Outcome

Chosen option: "Hexagonal", with Cobra subcommands `api`, `bot`, `worker`, `migrate` in `cmd/bragdoc`. `domain` imports stdlib only; `app` and `ports` import `domain`; adapters implement ports and never import each other; `cmd` wires everything. Optional dependencies (Redis, Gotenberg, Telegram) are wrapped in degrading adapters so their failure never fails a request.

### Consequences

* Good, because use cases are tested with fakes of small interfaces.
* Good, because depguard enforces the layering mechanically.
* Good, because degradation lives in one decorator per optional dependency, not in every feature.
* Bad, because each feature touches four packages; a project skill (`.claude/skills/backend-endpoint`) documents the steps so the ceremony is a checklist, not a design exercise.
* Neutral, because this supersedes the layout sketch in the "More Information" section of ADR-0001; the decision for Go and Gin stands.

### Confirmation

`.golangci.yml` depguard rules per package; `go test ./...` runs with no external services; `make test-integration` exercises adapters.

## More Information

Design: [docs/plans/2026-10-08-backend-bootstrap-design.md](../plans/2026-10-08-backend-bootstrap-design.md).
