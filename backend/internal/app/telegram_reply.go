package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func undoKey(telegramID int64) string { return "tg:undo:" + strconv.FormatInt(telegramID, 10) }

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

// splitCommand returns "/cmd" (bot-name suffix dropped, lowercased) and its
// argument, or "" and the text for a plain message.
func splitCommand(text string) (cmd, arg string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	cmd, arg = text, ""
	if i := strings.IndexFunc(text, unicode.IsSpace); i >= 0 {
		cmd, arg = text[:i], strings.TrimSpace(text[i:])
	}
	cmd, _, _ = strings.Cut(cmd, "@")
	return strings.ToLower(cmd), arg
}

// activeDocs is the active documents the user owns or edits, in web list order; /use indexes into it.
func (t *Telegram) activeDocs(ctx context.Context, userID string) ([]domain.Document, error) {
	return t.docs.ListWritable(ctx, userID)
}

func (t *Telegram) listDocs(ctx context.Context, link domain.TelegramLink) string {
	docs, err := t.activeDocs(ctx, link.UserID)
	if err != nil {
		return t.failed(ctx, err)
	}
	if len(docs) == 0 {
		return "You have no active documents. Create one in the web app."
	}
	var b strings.Builder
	for i, d := range docs {
		mark := ""
		if d.ID == link.DocumentID {
			mark = " ✓"
		}
		fmt.Fprintf(&b, "%d. %s%s\n", i+1, d.Title, mark)
	}
	b.WriteString("\nPick one with /use <number>.")
	return b.String()
}

func (t *Telegram) use(ctx context.Context, link domain.TelegramLink, arg string) string {
	docs, err := t.activeDocs(ctx, link.UserID)
	if err != nil {
		return t.failed(ctx, err)
	}
	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 || n > len(docs) {
		return "Send /use with a number from /docs."
	}
	d := docs[n-1]
	if err := t.links.SetDocument(ctx, link.UserID, d.ID); err != nil {
		return t.failed(ctx, err)
	}
	return fmt.Sprintf("Now writing to “%s”.", d.Title)
}

type undoEntry struct {
	UserID     string `json:"user_id"` // the Telegram id may be relinked to another user within the TTL
	DocumentID string `json:"document_id"`
	LogID      string `json:"log_id"`
	Name       string `json:"name"`
}

func (t *Telegram) capture(ctx context.Context, link domain.TelegramLink, telegramID int64, text string) string {
	if link.DocumentID == "" {
		return msgPickDoc
	}
	d, err := domain.ParseLogMessage(text)
	if err != nil {
		return t.failed(ctx, err)
	}
	l, err := t.logs.Create(ctx, CreateLogInput{DocumentID: link.DocumentID, UserID: link.UserID,
		Name: d.Name, Description: d.Description, Impact: d.Impact, Tags: d.Tags, Links: d.Links})
	if err != nil {
		return t.failed(ctx, err)
	}
	v, _ := json.Marshal(undoEntry{UserID: link.UserID, DocumentID: l.DocumentID, LogID: l.ID, Name: l.Name})
	_ = t.undo.Set(ctx, undoKey(telegramID), v, undoTTL) // Degrading: never fails

	var b strings.Builder
	fmt.Fprintf(&b, "Logged: %s\nImpact: %s", l.Name, l.Impact)
	if len(l.Tags) > 0 {
		b.WriteString("\nTags: #" + strings.Join(l.Tags, " #"))
	}
	for _, k := range l.Links {
		b.WriteString("\nLink: " + k.URL)
	}
	if l.ImpactStatement != nil && *l.ImpactStatement == "" {
		b.WriteString("\n\nNo impact stated. What changed because of this?")
	}
	if t.appURL != "" {
		fmt.Fprintf(&b, "\nEdit: %s/documents/%s?edit=%s", t.appURL, l.DocumentID, l.ID)
	}
	b.WriteString("\n/undo to remove it.")
	return b.String()
}

func (t *Telegram) last(ctx context.Context, link domain.TelegramLink) string {
	if link.DocumentID == "" {
		return msgPickDoc
	}
	page, err := t.logs.List(ctx, link.DocumentID, link.UserID, domain.LogFilter{PerPage: 5})
	if err != nil {
		return t.failed(ctx, err)
	}
	if len(page.Items) == 0 {
		return "No logs yet. Send a message to create one."
	}
	var b strings.Builder
	for _, l := range page.Items {
		fmt.Fprintf(&b, "• %s (%s, %s)\n", l.Name, l.Impact, l.CreatedAt.Format("Jan 2"))
	}
	return strings.TrimSpace(b.String())
}

func (t *Telegram) undoLast(ctx context.Context, link domain.TelegramLink, telegramID int64) string {
	v, ok, _ := t.undo.Get(ctx, undoKey(telegramID))
	var e undoEntry
	if !ok || json.Unmarshal(v, &e) != nil || e.UserID != link.UserID {
		return msgNothingToUndo
	}
	err := t.logs.Delete(ctx, e.DocumentID, e.LogID, link.UserID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return t.failed(ctx, err)
	}
	_ = t.undo.Delete(ctx, undoKey(telegramID))
	if err != nil {
		return msgNothingToUndo // already deleted elsewhere
	}
	return "Removed: " + e.Name
}

// failed turns a use-case error into a reply (FR-10, NFR-3).
func (t *Telegram) failed(ctx context.Context, err error) string {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		keys := slices.Sorted(maps.Keys(ve.Fields))
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+": "+ve.Fields[k])
		}
		return "Not saved. " + strings.Join(parts, "; ")
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrNotFound):
		return msgNoAccess
	case errors.Is(err, domain.ErrConflict):
		return msgArchived
	}
	slog.ErrorContext(ctx, "telegram command failed", slog.Any("err", err))
	return msgTryAgain
}
