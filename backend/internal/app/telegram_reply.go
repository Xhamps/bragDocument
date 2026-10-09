package app

import (
	"context"
	"errors"
	"strings"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Reply handles one private message from telegramID and returns the answer.
func (t *Telegram) Reply(ctx context.Context, telegramID int64, text string) string {
	cmd, arg := splitCommand(text)
	if cmd == "/start" && arg != "" {
		return t.start(ctx, telegramID, arg)
	}
	if _, err := t.links.FindByTelegramID(ctx, telegramID); errors.Is(err, domain.ErrNotFound) {
		return msgNotLinked
	}
	return msgHelp
}

// splitCommand returns "/cmd" (bot-name suffix dropped, lowercased) and its
// argument, or "" and the text for a plain message.
func splitCommand(text string) (cmd, arg string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	cmd, arg, _ = strings.Cut(text, " ")
	cmd, _, _ = strings.Cut(cmd, "@")
	return strings.ToLower(cmd), strings.TrimSpace(arg)
}
