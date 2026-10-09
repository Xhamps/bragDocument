---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# Redis as cache

> **Amended 2026-10-09:** Redis also carries event streams for the transactional outbox ([ADR-0015](0015-transactional-outbox-with-redis-streams.md)). Streams are not a cache, but the degradation rule still holds: the outbox in PostgreSQL is the source of truth, so Redis being down only delays delivery (audit entries appear once it is back).

## Context and Problem Statement

The logs list, dashboard aggregates, and per-request permission lookups are read far more often than written. The project requires Redis provisioned locally with Docker. What do we cache, and how is it kept correct?

## Decision Drivers

* Project requirement: Redis via Docker
* Dashboard and list p95 targets ([PRD-0002](../prd/0002-logs.md), [PRD-0005](../prd/0005-dashboard.md))
* Correctness: a write must never serve stale data for long
* Reuse Redis for short-lived state (Telegram link codes, export job status) to avoid a second store

## Considered Options

* Redis 7 in Docker, cache-aside with per-document version keys
* In-process cache (e.g. `ristretto`) per API instance
* No cache; rely on PostgreSQL indexes

## Decision Outcome

Chosen option: "Redis 7 with cache-aside and per-document version keys", because it is the requirement, it is shared across API instances, and version keys make invalidation one `INCR` on any write to a document: cache keys embed `doc:{id}:v{n}`, so old entries expire naturally (TTL 60 s) without tracking them.

### Consequences

* Good, because invalidation logic is one line in the write path.
* Good, because Telegram link codes (TTL 10 min) and export job progress live in Redis too.
* Neutral, because the cache is optional at runtime: if Redis is down the API logs a warning and reads from PostgreSQL.
* Bad, because cached authorization decisions must be short-lived (≤ 30 s) so revocation is prompt ([PRD-0004](../prd/0004-sharing-and-rbac.md) FR-5).

### Confirmation

Test: write a log, read the list, assert the new log appears immediately. Chaos test: stop Redis, API still serves requests.

## Pros and Cons of the Options

### Redis with version keys

* Good, because shared, simple invalidation.
* Bad, because one more container.

### In-process cache

* Good, because no network hop.
* Bad, because inconsistent across instances; no shared short-lived state.

### No cache

* Good, because simplest.
* Bad, because dashboard aggregates over 10k logs on every view miss the latency target; also not the requirement.
