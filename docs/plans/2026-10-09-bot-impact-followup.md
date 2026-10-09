# Bot impact follow-up Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** When a bot-created log states no impact, the bot offers "Add to description" / "Replace description" buttons; the user's next message updates the description and the impact is re-extracted. The web warning gains its missing "Close" button.

**Architecture:** Design: [2026-10-09-bot-impact-followup-design.md](2026-10-09-bot-impact-followup-design.md). `app.Telegram.Reply` returns `BotReply{Text, Buttons}`; a new `app.Telegram.Callback` handles taps. One cache key, `tg:impact:<telegramID>`, holds the last no-impact bot log (the `/undo` entry shape) plus the answer mode once a button is tapped; the next plain message consumes it and calls `Logs.Update`, which re-extracts. The Telegram adapter answers callback queries and renders buttons as an inline keyboard.

**Tech Stack:** Go 1.24, `github.com/go-telegram/bot` v1.27.0, testify; React + Vitest + Testing Library.

**Branch:** `feat/bot-impact-followup` (already created, design doc committed). Finish with a GitHub PR to `main`.

**Conventions:** Run Go commands from `backend/`, npm from `frontend/`. Commit messages end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

**Deviation from the design doc:** the design names the key `tg:pending:`. The plan uses one key, `tg:impact:<telegramID>`, set at capture (so a tap can verify the log) and given a mode on tap. Task 7 updates the design doc.

---

### Task 1: `Reply` returns `BotReply` (refactor, no behaviour change)

**Files:**
- Modify: `backend/internal/app/telegram.go` (types)
- Modify: `backend/internal/app/telegram_reply.go:19-50` (`Reply`)
- Modify: `backend/internal/app/telegram_test.go` (helper + mechanical rename)
- Modify: `backend/internal/adapters/telegram/bot.go`, `bot_test.go` (compile only)

**Step 1: Add the types** to `backend/internal/app/telegram.go`, after the `const` blocks:

```go
// BotReply is what the bot sends back: text, plus inline buttons when the
// user can act on it.
type BotReply struct {
	Text    string
	Buttons []BotButton
}

// BotButton is one inline button; Data comes back to Callback when tapped.
type BotButton struct{ Label, Data string }
```

**Step 2: Split `Reply`** in `backend/internal/app/telegram_reply.go`. Replace the whole `Reply` function with:

```go
// Reply handles one private message from telegramID and returns the answer.
// Nothing is stored for an unlinked account (FR-9); every failure gets a reply (NFR-3).
func (t *Telegram) Reply(ctx context.Context, telegramID int64, text string) BotReply {
	cmd, arg := splitCommand(text)
	if cmd == "/start" && arg != "" {
		return BotReply{Text: t.start(ctx, telegramID, arg)}
	}
	link, failure := t.lookup(ctx, telegramID)
	if failure != "" {
		return BotReply{Text: failure}
	}
	ctx = t.scope(ctx, link.TenantID)
	if cmd == "" {
		return BotReply{Text: t.capture(ctx, link, telegramID, arg)}
	}
	return BotReply{Text: t.command(ctx, link, telegramID, cmd, arg)}
}

// lookup finds the caller's link; a non-empty reply means stop and send it.
func (t *Telegram) lookup(ctx context.Context, telegramID int64) (domain.TelegramLink, string) {
	link, err := t.links.FindByTelegramID(ctx, telegramID)
	if errors.Is(err, domain.ErrNotFound) {
		return link, msgNotLinked
	}
	if err != nil {
		slog.ErrorContext(ctx, "telegram link lookup failed", slog.Any("err", err))
		return link, msgTryAgain
	}
	return link, ""
}

func (t *Telegram) command(ctx context.Context, link domain.TelegramLink, telegramID int64, cmd, arg string) string {
	switch cmd {
	case "/start", "/help":
		return msgHelp
	case "/docs":
		return t.listDocs(ctx, link)
	case "/use":
		return t.use(ctx, link, arg)
	case "/last":
		return t.last(ctx, link)
	case "/undo":
		return t.undoLast(ctx, link, telegramID)
	}
	return "Unknown command. Send /help."
}
```

