package app

import "context"

// Delete removes a document the caller owns. Logs and grants cascade in the database.
func (s *Documents) Delete(ctx context.Context, id, userID string) error {
	if _, err := s.owned(ctx, id, userID); err != nil {
		return err
	}
	return s.docs.Delete(ctx, id)
}
