package postgres

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// SharingRepo implements ports.SharingRepo for the tenant in the context.
type SharingRepo struct{ db *DB }

// NewSharingRepo wires the repository to the pool.
func NewSharingRepo(db *DB) *SharingRepo { return &SharingRepo{db: db} }

func (r *SharingRepo) tx(ctx context.Context, fn func(ctx context.Context, q *sqlcgen.Queries) error) error {
	return withQueries(ctx, r.db, fn)
}

// rowsOrNotFound turns an :execrows result into domain.ErrNotFound when nothing matched.
func rowsOrNotFound(n int64, err error) error {
	if err != nil {
		return wrap(err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SharingRepo) Get(ctx context.Context, docID string) (domain.Sharing, error) {
	did, err := parseID(docID)
	if err != nil {
		return domain.Sharing{}, err
	}
	out := domain.Sharing{Grants: []domain.Grant{}, Invitations: []domain.DocumentInvitation{}}
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		gs, err := q.ListGrants(ctx, did)
		if err != nil {
			return wrap(err)
		}
		for _, g := range gs {
			out.Grants = append(out.Grants, domain.Grant{DocumentID: docID, UserID: g.UserID.String(), Email: g.Email,
				DisplayName: g.DisplayName, Role: domain.Role(g.Role), GrantedBy: g.GrantedBy.String(), GrantedAt: g.GrantedAt})
		}
		is, err := q.ListPendingDocumentInvitations(ctx, did)
		if err != nil {
			return wrap(err)
		}
		for _, i := range is {
			out.Invitations = append(out.Invitations, toDocumentInvitation(i))
		}
		return nil
	})
	return out, err
}

func (r *SharingRepo) MemberByEmail(ctx context.Context, email string) (domain.User, error) {
	var out domain.User
	err := r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		u, err := q.GetUserByEmail(ctx, email)
		if err != nil {
			return wrap(err)
		}
		out = toUser(u)
		return nil
	})
	return out, err
}

func (r *SharingRepo) Member(ctx context.Context, id string) (domain.User, error) {
	uid, err := parseID(id)
	if err != nil {
		return domain.User{}, err
	}
	var out domain.User
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		u, err := q.GetUser(ctx, uid) // RLS: other tenants' users are not found
		if err != nil {
			return wrap(err)
		}
		out = toUser(u)
		return nil
	})
	return out, err
}

func (r *SharingRepo) Grant(ctx context.Context, g domain.Grant, a domain.AuditEntry) error {
	did, err := parseID(g.DocumentID)
	if err != nil {
		return err
	}
	uid, err := parseID(g.UserID)
	if err != nil {
		return err
	}
	by, err := parseID(g.GrantedBy)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if _, err := q.CreateGrant(ctx, sqlcgen.CreateGrantParams{DocumentID: did, UserID: uid, Role: string(g.Role), GrantedBy: by}); err != nil {
			return wrap(err)
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) Invite(ctx context.Context, inv domain.DocumentInvitation, a domain.AuditEntry) (domain.DocumentInvitation, error) {
	did, err := parseID(inv.DocumentID)
	if err != nil {
		return domain.DocumentInvitation{}, err
	}
	by, err := parseID(inv.InvitedBy)
	if err != nil {
		return domain.DocumentInvitation{}, err
	}
	var out domain.DocumentInvitation
	err = r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		row, err := q.CreateDocumentInvitation(ctx, sqlcgen.CreateDocumentInvitationParams{DocumentID: did,
			Email: inv.Email, Role: string(inv.Role), InvitedBy: by})
		if err != nil {
			return wrap(err)
		}
		out = toDocumentInvitation(row)
		a.TargetID = out.ID // known only after insert
		return audit(ctx, q, a)
	})
	return out, err
}

func (r *SharingRepo) SetRole(ctx context.Context, docID, userID string, role domain.Role, a domain.AuditEntry) error {
	did, err := parseID(docID)
	if err != nil {
		return err
	}
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if err := rowsOrNotFound(q.UpdateGrantRole(ctx, sqlcgen.UpdateGrantRoleParams{DocumentID: did, UserID: uid, Role: string(role)})); err != nil {
			return err
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) Revoke(ctx context.Context, docID, userID string, a domain.AuditEntry) error {
	did, err := parseID(docID)
	if err != nil {
		return err
	}
	uid, err := parseID(userID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if err := rowsOrNotFound(q.DeleteGrant(ctx, sqlcgen.DeleteGrantParams{DocumentID: did, UserID: uid})); err != nil {
			return err
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) CancelInvitation(ctx context.Context, docID, invID string, a domain.AuditEntry) error {
	did, err := parseID(docID)
	if err != nil {
		return err
	}
	iid, err := parseID(invID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		if err := rowsOrNotFound(q.DeletePendingDocumentInvitation(ctx, sqlcgen.DeletePendingDocumentInvitationParams{ID: iid, DocumentID: did})); err != nil {
			return err
		}
		return audit(ctx, q, a)
	})
}

func (r *SharingRepo) Transfer(ctx context.Context, docID, fromUserID, toUserID string, a domain.AuditEntry) error {
	did, err := parseID(docID)
	if err != nil {
		return err
	}
	from, err := parseID(fromUserID)
	if err != nil {
		return err
	}
	to, err := parseID(toUserID)
	if err != nil {
		return err
	}
	return r.tx(ctx, func(ctx context.Context, q *sqlcgen.Queries) error {
		// A from that is no longer the owner (stale or concurrent transfer) matches nothing: ErrNotFound.
		if err := rowsOrNotFound(q.SetDocumentOwner(ctx, sqlcgen.SetDocumentOwnerParams{ID: did, OwnerID: to, FromID: from})); err != nil {
			return err
		}
		if _, err := q.DeleteGrant(ctx, sqlcgen.DeleteGrantParams{DocumentID: did, UserID: to}); err != nil {
			return wrap(err)
		}
		if _, err := q.CreateGrant(ctx, sqlcgen.CreateGrantParams{DocumentID: did, UserID: from, Role: string(domain.RoleEditor), GrantedBy: from}); err != nil {
			return wrap(err)
		}
		return audit(ctx, q, a)
	})
}
