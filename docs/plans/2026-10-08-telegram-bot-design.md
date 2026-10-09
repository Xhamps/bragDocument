# Telegram bot: design

Date: 2026-10-08. Status: approved. Implements [PRD-0003](../prd/0003-telegram-bot.md) under [ADR-0009](../adr/0009-telegram-bot-integration.md), [ADR-0006](../adr/0006-redis-as-cache.md), [ADR-0007](../adr/0007-multi-tenancy-strategy.md), and [ADR-0012](../adr/0012-hexagonal-backend-layout.md).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Scope | FR-1..FR-10 including `/last` and `/undo`. Polling transport only; webhook mode deferred. |
| Architecture | Bot calls the `app` use cases in-process (ADR-0009). Rejected: bot as HTTP client with a service token (new impersonation auth for nothing), bot inside the `api` process (ties API scaling to one poller). |
| Link codes and undo pointer | Redis with TTLs, as ADR-0009 says. Linking therefore needs Redis: code generation and redemption use the raw cache and report "temporarily unavailable" instead of going through `Degrading` (which would hand out codes that never worked). `/undo` goes through `Degrading`; a miss means "nothing to undo". Documented as an exception to the degradation rule in ADR-0009. |
| Cross-tenant lookup | Telegram user id → user lookup runs under the existing `WithProvisioning` flag; everything after it under `WithTenant`. |
| Re-linking | A Telegram account already linked to another user is refused, never taken over. |
| Latency (NFR-1) | The bot builds its extractor with `min(LLM_TIMEOUT, 2s)`. |
| Deep link | New `APP_URL` config; link `APP_URL/documents/<doc>?edit=<log>` opens the log dialog. |

## 1. Data

Migration `0004_telegram`:

```
telegram_links  user_id uuid pk fk users ON DELETE CASCADE, tenant_id fk tenants,
                telegram_user_id bigint UNIQUE, document_id uuid NULL fk documents ON DELETE SET NULL,
                linked_at timestamptz
```

RLS policy `tenant_id = app_tenant_id() OR app_provisioning()`.

Redis keys:

- `tg:code:<code>` → `{user_id, tenant_id}`, TTL 10 min. Code: 8 characters, unambiguous alphabet, `crypto/rand`. Consumed with `GETDEL` (single use). A new code does not invalidate older ones; all expire in 10 min.
- `tg:undo:<telegram_user_id>` → `{document_id, log_id}`, TTL 5 min.

Deleting the target document sets `document_id` to NULL; the bot then asks for `/use`.

## 2. Bot process and commands

`cmd/bragdoc/bot.go` replaces the placeholder: connects Postgres and Redis, builds `app.Logs`, `app.Documents`, and a new `app.Telegram`, and starts `go-telegram/bot` in polling mode (new dependency). Empty `TELEGRAM_BOT_TOKEN` → warn and idle, so `compose up` works without a token. Only private chats are handled.

`adapters/telegram` maps an update to one `app.Telegram` method and sends the returned text. All logic is in `app`, so a webhook transport is a different adapter entry point later.

| Input | Behaviour |
|---|---|
| `/start <code>` | `GETDEL` the code → insert link → "Linked. /docs to pick a document". Unknown or expired → "Code invalid or expired, generate a new one in Settings". Redis down → "Linking temporarily unavailable". |
| Any message, unlinked | Link instructions only; nothing stored (FR-9). |
| `/help` | Commands plus a two-line example message. |
| `/docs` | Numbered active owned documents (web list order), current target marked. |
| `/use <n>` | Re-list, pick nth, save `document_id`. Bad n → clear reply. |
| Plain text | Parse → `Logs.Create` as the linked user → reply with name, impact, tags, links, deep link. `impact_statement == ""` adds "No impact stated, add one: <link>". Sets the undo pointer. |
| `/last` | `Logs.List` on the target, 5 newest. |
| `/undo` | Read pointer → `Logs.Delete` → clear pointer. None → "Nothing to undo (only the last bot log, within 5 min)". |

