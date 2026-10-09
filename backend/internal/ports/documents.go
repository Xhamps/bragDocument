package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentRepo stores documents of the tenant in the context.
type DocumentRepo interface {
	// ListByOwner sets LogCount and LastLogAt.
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Document, error)
	// ListShared returns documents granted to the user, with Role, OwnerName,
	// IsNew, LogCount, and LastLogAt.
	ListShared(ctx context.Context, userID string) ([]domain.Document, error)
	// ListWritable returns active documents the user owns or edits, most recently updated first.
	ListWritable(ctx context.Context, userID string) ([]domain.Document, error)
	// GetForUser sets Role, OwnerName, and IsNew. It returns domain.ErrNotFound
	// when no row matches or the user has no role on the document.
	GetForUser(ctx context.Context, id, userID string) (domain.Document, error)
	// MarkSeen clears the user's "New" badge on a shared document; a no-op otherwise.
	MarkSeen(ctx context.Context, id, userID string) error
	// Create stores the document, its starting example logs, and a in one
	// transaction; a.DocumentID is set to the new id.
	Create(ctx context.Context, d domain.Document, examples []domain.Log, a domain.AuditEntry) (domain.Document, error)
	// Update and Delete write a in the same transaction (PRD-0009 FR-1).
	Update(ctx context.Context, d domain.Document, a domain.AuditEntry) (domain.Document, error)
	Delete(ctx context.Context, id string, a domain.AuditEntry) error
}
