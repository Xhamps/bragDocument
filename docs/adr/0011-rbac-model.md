---
status: proposed
date: 2026-10-08
decision-makers: Engineering, Security
---

# RBAC model for documents

## Context and Problem Statement

Documents are shared with roles ([PRD-0004](../prd/0004-sharing-and-rbac.md)). Where are roles stored, how are permissions evaluated, and how do we keep the UI, API, and database consistent?

## Decision Drivers

* Enforcement in the API for every operation; DB isolation as the backstop
* Fixed roles for v1, extendable later
* Fast per-request checks
* Auditability

## Considered Options

* `document_grants(document_id, user_id, role)` table, a static role→permission matrix in Go, a Gin middleware that loads the caller's role per document
* Policy engine (Casbin, OPA)
* Supabase RLS policies as the sole authorization layer

## Decision Outcome

Chosen option: "Grants table + static matrix + middleware", because three fixed roles do not justify a policy engine, and keeping the matrix in code makes it testable and reviewable. The middleware resolves `(user, document)` → role (cached in Redis ≤ 30 s), attaches it to the request context, and handlers call `require(ctx, PermEditLogs)`. Ownership is a grant with role `owner`, constrained by a partial unique index to one per document. Every grant change writes an audit row.

### Consequences

* Good, because the permission matrix is one Go map, mirrored in the PRD table.
* Good, because tenant RLS ([ADR-0007](0007-multi-tenancy-strategy.md)) still protects data if a check is missed.
* Neutral, because the frontend fetches the caller's role with the document and hides controls; it never decides access.
* Bad, because custom roles would require a schema change; acceptable for v1.

### Confirmation

Table-driven tests over the matrix for every (role, action). An HTTP test suite asserts 403 for each forbidden combination and 404 for documents outside the tenant.

## Pros and Cons of the Options

### Grants table + static matrix

* Good, because simple, fast, testable.
* Bad, because no dynamic policies.

### Policy engine

* Good, because flexible.
* Bad, because a DSL and a dependency for three roles.

### Supabase RLS only

* Bad, because our data is not in Supabase ([ADR-0005](0005-postgresql-as-primary-database.md)), and DB-only policies hide the matrix from code review.
