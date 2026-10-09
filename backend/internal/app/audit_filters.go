package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Filters returns picker options from the entries the actor may read.
func (s *Audit) Filters(ctx context.Context, actor domain.User) ([]domain.AuditActor, []domain.AuditDocument, error) {
	owner, err := s.scope(ctx, actor)
	if err != nil {
		return nil, nil, err
	}
	return s.repo.Filters(ctx, owner)
}
