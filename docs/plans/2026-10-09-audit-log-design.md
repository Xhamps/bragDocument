# Audit log: design

Date: 2026-10-09. Status: approved. Implements [PRD-0009](../prd/0009-audit-log.md) under [ADR-0007](../adr/0007-multi-tenancy-strategy.md), [ADR-0011](../adr/0011-rbac-model.md), and [ADR-0012](../adr/0012-hexagonal-backend-layout.md). Extends the PRD-0004 `audit_entries` table.

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Editors and Activity | No. Only owners and tenant admins read entries (PRD open question, default kept). |
| Previous owner after transfer | No access. Read access follows `documents.owner_id` at read time. |
| Write mechanism | Extend the PRD-0004 pattern: every repo write method takes a `domain.AuditEntry` and writes it in its own transaction. Rejected: Postgres triggers (tags and links live in side tables, so field lists would be wrong; logic leaves the domain), entry carried in ctx (a forgotten entry fails silently, breaking FR-1). |
| Source | A domain context value: `cmd` sets `domain.WithSource` per service (api → `web`, bot → `telegram`, worker → `system`); `app` reads `domain.SourceOf`. Keeps `app` free of `telemetry` (ADR-0012). |
| One entry per action | A document update that renames and archives writes one entry; the action is the biggest change (archive/unarchive > rename > edit) and `changed_fields` lists every field. |
| Extra action | `document.edited` for a description-only change; every state change must be audited and "renamed" would mislead. |
| Example logs | Covered by `document.created`; `DeleteExamples` writes one `log.deleted` with target "example logs". |
| Pagination | Keyset on `id` (`before` cursor), 50 per page, max 100. |
| Picker options | `GET /audit/filters` returns distinct actors and documents among visible entries; covers removed users and deleted documents, and works for owners who cannot list members. |
| Old endpoint | `GET /tenant/audit` removed; the Tenant page section and SharePanel history move to the Audit page and the Activity section. |

## Data

Migration `0007_audit_log` alters `audit_entries`:

- `actor_id`, `document_id`, `document_title` become nullable (Telegram link has no document; `system` has no actor).
- New columns: `actor_name text NOT NULL DEFAULT ''`, `source text NOT NULL DEFAULT 'web' CHECK (source IN ('web','telegram','system'))`, `target_type text NOT NULL DEFAULT ''` (`log`, `user`, `invitation`, or empty), `target_id text NOT NULL DEFAULT ''`, `changed_fields text[] NOT NULL DEFAULT '{}'`.
- `target` stays: the target's name at the time (log name or email), satisfying FR-12. `role` stays.
- Existing actions are renamed to namespaced ones (`grant` → `sharing.granted`, `invite` → `sharing.invitation_sent`, `role_change` → `sharing.role_changed`, `revoke` → `sharing.revoked`, `invite_cancel` → `sharing.invitation_cancelled`, `invite_accept` → `sharing.invitation_accepted`, `transfer` → `sharing.ownership_transferred`).
- New indexes: `(tenant_id, actor_id, id DESC)`, `(tenant_id, action, id DESC)`. Existing `(tenant_id, id DESC)` and `(document_id, id DESC)` stay.
- Privileges unchanged: the app role has SELECT and INSERT only (FR-4, NFR-1). RLS unchanged.

## Backend

**Domain.** `AuditEntry` moves to `domain/audit.go` and gains `ActorName`, `Source`, `TargetType`, `TargetID`, `ChangedFields`. Action constants:

- `document.created|renamed|edited|archived|unarchived|deleted`
- `sharing.*` (above; includes ownership transfer)
- `log.created|edited|deleted|status_changed`
- `telegram.linked|unlinked`
- `export.requested`

`AuditFilter{ActorID, DocumentID, Action, From, To, OwnerID, Before, Limit}` with `Validate()`.

**Writes (FR-1).** *Superseded by the outbox: see `2026-10-09-audit-outbox-design.md` and ADR-0015. The original direct-insert design follows.* `DocumentRepo.Create/Update/Delete`, `LogRepo.Create/Update/Delete/DeleteExamples`, `TelegramLinkRepo.Link/Delete`, and `ExportRepo.Create` take `a domain.AuditEntry` and call the shared `audit()` helper inside their transaction. An insert failure rolls back the action. `audit()` copies the document title from `documents` when `DocumentID` is set, and skips it otherwise. Document delete writes its entry before deleting, in the same transaction.

**Building entries (app layer).** The app layer holds the old and new values, so it picks the action and `changed_fields`:

- Log update: only `status` changed → `log.status_changed`; otherwise `log.edited` with field names only (FR-13).
- Document update: biggest change wins, all changed fields listed.
- Actor name and email, and the document title, are copied by the insert itself (`INSERT … SELECT` joining `users` and `documents`); callers pass ids only. An unknown actor or document id inserts nothing (`ErrNotFound`).
- Entries for the bot get `source=telegram` through the service mapping.

**Reads.** `AuditRepo.List(ctx, f)` returns entries newest first plus the next cursor. `OwnerID`, when set, restricts to `document_id IN (SELECT id FROM documents WHERE owner_id = $owner)`. `AuditRepo.Filters(ctx, ownerID)` returns distinct actors and documents. `ponytail: DISTINCT over visible entries; cache or a summary table if pickers get slow on huge tenants.`