**Step 3: Test helper + rename.** Add to `backend/internal/app/telegram_test.go` after `linked`:

```go
// say sends text as Telegram user id and returns the reply text.
func (f *tgFixture) say(id int64, text string) string {
	return f.tg.Reply(context.Background(), id, text).Text
}
```

Then:

```bash
sed -i '' 's/f\.tg\.Reply(context\.Background(), /f.say(/g' internal/app/telegram_test.go
grep -n "tg.Reply(" internal/app/*_test.go   # expect: only the line inside say()
```

**Step 4: Keep the adapter compiling.** In `backend/internal/adapters/telegram/bot.go`, import `"github.com/xhamps/bragdocument/backend/internal/app"`, change `Replier.Reply` to return `app.BotReply`, and in `reply()` return `r.Reply(ctx, m.From.ID, m.Text).Text`. In `bot_test.go`, `fakeReplier.Reply` returns `app.BotReply{Text: "ok: " + text}` (import `app`). Task 5 rewrites this properly.

**Step 5: Run**

```bash
go build ./... && go test ./internal/app/ ./internal/adapters/telegram/
```
Expected: PASS (pure refactor).

**Step 6: Commit**

```bash
git add -A backend && git commit -m "refactor(app): bot Reply returns BotReply with optional buttons"
```

---

### Task 2: Warning carries the impact buttons

**Files:**
- Modify: `backend/internal/app/telegram.go` (consts)
- Modify: `backend/internal/app/telegram_reply.go` (`capture`, new helpers)
- Test: `backend/internal/app/telegram_test.go` (`TestTelegramNoImpactWarning`)

**Step 1: Write the failing test.** Replace `TestTelegramNoImpactWarning`:

```go
func TestTelegramNoImpactWarning(t *testing.T) {
	f := newTGFixture()
	f.linked("d1")
	f.lf.impact.statement = ""
	r := f.tg.Reply(context.Background(), 42, "Did a thing")
	require.Contains(t, r.Text, msgNoImpact)
	require.Equal(t, []BotButton{
		{Label: "Add to description", Data: "impact:add:l1"},
		{Label: "Replace description", Data: "impact:replace:l1"},
	}, r.Buttons)
	require.Equal(t, undoTTL, f.undo.ttl["tg:impact:42"])

	f.lf.impact.err = domain.ErrUnavailable // extraction off: nil statement, no nagging
	r = f.tg.Reply(context.Background(), 42, "Did a thing")
	require.NotContains(t, r.Text, msgNoImpact)
	require.Empty(t, r.Buttons)
}
```

**Step 2: Run** `go test ./internal/app/ -run TestTelegramNoImpactWarning` → FAIL (`msgNoImpact` undefined).

**Step 3: Implement.** In `telegram.go` add to the message consts:

```go
	msgNoImpact        = "No impact stated. What changed because of this?"
```

and to the first const block:

```go
	impactAdd     = "add"
	impactReplace = "replace"
```

In `telegram_reply.go`, after `undoKey`:

```go
// impactKey holds the last bot log that stated no impact, and the answer mode
// once one of its buttons is tapped (PRD-0007 FR-4 on the bot).
func impactKey(telegramID int64) string { return "tg:impact:" + strconv.FormatInt(telegramID, 10) }

type impactEntry struct {
	undoEntry
	Mode string `json:"mode,omitempty"` // "" until a button is tapped, then impactAdd or impactReplace
}

func impactButtons(logID string) []BotButton {
	return []BotButton{
		{Label: "Add to description", Data: "impact:" + impactAdd + ":" + logID},
		{Label: "Replace description", Data: "impact:" + impactReplace + ":" + logID},
	}
}

// askImpact remembers e so a tapped button can find its log, and returns the buttons.
func (t *Telegram) askImpact(ctx context.Context, telegramID int64, e undoEntry) []BotButton {
	v, _ := json.Marshal(impactEntry{undoEntry: e})
	_ = t.undo.Set(ctx, impactKey(telegramID), v, undoTTL) // Degrading: a miss only expires the buttons
	return impactButtons(e.LogID)
}
```

