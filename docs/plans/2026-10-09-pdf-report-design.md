# PDF report: design

Date: 2026-10-09. Status: approved. Implements [PRD-0006](../prd/0006-pdf-report.md) under [ADR-0010](../adr/0010-pdf-generation-with-gotenberg.md), [ADR-0011](../adr/0011-rbac-model.md), and [ADR-0012](../adr/0012-hexagonal-backend-layout.md).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Job queue | `export_jobs` in PostgreSQL, claimed with `FOR UPDATE SKIP LOCKED`. Progress (0–100) in Redis, optional through `Degrading`. |
| Summary table | Page 2, after the cover and goals (PRD open question). |
| Branding | Tenant name only; no logo until tenants can upload one. |
| Multi-tag logs | A log appears once, in its first matching section in template order. Section counts add up to the summary. |
| Tag→section mapping | Saved per document with the goals; the dialog prefills from it, falling back to the default mapping. |
| Encryption at rest | App-level AES-GCM with `EXPORT_KEY` (32 bytes) for any storage backend (NFR-2). |
| Accessibility | Gotenberg `pdfua=true` (NFR-3). |

## 1. Backend

Migration `0006_exports`, both tables with RLS `tenant_isolation`:

- `export_jobs`: id, tenant_id, document_id, requested_by, `params jsonb` (filters, period, goals, mapping as sent), status (`queued|running|done|failed`), error, file_key, created_at, finished_at, expires_at.
- `document_report_settings`: document_id PK, goals_this_year, goals_next_year, `section_map jsonb`. Upserted on each export.

Domain (`domain/report.go`):

- `Sections`: Projects, Collaboration & mentorship, Design & documentation, Company building, What you learned, Outside of work, Other.
- `DefaultSectionMap`: project → Projects; collaboration, mentorship → Collaboration & mentorship; design, documentation → Design & documentation; company-building → Company building; learning → What you learned; outside-of-work → Outside of work.
- `BuildReport(logs, map)`: first matching section in template order, unmatched → Other; impact × status counts.
- `ExportParams.Validate()`: reuses `LogFilter.Validate`, caps goal text, rejects unknown sections.

Use cases (`app/exports.go`), all `access(PermRead)` (FR-6):

- `Create`: counts matches through `LogRepo.List` (PerPage 1, read `Total`); over 2,000 is a validation error (FR-7); upserts settings; inserts a `queued` job.
- `Get`, `List`: one job or the caller's recent jobs on the document.
- `Open(jobID)`: access, `done`, not expired → decrypting reader.
- `Settings`, `SaveSettings`: dialog prefill.

Worker (replaces the `daemonCmd("worker")` placeholder): a loop claims one job under `WithProvisioning`, then works inside the job's tenant: re-check the requester's `PermRead` → page logs (examples excluded) → `BuildReport` → render `html/template` → Gotenberg → AES-GCM → `FileStore` → `done`, `expires_at = now + 24h`. Progress goes to Redis `export:{id}:progress` per step. Failure marks `failed` with a short error. Each tick deletes expired jobs and their files.

Ports: `ExportRepo`, `PDFRenderer` (Gotenberg over `net/http`), `FileStore` (local directory; S3 later). Encryption is a small helper around the store.

HTTP: `POST/GET /documents/:id/exports`, `GET /exports/:id` (status + progress), `GET /exports/:id/file`, `GET/PUT /documents/:id/report-settings`. Compose mounts the `exports` volume on the api too and adds `EXPORT_KEY`.

## 2. Report template

`report.html` (`embed`, print CSS):

- Page 1 cover: document title, author, tenant name, period, generation date.
- Page 2: goals this year / next year (omitted when empty), summary table impact × status with totals.
- Sections in template order with `<h2>`; empty sections are omitted.
- Each log: `<h3>` name; date, impact, status; description as escaped text with `white-space: pre-wrap`; tags; `<a href>` links (FR-3). `break-inside: avoid`.
- Footer page numbers through Gotenberg's footer HTML. `POST /forms/chromium/convert/html` with `pdfua=true`.

## 3. Frontend

- "Export PDF" in the document header (`DocumentTabs`), so on both logs and dashboard; prefilled from the URL's filters and period (FR-1).
- `ExportDialog`: period and filters (reusing `LogFilters`, `periods.ts`), two goal textareas, mapping table (one row per used tag, native `<select>` of sections), Generate, recent exports with Download while valid.
- A 422 for too many logs shows inline: "Narrow your filters".
- `useExport(jobId)` polls `GET /exports/:id` every second until done/failed; progress bar.
- No toast library: a `role="status"` notice in the document header, "Report ready — Download", survives closing the dialog.
- Download: authenticated `fetch` → blob → `a.download`.

## 4. Testing

- Domain: section assignment (first match, unmatched, untagged, custom map), summary counts, params validation.
- App: exports in the access matrix (read); over-limit rejected; viewer can export and download; outsider cannot open another's job; expired job not downloadable.
- Crypto: round trip; tampered ciphertext fails.
- Template: section order, omitted empties, link hrefs, HTML in descriptions escaped.
- Integration: claim with two workers takes each job once; full job against Gotenberg produces a multi-page PDF whose text holds section headings (skipped without Gotenberg); expired cleanup removes row and file; 500 logs timed against NFR-1.
- HTTP: handlers and param parsing.
- Vitest: dialog prefill from URL, mapping edit, over-limit message, polling to done, download button.

## 5. Delivery

Branch `feat/pdf-report`, conventional commits per layer. PRD-0006 → `accepted` with the decisions above. ADR-0010 amended (PostgreSQL queue, Redis progress, AES-GCM, `pdfua`). `openapi.yaml` and `docs/README.md` updated. Pull request against `main` linking PRD-0006 and ADR-0010.
