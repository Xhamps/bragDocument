# Logs: design

Date: 2026-10-08. Status: approved. Implements [PRD-0002](../prd/0002-logs.md) and a new PRD-0007 (impact extraction) under [ADR-0005](../adr/0005-postgresql-as-primary-database.md), [ADR-0007](../adr/0007-multi-tenancy-strategy.md), [ADR-0012](../adr/0012-hexagonal-backend-layout.md), and a new ADR-0013 (OpenAI for impact extraction).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Scope | Backend and frontend in one PR, including FR-9 (tag autocomplete), FR-10 (suggested tags), example logs, and the document card counters deferred from PRD-0001. |
| Permissions | Owner only until PRD-0004, reusing the documents rule: other tenants get 404 (RLS), a same-tenant non-owner gets 403 (PRD-0004 FR-7). Log writes on an archived document get 409. |
| Cache (NFR-2) | Deferred. Indexed Postgres queries are expected to meet NFR-1 at 10k logs; add the Redis cache when a measurement says otherwise. |
| Impact | Closes PRD-0002's open question: a fixed level (required, filterable) plus a statement that an LLM extracts from the description. No separate input. When no impact is found, the user is warned. |
| Extraction timing | Synchronous on save, short timeout. A save never fails because of the LLM. |
| LLM provider | OpenAI via the official `openai-go` SDK. Recorded in ADR-0013. Requirements in PRD-0007. |
| List query | One sqlc query with optional parameters, `CASE`-based whitelisted sort, `LIMIT/OFFSET` with `count(*) OVER ()`, `pg_trgm` for text search. Rejected: hand-built SQL with keyset paging (more code, only pays off far past 10k rows), full-text `tsvector` (stemming hurts substring search). |
| Example logs | Inserted by `DocumentCreate` in the same transaction with `is_example = true`; removable in one call. Existing documents get none. |

## 1. Schema

Migration `0003_logs`.

```
logs        id uuid pk, tenant_id fk tenants, document_id fk documents ON DELETE CASCADE,
            name text (1–120), description text default '',
            impact text check in ('low','medium','high','critical'),
            impact_statement text NULL,
            status text check in ('idea','in_progress','done','dropped') default 'done',
            is_example bool default false,
            created_at timestamptz (editable), created_by fk users,
            updated_at timestamptz, updated_by fk users
tags        tenant_id fk tenants, name text (lowercase); pk (tenant_id, name)
log_tags    tenant_id, log_id fk logs ON DELETE CASCADE, tag_name;
            pk (log_id, tag_name); fk (tenant_id, tag_name) → tags
log_links   id uuid pk, tenant_id, log_id fk logs ON DELETE CASCADE,
            url text, label text default '', host text
```

`impact_statement`: NULL means not checked (LLM disabled, timed out, or failed); `''` means checked and none found; text is the extracted statement.

RLS `tenant_id = app_tenant_id()` on all four tables, as on `documents`. Grants come from the existing default privilege. Indexes: `logs(document_id, created_at DESC)`, trigram GIN on `logs.name` and `logs.description`, `log_tags(tag_name)`, `log_links(host)`. The `tags` table is append-only; unused tags remain for autocomplete.

## 2. Impact extraction (PRD-0007, ADR-0013)

- Port `ports.ImpactExtractor { Extract(ctx, name, description) (string, error) }`; `""` means none found.
- Adapter `adapters/llm/openai.go`: structured JSON-schema output `{found bool, statement string}`. The prompt asks to quote or tightly paraphrase the outcome stated in the text and never invent one. Statement capped at 280 chars.
- Config: `OPENAI_API_KEY` (optional), `OPENAI_MODEL` (default: a current small model, verified at build time), `LLM_TIMEOUT` (default 5s).
- Degradation: no key → `cmd` wires a disabled extractor returning `domain.ErrUnavailable`. `LogCreate` and `LogUpdate` call the extractor only when name or description changed. On error: log, increment `impact_extraction_failures_total{reason}`, store NULL, save anyway. An edit that changes name or description re-extracts, which retries NULLs.
- Example logs skip extraction; their statements are hard-coded.
- Privacy: descriptions are sent to OpenAI. PRD-0007 states it.

## 3. API

Owner-only: other tenant → 404, same-tenant non-owner → 403, write on an archived document → 409.

