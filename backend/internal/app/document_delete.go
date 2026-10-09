package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Delete removes a document; owner only. Rows referencing the document cascade in the database.
func (s *Documents) Delete(ctx context.Context, id, userID string) error {
	if _, err := access(ctx, s.docs, id, userID, domain.PermDelete); err != nil {
		return err
	}
	return s.docs.Delete(ctx, id, entry(ctx, userID, domain.AuditDocumentDeleted, id))
}
