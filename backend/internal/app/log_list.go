package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// List returns one page of the document's logs. Archived documents are readable.
func (s *Logs) List(ctx context.Context, docID, userID string, f domain.LogFilter) (domain.LogPage, error) {
	if _, err := ownedDocument(ctx, s.docs, docID, userID); err != nil {
		return domain.LogPage{}, err
	}
	if err := f.Validate(); err != nil {
		return domain.LogPage{}, err
	}
	return s.logs.List(ctx, docID, f)
}
