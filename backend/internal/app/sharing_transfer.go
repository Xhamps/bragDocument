package app

import (
	"context"
	"errors"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Transfer hands ownership to another tenant member (FR-2). The previous
// owner keeps an editor grant; the new owner's grant, if any, goes away.
func (s *Sharing) Transfer(ctx context.Context, actor domain.User, docID, toUserID string) error {
	d, err := access(ctx, s.docs, docID, actor.ID, domain.PermTransfer)
	if err != nil {
		return err
	}
	if toUserID == actor.ID {
		return domain.NewValidationError(map[string]string{"user_id": "you already own this document"})
	}
	u, err := s.repo.Member(ctx, toUserID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.NewValidationError(map[string]string{"user_id": "not a member of this tenant"})
	}
	if err != nil {
		return err
	}
	return s.repo.Transfer(ctx, d.ID, actor.ID, u.ID, auditBy(actor, domain.AuditTransfer, d.ID, u.Email, domain.RoleOwner))
}
