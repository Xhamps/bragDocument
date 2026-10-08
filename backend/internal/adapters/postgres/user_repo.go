package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// UserRepo implements ports.UserRepo.
type UserRepo struct{ db *DB }

// NewUserRepo wires the repository to the pool.
func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

// Provision runs fn in the provisioning transaction (see DB.WithProvisioning).
func (r *UserRepo) Provision(ctx context.Context, fn func(ctx context.Context, tx ports.ProvisionTx) error) error {
	return r.db.WithProvisioning(ctx, func(ctx context.Context, tx pgx.Tx) error {
		return fn(ctx, provisionTx{q: sqlcgen.New(tx)})
	})
}

type provisionTx struct{ q *sqlcgen.Queries }

func (p provisionTx) GetUser(ctx context.Context, id string) (domain.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return domain.User{}, err
	}
	u, err := p.q.GetUser(ctx, uid)
	if err != nil {
		return domain.User{}, wrap(err)
	}
	return toUser(u), nil
}

func (p provisionTx) GetTenant(ctx context.Context, id string) (domain.Tenant, error) {
	tid, err := parseID(id)
	if err != nil {
		return domain.Tenant{}, err
	}
	t, err := p.q.GetTenant(ctx, tid)
	if err != nil {
		return domain.Tenant{}, wrap(err)
	}
	return toTenant(t), nil
}

func (p provisionTx) FindInvitationByEmail(ctx context.Context, email string) (domain.Invitation, error) {
	inv, err := p.q.GetInvitationByEmail(ctx, email)
	if err != nil {
		return domain.Invitation{}, wrap(err)
	}
	return toInvitation(inv), nil
}

func (p provisionTx) CreateTenant(ctx context.Context, name string) (domain.Tenant, error) {
	t, err := p.q.CreateTenant(ctx, name)
	if err != nil {
		return domain.Tenant{}, wrap(err)
	}
	return toTenant(t), nil
}

func (p provisionTx) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	uid, err := parseID(u.ID)
	if err != nil {
		return domain.User{}, err
	}
	tid, err := parseID(u.TenantID)
	if err != nil {
		return domain.User{}, err
	}
	row, err := p.q.CreateUser(ctx, sqlcgen.CreateUserParams{
		ID: uid, TenantID: tid, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role,
	})
	if err != nil {
		return domain.User{}, wrap(err)
	}
	return toUser(row), nil
}

func (p provisionTx) DeleteInvitation(ctx context.Context, id string) error {
	iid, err := parseID(id)
	if err != nil {
		return err
	}
	n, err := p.q.DeleteInvitation(ctx, iid)
	if err != nil {
		return wrap(err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