| Method | Path | Use case | Notes |
|---|---|---|---|
| GET | `/documents/:id/logs` | `LogList` | `q`, `tag` (repeat, any-of), `status` (repeat), `impact` (repeat), `from`/`to` (dates on `created_at`), `domain`, `sort` ∈ `created_at,name,impact,status` with `-` prefix for desc (default `-created_at`), `page`, `per_page` (default 50, max 100). Returns `{items, total}`. |
| POST | `/documents/:id/logs` | `LogCreate` | `name, description, impact, status, tags[], links[{url,label}], created_at?`. Extracts impact. 201. |
| PATCH | `/documents/:id/logs/:logId` | `LogUpdate` | All fields optional; `tags`/`links` replace the set when present. Re-extracts only if name or description changed. |
| DELETE | `/documents/:id/logs/:logId` | `LogDelete` | 204 |
| DELETE | `/documents/:id/example-logs` | `LogDeleteExamples` | 204 |
| GET | `/tags` | `TagList` | Tenant tag names, sorted. |
| GET | `/documents` | existing | Adds `log_count` and `last_log_at` (examples excluded) via a `LEFT JOIN` aggregate. |

Validation (domain, 422): name 1–120 after trim; description ≤ 20,000; enums; tags trimmed, lowercased, 1–50 chars, deduplicated, ≤ 20; links `http`/`https` only, ≤ 20, label ≤ 100, `host` lowercased with `www.` stripped. `from`/`to` are UTC calendar days (`to` inclusive). Back-dated logs are stored at noon local time, so they land on the chosen day; a log created late in the evening west of UTC may fall on the next UTC day. Client-local ranges can come later via RFC 3339 instants.

Sort order: impact `critical > high > medium > low`, status `idea → in_progress → done → dropped`, tiebreak on `id`.

`backend/api/openapi.yaml` is updated.

## 4. Frontend

- Route `/documents/:id` → `routes/DocumentLogs.tsx`. Document cards link there and show "N logs · last on {date}".
- Header: document title from the cached `useDocuments` list; **New log** button (FR-3). Archived documents are read-only.
- Filter bar: debounced search, tags input, Status and Impact toggle buttons (`aria-pressed`), native date inputs, domain input, sort select. Active filters as removable chips; result count.
- All filter, sort, and page state lives in `useSearchParams` (FR-6). Filter changes reset `page`.
- Rows: name with the impact statement below, or an amber "No impact stated" when `''`; impact and status badges; tags; date; row menu Edit and Delete. Expanding a row renders the description as Markdown and the links with `target="_blank" rel="noopener noreferrer"`. Example logs carry an "Example" badge and a "Remove examples" banner.
- Pagination Prev/Next with "page X of Y". Empty states for "no logs yet" and "no matches".
- `LogFormDialog` (create and edit), ordered per PRD §9: name, impact; description with Write/Preview; tags with `<datalist>` from `/tags` plus the 8 suggested tags; link rows; status, date. Cmd/Ctrl+Enter submits. Labels and `aria-describedby` on errors.
- On save with `impact_statement === ''` the dialog stays open with a warning, "Add impact" (focuses description) and "Close".
- `logs/useLogs.ts`: react-query hooks keyed by `[docId, searchParams]`; mutations invalidate logs, documents, and tags.
- New dependency: `react-markdown` (no raw HTML by default).

## 5. Testing

- Backend unit: domain validation tables; one test per use case with fakes, including extraction found / none / error → NULL / unchanged fields → no call, and the owner check (404/403) and archived 409.
- Backend HTTP: each handler with a fake use case; `LogList` query parsing (repeated params, bad sort → 422, `per_page` clamp).
- OpenAI adapter: `httptest` server with canned found, not found, 500, and slow responses. No live calls in CI.
- Integration (`integration` tag): RLS isolation on the four tables; each filter and sort; cascade on document delete; examples on create; a 10k-log timing reported in the PR, not asserted.
- Frontend (Vitest, `fetch` mocked): filters update URL and request; chip removal; Cmd+Enter submit; dialog stays open on `''`; empty state and remove examples.

## 6. Delivery

Branch `feat/logs`. Conventional commits per layer: docs (this design, PRD-0007, ADR-0013), migration, queries, domain, ports and app, openai adapter, http, frontend. PRD-0002 → `accepted` with the impact decision and deferred NFR-2 logged; PRD-0007 and ADR-0013 → `accepted`. `docs/README.md` index, `.env.example`, `docker-compose.yml`, and README updated. Pull request against `main` linking PRD-0002, PRD-0007, ADR-0013, noting owner-only scope until PRD-0004 and the deferred cache.
