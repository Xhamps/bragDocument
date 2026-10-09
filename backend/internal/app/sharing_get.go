package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Get returns the document's grants and pending invitations; owner only.
func (s *Sharing) Get(ctx context.Context, actor domain.User, docID string) (domain.Sharing, error) {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return domain.Sharing{}, err
	}
	return s.repo.Get(ctx, docID)
}
