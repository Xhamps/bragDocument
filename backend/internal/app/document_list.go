package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentListOutput separates owned documents from documents shared with the caller.
type DocumentListOutput struct {
	Owned  []domain.Document
	Shared []domain.Document
}

// List returns the caller's documents. Shared is always empty until PRD-0004
// adds grants; the shape is fixed now so the client does not change later.
func (s *Documents) List(ctx context.Context, ownerID string) (DocumentListOutput, error) {
	owned, err := s.docs.ListByOwner(ctx, ownerID)
	if err != nil {
		return DocumentListOutput{}, err
	}
	return DocumentListOutput{Owned: owned, Shared: []domain.Document{}}, nil
}
