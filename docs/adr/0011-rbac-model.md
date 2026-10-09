---
status: accepted
date: 2026-10-09
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

* `document_grants(document_id, user_id, role)` table, a static role→permission matrix in Go, and a role resolver in the application layer (`app.access`)
* Policy engine (Casbin, OPA)
* Supabase RLS policies as the sole authorization layer

## Decision Outcome

Chosen option: "Grants table + static matrix + app-layer resolver", because three fixed roles do not justify a policy engine, and keeping the matrix in code makes it testable and reviewable. `app.access(ctx, docs, docID, userID, perm)` loads the document and the caller's role in one query and checks `domain.Can(role, perm)`; handlers and the Telegram bot both go through the use cases, so there is no HTTP middleware (the bot calls `app` in-process and would bypass one). There is no Redis cache: FR-5 requires immediate revocation, and the single indexed query meets NFR-2. The owner stays in `documents.owner_id` (decided in the tenants design); `document_grants` holds `editor` and `viewer` only, so "exactly one owner" is a column, not a partial unique index. No role → 404 (FR-7); a role without the permission → 403 whose message names the role (`domain.AccessError`). Every grant change writes an `audit_entries` row in the same transaction; the app role has no UPDATE or DELETE on that table.

### Consequences

* Good, because the permission matrix is one Go map, mirrored in the PRD table.
* Good, because tenant RLS ([ADR-0007](0007-multi-tenancy-strategy.md)) still protects data if a check is missed.
* Neutral, because the frontend fetches the caller's role with the document and hides controls; it never decides access.
* Bad, because custom roles would require a schema change; acceptable for v1.

### Confirmation

`internal/domain/permission_test.go` checks every (role, permission) against the PRD table. `internal/app/access_matrix_test.go` drives every use case as owner, editor, viewer, and an ungranted member, asserting success, 403 (`AccessError`), or 404.

## Pros and Cons of the Options

### Grants table + static matrix + app-layer resolver

* Good, because simple, fast, testable.
* Bad, because no dynamic policies.

### Policy engine

* Good, because flexible.
* Bad, because a DSL and a dependency for three roles.

### Supabase RLS only

* Bad, because our data is not in Supabase ([ADR-0005](0005-postgresql-as-primary-database.md)), and DB-only policies hide the matrix from code review.
