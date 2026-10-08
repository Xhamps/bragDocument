package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// RemoveMember deletes a member. Admins cannot remove themselves; the
// repository refuses members who still own documents (ErrConflict).
func (s *Tenants) RemoveMember(ctx context.Context, actor domain.User, id string) error {
	if err := requireAdmin(actor); err != nil {
		return err
	}
	if id == actor.ID {
		return domain.NewValidationError(map[string]string{"id": "cannot remove yourself"})
	}
	return s.repo.DeleteMember(ctx, id)
}
