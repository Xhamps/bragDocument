package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type matrixFixture struct {
	docs    *Documents
	logs    *Logs
	sharing *Sharing
	exports *Exports
	logID   string
	invID   string
}

// newMatrixFixture: d1 owned by u1; u2 holds the role under test; u5 is a
// viewer to act on; one pending invitation; one log.
func newMatrixFixture(t *testing.T, role domain.Role) matrixFixture {
	t.Helper()
	fd := newFakeDocs()
	fd.docs["d1"] = domain.Document{ID: "d1", TenantID: "t1", OwnerID: "u1", Title: "2026", State: domain.DocumentActive}
	if role.Grantable() {
		fd.grant("d1", "u2", role)
	}
	fd.grant("d1", "u5", domain.RoleViewer)
	repo := newFakeSharing(fd)
	for _, id := range []string{"u1", "u2", "u5", "u9"} {
		repo.members[id] = domain.User{ID: id, TenantID: "t1", Email: id + "@acme.com"}
	}
	inv, err := repo.Invite(context.Background(), domain.DocumentInvitation{DocumentID: "d1", Email: "new@acme.com", Role: domain.RoleViewer}, domain.AuditEntry{})
	require.NoError(t, err)
	fl := newFakeLogs()
	l, err := fl.Create(context.Background(), domain.Log{DocumentID: "d1", Name: "x", Impact: "low", Status: domain.StatusDone}, domain.AuditEntry{})
	require.NoError(t, err)
	return matrixFixture{docs: NewDocuments(fd), logs: NewLogs(fd, fl, &fakeImpact{}, newFakeCache()),
		sharing: NewSharing(fd, repo, &fakeMailer{}, ""), logID: l.ID, invID: inv.ID,
		exports: NewExports(fd, fl, newFakeExports(), &fakeRenderer{}, newFakeFiles(), newFakeCache(),
			func(ctx context.Context, _ string) context.Context { return ctx })}
}

func TestAccessMatrix(t *testing.T) {
	ctx := context.Background()
	name, archived := "renamed", domain.DocumentArchived
	actions := []struct {
		name string
		perm domain.Permission
		run  func(f matrixFixture, u domain.User) error
	}{
		{"get document", domain.PermRead, func(f matrixFixture, u domain.User) error { _, err := f.docs.Get(ctx, "d1", u.ID); return err }},
		{"list logs", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.logs.List(ctx, "d1", u.ID, domain.LogFilter{})
			return err
		}},
		{"get log", domain.PermRead, func(f matrixFixture, u domain.User) error { _, err := f.logs.Get(ctx, "d1", f.logID, u.ID); return err }},
		{"create log", domain.PermWriteLogs, func(f matrixFixture, u domain.User) error {
			_, err := f.logs.Create(ctx, CreateLogInput{DocumentID: "d1", UserID: u.ID, Name: "y", Impact: "low"})
			return err
		}},
		{"update log", domain.PermWriteLogs, func(f matrixFixture, u domain.User) error {
			_, err := f.logs.Update(ctx, UpdateLogInput{ID: f.logID, DocumentID: "d1", UserID: u.ID, Name: &name})
			return err
		}},
		{"delete log", domain.PermWriteLogs, func(f matrixFixture, u domain.User) error { return f.logs.Delete(ctx, "d1", f.logID, u.ID) }},
		{"delete examples", domain.PermWriteLogs, func(f matrixFixture, u domain.User) error { return f.logs.DeleteExamples(ctx, "d1", u.ID) }},
		{"dashboard", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.logs.Dashboard(ctx, "d1", u.ID, nil, nil)
			return err
		}},
		{"rename", domain.PermManage, func(f matrixFixture, u domain.User) error {
			_, err := f.docs.Update(ctx, UpdateDocumentInput{ID: "d1", UserID: u.ID, Title: &name})
			return err
		}},
		{"archive", domain.PermManage, func(f matrixFixture, u domain.User) error {
			_, err := f.docs.Update(ctx, UpdateDocumentInput{ID: "d1", UserID: u.ID, State: &archived})
			return err
		}},
		{"delete document", domain.PermDelete, func(f matrixFixture, u domain.User) error { return f.docs.Delete(ctx, "d1", u.ID) }},
		{"view sharing", domain.PermShare, func(f matrixFixture, u domain.User) error { _, err := f.sharing.Get(ctx, u, "d1"); return err }},
		{"share", domain.PermShare, func(f matrixFixture, u domain.User) error {
			_, err := f.sharing.Share(ctx, ShareInput{Actor: u, DocumentID: "d1", Email: "u9@acme.com", Role: domain.RoleViewer})
			return err
		}},
		{"change role", domain.PermShare, func(f matrixFixture, u domain.User) error {
			return f.sharing.ChangeRole(ctx, u, "d1", "u5", domain.RoleEditor)
		}},
		{"revoke", domain.PermShare, func(f matrixFixture, u domain.User) error { return f.sharing.Revoke(ctx, u, "d1", "u5") }},
		{"cancel invitation", domain.PermShare, func(f matrixFixture, u domain.User) error { return f.sharing.CancelInvitation(ctx, u, "d1", f.invID) }},
		{"document audit", domain.PermShare, func(f matrixFixture, u domain.User) error {
			_, err := f.sharing.DocumentAudit(ctx, u, "d1")
			return err
		}},
		{"export pdf", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.exports.Create(ctx, CreateExportInput{DocumentID: "d1", UserID: u.ID})
			return err
		}},
		{"export settings", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.exports.Settings(ctx, "d1", u.ID)
			return err
		}},
		{"export history", domain.PermRead, func(f matrixFixture, u domain.User) error {
			_, err := f.exports.List(ctx, "d1", u.ID)
			return err
		}},
		{"transfer", domain.PermTransfer, func(f matrixFixture, u domain.User) error { return f.sharing.Transfer(ctx, u, "d1", "u5") }},
	}
	for _, role := range []domain.Role{domain.RoleOwner, domain.RoleEditor, domain.RoleViewer, ""} {
		user := domain.User{ID: "u2", TenantID: "t1", Email: "u2@acme.com"}
		if role == domain.RoleOwner {
			user = domain.User{ID: "u1", TenantID: "t1", Email: "u1@acme.com"}
		}
		for _, a := range actions {
			t.Run(string(role)+"/"+a.name, func(t *testing.T) {
				err := a.run(newMatrixFixture(t, role), user)
				switch {
				case role == "":
					require.ErrorIs(t, err, domain.ErrNotFound, "no grant: no trace of the document")
				case domain.Can(role, a.perm):
					require.NoError(t, err)
				default:
					var ae *domain.AccessError
					require.ErrorAs(t, err, &ae)
					require.Equal(t, role, ae.Role)
					require.ErrorIs(t, err, domain.ErrForbidden)
				}
			})
		}
	}
}

// FR-9: the tenant admin role grants no document access; reads need a grant.
func TestTenantAdminWithoutGrantCannotRead(t *testing.T) {
	f := newMatrixFixture(t, "")
	admin := domain.User{ID: "u2", TenantID: "t1", Email: "u2@acme.com", Role: domain.RoleAdmin}
	_, err := f.docs.Get(context.Background(), "d1", admin.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = f.logs.List(context.Background(), "d1", admin.ID, domain.LogFilter{})
	require.ErrorIs(t, err, domain.ErrNotFound)
}
