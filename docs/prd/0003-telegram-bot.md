---
status: proposed
date: 2026-10-08
owner: Product
stakeholders: Engineering
related-adrs: [ADR-0009]
---

# PRD-0003: Adding logs through a Telegram bot

## 1. Summary

A user links their Telegram account once, then sends messages to the bot to create logs in a chosen document without opening the web app.

## 2. Problem

The article's main advice is to record things as they happen. The moment of "I just shipped that" is rarely at a browser tab with the app open. A chat bot lowers the capture cost to one message.

## 3. Goals

- A log can be captured from Telegram in one message.
- Logs created via the bot are indistinguishable from UI logs afterwards (same fields, same permissions).

## 4. Non-goals

- Slack, Discord, email, or SMS channels in v1.
- Reading or editing logs from Telegram beyond listing the last few.
- Group-chat usage. The bot works in a private chat with the user.

## 5. Users and personas

- **Author** with a Telegram account.

## 6. User stories

- As an author, I want to link my Telegram account from settings so that the bot knows who I am.
- As an author, I want to pick which document the bot writes to.
- As an author, I want to send a message and have a log created with the message as name.
- As an author, I want to add tags, impact, and a link in the same message.
- As an author, I want the bot to confirm what it created, with a link to edit it.

## 7. Functional requirements

| ID | Requirement | Priority |
|---|---|---|
| FR-1 | Settings MUST show a one-time link code; sending `/start <code>` to the bot MUST link the Telegram account to the user. Codes expire after 10 minutes. | Must |
| FR-2 | A linked user MUST be able to unlink from settings. | Must |
| FR-3 | `/docs` MUST list the user's active documents; `/use <n>` MUST set the target document for subsequent messages. | Must |
| FR-4 | Any plain message MUST create a log in the target document. First line is the name; the rest is the description. | Must |
| FR-5 | Inline markers MUST be parsed: `#tag` → tag, `!low` `!medium` `!high` `!critical` → impact level, URLs → reference links. Default impact `medium`, default status `done`. | Must |
| FR-6 | The bot MUST reply with a summary of the created log and a deep link to edit it in the UI. | Must |
| FR-7 | `/last` SHOULD list the last 5 logs in the target document. | Should |
| FR-8 | `/undo` SHOULD delete the last log created via the bot within 5 minutes. | Should |
| FR-9 | Messages from unlinked accounts MUST receive only instructions on how to link; nothing is stored. | Must |
| FR-10 | Logs created via the bot MUST respect the user's role on the target document; if the user lost write access, the bot MUST say so. | Must |

## 8. Non-functional requirements

- NFR-1: Bot reply within 3 s of message receipt.
- NFR-2: Telegram user IDs are stored only as the link; message content is stored only as the resulting log.
- NFR-3: A message that fails to create a log MUST be reported to the user; nothing is silently dropped.

## 9. UX notes

- Settings → "Telegram" card: status (linked / not linked), link code with copy button, unlink button.
- Bot `/help` text includes a two-line example message.

## 10. Data

- **Telegram link**: user, telegram user id, target document, linked at.
- **Link code**: code, user, expires at.

## 11. Success metrics

- 30 % of active authors link Telegram within the first month.
- Linked authors create at least twice as many logs per month as unlinked ones.

## 12. Open questions

| Question | Owner | Due |
|---|---|---|
| Should the bot ask follow-up questions (impact? tags?) or stay one-shot? Proposal: one-shot, with `/edit` later. | Product | before build |

## 13. Decisions log

| Date | Decision | Why |
|---|---|---|
| 2026-10-08 | One-shot messages with inline markers | Lowest capture cost, matches the "write it down now" goal |
