// Package telegram is the Telegram Bot API transport (ADR-0009): it receives
// private text messages and button taps and sends back what the use case replies.
package telegram

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/xhamps/bragdocument/backend/internal/app"
)

// replyTimeout bounds one reply (DB, LLM, send). It is detached from the polling
// context so a reply in flight at shutdown still finishes.
const replyTimeout = 10 * time.Second

// failedReply is sent when Reply or Callback panics.
const failedReply = "Something went wrong and nothing was saved. Try again."

// Replier is app.Telegram.
type Replier interface {
	Reply(ctx context.Context, telegramID int64, text string) app.BotReply
	Callback(ctx context.Context, telegramID int64, data string) app.BotReply
}

// Bot polls Telegram and answers private text messages.
type Bot struct{ b *tg.Bot }

// Option configures New; WithServerURL points at a fake Bot API in tests.
type Option = tg.Option

// WithServerURL overrides the Bot API URL.
func WithServerURL(u string) Option { return tg.WithServerURL(u) }

// New builds the bot. It skips the getMe call so a Telegram outage never blocks startup.
// ctx carries log attributes (service) for errors the library reports without one.
func New(ctx context.Context, token string, r Replier, opts ...Option) (*Bot, error) {
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
	b, err := tg.New(token, append([]tg.Option{
		tg.WithSkipGetMe(),
		tg.WithDefaultHandler(handler),
		// One worker, handlers inline: updates are handled in order, so /undo
		// never races a capture still waiting on the LLM.
		tg.WithNotAsyncHandlers(),
		tg.WithErrorsHandler(func(err error) { slog.ErrorContext(ctx, "telegram polling failed", slog.Any("err", err)) }),
	}, opts...)...)
	if err != nil {
		return nil, err
	}
	return &Bot{b: b}, nil
}

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

// Run long-polls until ctx is done (ADR-0009: polling; webhook later).
func (b *Bot) Run(ctx context.Context) { b.b.Start(ctx) }
