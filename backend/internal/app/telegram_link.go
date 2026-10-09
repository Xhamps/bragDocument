package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// LinkCode is shown in Settings; the user sends "/start <Code>" to the bot.
type LinkCode struct {
	Code      string
	ExpiresAt time.Time
}

type codeOwner struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
}

// NewCode issues a one-time link code (FR-1). Redis down → domain.ErrUnavailable.
func (t *Telegram) NewCode(ctx context.Context, userID, tenantID string) (LinkCode, error) {
	code := t.newCode()
	v, _ := json.Marshal(codeOwner{UserID: userID, TenantID: tenantID})
	if err := t.codes.Set(ctx, codeKey(code), v, linkCodeTTL); err != nil {
		return LinkCode{}, fmt.Errorf("%w: link code: %v", domain.ErrUnavailable, err)
	}
	return LinkCode{Code: code, ExpiresAt: t.now().Add(linkCodeTTL)}, nil
}

// Status returns the caller's link; ok is false when not linked.
func (t *Telegram) Status(ctx context.Context, userID string) (domain.TelegramLink, bool, error) {
	l, err := t.links.Get(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.TelegramLink{}, false, nil
	}
	return l, err == nil, err
}

// Unlink removes the caller's link (FR-2). Idempotent.
func (t *Telegram) Unlink(ctx context.Context, userID string) error {
	return t.links.Delete(ctx, userID, entry(ctx, userID, domain.AuditTelegramUnlinked, ""))
}

// start redeems a link code sent as "/start <code>".
func (t *Telegram) start(ctx context.Context, telegramID int64, code string) string {
	v, ok, err := t.codes.GetDel(ctx, codeKey(normalizeCode(code)))
	if err != nil {
		slog.WarnContext(ctx, "telegram link code lookup failed", slog.Any("err", err))
		return msgLinkUnavailable
	}
	var o codeOwner
	if !ok || json.Unmarshal(v, &o) != nil {
		return msgCodeInvalid
	}
	err = t.links.Link(ctx, domain.TelegramLink{UserID: o.UserID, TenantID: o.TenantID, TelegramUserID: telegramID, LinkedAt: t.now()},
		entry(ctx, o.UserID, domain.AuditTelegramLinked, ""))
	switch {
	case errors.Is(err, domain.ErrConflict):
		return msgLinkedElsewhere
	case err != nil:
		slog.ErrorContext(ctx, "telegram link failed", slog.Any("err", err))
		return msgTryAgain
	}
	return "Linked. Send /docs to pick the document I write to."
}

// normalizeCode accepts the code as typed: any case, with spaces or dashes.
func normalizeCode(code string) string {
	return strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(code))
}
