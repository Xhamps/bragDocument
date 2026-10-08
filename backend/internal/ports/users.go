package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// UserRepo provisions callers. Provision runs fn inside one transaction that
// may read and write across tenants; only the sign-in path uses it.
type UserRepo interface {
	Provision(ctx context.Context, fn func(ctx context.Context, tx ProvisionTx) error) error
}

// ProvisionTx is what the sign-in use case can do inside Provision.
// GetUser, GetTenant, and FindInvitationByEmail return domain.ErrNotFound when no row matches.
type ProvisionTx interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
	GetTenant(ctx context.Context, id string) (domain.Tenant, error)
	FindInvitationByEmail(ctx context.Context, email string) (domain.Invitation, error)
	CreateTenant(ctx context.Context, name string) (domain.Tenant, error)
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
	DeleteInvitation(ctx context.Context, id string) error
}
