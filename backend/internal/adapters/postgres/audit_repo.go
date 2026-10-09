package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
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

// AuditRepo implements ports.AuditRepo for the tenant in the context.
type AuditRepo struct{ db *DB }

// NewAuditRepo wires the repository to the pool.
func NewAuditRepo(db *DB) *AuditRepo { return &AuditRepo{db: db} }

func (r *AuditRepo) List(ctx context.Context, f domain.AuditFilter) (domain.AuditPage, error) {
	p := sqlcgen.ListAuditParams{Lim: int32(f.Limit + 1)} // one extra row tells whether a next page exists
	var err error
	for _, id := range []struct {
		s   string
		dst *pgtype.UUID
	}{{f.ActorID, &p.ActorID}, {f.DocumentID, &p.DocumentID}, {f.OwnerID, &p.OwnerID}} {
		if *id.dst, err = optID(id.s); err != nil {
			return domain.AuditPage{}, err
		}
	}
	if f.Action != "" {
		p.Action = pgtype.Text{String: f.Action, Valid: true}
	}
	if f.From != nil {
		p.FromAt = pgtype.Timestamptz{Time: *f.From, Valid: true}
	}
	if f.To != nil {
		p.ToAt = pgtype.Timestamptz{Time: *f.To, Valid: true}
	}
	if f.Before > 0 {
		p.Before = pgtype.Int8{Int64: f.Before, Valid: true}
	}
	out := domain.AuditPage{Entries: []domain.AuditEntry{}}
	err = r.db.WithTenant(ctx, telemetry.TenantID(ctx), func(ctx context.Context, tx pgx.Tx) error {
		// The optional filters defeat a generic plan (a full seq scan per page); a custom
		// plan picks the matching index. Auto mode already keeps custom here; this pins it (NFR-3).
		if _, err := tx.Exec(ctx, "SET LOCAL plan_cache_mode = force_custom_plan"); err != nil {
			return wrap(err)
		}
		rows, err := sqlcgen.New(tx).ListAudit(ctx, p)
		if err != nil {
			return wrap(err)
		}
		if len(rows) > f.Limit {
			rows = rows[:f.Limit]
			out.NextBefore = rows[len(rows)-1].ID
		}
		for _, row := range rows {
			out.Entries = append(out.Entries, toAuditEntry(row))
		}
		return nil
	})
	return out, err
}

func (r *AuditRepo) Filters(ctx context.Context, ownerID string) ([]domain.AuditActor, []domain.AuditDocument, error) {
	oid, err := optID(ownerID)
	if err != nil {
		return nil, nil, err
	}
	actors, docs := []domain.AuditActor{}, []domain.AuditDocument{}
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		as, err := q.ListAuditActors(ctx, oid)
		if err != nil {
			return wrap(err)
		}
		for _, a := range as {
			actors = append(actors, domain.AuditActor{ID: idString(a.ActorID), Name: a.ActorName, Email: a.ActorEmail})
		}
		ds, err := q.ListAuditDocuments(ctx, oid)
		if err != nil {
			return wrap(err)
		}
		for _, d := range ds {
			docs = append(docs, domain.AuditDocument{ID: idString(d.DocumentID), Title: d.DocumentTitle.String})
		}
		return nil
	})
	return actors, docs, err
}
