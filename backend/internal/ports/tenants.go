package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TenantRepo manages membership of the tenant in the context.
type TenantRepo interface {
	ListMembers(ctx context.Context) ([]domain.User, error)
	// DeleteMember returns domain.ErrConflict when the member still owns documents.
	DeleteMember(ctx context.Context, id string) error
	ListInvitations(ctx context.Context) ([]domain.Invitation, error)
	// CreateInvitation returns domain.ErrConflict when the email is already invited.
	CreateInvitation(ctx context.Context, inv domain.Invitation) (domain.Invitation, error)
	DeleteInvitation(ctx context.Context, id string) error
}
