---
status: proposed            # proposed | accepted | rejected | deprecated | superseded by PRD-NNNN
date: YYYY-MM-DD            # last update
owner: {product owner}
stakeholders: {people or teams consulted}
related-adrs: []            # e.g. [ADR-0004]
---

# PRD-NNNN: {Short title}

## 1. Summary

One paragraph. What is being built and for whom. Someone reading only this paragraph should understand the feature.

## 2. Problem

What is hard, slow, or impossible for the user today. Cite evidence where it exists (support requests, interviews, metrics, the source article).

## 3. Goals

- Measurable outcomes this PRD commits to.

## 4. Non-goals

- Things deliberately out of scope, so nobody has to ask.

## 5. Users and personas

Who uses this, and in which situation. One line per persona.

## 6. User stories

- As a {persona}, I want {capability} so that {outcome}.

## 7. Functional requirements

Numbered, testable statements. Use MUST / SHOULD / MAY (RFC 2119).

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | The system MUST … | Must |

## 8. Non-functional requirements

Performance, security, privacy, accessibility, observability. Numbered like NFR-1.

## 9. UX notes

Flows, screens, and states (empty, loading, error). Link mockups when they exist. No implementation detail.

## 10. Data

Entities and fields this feature introduces or changes, in business terms. The schema itself belongs in code and ADRs.

## 11. Success metrics

How we know the feature works after release. Each metric has a target and a measurement source.

## 12. Open questions

| Question | Owner | Due |
|---|---|---|

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
