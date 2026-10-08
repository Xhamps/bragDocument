package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ListInvitations returns pending invitations of the actor's tenant.
func (s *Tenants) ListInvitations(ctx context.Context, actor domain.User) ([]domain.Invitation, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.repo.ListInvitations(ctx)
}
