package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// maxNameLen caps display names and derived tenant names; longer values are truncated, not rejected.
const maxNameLen = 200

// EnsureUserInput is what the verified token tells us about the caller.
type EnsureUserInput struct {
	ID          string // token "sub"
	Email       string
	DisplayName string
}

// Principal is the authenticated caller and their tenant.
type Principal struct {
	User   domain.User
	Tenant domain.Tenant
}

// UserEnsure loads the caller or provisions them: into the tenant that invited
// their email, or into a brand-new tenant as its admin.
type UserEnsure struct{ users ports.UserRepo }

// NewUserEnsure wires the use case.
func NewUserEnsure(users ports.UserRepo) *UserEnsure { return &UserEnsure{users: users} }

// Execute runs on every authenticated request; the common path is one
// transaction with two lookups (user, tenant).
func (uc *UserEnsure) Execute(ctx context.Context, in EnsureUserInput) (Principal, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return Principal{}, err
	}
	p, err := uc.provision(ctx, in, email)
	if errors.Is(err, domain.ErrConflict) {
		// A concurrent first sign-in won the users.id insert; the second pass finds the row.
		p, err = uc.provision(ctx, in, email)
	}
	if err != nil {
		return Principal{}, fmt.Errorf("provision user %s: %w", in.ID, err)
	}
	return p, nil
}

func (uc *UserEnsure) provision(ctx context.Context, in EnsureUserInput, email string) (Principal, error) {
	var p Principal
	err := uc.users.Provision(ctx, func(ctx context.Context, tx ports.ProvisionTx) error {
		u, err := tx.GetUser(ctx, in.ID)
		if err == nil {
			t, err := tx.GetTenant(ctx, u.TenantID)
			if err != nil {
				return fmt.Errorf("get tenant %s: %w", u.TenantID, err)
			}
			p = Principal{User: u, Tenant: t}
			return nil
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		user := domain.User{ID: in.ID, Email: email, DisplayName: truncate(strings.TrimSpace(in.DisplayName)), Role: domain.RoleMember}
		inv, err := tx.FindInvitationByEmail(ctx, email)
		switch {
		case err == nil:
			user.TenantID = inv.TenantID
			if p.Tenant, err = tx.GetTenant(ctx, inv.TenantID); err != nil {
				return fmt.Errorf("get tenant %s: %w", inv.TenantID, err)
			}
			// Insert the user before consuming the invitation: a concurrent loser then
			// hits the users pkey conflict (retried) rather than a missing invitation.
			if p.User, err = tx.CreateUser(ctx, user); err != nil {
				return err
			}
			if err := tx.DeleteInvitation(ctx, inv.ID); err != nil {
				return fmt.Errorf("delete invitation %s: %w", inv.ID, err)
			}
			return nil
		case errors.Is(err, domain.ErrNotFound):
			if p.Tenant, err = tx.CreateTenant(ctx, tenantName(user.DisplayName, email)); err != nil {
				return err
			}
			user.TenantID = p.Tenant.ID
			user.Role = domain.RoleAdmin
			p.User, err = tx.CreateUser(ctx, user)
			return err
		default:
			return err
		}
	})
	return p, err
}

func tenantName(displayName, email string) string {
	if displayName != "" {
		return displayName
	}
	local, _, _ := strings.Cut(email, "@")
	return truncate(local)
}

// truncate keeps the first maxNameLen runes.
func truncate(s string) string {
	if utf8.RuneCountInString(s) <= maxNameLen {
		return s
	}
	r := []rune(s)
	return string(r[:maxNameLen])
}
