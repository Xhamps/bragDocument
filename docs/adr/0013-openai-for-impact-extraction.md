---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# OpenAI for impact extraction

## Context and Problem Statement

[PRD-0007](../prd/0007-impact-extraction.md) extracts an impact statement from a log's description when it is saved. Which LLM provider and integration shape do we use, given that saving a log must never fail because of it ([ADR-0012](0012-hexagonal-backend-layout.md) degradation rule)?

## Decision Drivers

* Short, single-field extraction; latency on the save path matters more than reasoning depth
* Structured output so the response is parseable without heuristics
* Optional at runtime: no key, no feature, no failure
* Testable without network access

## Considered Options

* OpenAI Chat Completions with a JSON-schema response format, official `openai-go` SDK
* Anthropic Claude
* Local model through Ollama in Docker Compose

## Decision Outcome

Chosen option: "OpenAI with the official Go SDK", because it is the team's choice, supports strict JSON-schema output, and the SDK accepts a base URL so tests run against `httptest`.

The integration is a port, `ports.ImpactExtractor`, with two adapters in `internal/adapters/llm`: `OpenAIExtractor` and `Disabled` (wired when `OPENAI_API_KEY` is empty). Configuration: `OPENAI_API_KEY`, `OPENAI_MODEL`, `LLM_TIMEOUT` (default 5 s). The SDK's retries are disabled on the save path.

### Consequences

* Good, because the port keeps the provider swappable; a second adapter is one file.
* Good, because a missing key or an outage degrades to "impact not checked" instead of an error.
* Bad, because log text leaves our infrastructure; PRD-0007 NFR-3 documents it and the key is opt-in.
* Bad, because saves can take up to `LLM_TIMEOUT` longer when OpenAI is slow.

### Confirmation

Unit tests for the use case with a fake extractor (found, none, error); adapter tests against `httptest` (found, not found, 500, slow); `impact_extraction_failures_total` visible on `/metrics`.

## Pros and Cons of the Options

### OpenAI

* Good, because strict JSON-schema responses.
* Bad, because a paid external dependency.

### Anthropic Claude

* Good, because comparable quality and structured output.
* Neutral, because not the team's choice for this feature.

### Ollama

* Good, because no data leaves the machine and no cost.
* Bad, because a heavy container, slower on CPU, and weaker extraction quality.