Error replies (FR-10, NFR-3): 403/404 → "You no longer have access to that document. /docs". 409 archived → "That document is archived. /docs". No target → "Pick a document first: /docs". Validation errors listed per field. Anything else → "Couldn't save, try again", plus a log line.

## 3. Message parser

`domain.ParseLogMessage(text) (LogDraft, error)`, pure:

- Markers anywhere: `#tag` (word characters and `-`, no space after `#`, so Markdown `# Heading` is untouched); `!low|!medium|!high|!critical`, case-insensitive, last wins; `http(s)://` URLs → links with empty label, deduplicated.
- Name: first line with markers and URLs removed, whitespace collapsed. Empty → validation error "First line needs some text besides tags/links".
- Description: remaining lines verbatim, only `!impact` markers removed.
- Defaults: impact `medium`, status `done`. Limits come from `domain.Log.Validate`, so errors match the UI.

Example: `Shipped SSO #auth !high https://github.com/x/pr/1` / `Cut login tickets by 40%` → name "Shipped SSO", tag `auth`, impact high, one link, that description.

## 4. API and frontend

Behind `Auth`, caller only:

| Method | Path | Use case | Notes |
|---|---|---|---|
| GET | `/me/telegram` | `Telegram.Status` | `{linked, linked_at?, document_id?}` |
| POST | `/me/telegram/code` | `Telegram.NewCode` | 201 `{code, expires_at, bot_url?}`. Redis down → 503. |
| DELETE | `/me/telegram` | `Telegram.Unlink` | 204, idempotent. |
| GET | `/documents/:id/logs/:logId` | `Logs.Get` | For the deep link. |

`bot_url` = `https://t.me/<TELEGRAM_BOT_USERNAME>?start=<code>`; omitted when `TELEGRAM_BOT_USERNAME` is empty. `openapi.yaml` updated.

Frontend:

- Route `/settings` (`routes/Settings.tsx`), "Settings" nav link.
- Telegram card. Not linked: "Generate link code" → code with Copy, expiry, "Open in Telegram" when `bot_url` is set, and "send `/start CODE` to the bot". Linked: "Linked since {date}" and Unlink with confirm. Status refetches on window focus.
- `DocumentLogs` reads `?edit=<logId>`, fetches the log, opens `LogFormDialog`, drops the param on close.
- `settings/useTelegram.ts` react-query hooks.

## 5. Testing

- Domain: parser table test (markers, stripping, empty name, last impact wins, `# heading`, URL dedupe).
- App, hand-written fakes: redeem valid / expired / linked elsewhere / Redis error; unlinked → instructions, nothing stored; `/use` bounds; message → `Create` with parsed fields; FR-10 replies for 403, 404, 409, no target; `/undo` within window and empty; `/last`.
- Telegram adapter: fake Bot API on `httptest`, link → `/use` → message, asserting the create call and the reply.
- Postgres (`integration` tag): RLS on `telegram_links`, provisioning lookup, unique Telegram id, `SET NULL` on document delete.
- HTTP: the four handlers with fakes, 503 on Redis down.
- Frontend (Vitest): Settings card states, copy, unlink confirm; `?edit=` opens the dialog.

## 6. Delivery

Branch `feat/telegram-bot`. Conventional commits per layer: docs, migration, queries, domain parser, app, telegram adapter, http, cmd, frontend. PRD-0003 → `accepted` with the one-shot decision and the open question closed. ADR-0009 → `accepted`, noting Redis-backed linking as a degradation exception and webhook deferred. `docs/README.md`, `.env.example` (`APP_URL`, `TELEGRAM_BOT_USERNAME`), `docker-compose.yml`, and README updated. Pull request against `main` linking PRD-0003 and ADR-0009, noting webhook mode deferred.
