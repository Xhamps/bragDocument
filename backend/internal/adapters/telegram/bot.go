// Package telegram is the Telegram Bot API transport (ADR-0009): it receives
// private text messages and sends back what the use case replies.
package telegram

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	tg "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// replyTimeout bounds one reply (DB, LLM, send). It is detached from the polling
// context so a reply in flight at shutdown still finishes.
const replyTimeout = 10 * time.Second

// failedReply is sent when Reply panics.
const failedReply = "Something went wrong and nothing was saved. Try again."

// Replier is app.Telegram.Reply.
type Replier interface {
	Reply(ctx context.Context, telegramID int64, text string) string
}

// Bot polls Telegram and answers private text messages.
type Bot struct{ b *tg.Bot }

// Option configures New; WithServerURL points at a fake Bot API in tests.
type Option = tg.Option

// WithServerURL overrides the Bot API URL.
func WithServerURL(u string) Option { return tg.WithServerURL(u) }

// New builds the bot. It skips the getMe call so a Telegram outage never blocks startup.
func New(token string, r Replier, opts ...Option) (*Bot, error) {
	handler := func(ctx context.Context, b *tg.Bot, u *models.Update) {
		m := u.Message
		if m == nil || m.Text == "" || m.From == nil || m.Chat.Type != models.ChatTypePrivate {
			return // PRD-0003 non-goal: groups; non-text messages are ignored
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), replyTimeout)
		defer cancel()
		text := reply(ctx, r, m)
		// No ParseMode: replies echo user-written titles, so they go out as plain text.
		if _, err := b.SendMessage(ctx, &tg.SendMessageParams{ChatID: m.Chat.ID, Text: text}); err != nil {
			slog.ErrorContext(ctx, "telegram send failed", slog.Int64("chat_id", m.Chat.ID), slog.Any("err", err))
		}
	}
	b, err := tg.New(token, append([]tg.Option{
		tg.WithSkipGetMe(),
		tg.WithDefaultHandler(handler),
		// One worker, handlers inline: updates are handled in order, so /undo
		// never races a capture still waiting on the LLM.
		tg.WithNotAsyncHandlers(),
		tg.WithErrorsHandler(func(err error) { slog.Error("telegram polling failed", slog.Any("err", err)) }),
	}, opts...)...)
	if err != nil {
		return nil, err
	}
	return &Bot{b: b}, nil
}

// reply calls r, turning a panic into failedReply: the library does not
// recover handler panics, so one bad update would otherwise kill the bot.
func reply(ctx context.Context, r Replier, m *models.Message) (text string) {
	defer func() {
		if p := recover(); p != nil {
			slog.ErrorContext(ctx, "telegram reply panicked", slog.Any("panic", p), slog.String("stack", string(debug.Stack())))
			text = failedReply
		}
	}()
	return r.Reply(ctx, m.From.ID, m.Text)
}

// Run long-polls until ctx is done (ADR-0009: polling; webhook later).
func (b *Bot) Run(ctx context.Context) { b.b.Start(ctx) }
