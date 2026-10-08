---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# Monorepo layout

## Context and Problem Statement

The project requires backend and frontend in one repository. How is the repository organized so that both can be built, tested, and provisioned together without coupling their toolchains?

## Decision Drivers

* Project requirement: monorepo
* One `docker-compose.yml` at the root builds everything
* Independent toolchains: Go modules and npm
* Shared API contract between backend and frontend

## Considered Options

* Two top-level folders (`backend/`, `frontend/`) with root-level compose and docs
* Workspace tooling (Nx, Turborepo, Bazel)
* Go module at the root with the frontend embedded via `embed`

## Decision Outcome

Chosen option: "Two top-level folders with root-level compose and docs", because it is the smallest layout that satisfies the drivers. Each folder owns its toolchain; the root owns orchestration and documentation.

```
.
├── docker-compose.yml
├── docs/
├── backend/        # go.mod here
│   ├── cmd/
│   ├── internal/
│   ├── migrations/
│   └── Dockerfile
└── frontend/       # package.json here
    ├── src/
    └── Dockerfile
```

### Consequences

* Good, because a change spanning API and UI is one PR.
* Good, because CI runs backend and frontend jobs filtered by path.
* Neutral, because the API contract is shared as an OpenAPI file at `backend/api/openapi.yaml`; the frontend generates its client from it.
* Bad, because repository size and CI time grow together; path filters mitigate.

### Confirmation

`docker compose build` succeeds from the root. CI defines separate jobs for `backend/**` and `frontend/**`.

## Pros and Cons of the Options

### Two top-level folders

* Good, because no extra tooling.
* Bad, because no task graph; acceptable at two packages.

### Workspace tooling

* Good, because caching and task orchestration.
* Bad, because adds a tool for two packages with different ecosystems.

### Go `embed` of the frontend

* Good, because one deployable binary.
* Bad, because couples release cycles and makes frontend hot reload awkward in Docker.