Change `capture` to return `BotReply`: every `return t.failed(ctx, err)` / `return msgPickDoc` becomes `return BotReply{Text: ...}`; build the undo entry once as `e := undoEntry{...}` and marshal `e`; replace the warning block with

```go
	var buttons []BotButton
	if l.ImpactStatement != nil && *l.ImpactStatement == "" {
		b.WriteString("\n\n" + msgNoImpact)
		buttons = t.askImpact(ctx, telegramID, e)
	}
```

and end with `return BotReply{Text: b.String(), Buttons: buttons}`. In `Reply`, the plain-message branch becomes `return t.capture(ctx, link, telegramID, arg)`.

**Step 4: Run** `go test ./internal/app/` → PASS (`TestTelegramCapture`'s exact text is unchanged).

**Step 5: Commit**

```bash
git add backend/internal/app && git commit -m "feat(app): bot no-impact warning offers add/replace description buttons"
```

---

### Task 3: `Callback` records which button was tapped

**Files:**
- Modify: `backend/internal/app/telegram.go` (messages)
- Modify: `backend/internal/app/telegram_reply.go` (`Callback`)
- Test: `backend/internal/app/telegram_test.go`

**Step 1: Write the failing test.**

```go
func TestTelegramImpactCallback(t *testing.T) {
	ctx := context.Background()
	f := newTGFixture()
	f.linked("d1")
	f.lf.impact.statement = ""
	f.say(42, "Did a thing")

	require.Equal(t, msgSendAddition, f.tg.Callback(ctx, 42, "impact:add:l1").Text)
	require.Contains(t, string(f.undo.data["tg:impact:42"]), `"mode":"add"`)
	require.Equal(t, msgSendReplacement, f.tg.Callback(ctx, 42, "impact:replace:l1").Text, "a second tap switches mode")
	require.Contains(t, string(f.undo.data["tg:impact:42"]), `"mode":"replace"`)

	for _, data := range []string{"impact:add:l9", "impact:bogus:l1", "nope", ""} {
		require.Equal(t, msgButtonExpired, f.tg.Callback(ctx, 42, data).Text, data)
	}
	require.Equal(t, msgNotLinked, f.tg.Callback(ctx, 43, "impact:add:l1").Text)
}
```

**Step 2: Run** `go test ./internal/app/ -run TestTelegramImpactCallback` → FAIL (undefined).

**Step 3: Implement.** Messages in `telegram.go`:

```go
	msgButtonExpired   = "That button has expired. Edit the log in the web app."
	msgSendAddition    = "Send the text to add to the description."
	msgSendReplacement = "Send the new description."
	msgLogGone         = "That log is gone. Send it again to log it."
```

In `telegram_reply.go`, after `Reply`:

```go
// Callback handles a tapped inline button; only the impact buttons exist.
// The tap is valid for the last no-impact log, within undoTTL, for the same user.
func (t *Telegram) Callback(ctx context.Context, telegramID int64, data string) BotReply {
	rest, ok := strings.CutPrefix(data, "impact:")
	mode, logID, _ := strings.Cut(rest, ":")
	if !ok || (mode != impactAdd && mode != impactReplace) {
		return BotReply{Text: msgButtonExpired}
	}
	link, failure := t.lookup(ctx, telegramID)
	if failure != "" {
		return BotReply{Text: failure}
	}
	e, ok := t.pendingImpact(ctx, link, telegramID)
	if !ok || e.LogID != logID {
		return BotReply{Text: msgButtonExpired}
	}
	e.Mode = mode
	v, _ := json.Marshal(e)
	_ = t.undo.Set(ctx, impactKey(telegramID), v, undoTTL)
	if mode == impactAdd {
		return BotReply{Text: msgSendAddition}
	}
	return BotReply{Text: msgSendReplacement}
}

// pendingImpact reads the caller's impact entry; another user's (relinked
// account) or an unreadable one counts as none.
func (t *Telegram) pendingImpact(ctx context.Context, link domain.TelegramLink, telegramID int64) (impactEntry, bool) {
	v, found, _ := t.undo.Get(ctx, impactKey(telegramID))
	var e impactEntry
	if !found || json.Unmarshal(v, &e) != nil || e.UserID != link.UserID {
		return impactEntry{}, false
	}
	return e, true
}
```

**Step 4: Run** `go test ./internal/app/` → PASS.

**Step 5: Commit**

```bash
git add backend/internal/app && git commit -m "feat(app): bot Callback arms an add/replace answer for the no-impact log"
```

---

### Task 4: The next message answers

**Files:**
- Modify: `backend/internal/app/telegram_reply.go` (`Reply`, new `answer`)
- Test: `backend/internal/app/telegram_test.go`

**Step 1: Write the failing tests.**

```go
func TestTelegramImpactAnswerAdds(t *testing.T) {
	ctx := context.Background()
	f := newTGFixture()
	f.linked("d1")
	f.lf.impact.statement = ""
	f.say(42, "Migrated CI\nMoved to GitHub Actions")

	f.tg.Callback(ctx, 42, "impact:add:l1")
	f.lf.impact.statement = "Build time dropped from 20 to 6 min"
	require.Equal(t, "Updated: Migrated CI\nImpact found: Build time dropped from 20 to 6 min",
		f.say(42, "Build time dropped from 20 to 6 min"))
	require.Equal(t, "Moved to GitHub Actions\n\nBuild time dropped from 20 to 6 min", f.lf.logs.logs["l1"].Description)
	require.Len(t, f.lf.logs.logs, 1, "the answer is not a new log")

	require.Contains(t, f.say(42, "Another thing"), "Logged: Another thing", "the answer was consumed")
	require.Len(t, f.lf.logs.logs, 2)
}

func TestTelegramImpactAnswerReplacesAndAsksAgain(t *testing.T) {
	ctx := context.Background()
	f := newTGFixture()
	f.linked("d1")
	f.lf.impact.statement = ""
	f.say(42, "Migrated CI\nMoved to GitHub Actions")

	f.tg.Callback(ctx, 42, "impact:replace:l1")
	r := f.tg.Reply(ctx, 42, "Still vague")
	require.Equal(t, "Updated: Migrated CI\n\n"+msgNoImpact, r.Text)
	require.Equal(t, impactButtons("l1"), r.Buttons, "still none: ask again")
	require.Equal(t, "Still vague", f.lf.logs.logs["l1"].Description)

	f.tg.Callback(ctx, 42, "impact:add:l1")
	f.lf.impact.err = domain.ErrUnavailable
	require.Equal(t, "Updated: Migrated CI", f.say(42, "Saved a day"), "not checked: no impact line")
	require.Equal(t, "Still vague\n\nSaved a day", f.lf.logs.logs["l1"].Description)
}

func TestTelegramImpactAnswerIgnoredOrGone(t *testing.T) {
	ctx := context.Background()
	f := newTGFixture()
	f.linked("d1")
	f.lf.impact.statement = ""
	f.say(42, "Did a thing") // l1

	require.Contains(t, f.say(42, "Second"), "Logged: Second", "no tap: a plain message is a new log") // l2

	f.say(42, "Third") // l3
	f.tg.Callback(ctx, 42, "impact:add:l3")
	f.say(42, "/undo")
	require.Equal(t, msgLogGone, f.say(42, "Saved a day"))
	require.Len(t, f.lf.logs.logs, 2)

	f.say(42, "Fourth") // l4
	f.tg.Callback(ctx, 42, "impact:add:l4")
	require.NoError(t, f.tg.Unlink(ctx, "u1"))
	f.lf.docs.docs["d7"] = domain.Document{ID: "d7", TenantID: "t2", OwnerID: "u2", State: domain.DocumentActive}
	f.links.byUser["u2"] = domain.TelegramLink{UserID: "u2", TenantID: "t2", TelegramUserID: 42, DocumentID: "d7"}
	require.Contains(t, f.say(42, "Mine now"), "Logged: Mine now", "relinked: u1's pending answer is ignored")
	require.Equal(t, "", f.lf.logs.logs["l4"].Description)
}
```

**Step 2: Run** `go test ./internal/app/ -run TestTelegramImpactAnswer` → FAIL (messages become new logs).

**Step 3: Implement.** In `Reply`, the plain-message branch:

```go
	if cmd == "" {
		if r, ok := t.answer(ctx, link, telegramID, arg); ok {
			return r
		}
		return t.capture(ctx, link, telegramID, arg)
	}
```

After `capture`:

```go
// answer applies a plain message to the log whose impact button was tapped.
// ok is false when no answer is pending, so the message is a new log.
func (t *Telegram) answer(ctx context.Context, link domain.TelegramLink, telegramID int64, text string) (BotReply, bool) {
	e, ok := t.pendingImpact(ctx, link, telegramID)
	if !ok || e.Mode == "" {
		return BotReply{}, false
	}
	_ = t.undo.Delete(ctx, impactKey(telegramID))
	l, err := t.logs.Get(ctx, e.DocumentID, e.LogID, link.UserID)
	if err == nil {
		desc := text
		if e.Mode == impactAdd && strings.TrimSpace(l.Description) != "" {
			desc = l.Description + "\n\n" + text
		}
		l, err = t.logs.Update(ctx, UpdateLogInput{ID: l.ID, DocumentID: l.DocumentID, UserID: link.UserID, Description: &desc})
	}
	if errors.Is(err, domain.ErrNotFound) {
		return BotReply{Text: msgLogGone}, true
	}
	if err != nil {
		return BotReply{Text: t.failed(ctx, err)}, true
	}
	r := BotReply{Text: "Updated: " + l.Name}
	switch {
	case l.ImpactStatement == nil: // not checked: nothing to say
	case *l.ImpactStatement == "":
		r.Text += "\n\n" + msgNoImpact
		r.Buttons = t.askImpact(ctx, telegramID, e.undoEntry)
	default:
		r.Text += "\nImpact found: " + *l.ImpactStatement
	}
	return r, true
}
```

**Step 4: Run** `go test ./internal/app/` → PASS.

**Step 5: Commit**

```bash
git add backend/internal/app && git commit -m "feat(app): bot uses the next message to add to or replace a no-impact description"
```

---

### Task 5: Telegram adapter — inline keyboard and callback queries

**Files:**
- Modify: `backend/internal/adapters/telegram/bot.go`
- Test: `backend/internal/adapters/telegram/bot_test.go`

**Step 1: Write the failing test.** In `bot_test.go`, give `fakeReplier` a `Callback` and buttons on "Shipped X":

```go
func (f *fakeReplier) Reply(_ context.Context, telegramID int64, text string) app.BotReply {
	f.mu.Lock()
	f.from, f.texts = telegramID, append(f.texts, text)
	f.mu.Unlock()
	if text == "boom" {
		panic("replier exploded")
	}
	return app.BotReply{Text: "ok: " + text, Buttons: []app.BotButton{{Label: "Add", Data: "impact:add:l1"}}}
}

func (f *fakeReplier) Callback(_ context.Context, telegramID int64, data string) app.BotReply {
	f.mu.Lock()
	f.from, f.texts = telegramID, append(f.texts, "tap:"+data)
	f.mu.Unlock()
	return app.BotReply{Text: "tapped " + data}
}
```

In the test server:
- `sentMsg` gains `markup string`, read from `r.FormValue("reply_markup")` (JSON fallback: `ReplyMarkup json.RawMessage \`json:"reply_markup"\``, stored as `string(body.ReplyMarkup)`).
- Add a fourth update to the first `getUpdates` result:
  `{"update_id":4,"callback_query":{"id":"cb1","from":{"id":42,"is_bot":false,"first_name":"A"},"chat_instance":"x","data":"impact:add:l1"}}`
- Add a case recording answered callbacks:

```go
		case strings.HasSuffix(r.URL.Path, "/answerCallbackQuery"):
			_ = r.ParseMultipartForm(1 << 20)
			answered <- r.FormValue("callback_query_id")
			_, _ = io.WriteString(w, `{"ok":true,"result":true}`)
```
  with `answered := make(chan string, 1)` declared next to `sent`. If the library posts JSON here, decode `{"callback_query_id"}` like `sendMessage` does.

Replace the reply loop and final assertions:

```go
	for _, want := range []string{failedReply, "ok: Shipped X", "tapped impact:add:l1"} {
		select {
		case got := <-sent:
			require.Equal(t, want, got.text)
			require.Empty(t, got.parseMode, "replies carry user text; send them as plain text")
			if want == "ok: Shipped X" {
				require.JSONEq(t, `{"inline_keyboard":[[{"text":"Add","callback_data":"impact:add:l1"}]]}`, got.markup)
			} else {
				require.Empty(t, got.markup)
			}
		case <-ctx.Done():
			t.Fatalf("no reply %q sent", want)
		}
	}
	require.Equal(t, "cb1", <-answered, "the button spinner is stopped")
	rep.mu.Lock()
	defer rep.mu.Unlock()
	require.Equal(t, int64(42), rep.from)
	require.Equal(t, []string{"boom", "Shipped X", "tap:impact:add:l1"}, rep.texts, "the group message never reaches the replier")
```

**Step 2: Run** `go test ./internal/adapters/telegram/` → FAIL (no markup, callback ignored).

**Step 3: Implement** in `bot.go`:

```go
// Replier is app.Telegram.
type Replier interface {
	Reply(ctx context.Context, telegramID int64, text string) app.BotReply
	Callback(ctx context.Context, telegramID int64, data string) app.BotReply
}
```

Handler:

```go
	handler := func(ctx context.Context, b *tg.Bot, u *models.Update) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), replyTimeout)
		defer cancel()
		if q := u.CallbackQuery; q != nil {
			// Stop the button's spinner; the answer comes as a message.
			if _, err := b.AnswerCallbackQuery(ctx, &tg.AnswerCallbackQueryParams{CallbackQueryID: q.ID}); err != nil {
				slog.ErrorContext(ctx, "telegram callback answer failed", slog.Any("err", err))
			}
			// ponytail: buttons are only sent in private chats, where the chat id is the user id.
			send(ctx, b, q.From.ID, safely(ctx, func() app.BotReply { return r.Callback(ctx, q.From.ID, q.Data) }))
			return
		}
		m := u.Message
		if m == nil || m.Text == "" || m.From == nil || m.Chat.Type != models.ChatTypePrivate {
			return // PRD-0003 non-goal: groups; non-text messages are ignored
		}
		send(ctx, b, m.Chat.ID, safely(ctx, func() app.BotReply { return r.Reply(ctx, m.From.ID, m.Text) }))
	}
```

Replace `reply` with:

```go
// safely calls f, turning a panic into failedReply: the library does not
// recover handler panics, so one bad update would otherwise kill the bot.
func safely(ctx context.Context, f func() app.BotReply) (out app.BotReply) {
	defer func() {
		if p := recover(); p != nil {
			slog.ErrorContext(ctx, "telegram reply panicked", slog.Any("panic", p), slog.String("stack", string(debug.Stack())))
			out = app.BotReply{Text: failedReply}
		}
	}()
	return f()
}

// send delivers rep, with its buttons as one inline keyboard row.
func send(ctx context.Context, b *tg.Bot, chatID int64, rep app.BotReply) {
	// No ParseMode: replies echo user-written titles, so they go out as plain text.
	p := &tg.SendMessageParams{ChatID: chatID, Text: rep.Text}
	if len(rep.Buttons) > 0 {
		row := make([]models.InlineKeyboardButton, len(rep.Buttons))
		for i, btn := range rep.Buttons {
			row[i] = models.InlineKeyboardButton{Text: btn.Label, CallbackData: btn.Data}
		}
		p.ReplyMarkup = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{row}}
	}
	if _, err := b.SendMessage(ctx, p); err != nil {
		slog.ErrorContext(ctx, "telegram send failed", slog.Int64("chat_id", chatID), slog.Any("err", err))
	}
}
```

Update the package doc: "it receives private text messages and button taps and sends back what the use case replies."

**Step 4: Run** `go build ./... && go test ./internal/adapters/telegram/ ./internal/app/` → PASS. `cmd/bragdoc/bot.go` needs no change (`*app.Telegram` satisfies `Replier`).

**Step 5: Commit**

```bash
git add backend/internal/adapters/telegram && git commit -m "feat(telegram): inline keyboards and button taps"
```

---

### Task 6: Web warning gets its "Close" button

**Files:**
- Modify: `frontend/packages/app/src/logs/LogFormDialog.tsx:134-155`
- Test: `frontend/packages/app/src/routes/DocumentLogs.test.tsx` ("Cmd+Enter creates; the dialog stays open…")

**Step 1: Write the failing test.** At the end of that test add (import `within` from `@testing-library/react` if missing):

```tsx
  await userEvent.click(
    within(screen.getByRole("status")).getByRole("button", { name: "Close" }),
  );
  await waitFor(() =>
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
  );
```

`within(status)` matters: the dialog's own X button is also named "Close".

**Step 2: Run** `npm test -- DocumentLogs` → FAIL (no Close in the warning).

**Step 3: Implement.** In `LogFormDialog.tsx`, wrap the "Add impact" button in a row and add Close, which calls the existing `onCancel`:

```tsx
          <div className="flex gap-2">
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => {
                setPreview(false);
                requestAnimationFrame(() => descRef.current?.focus());
              }}
            >
              Add impact
            </Button>
            <Button type="button" size="sm" variant="ghost" onClick={onCancel}>
              Close
            </Button>
          </div>
```

(drop `className="w-fit"` from "Add impact"; the flex row sizes it).

**Step 4: Run** `npm test && npm run typecheck && npm run lint && npm run fmt:check` → PASS.

**Step 5: Commit**

```bash
git add frontend/packages/app/src && git commit -m "feat(frontend): no-impact warning gets its Close button (PRD-0007 §9)"
```

---

### Task 7: Docs

**Files:**
- Modify: `docs/prd/0003-telegram-bot.md` (FR table, decisions log)
- Modify: `docs/prd/0007-impact-extraction.md` (decisions log)
- Modify: `docs/plans/2026-10-09-bot-impact-followup-design.md` (key name)
- Modify: `backend/internal/app/telegram.go` `msgHelp` only if it lists every reply (it does not; leave it)

**Step 1:** Add to the PRD-0003 FR table after FR-10:

```markdown
| FR-11 | When a created log states no impact (PRD-0007), the reply MUST offer "Add to description" and "Replace description"; after a tap, the user's next message updates the description and the impact is re-extracted. Valid for 5 minutes. | Must |
```

and to its decisions log:

```markdown
| 2026-10-09 | No-impact follow-up via inline buttons, then the next message | Keeps plain messages as new logs unless the user explicitly chose to answer |
```

**Step 2:** Add to the PRD-0007 decisions log:

```markdown
| 2026-10-09 | The bot asks too: inline buttons to add to or replace the description (PRD-0003 FR-11) | FR-4's warning, for logs written in Telegram |
```

**Step 3:** In the design doc's Flow step 2, replace `tg:pending:<telegramID>` with: "the `tg:impact:<telegramID>` entry, written when the warning is sent, gains the mode".

**Step 4: Commit**

```bash
git add docs && git commit -m "docs: bot no-impact follow-up in PRD-0003 FR-11 and PRD-0007"
```

---

### Task 8: Verify and open the PR

**Step 1:** Use @superpowers:verification-before-completion. From `backend/`: `go test ./... && make lint`. From `frontend/`: `npm test && npm run typecheck && npm run lint && npm run fmt:check`. All must pass; paste failures, don't paper over them.

**Step 2:** `git push -u origin feat/bot-impact-followup`

**Step 3:** `gh pr create --base main --title "feat: bot asks for the impact when a log states none" --body ...` with a summary (bot buttons → next message adds or replaces → re-extract; web Close button; docs) and a test plan, ending with:

```
🤖 Generated with [Claude Code](https://claude.com/claude-code)
```
