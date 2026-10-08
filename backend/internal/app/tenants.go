package app

import (
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// Tenants groups the membership use cases; admin only.
type Tenants struct{ repo ports.TenantRepo }

// NewTenants wires the use cases.
func NewTenants(repo ports.TenantRepo) *Tenants { return &Tenants{repo: repo} }

func requireAdmin(actor domain.User) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	return nil
}
