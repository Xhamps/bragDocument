package app

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/ports"
)

const (
	linkCodeTTL = 10 * time.Minute // PRD-0003 FR-1
	undoTTL     = 5 * time.Minute  // PRD-0003 FR-8
	// No 0/O/1/I/L: codes are read off a screen and typed.
	linkCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	linkCodeLen      = 8
)

const (
	msgNotLinked       = "I don't know you yet. In the web app open Settings → Telegram, generate a code, and send it here as /start CODE."
	msgCodeInvalid     = "That code is invalid or expired. Generate a new one in Settings → Telegram."
	msgLinkUnavailable = "Linking is temporarily unavailable. Try again in a minute."
	msgLinkedElsewhere = "This Telegram account is already linked to another account. Unlink it there first."
	msgTryAgain        = "Something went wrong and nothing was saved. Try again."
	msgPickDoc         = "Pick a document first: /docs, then /use <number>."
	msgNoAccess        = "You no longer have access to that document. Pick another with /docs."
	msgArchived        = "That document is archived. Pick another with /docs."
	msgNothingToUndo   = "Nothing to undo. /undo removes the last log I created, within 5 minutes."
	msgHelp            = `Send a message to log it: first line is the name, the rest the description.
#tag adds a tag, !low !medium !high !critical sets impact (default medium), links are kept.

Example:
Shipped SSO #auth !high https://github.com/acme/app/pull/12
Cut login support tickets by 40%.

/docs list documents · /use <n> pick one · /last last 5 · /undo remove the last one`
)

// Telegram is the bot's use cases (PRD-0003, ADR-0009) and the web side of
// linking. Reply returns the text to send back; the adapter only transports it.
type Telegram struct {
	links  ports.TelegramLinkRepo
	docs   ports.DocumentRepo
	logs   *Logs
	codes  ports.Cache
	undo   ports.Cache
	appURL string
	// scope puts the tenant in the context (telemetry.WithTenantID); injected
	// because app may not import telemetry.
	scope   func(ctx context.Context, tenantID string) context.Context
	now     func() time.Time
	newCode func() string
}

// NewTelegram wires the use cases. appURL builds the edit deep links; "" omits them.
// codes must be the raw cache (linking needs Redis, its errors must surface);
// undo should be Degrading (a miss only means nothing to undo).
func NewTelegram(links ports.TelegramLinkRepo, docs ports.DocumentRepo, logs *Logs, codes, undo ports.Cache,
	appURL string, scope func(context.Context, string) context.Context) *Telegram {
	return &Telegram{links: links, docs: docs, logs: logs, codes: codes, undo: undo, appURL: appURL,
		scope: scope, now: time.Now, newCode: randomCode}
}

func randomCode() string {
	b := make([]byte, linkCodeLen)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error (Go 1.24+)
	for i := range b {
		b[i] = linkCodeAlphabet[int(b[i])%len(linkCodeAlphabet)] // ponytail: modulo bias is negligible for a 10-minute single-use code
	}
	return string(b)
}

func codeKey(code string) string { return "tg:code:" + code }
