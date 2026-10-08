package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ListMembers returns every user of the actor's tenant.
func (s *Tenants) ListMembers(ctx context.Context, actor domain.User) ([]domain.User, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx)
}
