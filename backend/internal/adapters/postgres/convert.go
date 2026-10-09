package postgres

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

func toLog(l sqlcgen.Log) domain.Log {
	out := domain.Log{ID: l.ID.String(), TenantID: l.TenantID.String(), DocumentID: l.DocumentID.String(),
		Name: l.Name, Description: l.Description, Impact: l.Impact, Status: l.Status, IsExample: l.IsExample,
		Tags: []string{}, Links: []domain.Link{},
		CreatedAt: l.CreatedAt, CreatedBy: l.CreatedBy.String(), UpdatedAt: l.UpdatedAt, UpdatedBy: l.UpdatedBy.String()}
	if l.ImpactStatement.Valid {
		s := l.ImpactStatement.String
		out.ImpactStatement = &s
	}
	return out
}

func textOrNull(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// orEmpty turns nil into an empty slice: pgx encodes nil as NULL, and JSON as null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func toDocumentInvitation(i sqlcgen.DocumentInvitation) domain.DocumentInvitation {
	return domain.DocumentInvitation{ID: i.ID.String(), DocumentID: i.DocumentID.String(), Email: i.Email,
		Role: domain.Role(i.Role), InvitedBy: i.InvitedBy.String(), CreatedAt: i.CreatedAt}
}

func toAuditEntry(a sqlcgen.AuditEntry) domain.AuditEntry {
	return domain.AuditEntry{ID: a.ID, ActorID: a.ActorID.String(), ActorEmail: a.ActorEmail, Action: a.Action,
		DocumentID: a.DocumentID.String(), DocumentTitle: a.DocumentTitle, Target: a.Target, Role: domain.Role(a.Role), At: a.At}
}
