---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# PDF generation with Gotenberg

## Context and Problem Statement

The report export ([PRD-0006](../prd/0006-pdf-report.md)) must produce a readable, sectioned, hyperlinked, accessible PDF of up to 2,000 logs. How is the PDF produced?

## Decision Drivers

* Layout fidelity: headings, tables, links, page breaks
* Reuse the same template logic as the web app where possible
* Runs in Docker locally ([ADR-0008](0008-docker-compose-for-local-provisioning.md))
* Asynchronous generation with progress

## Considered Options

* Render HTML with Go `html/template`, convert with Gotenberg (headless Chromium in a container)
* Pure Go PDF library (`maroto`, `go-pdf/fpdf`)
* Headless Chromium driven in-process via `chromedp`

## Decision Outcome

Chosen option: "HTML template + Gotenberg", because HTML and CSS give the layout quality the report needs (sections, tables, page headers, clickable links) with the least code, and Gotenberg is a single container in the compose file with a simple HTTP API.

Flow: the API inserts a `queued` row in `export_jobs` (PostgreSQL). The `worker` claims it with `FOR UPDATE SKIP LOCKED` under `app.provisioning`, then works inside the job's tenant: it renders `report.html` from the template with the filtered logs, POSTs it to Gotenberg with `generateDocumentOutline=true` (which implies a tagged PDF), encrypts the result with AES-256-GCM (`EXPORT_KEY`), and writes it under `EXPORT_DIR/<tenant>/<job>.pdf` (local: a Docker volume shared by api and worker; production: the same through a mounted volume until an S3 store is needed). The job is then `done` with a 24 h expiry. Progress (0–100) lives in Redis through the `Degrading` cache; a Redis outage hides the bar and nothing else. Each worker tick deletes expired jobs and their files.

### Consequences

* Good, because designers can style the report in CSS; print styles are standard.
* Good, because Gotenberg is stateless and horizontally scalable.
* Neutral, because the export is a background job; job state is in PostgreSQL, progress in Redis.
* Bad, because one more container and ~400 MB image. Acceptable.
* Neutral, because tagged (accessible) output comes from Chromium's `generateTaggedPdf`, enabled through `generateDocumentOutline`; no post-processing step.

### Confirmation

Golden test: render a fixture document and compare page count and text extraction. Load test: 500 logs under 15 s.

## Pros and Cons of the Options

### HTML + Gotenberg

* Good, because best layout per line of code.
* Bad, because an extra service.

### Pure Go PDF library

* Good, because no extra service.
* Bad, because manual layout code for every element; links and page flow are tedious.

### `chromedp` in-process

* Good, because no extra service.
* Bad, because Chromium must be inside the API image; heavier image and process management in the API.
