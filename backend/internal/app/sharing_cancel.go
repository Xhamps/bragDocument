package app

import (
	"context"
	"slices"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// CancelInvitation withdraws a pending document invitation; owner only.
func (s *Sharing) CancelInvitation(ctx context.Context, actor domain.User, docID, invID string) error {
	if _, err := access(ctx, s.docs, docID, actor.ID, domain.PermShare); err != nil {
		return err
	}
	sh, err := s.repo.Get(ctx, docID)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(sh.Invitations, func(inv domain.DocumentInvitation) bool { return inv.ID == invID })
	if i < 0 {
		return domain.ErrNotFound
	}
	inv := sh.Invitations[i]
	return s.repo.CancelInvitation(ctx, docID, invID, auditBy(ctx, actor, domain.AuditInviteCancel, docID, domain.TargetInvitation, invID, inv.Email, inv.Role))
}
