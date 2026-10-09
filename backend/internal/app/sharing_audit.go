package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// DocumentAudit lists a document's sharing history, newest first; owner only (FR-8).
func (s *Sharing) DocumentAudit(ctx context.Context, actor domain.User, docID string) ([]domain.AuditEntry, error) {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return nil, err
	}
	return s.repo.Audit(ctx, docID)
}

// TenantAudit lists every sharing change in the tenant; admins only (FR-8).
// Entries carry titles and emails, never document content (FR-9).
func (s *Sharing) TenantAudit(ctx context.Context, actor domain.User) ([]domain.AuditEntry, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.repo.Audit(ctx, "")
}
