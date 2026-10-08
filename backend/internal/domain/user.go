package domain

import "time"

// Tenant roles. The first user of a tenant is its admin.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// User mirrors the identity provider's subject and belongs to exactly one tenant.
type User struct {
	ID          string // Supabase "sub"
	TenantID    string
	Email       string
	DisplayName string
	Role        string
	CreatedAt   time.Time
}

// IsAdmin reports whether the user manages tenant membership.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }
