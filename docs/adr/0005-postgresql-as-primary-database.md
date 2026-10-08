---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# PostgreSQL as primary database

## Context and Problem Statement

Tenants, users, documents, logs, tags, links, grants, invitations, audit entries, and export jobs need durable relational storage with strong filtering. The project requires PostgreSQL provisioned locally with Docker.

## Decision Drivers

* Project requirement: PostgreSQL via Docker
* Filtering by every log field, including tags (many-to-many) and text search
* Row-level security for tenant isolation ([ADR-0007](0007-multi-tenancy-strategy.md))
* Simple migrations and local reset

## Considered Options

* PostgreSQL 16 in Docker, migrations with `golang-migrate`, access via `pgx`
* PostgreSQL via Supabase's bundled database
* SQLite for local, PostgreSQL in production

## Decision Outcome

Chosen option: "PostgreSQL 16 in Docker with `golang-migrate` and `pgx`", because it is the requirement, keeps application data separate from the IdP, and gives us full-text search (`tsvector`), array and JSONB columns, and RLS.

### Consequences

* Good, because `docker compose up postgres` gives a clean database; `make migrate` applies `backend/migrations/*.sql`.
* Good, because text filter uses a generated `tsvector` column with a GIN index; tag filter uses a join table with a composite index.
* Neutral, because we use SQL directly with `pgx` and `sqlc`-generated code rather than an ORM; schema is the source of truth.
* Bad, because developers need Docker running; documented in the root README.

### Confirmation

Migrations run on container start in the `app` profile; a smoke test inserts and filters 1,000 logs and asserts p95 < 300 ms on the list query ([PRD-0002](../prd/0002-logs.md) NFR-1).

## Pros and Cons of the Options

### PostgreSQL in Docker

* Good, because identical engine locally and in production.
* Bad, because one more container; acceptable.

### Supabase's bundled database

* Good, because one less service.
* Bad, because couples our schema and RLS to the IdP project and its hosted limits; the requirement separates them.

### SQLite locally

* Good, because zero infrastructure.
* Bad, because no RLS, different SQL dialect, bugs found only in production.
