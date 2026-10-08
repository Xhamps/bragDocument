---
status: proposed
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
* Good, because switching to webhook is a configuration flag (`TELEGRAM_MODE=webhook`).
* Neutral, because link codes and the last-created-log pointer for `/undo` live in Redis with TTLs ([ADR-0006](0006-redis-as-cache.md)).
* Bad, because long polling means one bot instance; production uses webhook to scale. Telegram retries undelivered webhook updates, so restarts do not lose messages.

### Confirmation

Unit tests for the message parser. Integration test with a fake Bot API server: link, `/use`, message → log row exists with parsed fields.

## Pros and Cons of the Options

### Long polling + webhook

* Good, because local-friendly, production-scalable.
* Bad, because two transports to keep working; thin adapter layer.

### Webhook only

* Bad, because every developer needs a tunnel.

### Third-party automation

* Bad, because external dependency on the capture path, weaker auth story.
