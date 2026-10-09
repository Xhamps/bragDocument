---
status: accepted
date: 2026-10-08
decision-makers: Engineering
---

# Telegram bot integration

## Context and Problem Statement

Users add logs by messaging a Telegram bot ([PRD-0003](../prd/0003-telegram-bot.md)). How does the bot receive messages, how is it linked to a user, and where does it run?

## Decision Drivers

* Works locally without a public URL
* One codebase with the API: same authorization, same log creation path
* No message loss on restart

## Considered Options

* Long polling (`getUpdates`) in a `cmd/bot` process, webhook optional in production, using `go-telegram/bot`
* Webhook only, with a tunnel for local development
* Third-party automation (n8n, Zapier) calling our API

## Decision Outcome

Chosen option: "Long polling locally, webhook in production, same binary", because polling needs no public endpoint and the Bot API lets the same handler code run behind either transport. The bot process calls the internal `log` service directly, in-process, with the user's identity resolved from the Telegram link, so authorization is identical to the HTTP path.

Message parsing: first line → name, remaining lines → description, `#word` → tag, `!low|!medium|!high|!critical` → impact, URLs → reference links.

### Consequences

* Good, because `docker compose --profile app up` runs the bot without ngrok.
* Good, because switching to webhook is meant to be a configuration flag (`TELEGRAM_MODE=webhook`) once that transport is built.
* Neutral, because link codes and the last-created-log pointer for `/undo` live in Redis with TTLs ([ADR-0006](0006-redis-as-cache.md)).
* Neutral, because linking depends on Redis: code generation and redemption use the raw cache and report "temporarily unavailable" (HTTP 503) instead of degrading, a documented exception to [ADR-0012](0012-hexagonal-backend-layout.md)'s degradation rule; `/undo` goes through the degrading cache, so a Redis failure reads as "nothing to undo".
* Neutral, because a link code is 8 characters from a 31-symbol alphabet (about 8.5e11 values), expires after 10 minutes, and is single use (`GETDEL`). There is no per-account throttle on `/start` yet; add a failure counter if abuse appears.
* Neutral, because `telegram_links` has composite foreign keys to `users(id, tenant_id)` and `documents(id, tenant_id)`, so a link cannot point across tenants; the lookup by Telegram id is cross-tenant and runs under `app.provisioning`.
* Neutral, because the bot handles updates sequentially (one worker), recovers from panics, bounds each reply at 10 s, replies in plain text, and caps impact extraction at `min(LLM_TIMEOUT, 2s)` for NFR-1.
* Bad, because webhook mode is not built yet; `TELEGRAM_MODE` other than `polling` fails at startup (checked only when `TELEGRAM_BOT_TOKEN` is set; without a token the bot idles).
* Bad, because long polling means one bot instance; production uses webhook to scale. Telegram retries undelivered webhook updates, so restarts do not lose messages.

### Confirmation

Table tests for the message parser. Use-case tests for `app.Telegram` with fake ports. Transport test against a fake Bot API (`httptest`). Postgres integration test for RLS, the provisioning lookup, and the tenant-safe foreign keys on `telegram_links`. There is no end-to-end test of Postgres and the bot together.

## Pros and Cons of the Options

### Long polling + webhook

* Good, because local-friendly, production-scalable.
* Bad, because two transports to keep working; thin adapter layer.

### Webhook only

* Bad, because every developer needs a tunnel.

### Third-party automation

* Bad, because external dependency on the capture path, weaker auth story.
