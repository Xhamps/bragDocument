package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ChangeRole switches a grant between editor and viewer (FR-5); owner only.
func (s *Sharing) ChangeRole(ctx context.Context, actor domain.User, docID, userID string, role domain.Role) error {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return err
	}
	if err := grantableRole(role); err != nil {
		return err
	}
	g, err := s.grantOf(ctx, docID, userID)
	if err != nil {
		return err
	}
	if g.Role == role {
		return nil
	}
	return s.repo.SetRole(ctx, docID, userID, role, auditBy(actor, domain.AuditRoleChange, docID, g.Email, role))
}
