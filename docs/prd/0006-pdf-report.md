---
status: accepted
date: 2026-10-08
owner: Product
stakeholders: Engineering
related-adrs: [ADR-0010]
---

# PRD-0006: PDF report for sharing with the team

## 1. Summary

Any member of a document can generate a PDF of the document, optionally filtered, organized in the sections the article recommends. It is the artifact handed to a manager, a peer reviewer, or a promotion committee.

## 2. Problem

Review processes still run on documents attached to forms and emails. The article suggests a 5 to 10 page yearly write-up. Producing it by hand from a list of logs is tedious and error-prone; a generated PDF makes it a click.

## 3. Goals

- A review-ready document in one click.
- The output reads like the article's template, not like a database dump.

## 4. Non-goals

- Editing the PDF in-app, or Word/Google Docs export in v1.
- Scheduled or emailed reports.
- Branding or theme customization beyond tenant name and logo.

## 5. Users and personas

- **Author** preparing for a review or promotion.
- **Viewer** (manager) who wants an offline copy.

## 6. User stories

- As an author, I want a PDF of this year's logs grouped by the article's sections so that I can attach it to my review.
- As an author, I want to include only `done` logs with impact `high` or above so that the report is focused.
- As an author, I want a short intro (goals for this year, goals for next year) at the top.
- As a viewer, I want to download the same report.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | "Export PDF" MUST be available on the document page and on the logs list, carrying the active filters. | Must |
| FR-2 | The report MUST contain: cover (document title, author, tenant, period, generation date), optional "Goals for this year" and "Goals for next year" text, sections per the article's template (Projects, Collaboration & mentorship, Design & documentation, Company building, What you learned, Outside of work, Other) populated by tag, and a summary table (counts by impact and status). | Must |
| FR-3 | Each log entry MUST render name, date, impact, status, description, tags, and reference links as clickable URLs. | Must |
| FR-4 | The mapping from tags to sections MUST be shown in the export dialog and editable per export; untagged or unmapped logs go to "Other". | Must |
| FR-5 | Generation MUST be asynchronous with a progress state; the file MUST be downloadable from the UI when ready and for 24 hours after. | Must |
| FR-6 | The report MUST respect the caller's role: only logs the caller can read are included. | Must |
| FR-7 | Reports SHOULD be limited to 2,000 logs; above that the UI asks the user to narrow filters. | Should |

## 8. Non-functional requirements

- NFR-1: A 500-log report generates in under 15 s.
- NFR-2: Generated files are stored per tenant, encrypted at rest, and deleted after 24 hours.
- NFR-3: Tagged PDF output (accessible headings and reading order).

## 9. UX notes

- Export dialog: period, filters (prefilled), goals text areas, tag-to-section mapping table, "Generate" button. Toast with download link when ready; history of recent exports in the dialog.

## 10. Data

- **Export job**: document, requested by, filters, status (`queued`, `running`, `done`, `failed`), file key, created at, expires at.
- **Document goals**: document, goals this year, goals next year (persisted so they do not need re-typing).

## 11. Success metrics

- 40 % of shared documents have at least one export per review cycle.
- Fewer than 1 % of export jobs fail.

## 12. Open questions

| Question | Owner | Due |
|---|---|---|
| Does the summary table belong on page 2 or at the end? | Design | answered 2026-10-09: page 2 |

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
| 2026-10-08 | Sections follow the article's template | That is the format managers already recognize |
| 2026-10-09 | Summary table on page 2, after the cover and goals | The overview comes before the detail, as in promotion packets |
| 2026-10-09 | A log appears once, in its first matching section in template order | Section counts add up to the summary |
| 2026-10-09 | The tag→section mapping and the goals are saved per document | The next export starts where the last one ended |
| 2026-10-09 | Cover shows the tenant name; no logo in v1 | Tenants have no logo yet |
| 2026-10-09 | Example logs are never in the report | They are starter content, not work |
