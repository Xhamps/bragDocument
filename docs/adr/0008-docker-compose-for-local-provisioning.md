---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# Docker Compose for local provisioning

## Context and Problem Statement

Developers need PostgreSQL, Redis, the PDF renderer, the API, and the frontend running locally with one command. The project requires a `docker-compose.yml` at the root.

## Decision Drivers

* Project requirement: compose file provisions all services
* Infrastructure services must start without building application code
* Fast inner loop for backend and frontend code

## Considered Options

* One `docker-compose.yml` with profiles: infra by default, `app` profile for backend and frontend
* Separate compose files per concern
* Tilt / Skaffold on a local Kubernetes

## Decision Outcome

Chosen option: "One compose file with profiles", because `docker compose up` gives postgres, redis, and gotenberg immediately, and `docker compose --profile app up` adds the API and frontend containers for a full stack run. Day-to-day, developers run the Go API and Vite dev server on the host against the infra containers.

### Consequences

* Good, because new contributors are productive after `docker compose up`.
* Good, because health checks gate the API on postgres and redis readiness.
* Neutral, because Supabase is not in the compose file; local identity uses the Supabase CLI stack or a hosted dev project, configured through `.env` ([ADR-0004](0004-supabase-as-identity-provider.md)).
* Bad, because two ways to run the app (host vs container) can drift; the API reads all configuration from environment variables, documented in `.env.example`.

### Confirmation

CI job runs `docker compose config` and `docker compose up -d postgres redis gotenberg` then the backend integration tests.

## Pros and Cons of the Options

### One compose file with profiles

* Good, because one file, one command.
* Bad, because profiles are less known; documented in the README.

### Separate compose files

* Bad, because more files for the same result.

### Local Kubernetes

* Bad, because far heavier than the project needs.
