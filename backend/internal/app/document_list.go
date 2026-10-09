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

// List returns the caller's own documents and those shared with them (PRD-0001 FR-9).
func (s *Documents) List(ctx context.Context, userID string) (DocumentListOutput, error) {
	owned, err := s.docs.ListByOwner(ctx, userID)
	if err != nil {
		return DocumentListOutput{}, err
	}
	for i := range owned {
		owned[i].Role = domain.RoleOwner
	}
	shared, err := s.docs.ListShared(ctx, userID)
	if err != nil {
		return DocumentListOutput{}, err
	}
	return DocumentListOutput{Owned: owned, Shared: shared}, nil
}
