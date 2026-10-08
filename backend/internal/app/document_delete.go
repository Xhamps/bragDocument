package app

import "context"

// Delete removes a document the caller owns. Rows referencing the document cascade in the database.
func (s *Documents) Delete(ctx context.Context, id, userID string) error {
	if _, err := ownedDocument(ctx, s.docs, id, userID); err != nil {
		return err
	}
	return s.docs.Delete(ctx, id)
}
