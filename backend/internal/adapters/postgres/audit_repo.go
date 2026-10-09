package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// optID turns "" into NULL and anything else into a uuid (bad → ErrNotFound).
func optID(s string) (pgtype.UUID, error) {
	if s == "" {
		return pgtype.UUID{}, nil
	}
	id, err := parseID(s)
	return pgtype.UUID{Bytes: id, Valid: err == nil}, err
}

func idString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// audit writes one entry in the caller's transaction (PRD-0009 FR-1): a failed
// insert fails the action. A DocumentID that matches nothing is ErrNotFound.
func audit(ctx context.Context, q *sqlcgen.Queries, a domain.AuditEntry) error {
	did, err := optID(a.DocumentID)
	if err != nil {
		return err
	}
	aid, err := optID(a.ActorID)
	if err != nil {
		return err
	}
	n, err := q.CreateAuditEntry(ctx, sqlcgen.CreateAuditEntryParams{
		ActorID: aid, DocumentID: did, Source: a.Source, Action: a.Action,
		TargetType: a.TargetType, TargetID: a.TargetID, Target: a.Target, Role: string(a.Role),
		ChangedFields: orEmpty(a.ChangedFields),
	})
	return rowsOrNotFound(n, err)
}
