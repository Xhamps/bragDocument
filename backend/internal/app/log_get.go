package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Get returns one log. Archived documents are readable.
func (s *Logs) Get(ctx context.Context, docID, id, userID string) (domain.Log, error) {
	if _, err := ownedDocument(ctx, s.docs, docID, userID); err != nil {
		return domain.Log{}, err
	}
	return s.logs.Get(ctx, docID, id)
}
