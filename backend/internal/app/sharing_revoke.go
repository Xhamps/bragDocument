package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Revoke removes a grant; the next request of that user is denied (FR-5).
func (s *Sharing) Revoke(ctx context.Context, actor domain.User, docID, userID string) error {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return err
	}
	g, err := s.grantOf(ctx, docID, userID)
	if err != nil {
		return err
	}
	return s.repo.Revoke(ctx, docID, userID, auditBy(ctx, actor, domain.AuditRevoke, docID, domain.TargetUser, userID, g.Email, g.Role))
}
