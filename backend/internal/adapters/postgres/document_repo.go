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

func (r *DocumentRepo) tx(ctx context.Context, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return r.db.WithTenant(ctx, telemetry.TenantID(ctx), func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, sqlcgen.New(tx))
	})
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
		for _, d := range rows {
			out = append(out, toDocument(d))
		}
		return nil
	})
	return out, err
}

func (r *DocumentRepo) Get(ctx context.Context, id string) (domain.Document, error) {
	did, err := parseID(id)
	if err != nil {
		return domain.Document{}, err
	}
	var out domain.Document
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.GetDocument(ctx, did)
		if err != nil {
			return wrap(err)
		}
		out = toDocument(row)
		return nil
	})
	return out, err
}

func (r *DocumentRepo) Create(ctx context.Context, d domain.Document) (domain.Document, error) {
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
