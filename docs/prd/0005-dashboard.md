---
status: proposed
date: 2026-10-08
owner: Product
stakeholders: Engineering
related-adrs: [ADR-0006]
---

# PRD-0005: Dashboard with log metrics

## 1. Summary

A per-document dashboard summarizes logs: how many, when, of what kind, and with what impact. It helps the author see themes and gaps, which the article names as a benefit beyond promotion.

## 2. Problem

A list of 200 logs does not show that 80 % of them are `project` and none are `mentorship`, or that nothing was logged in Q2. The dashboard makes those patterns visible so the author can act on them before a review.

## 3. Goals

- An author sees the shape of their year in one screen.
- Gaps against the article's recommended sections are obvious.

## 4. Non-goals

- Cross-user or cross-document comparisons in v1.
- Manager roll-ups across reports.
- Custom charts or query builders.

## 5. Users and personas

- **Author** reviewing their own document.
- **Viewer** (manager) getting an overview before reading logs.

## 6. User stories

- As an author, I want to see logs per month so that I notice quiet periods.
- As an author, I want to see the split by tag so that I notice work I forgot to log.
- As an author, I want to see the split by impact and status so that I can prioritize finishing `in_progress` high-impact items.
- As a viewer, I want the same dashboard, read-only.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | The dashboard MUST show: total logs, logs in the selected period, logs by month (bar), logs by tag (top 10), logs by status, logs by impact level. | Must |
| FR-2 | A period selector MUST offer: last 30 days, last quarter, last 12 months, calendar year, custom range. | Must |
| FR-3 | Each chart element MUST link to the logs list pre-filtered to that slice ([PRD-0002](0002-logs.md) FR-6). | Must |
| FR-4 | A "coverage" panel SHOULD list the article's suggested tags with a count, highlighting those at zero. | Should |
| FR-5 | The dashboard MUST respect the viewer's role; viewers see it read-only. | Must |
| FR-6 | Numbers MUST match the logs list for the same filters. | Must |

## 8. Non-functional requirements

- NFR-1: Dashboard loads under 500 ms p95 for 10,000 logs; aggregates MAY be cached for 60 s and MUST be invalidated on writes ([ADR-0006](../adr/0006-redis-as-cache.md)).
- NFR-2: Charts have text alternatives (table toggle) for accessibility.
- NFR-3: Colors follow a single palette and remain legible in light and dark mode.

## 9. UX notes

- Top row: four stat tiles (total, this period, high/critical impact, in progress).
- Middle: logs per month bar chart.
- Bottom: tags, status, impact as horizontal bars; coverage panel on the right.

## 10. Data

No new entities. Read-only aggregates over Log, Tag, and their relations.

## 11. Success metrics

- 60 % of authors open the dashboard at least once a month.
- Logs tagged with a previously-zero coverage tag increase after the dashboard is viewed.

## 12. Open questions

| Question | Owner | Due |
|---|---|---|
| Is a document-level target (e.g. "2 logs/week") worth adding? | Product | after v1 |

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
| 2026-10-08 | Fixed set of six charts | Covers the article's reflection use-case; a query builder is speculative |
