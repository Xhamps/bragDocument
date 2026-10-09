package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// TenantRepo implements ports.TenantRepo for the tenant in the context.
type TenantRepo struct{ db *DB }

// NewTenantRepo wires the repository to the pool.
func NewTenantRepo(db *DB) *TenantRepo { return &TenantRepo{db: db} }

func (r *TenantRepo) tx(ctx context.Context, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return r.db.WithTenant(ctx, telemetry.TenantID(ctx), func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, sqlcgen.New(tx))
	})
}

func (r *TenantRepo) ListMembers(ctx context.Context) ([]domain.User, error) {
	tid, err := parseID(telemetry.TenantID(ctx))
	if err != nil {
		return nil, err
	}
	out := []domain.User{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListUsersByTenant(ctx, tid)
		if err != nil {
			return wrap(err)
		}
		for _, u := range rows {
			out = append(out, toUser(u))
		}
		return nil
	})
	return out, err
}

func (r *TenantRepo) DeleteMember(ctx context.Context, id string) error {
	uid, err := parseID(id)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.DeleteUser(ctx, uid) // documents.owner_id is ON DELETE RESTRICT → 23503 → ErrConflict
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *TenantRepo) ListInvitations(ctx context.Context) ([]domain.Invitation, error) {
	tid, err := parseID(telemetry.TenantID(ctx))
	if err != nil {
		return nil, err
	}
	out := []domain.Invitation{}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		rows, err := q.ListInvitationsByTenant(ctx, tid)
		if err != nil {
			return wrap(err)
		}
		for _, i := range rows {
			out = append(out, toInvitation(i))
		}
		docInvs, err := q.ListPendingDocumentInvitationsByTenant(ctx, tid)
		if err != nil {
			return wrap(err)
		}
		for _, i := range docInvs {
			out = append(out, domain.Invitation{ID: i.ID.String(), TenantID: tid.String(), Email: i.Email,
				CreatedAt: i.CreatedAt, ForDocument: true, DocumentTitle: i.DocumentTitle})
		}
		return nil
	})
	return out, err
}

func (r *TenantRepo) CreateInvitation(ctx context.Context, inv domain.Invitation) (domain.Invitation, error) {
	tid, err := parseID(inv.TenantID)
	if err != nil {
		return domain.Invitation{}, err
	}
	by, err := parseID(inv.CreatedBy)
	if err != nil {
		return domain.Invitation{}, err
	}
	var out domain.Invitation
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.CreateInvitation(ctx, sqlcgen.CreateInvitationParams{TenantID: tid, Email: inv.Email, CreatedBy: by})
		if err != nil {
			return wrap(err)
		}
		out = toInvitation(row)
		return nil
	})
	return out, err
}

func (r *TenantRepo) DeleteInvitation(ctx context.Context, id string) error {
	iid, err := parseID(id)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.DeleteInvitation(ctx, iid)
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
