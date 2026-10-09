package app

import (
	"context"
	"errors"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Share outcomes.
const (
	ShareGranted = "grant"      // a member got access at once
	ShareInvited = "invitation" // held until the email signs in (FR-4)
)

// ShareInput: the owner shares DocumentID with Email as Role.
type ShareInput struct {
	Actor      domain.User
	DocumentID string
	Email      string
	Role       domain.Role
}

// Share grants a tenant member a role at once, or holds an invitation for an
// email that is not a user yet (FR-3, FR-4). Either way the recipient is
// emailed; a failed email never fails the share (FR-6).
func (s *Sharing) Share(ctx context.Context, in ShareInput) (string, error) {
	d, err := access(ctx, s.docs, in.DocumentID, in.Actor.ID, domain.PermShare)
	if err != nil {
		return "", err
	}
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return "", err
	}
	if err := grantableRole(in.Role); err != nil {
		return "", err
	}
	if email == in.Actor.Email {
		return "", domain.NewValidationError(map[string]string{"email": "you already own this document"})
	}

	kind, link := ShareGranted, s.appURL+"/documents/"+d.ID
	u, err := s.repo.MemberByEmail(ctx, email)
	switch {
	case err == nil:
		g := domain.Grant{DocumentID: d.ID, UserID: u.ID, Role: in.Role, GrantedBy: in.Actor.ID}
		err = s.repo.Grant(ctx, g, auditBy(ctx, in.Actor, domain.AuditGrant, d.ID, domain.TargetUser, u.ID, email, in.Role))
	case errors.Is(err, domain.ErrNotFound):
		kind, link = ShareInvited, s.appURL+"/sign-in"
		inv := domain.DocumentInvitation{DocumentID: d.ID, Email: email, Role: in.Role, InvitedBy: in.Actor.ID}
		_, err = s.repo.Invite(ctx, inv, auditBy(ctx, in.Actor, domain.AuditInvite, d.ID, domain.TargetInvitation, "", email, in.Role))
	}
	if err != nil {
		return "", err
	}
	s.notify(ctx, in.Actor, d, email, in.Role, link)
	return kind, nil
}