App rule: admins read everything; non-admins get `OwnerID = self`, and must own at least one document (else 403). A `document` filter the caller cannot see is 404 (FR-7); one they see but do not own is 403.

## API

- `GET /audit?actor=&document=&action=&from=&to=&before=&limit=` → `{ entries, next_before }`. Bad filters → 422.
- `GET /documents/{id}/audit?before=&limit=` → same shape. Owner or tenant admin. Admins no longer get 404 here; they still cannot read content (PRD-0004 FR-9).
- `GET /audit/filters` → `{ actors: [{id, name, email}], documents: [{id, title}] }`.
- `GET /tenant/audit` removed.

Entry JSON: `{ id, at, source, action, actor: {id|null, name, email}, document: {id, title}|null, target: {type, id, name}, role, changed_fields }`.

## Frontend

- `audit/` folder: `useAudit(filters)` and `useDocumentActivity(id)` (`useInfiniteQuery` on `next_before`), `useAuditFilters()`, and `describe.ts` (one sentence per action, replacing `sharing/audit.ts`).
- Route `/audit` (`routes/Audit.tsx`): table *Time, User, Action, Target, Source*; filters for user, document, action, and from/to (`<input type="date">`) kept in the URL via `useSearchParams`; "Load more"; clicking a user or document applies that filter; relative time with the exact local timestamp in `title` (NFR-4); empty, filtered-empty with "Clear filters", skeleton, and inline error with Retry.
- `Root.tsx` nav link when `me.role === "admin"` or the user owns at least one document (from the documents list already loaded).
- `DocumentLogs.tsx`: an **Activity** section for owners and tenant admins who can open the page, the latest 10 entries, with "View all" → `/audit?document=<id>`.
- Removed: SharePanel history view and the Tenant page "Sharing audit" section.

## Errors

- Audit insert fails → transaction rolls back; the action returns the wrapped error like any failed save.
- Read 403/404 → existing `errorText` states.

## Testing

- App unit tests (fakes): each state-changing use case records exactly one entry with the right action, source, target, and `changed_fields`; status-only edit → `log.status_changed`; bot call → `source=telegram`. Read-permission matrix for admin, owner, editor, viewer, and member without grant.
- Postgres integration: one test per action type (PRD success metric); a failing audit insert rolls back the action; filters and keyset paging; the app role cannot UPDATE or DELETE; RLS hides other tenants; a deleted document's entry keeps its title.
- HTTP handler tests: query parsing and response shape.
- Frontend: Audit page filters sync to the URL; empty and filtered-empty states; Activity shows for owners and admins only.
- NFR-3: no load test in v1; run `EXPLAIN` on the four filter shapes against 1M seeded rows once, by hand.

## Performance check (NFR-3)

Run on 2026-10-09: throwaway `postgres:16-alpine`, laptop Docker, warm cache. One tenant has 1,000,005 entries, with 49 actors, 500 documents and 20 actions spread at random over one year. A second tenant has 1,000. One rare actor has 5 entries with the oldest ids. The query is the `ListAudit` SQL, `LIMIT 51`, run as `bragdoc_app` with `app.tenant_id` set, through `PREPARE`/`EXPLAIN (ANALYZE, BUFFERS) EXECUTE`.

| Shape | Generic plan | Custom plan (index) |
|---|---|---|
| no filter | 458 ms, seq scan + top-N sort | 0.4 ms, `audit_entries_pkey` backward |
| actor | 60 ms, seq scan | 0.1 ms, `audit_entries_actor_idx` |
| document | 54 ms, seq scan | 0.1 ms, `audit_entries_document_idx` |
| action | 83 ms, seq scan | 0.1 ms, `audit_entries_action_idx` |
| document + actor | 49 ms, seq scan | 2.5 ms, BitmapAnd of document + actor indexes |
| action + 1-month range | 66 ms, seq scan | 14 ms, `audit_entries_action_idx` |
| owner scope (10 docs) | 294 ms, seq scan | 1.1 ms, `pkey` backward + hashed subplan |
| rare actor (5 old rows) | 49 ms, seq scan | 0.03 ms, `audit_entries_actor_idx` |
| date only, old month | 100 ms, seq scan | 69 ms, `pkey` backward (filters 855k rows) |
| owner + rare actor (0 rows) | 50 ms, seq scan | 0.3 ms, `audit_entries_actor_idx` |

- **Generic plan.** The `$n IS NULL OR ...` filters cannot use an index in a generic plan, so every shape seq-scans the whole tenant. The unfiltered page sits at the budget (458 ms) and grows linearly with table size.
- **Auto mode.** Under the default `plan_cache_mode = auto`, which is what pgx's cached statements get, Postgres kept custom plans after 10 executions (`custom_plans = 10`, `generic_plans = 0`), because the generic plan's estimated cost is far higher.
- **The pin.** `AuditRepo.List` now runs `SET LOCAL plan_cache_mode = force_custom_plan` in its transaction, so the cost heuristic can no longer flip it to the generic plan.
- **Verdict.** NFR-3 holds: the worst custom shape is 69 ms, a date-only filter on an old month, which walks the id index back to it. If that range ever gets slow, the fix is an index on `(tenant_id, at)`.
