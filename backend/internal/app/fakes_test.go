package app

import (
	"context"
	"errors"
	"strconv"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

type fakeUsers struct {
	users               map[string]domain.User
	tenants             map[string]domain.Tenant
	invitations         map[string]domain.Invitation // by email
	seq                 int
	createUserErrOnce   error // returned by the first CreateUser, then cleared; a conflict inserts the winner's row
	createUserErrAlways error // returned by every CreateUser, no row inserted
	createUserCalls     int
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[string]domain.User{}, tenants: map[string]domain.Tenant{}, invitations: map[string]domain.Invitation{}}
}

func (f *fakeUsers) Provision(ctx context.Context, fn func(context.Context, ports.ProvisionTx) error) error {
	return fn(ctx, f)
}
func (f *fakeUsers) GetUser(_ context.Context, id string) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}
func (f *fakeUsers) GetTenant(_ context.Context, id string) (domain.Tenant, error) {
	t, ok := f.tenants[id]
	if !ok {
		return domain.Tenant{}, domain.ErrNotFound
	}
	return t, nil
}
func (f *fakeUsers) FindInvitationByEmail(_ context.Context, email string) (domain.Invitation, error) {
	i, ok := f.invitations[email]
	if !ok {
		return domain.Invitation{}, domain.ErrNotFound
	}
	return i, nil
}
func (f *fakeUsers) CreateTenant(_ context.Context, name string) (domain.Tenant, error) {
	f.seq++
	t := domain.Tenant{ID: "t" + strconv.Itoa(f.seq), Name: name}
	f.tenants[t.ID] = t
	return t, nil
}
func (f *fakeUsers) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	f.createUserCalls++
	if f.createUserErrAlways != nil {
		return domain.User{}, f.createUserErrAlways
	}
	if err := f.createUserErrOnce; err != nil {
		f.createUserErrOnce = nil
		if errors.Is(err, domain.ErrConflict) {
			// The concurrent winner's row: in the invited tenant if there is an invitation, else t9.
			tenantID := "t9"
			if inv, ok := f.invitations[u.Email]; ok {
				tenantID = inv.TenantID
				delete(f.invitations, u.Email) // the winner consumed it
			}
			f.users[u.ID] = domain.User{ID: u.ID, TenantID: tenantID, Email: u.Email}
		}
		return domain.User{}, err
	}
	f.users[u.ID] = u
	return u, nil
}
func (f *fakeUsers) DeleteInvitation(_ context.Context, id string) error {
	for email, i := range f.invitations {
		if i.ID == id {
			delete(f.invitations, email)
			return nil
		}
	}
	return domain.ErrNotFound
}

type fakeDocs struct {
	docs map[string]domain.Document
	seq  int
}

func newFakeDocs() *fakeDocs { return &fakeDocs{docs: map[string]domain.Document{}} }

func (f *fakeDocs) ListByOwner(_ context.Context, ownerID string) ([]domain.Document, error) {
	out := []domain.Document{}
	for _, d := range f.docs {
		if d.OwnerID == ownerID {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDocs) Get(_ context.Context, id string) (domain.Document, error) {
	d, ok := f.docs[id]
	if !ok {
		return domain.Document{}, domain.ErrNotFound
	}
	return d, nil
}
func (f *fakeDocs) Create(_ context.Context, d domain.Document) (domain.Document, error) {
	f.seq++
	d.ID = "d" + strconv.Itoa(f.seq)
	f.docs[d.ID] = d
	return d, nil
}
func (f *fakeDocs) Update(_ context.Context, d domain.Document) (domain.Document, error) {
	if _, ok := f.docs[d.ID]; !ok {
		return domain.Document{}, domain.ErrNotFound
	}
	f.docs[d.ID] = d
	return d, nil
}
func (f *fakeDocs) Delete(_ context.Context, id string) error {
	if _, ok := f.docs[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.docs, id)
	return nil
}

type fakeTenants struct {
	members     map[string]domain.User
	invitations map[string]domain.Invitation
	seq         int
}

func newFakeTenants() *fakeTenants {
	return &fakeTenants{members: map[string]domain.User{}, invitations: map[string]domain.Invitation{}}
}

func (f *fakeTenants) ListMembers(context.Context) ([]domain.User, error) {
	out := []domain.User{}
	for _, u := range f.members {
		out = append(out, u)
	}
	return out, nil
}
func (f *fakeTenants) DeleteMember(_ context.Context, id string) error {
	if _, ok := f.members[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.members, id)
	return nil
}
func (f *fakeTenants) ListInvitations(context.Context) ([]domain.Invitation, error) {
	out := []domain.Invitation{}
	for _, i := range f.invitations {
		out = append(out, i)
	}
	return out, nil
}
func (f *fakeTenants) CreateInvitation(_ context.Context, inv domain.Invitation) (domain.Invitation, error) {
	for _, i := range f.invitations {
		if i.Email == inv.Email {
			return domain.Invitation{}, domain.ErrConflict
		}
	}
	f.seq++
	inv.ID = "i" + strconv.Itoa(f.seq)
	f.invitations[inv.ID] = inv
	return inv, nil
}
func (f *fakeTenants) DeleteInvitation(_ context.Context, id string) error {
	if _, ok := f.invitations[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.invitations, id)
	return nil
}
