package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Invite records that an email may join the actor's tenant on first sign-in.
// No email is sent (PRD-0004 FR-6). Existing members conflict.
func (s *Tenants) Invite(ctx context.Context, actor domain.User, email string) (domain.Invitation, error) {
	if err := requireAdmin(actor); err != nil {
		return domain.Invitation{}, err
	}
	email, err := domain.NormalizeEmail(email)
	if err != nil {
		return domain.Invitation{}, err
	}
	members, err := s.repo.ListMembers(ctx)
	if err != nil {
		return domain.Invitation{}, err
	}
	for _, m := range members {
		if m.Email == email {
			return domain.Invitation{}, domain.ErrConflict
		}
	}
	return s.repo.CreateInvitation(ctx, domain.Invitation{TenantID: actor.TenantID, Email: email, CreatedBy: actor.ID})
}
