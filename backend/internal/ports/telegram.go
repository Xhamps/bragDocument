package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TelegramLinkRepo stores Telegram links (PRD-0003).
type TelegramLinkRepo interface {
	// FindByTelegramID looks across tenants: the bot does not know the tenant yet.
	// Returns domain.ErrNotFound for an unlinked account.
	FindByTelegramID(ctx context.Context, telegramUserID int64) (domain.TelegramLink, error)
	// Get returns the caller's link in the context's tenant, or domain.ErrNotFound.
	Get(ctx context.Context, userID string) (domain.TelegramLink, error)
	// Link upserts the user's link in l.TenantID and writes a. domain.ErrConflict when the
	// Telegram account is linked to another user.
	Link(ctx context.Context, l domain.TelegramLink, a domain.AuditEntry) error
	SetDocument(ctx context.Context, userID, documentID string) error
	// Delete is idempotent; a is written only when a link was removed.
	Delete(ctx context.Context, userID string, a domain.AuditEntry) error
}
