package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// SharingRepo changes grants and document invitations of the tenant in the
// context. Every write stores its audit entry in the same transaction.
type SharingRepo interface {
	Get(ctx context.Context, docID string) (domain.Sharing, error)
	// MemberByEmail and Member return domain.ErrNotFound for anyone outside the tenant.
	MemberByEmail(ctx context.Context, email string) (domain.User, error)
	Member(ctx context.Context, id string) (domain.User, error)
	// Grant returns domain.ErrConflict when the user already has a grant.
	Grant(ctx context.Context, g domain.Grant, a domain.AuditEntry) error
	// Invite returns domain.ErrConflict when the email is already invited to the document.
	Invite(ctx context.Context, inv domain.DocumentInvitation, a domain.AuditEntry) (domain.DocumentInvitation, error)
	// SetRole, Revoke, and CancelInvitation return domain.ErrNotFound when nothing matched.
	SetRole(ctx context.Context, docID, userID string, role domain.Role, a domain.AuditEntry) error
	Revoke(ctx context.Context, docID, userID string, a domain.AuditEntry) error
	CancelInvitation(ctx context.Context, docID, invID string, a domain.AuditEntry) error
	// Transfer makes toUserID the owner, drops their grant, and makes fromUserID an editor.
	Transfer(ctx context.Context, docID, fromUserID, toUserID string, a domain.AuditEntry) error
	// Audit lists entries newest first: of one document, or of the tenant when docID is "".
	// It returns at most 200 entries per document and 500 per tenant.
	Audit(ctx context.Context, docID string) ([]domain.AuditEntry, error)
}

// Mailer sends one transactional email (ADR-0014). A disabled mailer returns
// domain.ErrUnavailable; callers never fail an action because of it.
type Mailer interface {
	Send(ctx context.Context, to, subject, html string) error
}
