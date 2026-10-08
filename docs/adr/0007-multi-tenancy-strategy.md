---
status: proposed
date: 2026-10-08
decision-makers: Engineering, Security
---

# Multi-tenancy: shared schema with row-level security

## Context and Problem Statement

The product is multi-tenant. Data of one tenant must never be visible to another, even through an application bug. How is tenant isolation implemented in PostgreSQL and the API?

## Decision Drivers

* Isolation enforced in the database, not only in code ([PRD-0001](../prd/0001-tenants-users-and-brag-documents.md) NFR-1)
* One schema and one migration path
* Simple local setup
* Reasonable path to stronger isolation later if a customer requires it

## Considered Options

* Shared database, shared schema, `tenant_id` on every table, PostgreSQL row-level security (RLS)
* Schema per tenant
* Database per tenant

## Decision Outcome

Chosen option: "Shared schema with `tenant_id` and RLS", because it meets the drivers with the least operational cost. Every tenant-scoped table has a `tenant_id` column with a foreign key and an index. RLS policies compare `tenant_id` with `current_setting('app.tenant_id')`. The API opens each request's transaction with `set_config('app.tenant_id', $1, true)` (the parameterised equivalent of `SET LOCAL`), taken from the authenticated user, before running any query. The application role is not the table owner and cannot bypass RLS.

### Consequences

* Good, because a query that forgets a `WHERE tenant_id` still returns nothing from other tenants.
* Good, because one database to back up and migrate.
* Neutral, because every query runs inside a transaction that sets the tenant; a helper in `backend/internal/db` is the only way to get a connection.
* Bad, because noisy-neighbor effects are possible; mitigated by indexes and the cache. Per-tenant databases remain a future option, recorded in a new ADR if needed.
* Bad, because background jobs (bot, export worker) must set the tenant explicitly from the job payload.

### Confirmation

Migration test: with RLS enabled, a connection set to tenant A selecting from each tenant-scoped table returns zero rows of tenant B. A CI check fails if a new table lacks a `tenant_id` column and policy.

## Pros and Cons of the Options

### Shared schema with RLS

* Good, because minimal ops, DB-enforced isolation.
* Bad, because shared resources.

### Schema per tenant

* Good, because clearer separation.
* Bad, because migrations multiply; connection routing per tenant.

### Database per tenant

* Good, because strongest isolation.
* Bad, because heaviest ops; not justified for v1.
