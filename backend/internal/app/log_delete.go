package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Delete removes one log.
func (s *Logs) Delete(ctx context.Context, docID, id, userID string) error {
	if _, err := s.writable(ctx, docID, userID); err != nil {
		return err
	}
	l, err := s.logs.Get(ctx, docID, id) // the entry names the log (FR-12)
	if err != nil {
		return err
	}
	a := entry(ctx, userID, domain.AuditLogDeleted, docID)
	a.TargetType, a.TargetID, a.Target = domain.TargetLog, id, l.Name
	err = s.logs.Delete(ctx, docID, id, a)
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
	a := entry(ctx, userID, domain.AuditLogDeleted, docID)
	a.TargetType, a.Target = domain.TargetLog, "example logs"
	err := s.logs.DeleteExamples(ctx, docID, a)
	if err == nil {
		s.touch(ctx, docID)
	}
	return err
}
