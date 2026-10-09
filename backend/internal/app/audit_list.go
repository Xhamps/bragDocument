package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// List returns one page of the entries the actor may read, newest first (FR-8, FR-11).
// A document filter needs the owner's PermShare for non-admins: a grant without
// it is 403, no access at all is 404 (FR-7).
func (s *Audit) List(ctx context.Context, actor domain.User, f domain.AuditFilter) (domain.AuditPage, error) {
	if !actor.IsAdmin() && f.DocumentID != "" {
		if _, err := access(ctx, s.docs, f.DocumentID, actor.ID, domain.PermShare); err != nil {
			return domain.AuditPage{}, err
		}
	}
	owner, err := s.scope(ctx, actor)
	if err != nil {
		return domain.AuditPage{}, err
	}
	f.OwnerID = owner
	if err := f.Validate(); err != nil {
		return domain.AuditPage{}, err
	}
	return s.repo.List(ctx, f)
}
