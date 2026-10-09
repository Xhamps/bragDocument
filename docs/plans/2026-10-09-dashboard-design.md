# Dashboard: design

Date: 2026-10-09. Status: approved. Implements [PRD-0005](../prd/0005-dashboard.md) under [ADR-0006](../adr/0006-redis-as-cache.md), [ADR-0011](../adr/0011-rbac-model.md), and [ADR-0012](../adr/0012-hexagonal-backend-layout.md).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Example logs | Excluded from every number. The logs list gains `examples=false` so chart links show the same count (FR-6). |
| Period | Everything except "Total" follows the period. Default: last 12 months. |
| Aggregation | One endpoint, one tenant transaction, a few `GROUP BY` queries. |
| Cache | Redis cache-aside, 60 s TTL, per-document version key bumped on every log write (ADR-0006). Optional at runtime through `Degrading`. |
| Charts | Recharts 3 with an always-available table view of real links. |
| Access | `access(PermRead)`; viewers see the same page, which has no write controls (FR-5). |

## 1. Backend

Domain:

```
Bucket    {Key string; Count int}
Dashboard {Total, InPeriod, HighImpact, InProgress int
           Months   []Bucket  // "2026-03", every month of the period, zeros included
           Tags     []Bucket  // top 10 in period, count desc then name
           Statuses []Bucket  // all four, fixed order
           Impacts  []Bucket  // all four, fixed order
           Coverage []Bucket} // domain.SuggestedTags, zeros included
```

`Total` is all-time; the rest is `created_at >= from AND < to`. All exclude `is_example`. `HighImpact` = high + critical; `InProgress` = status in_progress. Months are UTC. `domain.SuggestedTags` moves from the frontend constant (kept there, guarded by a test). A period longer than 5 years is a validation error; a missing period defaults to the last 12 months.

Use case `Logs.Dashboard(ctx, docID, userID, from, to)`: `access(PermRead)`, then cache, then `LogRepo.Dashboard`.

Repository: one `WithTenant` transaction, five sqlc queries (totals with `FILTER`, months via `date_trunc('month', created_at AT TIME ZONE 'UTC')`, tags via `log_tags`, statuses, impacts). Coverage counts come from a tag query restricted to the suggested names. Gap-filling of months and enum zeros happens in `domain`.

HTTP: `GET /documents/:id/dashboard?from=YYYY-MM-DD&to=YYYY-MM-DD`, parsed with the logs list's date helper (UTC, `to` inclusive of that day). `openapi.yaml` updated.

Logs list: `examples=false` adds `NOT is_example` to `ListLogs`.

## 2. Cache (NFR-1, ADR-0006)

- Version key `doc:{id}:v`: every `Logs.Create`, `Update`, `Delete`, `DeleteExamples` sets it to a fresh random value, TTL 24 h. Missing reads as `0`. A random value with `Set` invalidates like `INCR` without widening `ports.Cache`.
- Entry `dash:{docID}:v{ver}:{from}:{to}`: JSON of `Dashboard`, TTL 60 s. Old versions expire on their own.
- `app.Logs` takes a `ports.Cache`; `cmd` passes the existing `Degrading` cache to the api and the bot, so bot writes invalidate too.
- `access()` always runs before the cache, so revocation stays immediate. Redis down → misses → Postgres. A corrupt entry is a miss. A failed bump while Redis flaps → at most 60 s stale, the bound ADR-0006 accepts.

## 3. Frontend

- Route `/documents/:id/dashboard`; "Logs | Dashboard" tabs (links) in the document header.
- Period: native `<select>` (last 30 days, last quarter = previous full calendar quarter, last 12 months, this calendar year, custom with two date inputs). State lives in the URL (`from`, `to`).
- Layout per PRD §9: four tiles (total, in period, high/critical, in progress); monthly bar chart; tags, status, impact as horizontal bars; coverage panel listing the eight suggested tags, zeros highlighted.
- `BarList` wraps a Recharts `BarChart` (`accessibilityLayer`, fills `var(--chart-N)`). Clicking a bar navigates to `/documents/:id?…&from&to&examples=false`; a month bar narrows the dates to that month. "Show as table" swaps in a `<table>` whose rows are `<Link>`s (NFR-2). Tiles link the same way.
- Dependency: `recharts@^3` in `packages/app`.

## 4. Testing

- Domain: month gap-filling across years, period defaults and 5-year cap, enum zeros, suggested tags.
- App: dashboard in the access matrix (read); cache hit, miss, write invalidates, Redis error still answers, corrupt entry is a miss.
- Integration: seeded logs across months/tags/statuses/impacts plus an example log; every bucket asserted; a slice equals `ListLogs` total with the same filter and `examples=false` (FR-6); write a log then read the dashboard and see it (ADR-0006 confirmation); 10k logs timed and logged against NFR-1.
- HTTP: handler and date parsing.
- Vitest: tiles, table links carry the right query, table toggle, period select updates the URL, chart mode mounts.

## 5. Delivery

Branch `feat/dashboard`, conventional commits per layer. PRD-0005 → `accepted` with the decisions above. `openapi.yaml` and `docs/README.md` updated. Pull request against `main` linking PRD-0005 and ADR-0006.
