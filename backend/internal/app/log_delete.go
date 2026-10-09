package app

import "context"

// Delete removes one log.
func (s *Logs) Delete(ctx context.Context, docID, id, userID string) error {
	if _, err := s.writable(ctx, docID, userID); err != nil {
		return err
	}
	err := s.logs.Delete(ctx, docID, id)
	if err == nil {
		s.touch(ctx, docID)
	}
	return err
}

// DeleteExamples removes the document's example logs in one call (PRD-0002 §9).
func (s *Logs) DeleteExamples(ctx context.Context, docID, userID string) error {
	if _, err := s.writable(ctx, docID, userID); err != nil {
		return err
	}
	err := s.logs.DeleteExamples(ctx, docID)
	if err == nil {
		s.touch(ctx, docID)
	}
	return err
}
