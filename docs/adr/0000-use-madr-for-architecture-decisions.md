---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# Use MADR for architecture decisions

## Context and Problem Statement

The project requires every decision to be written down under `docs/`. Architecture decisions need a consistent, lightweight format that records the options considered and why one was chosen, so that future readers understand the reasoning and know when to revisit it.

## Decision Drivers

* Low writing cost, so decisions actually get recorded
* Options and trade-offs visible, not only the outcome
* Widely known format, no onboarding

## Considered Options

* MADR (Markdown Architectural Decision Records)
* Nygard-style ADR (Context / Decision / Status / Consequences)
* Free-form design docs

## Decision Outcome

Chosen option: "MADR", because it is the format the project requirements name, it keeps the considered options and their pros and cons in the record, and it is plain Markdown.

### Consequences

* Good, because every ADR has the same shape; `docs/adr/TEMPLATE.md` is the source.
* Good, because superseding is explicit through the `status` field.
* Bad, because the full template is verbose for trivial decisions; optional sections may be removed.

### Confirmation

Code review rejects changes to `docs/adr/` that do not follow the template. File names follow `NNNN-title-with-dashes.md`.

## More Information

Template and guidance: https://adr.github.io/madr/
