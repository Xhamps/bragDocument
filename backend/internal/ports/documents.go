package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentRepo stores documents of the tenant in the context.
type DocumentRepo interface {
	// ListByOwner sets LogCount and LastLogAt.
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Document, error)
	// Get returns domain.ErrNotFound when no row matches.
	Get(ctx context.Context, id string) (domain.Document, error)
	// Create stores the document and its starting example logs in one transaction.
	Create(ctx context.Context, d domain.Document, examples []domain.Log) (domain.Document, error)
	Update(ctx context.Context, d domain.Document) (domain.Document, error)
	Delete(ctx context.Context, id string) error
}
