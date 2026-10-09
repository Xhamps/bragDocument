package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// DocumentRepo implements ports.DocumentRepo for the tenant in the context.
type DocumentRepo struct{ db *DB }

// NewDocumentRepo wires the repository to the pool.
func NewDocumentRepo(db *DB) *DocumentRepo { return &DocumentRepo{db: db} }

// withQueries runs fn in a transaction scoped to the tenant in the context.
func withQueries(ctx context.Context, db *DB, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return db.WithTenant(ctx, telemetry.TenantID(ctx), func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, sqlcgen.New(tx))
	})
}

func (r *DocumentRepo) tx(ctx context.Context, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return withQueries(ctx, r.db, fn)
}

func (r *DocumentRepo) ListByOwner(ctx context.Context, ownerID string) ([]domain.Document, error) {
	oid, err := parseID(ownerID)
	if err != nil {
		return nil, err
	}
	out := []domain.Document{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListDocumentsByOwner(ctx, oid)
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			d := toDocument(row.Document)
			d.LogCount = int(row.LogCount)
			if d.LogCount > 0 {
				t := row.LastLogAt
				d.LastLogAt = &t
			}
			out = append(out, d)
		}
		return nil
	})
	return out, err
}

func (r *DocumentRepo) GetForUser(ctx context.Context, id, userID string) (domain.Document, error) {
	did, err := parseID(id)
	if err != nil {
		return domain.Document{}, err
	}
	uid, err := parseID(userID)
	if err != nil {
		return domain.Document{}, err
	}
	var out domain.Document
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetDocumentForUser(ctx, sqlcgen.GetDocumentForUserParams{ID: did, UserID: uid})
		if err != nil {
			return wrap(err)
		}
		if row.Role == "" {
			return domain.ErrNotFound // PRD-0004 FR-7: no role, no trace
		}
		out = toDocument(row.Document)
		out.Role, out.OwnerName, out.IsNew = domain.Role(row.Role), row.OwnerName, row.IsNew
		return nil
	})
	return out, err
}

func (r *DocumentRepo) ListShared(ctx context.Context, userID string) ([]domain.Document, error) {
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	out := []domain.Document{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListSharedDocuments(ctx, uid)
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			d := toDocument(row.Document)
			d.Role, d.OwnerName, d.IsNew = domain.Role(row.Role), row.OwnerName, row.IsNew
			d.LogCount = int(row.LogCount)
			if d.LogCount > 0 {
				t := row.LastLogAt
				d.LastLogAt = &t
			}
			out = append(out, d)
		}
		return nil
	})
	return out, err
}

func (r *DocumentRepo) ListWritable(ctx context.Context, userID string) ([]domain.Document, error) {
	uid, err := parseID(userID)
	if err != nil {
		return nil, err
	}
	out := []domain.Document{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListWritableDocuments(ctx, uid)
		if err != nil {
			return wrap(err)
		}
		for _, row := range rows {
			out = append(out, toDocument(row))
		}
		return nil
	})
	return out, err
}

func (r *DocumentRepo) MarkSeen(ctx context.Context, id, userID string) error {
	did, err := parseID(id)
	if err != nil {
		return err
	}
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		return wrap(q.MarkGrantSeen(ctx, sqlcgen.MarkGrantSeenParams{DocumentID: did, UserID: uid}))
	})
}

func (r *DocumentRepo) Create(ctx context.Context, d domain.Document, examples []domain.Log) (domain.Document, error) {
	tid, err := parseID(d.TenantID)
	if err != nil {
		return domain.Document{}, err
	}
	oid, err := parseID(d.OwnerID)
	if err != nil {
		return domain.Document{}, err
	}
	var out domain.Document
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.CreateDocument(ctx, sqlcgen.CreateDocumentParams{TenantID: tid, OwnerID: oid, Title: d.Title, Description: d.Description})
		if err != nil {
			return wrap(err)
		}
		for _, ex := range examples {
			ex.TenantID, ex.DocumentID = row.TenantID.String(), row.ID.String()
			if _, err := insertLog(ctx, q, ex); err != nil {
				return err
			}
		}
		out = toDocument(row)
		return nil
	})
	return out, err
}

func (r *DocumentRepo) Update(ctx context.Context, d domain.Document) (domain.Document, error) {
	did, err := parseID(d.ID)
	if err != nil {
		return domain.Document{}, err
	}
	var out domain.Document
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.UpdateDocument(ctx, sqlcgen.UpdateDocumentParams{ID: did, Title: d.Title, Description: d.Description, State: d.State})
		if err != nil {
			return wrap(err)
		}
		out = toDocument(row)
		return nil
	})
	return out, err
}

func (r *DocumentRepo) Delete(ctx context.Context, id string) error {
	did, err := parseID(id)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.DeleteDocument(ctx, did)
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
