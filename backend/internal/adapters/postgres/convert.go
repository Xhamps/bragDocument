package postgres

import (
	"github.com/google/uuid"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// parseID turns a path or token id into a uuid; anything else is a not-found,
// never a 500, because callers only ever send ids we issued.
func parseID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, domain.ErrNotFound
	}
	return id, nil
}

func toTenant(t sqlcgen.Tenant) domain.Tenant {
	return domain.Tenant{ID: t.ID.String(), Name: t.Name, CreatedAt: t.CreatedAt}
}

func toUser(u sqlcgen.User) domain.User {
	return domain.User{ID: u.ID.String(), TenantID: u.TenantID.String(), Email: u.Email,
		DisplayName: u.DisplayName, Role: u.Role, CreatedAt: u.CreatedAt}
}

func toInvitation(i sqlcgen.TenantInvitation) domain.Invitation {
	return domain.Invitation{ID: i.ID.String(), TenantID: i.TenantID.String(), Email: i.Email,
		CreatedBy: i.CreatedBy.String(), CreatedAt: i.CreatedAt}
}

func toDocument(d sqlcgen.Document) domain.Document {
	return domain.Document{ID: d.ID.String(), TenantID: d.TenantID.String(), OwnerID: d.OwnerID.String(),
		Title: d.Title, Description: d.Description, State: d.State, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}
