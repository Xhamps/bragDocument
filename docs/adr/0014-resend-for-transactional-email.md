---
status: accepted
date: 2026-10-09
decision-makers: Engineering
---

# Resend for transactional email

## Context and Problem Statement

PRD-0004 FR-6 requires an email when a document is shared. The product sends no email today. Which provider, and how does email fit the degradation rule of ADR-0012?

## Decision Drivers

* Few lines of integration; no SMTP server to run
* Email must never make a share fail
* Works without configuration in local development

## Considered Options

* Resend through `github.com/resend/resend-go/v2`
* SMTP with a Mailpit container locally
* Supabase Auth "invite user" emails

## Decision Outcome

Chosen option: "Resend", because it is one HTTP call behind a `ports.Mailer` port. `adapters/email.Resend` sends with `MAIL_TIMEOUT` and counts failures in `email_failures_total`. Without `RESEND_API_KEY` the API logs a warning at startup and uses `email.Disabled`, which returns `domain.ErrUnavailable`. `app.Sharing` sends after the grant or invitation is committed and only logs failures: Resend is an optional dependency like Redis and Gotenberg.

### Consequences

* Good, because sharing works with or without email configured.
* Neutral, because `MAIL_FROM` must be a sender on a domain verified in Resend; the default `onboarding@resend.dev` delivers only to the account owner's address.
* Bad, because the share request waits for the send (bounded by `MAIL_TIMEOUT`); move to a background job if latency matters.

## Pros and Cons of the Options

### SMTP + Mailpit

* Good, because provider-neutral.
* Bad, because another container and TLS/auth configuration for one email.

### Supabase invite emails

* Bad, because they only reach people without an account; existing members would get nothing.
