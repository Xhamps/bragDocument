---
status: accepted
date: 2026-10-08
owner: Product
stakeholders: Engineering, Security
related-adrs: [ADR-0013]
---

# PRD-0007: Impact statement extracted from the description

## 1. Summary

When a log is saved, an LLM reads its name and description and extracts the one sentence that states the impact of the work. The statement is stored on the log and shown under its name. When the text states no impact, the user is warned and asked to add one.

## 2. Problem

The article's central advice is to state the impact, not the activity. A separate "impact statement" input is one more field to skip. Writing the impact inside the description is natural; pulling it out automatically keeps the form short and still makes the impact visible and measurable (PRD-0002 §11: 90 % of logs have an impact statement).

## 3. Goals

- Every saved log gets a statement or an explicit "none found".
- The user sees a warning in the form, before leaving it, when none is found.
- Saving a log never fails or slows beyond the timeout because of the LLM.

## 4. Non-goals

- Rewriting or embellishing the user's text. The statement quotes or tightly paraphrases what is written.
- Editing the statement by hand in v1. The user edits the description instead.
- Back-filling statements for existing logs in bulk.

## 5. Users and personas

- **Author**: writes the log and reacts to the warning.

## 6. User stories

- As an author, I want my impact surfaced from what I already wrote so that I do not fill two fields.
- As an author, I want to be told when I forgot the impact so that the log is useful in my review.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | On create, and on edit when the name or description changed, the system MUST extract an impact statement from the name and description. | Must |
| FR-2 | The statement MUST be stored as: text (found), empty (checked, none found), or null (not checked: disabled, timed out, or failed). | Must |
| FR-3 | The statement MUST be at most 280 characters and MUST NOT state anything absent from the text. | Must |
| FR-4 | When the result is empty, the form MUST stay open with a warning and offer to add the impact. | Must |
| FR-5 | The list MUST show the statement under the log name, or a "No impact stated" marker when empty. | Must |
| FR-6 | An extraction failure MUST NOT fail the save; the log is stored with a null statement and retried on the next edit of its text. | Must |
| FR-7 | Example logs (PRD-0002 §9) MUST carry fixed statements and MUST NOT call the LLM. | Must |

## 8. Non-functional requirements

- NFR-1: Extraction timeout 5 s by default (`LLM_TIMEOUT`), no retries on the save path.
- NFR-2: Failures counted in `impact_extraction_failures_total{reason}`.
- NFR-3: Privacy: the log name and description are sent to OpenAI. No other data (user, tenant, links, tags) is sent. Operators who cannot accept this leave `OPENAI_API_KEY` unset, which disables extraction.
- NFR-4: The prompt treats the description as data; the output is only ever stored on the author's own log, so injected instructions can at worst produce a wrong statement on that log.

## 9. UX notes

- Warning copy: "We couldn't find an impact in the description. What changed because of this work?" with "Add impact" (focuses the description) and "Close".
- A null statement shows nothing; no warning, since the user did nothing wrong.

## 10. Data

- `logs.impact_statement text NULL` (PRD-0002 §10).

## 11. Success metrics

- 90 % of logs have a non-empty statement (PRD-0002 §11).
- Extraction failure rate under 1 % of saves.

## 12. Open questions

| Question | Owner | Due |
|---|---|---|

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
| 2026-10-08 | Synchronous on save, not async | The warning must appear while the user is still in the form |
| 2026-10-08 | OpenAI ([ADR-0013](../adr/0013-openai-for-impact-extraction.md)) | Team choice |
