# Bot impact follow-up: design

Date: 2026-10-09. Status: approved. Extends [PRD-0007](../prd/0007-impact-extraction.md) FR-4 to the Telegram bot ([PRD-0003](../prd/0003-telegram-bot.md), [ADR-0009](../adr/0009-telegram-bot-integration.md)).

## Decisions taken during brainstorming

| Topic | Decision |
|---|---|
| Web UI | Already warns (PRD-0007 FR-4). Only the missing "Close" button (§9) is added. |
| How the bot user answers | Inline buttons, then the next plain message is the answer. Not commands, not Telegram replies. |
| Answer modes | "Add to description" appends after a blank line; "Replace description" substitutes. |
| Answer parsing | Raw text; `#tag` and `!impact` markers are not parsed in an answer. |
| Pending window | 5 minutes, same as `/undo`. No `/cancel`; it expires. |
| Old keyboard | Not removed after a tap; a second tap only resets the pending answer. |

## Flow

1. A bot capture whose extraction returns "none found" replies with the existing warning plus two inline buttons. `callback_data` is `impact:add:<logID>` or `impact:replace:<logID>` (under Telegram's 64-byte limit).
2. A tap stores `tg:pending:<telegramID>` = `{user_id, document_id, log_id, mode}` in the cache with a 5-minute TTL and replies "Send the text to add." or "Send the new description.".
3. The next plain message, when a pending answer exists for the same linked user, consumes the key and calls `Logs.Update` with the new description. `Update` re-extracts because the description changed.
4. Reply: "Updated: <name>" plus "Impact found: <statement>"; still none → the warning and buttons again; not checked (disabled, failed) → "Updated" only.
5. Commands work normally while an answer is pending. An expired pending answer, or one stored for another user (relink), is ignored and the message creates a log. A log deleted in the meantime replies "That log is gone.".

## Components

- `app.Telegram.Reply` returns `BotReply{Text string; Buttons []BotButton}` (`BotButton{Label, Data}`) instead of `string`.
- New `app.Telegram.Callback(ctx, telegramID, data) BotReply` handles a button tap.
- Pending state uses the cache the bot already uses for `/undo`; no new port. Write access is enforced by `Logs.Update`, as on the web.
- `adapters/telegram/bot.go` also handles `Update.CallbackQuery`: `AnswerCallbackQuery` (stops the spinner), then `SendMessage`. Replies with buttons carry an `InlineKeyboardMarkup`.
- Web: `LogFormDialog` warning gains a "Close" button.
- Docs: PRD-0003 gains FR-11; PRD-0007 decisions log records the bot follow-up.

## Error handling

- Cache failures degrade: no pending answer means the message becomes a new log, never an error.
- `Logs.Update` errors go through `failed()`; `ErrNotFound` for the pending log replies "That log is gone.".
- Unknown or malformed `callback_data` replies "That button has expired.".
- An unlinked account tapping a button gets the usual not-linked reply.

## Testing

- App, with fakes: tap → answer → append; tap → answer → replace; answer still without impact → buttons again; expired pending → new log; pending of a relinked user ignored; log undone before the answer; malformed callback data.
- Adapter, against the existing fake Bot API: a callback query is answered and replied to; a reply with buttons carries the keyboard.
- Web: the "Close" button closes the dialog.
