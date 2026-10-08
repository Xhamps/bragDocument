package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Uninvite withdraws a pending invitation.
func (s *Tenants) Uninvite(ctx context.Context, actor domain.User, id string) error {
	if err := requireAdmin(actor); err != nil {
		return err
	}
	return s.repo.DeleteInvitation(ctx, id)
}
