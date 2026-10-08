package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentRepo stores documents of the tenant in the context.
type DocumentRepo interface {
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Document, error)
	Get(ctx context.Context, id string) (domain.Document, error)
	Create(ctx context.Context, d domain.Document) (domain.Document, error)
	Update(ctx context.Context, d domain.Document) (domain.Document, error)
	Delete(ctx context.Context, id string) error
}
