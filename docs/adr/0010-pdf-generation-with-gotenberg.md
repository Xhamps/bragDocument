---
status: proposed
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

Flow: API enqueues an export job → worker renders `report.html` from the template with the filtered logs → POSTs it to Gotenberg → stores the PDF in object storage (local: a Docker volume; production: S3-compatible) → marks the job done with a 24 h expiry.

### Consequences

* Good, because designers can style the report in CSS; print styles are standard.
* Good, because Gotenberg is stateless and horizontally scalable.
* Neutral, because the export is a background job; job state is in PostgreSQL, progress in Redis.
* Bad, because one more container and ~400 MB image. Acceptable.
* Bad, because tagged (accessible) PDF support depends on Chromium's output; verify in confirmation, fall back to a post-processing step if needed.

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
