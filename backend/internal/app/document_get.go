package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Get returns one document with the caller's role and clears its "New" badge
// (PRD-0004 FR-6, in-app part).
func (s *Documents) Get(ctx context.Context, id, userID string) (domain.Document, error) {
	d, err := access(ctx, s.docs, id, userID, domain.PermRead)
	if err != nil {
		return domain.Document{}, err
	}
	if d.IsNew {
		if err := s.docs.MarkSeen(ctx, id, userID); err != nil {
			return domain.Document{}, err
		}
	}
	return d, nil
}
