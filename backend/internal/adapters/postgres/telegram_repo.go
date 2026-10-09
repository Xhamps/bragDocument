package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// TelegramLinkRepo implements ports.TelegramLinkRepo.
type TelegramLinkRepo struct{ db *DB }

// NewTelegramLinkRepo wires the repository to the pool.
func NewTelegramLinkRepo(db *DB) *TelegramLinkRepo { return &TelegramLinkRepo{db: db} }

func toTelegramLink(l sqlcgen.TelegramLink) domain.TelegramLink {
	out := domain.TelegramLink{UserID: l.UserID.String(), TenantID: l.TenantID.String(),
		TelegramUserID: l.TelegramUserID, LinkedAt: l.LinkedAt}
	if l.DocumentID.Valid {
		out.DocumentID = uuid.UUID(l.DocumentID.Bytes).String()
	}
	return out
}

// FindByTelegramID runs under the provisioning flag: the tenant is what we are looking up.
func (r *TelegramLinkRepo) FindByTelegramID(ctx context.Context, telegramUserID int64) (domain.TelegramLink, error) {
	var out domain.TelegramLink
	err := r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		l, err := sqlcgen.New(tx).GetTelegramLinkByTelegramID(ctx, telegramUserID)
		if err != nil {
			return wrap(err)
		}
		out = toTelegramLink(l)
		return nil
	})
	return out, err
}

func (r *TelegramLinkRepo) Get(ctx context.Context, userID string) (domain.TelegramLink, error) {
	uid, err := parseID(userID)
	if err != nil {
		return domain.TelegramLink{}, err
	}
	var out domain.TelegramLink
	err = withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		l, err := q.GetTelegramLink(ctx, uid)
		if err != nil {
			return wrap(err)
		}
		out = toTelegramLink(l)
		return nil
	})
	return out, err
}

// Link uses l.TenantID, not the context: the bot links before any tenant is in scope.
func (r *TelegramLinkRepo) Link(ctx context.Context, l domain.TelegramLink, a domain.AuditEntry) error {
	uid, err := parseID(l.UserID)
	if err != nil {
		return err
	}
	tid, err := parseID(l.TenantID)
	if err != nil {
		return err
	}
	return r.db.WithTenant(ctx, l.TenantID, func(ctx context.Context, tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.UpsertTelegramLink(ctx, sqlcgen.UpsertTelegramLinkParams{
			UserID: uid, TenantID: tid, TelegramUserID: l.TelegramUserID, LinkedAt: l.LinkedAt}); err != nil {
			return wrap(err)
		}
		return audit(ctx, q, a)
	})
}

func (r *TelegramLinkRepo) SetDocument(ctx context.Context, userID, documentID string) error {
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	did, err := parseID(documentID)
	if err != nil {
		return err
	}
	return withQueries(ctx, r.db, func(ctx context.Context, q *sqlcgen.Queries) error {
		n, err := q.SetTelegramLinkDocument(ctx, sqlcgen.SetTelegramLinkDocumentParams{
			UserID: uid, DocumentID: pgtype.UUID{Bytes: did, Valid: true}})
		if err != nil {
			return wrap(err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// Delete writes a only when a link was removed: unlinking twice is one entry.
func (r *TelegramLinkRepo) Delete(ctx context.Context, userID string, a domain.AuditEntry) error {
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	_, err = write(ctx, r.db, a, func(ctx context.Context, q *sqlcgen.Queries, _ *domain.AuditEntry) (struct{}, error) {
		n, err := q.DeleteTelegramLink(ctx, uid)
		if err != nil {
			return struct{}{}, wrap(err)
		}
		if n == 0 {
			return struct{}{}, errNoChange // nothing removed, no entry
		}
		return struct{}{}, nil
	})
	return err
}
