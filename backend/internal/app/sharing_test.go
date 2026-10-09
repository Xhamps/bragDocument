package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

var (
	ada   = domain.User{ID: "u1", TenantID: "t1", Email: "ada@acme.com", DisplayName: "Ada", Role: domain.RoleMember}
	bob   = domain.User{ID: "u2", TenantID: "t1", Email: "bob@acme.com", Role: domain.RoleMember}
	carol = domain.User{ID: "u3", TenantID: "t1", Email: "carol@acme.com", Role: domain.RoleAdmin}
)

type sharingFixture struct {
	s    *Sharing
	docs *fakeDocs
	repo *fakeSharing
	mail *fakeMailer
}

func newSharingFixture() sharingFixture {
	f := sharingFixture{docs: newFakeDocs(), mail: &fakeMailer{}}
	f.repo = newFakeSharing(f.docs)
	for _, u := range []domain.User{ada, bob, carol} {
		f.repo.members[u.ID] = u
	}
	f.docs.docs["d1"] = domain.Document{ID: "d1", TenantID: "t1", OwnerID: ada.ID, Title: "2026", State: domain.DocumentActive}
	f.s = NewSharing(f.docs, f.repo, f.mail, "https://app.test")
	return f
}

func TestShareWithMemberGrantsAndEmails(t *testing.T) {
	f := newSharingFixture()
	ctx := context.Background()

	kind, err := f.s.Share(ctx, ShareInput{Actor: ada, DocumentID: "d1", Email: " BOB@acme.com ", Role: domain.RoleViewer})
	require.NoError(t, err)
	require.Equal(t, ShareGranted, kind)
	require.Equal(t, domain.RoleViewer, f.docs.grants["d1"]["u2"])
	require.Equal(t, []string{"bob@acme.com: Ada shared “2026” with you"}, f.mail.sent)
	require.Contains(t, f.mail.html, `href="https://app.test/documents/d1"`)
	require.Equal(t, domain.AuditEntry{ActorID: "u1", Source: domain.SourceWeb, Action: domain.AuditGrant,
		DocumentID: "d1", TargetType: domain.TargetUser, TargetID: "u2", Target: "bob@acme.com", Role: domain.RoleViewer}, f.repo.audit[0])

	_, err = f.s.Share(ctx, ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleEditor})
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestShareWithUnknownEmailInvites(t *testing.T) {
	f := newSharingFixture()
	kind, err := f.s.Share(context.Background(), ShareInput{Actor: ada, DocumentID: "d1", Email: "new@acme.com", Role: domain.RoleEditor})
	require.NoError(t, err)
	require.Equal(t, ShareInvited, kind)
	require.Len(t, f.repo.invs["d1"], 1)
	require.Equal(t, domain.AuditInvite, f.repo.audit[0].Action)
	require.Contains(t, f.mail.html, `href="https://app.test/sign-in"`)
}

func TestShareEscapesEmailHTML(t *testing.T) {
	f := newSharingFixture()
	d := f.docs.docs["d1"]
	d.Title = `<script>x</script>`
	f.docs.docs["d1"] = d
	_, err := f.s.Share(context.Background(), ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleViewer})
	require.NoError(t, err)
	require.NotContains(t, f.mail.html, "<script>")
}

func TestShareSurvivesMailFailure(t *testing.T) {
	f := newSharingFixture()
	f.mail.err = errors.New("resend down")
	_, err := f.s.Share(context.Background(), ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleViewer})
	require.NoError(t, err)
	require.Equal(t, domain.RoleViewer, f.docs.grants["d1"]["u2"])
}

func TestShareValidation(t *testing.T) {
	f := newSharingFixture()
	var ve *domain.ValidationError
	for name, in := range map[string]ShareInput{
		"self":      {Actor: ada, DocumentID: "d1", Email: "ada@acme.com", Role: domain.RoleViewer},
		"bad email": {Actor: ada, DocumentID: "d1", Email: "nope", Role: domain.RoleViewer},
		"owner":     {Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleOwner},
	} {
		_, err := f.s.Share(context.Background(), in)
		require.ErrorAs(t, err, &ve, name)
	}
	require.Empty(t, f.mail.sent)
}

func TestChangeRoleRevokeCancel(t *testing.T) {
	f := newSharingFixture()
	ctx := context.Background()
	f.docs.grant("d1", bob.ID, domain.RoleViewer)
	inv, err := f.repo.Invite(ctx, domain.DocumentInvitation{DocumentID: "d1", Email: "new@acme.com", Role: domain.RoleViewer}, domain.AuditEntry{})
	require.NoError(t, err)
	f.repo.audit = nil

	require.NoError(t, f.s.ChangeRole(ctx, ada, "d1", bob.ID, domain.RoleEditor))
	require.Equal(t, domain.RoleEditor, f.docs.grants["d1"][bob.ID])
	require.NoError(t, f.s.ChangeRole(ctx, ada, "d1", bob.ID, domain.RoleEditor), "same role is a no-op")
	var ve *domain.ValidationError
	require.ErrorAs(t, f.s.ChangeRole(ctx, ada, "d1", bob.ID, domain.RoleOwner), &ve)
	require.ErrorIs(t, f.s.ChangeRole(ctx, ada, "d1", carol.ID, domain.RoleViewer), domain.ErrNotFound)

	require.NoError(t, f.s.Revoke(ctx, ada, "d1", bob.ID))
	require.ErrorIs(t, f.s.Revoke(ctx, ada, "d1", bob.ID), domain.ErrNotFound)
	require.NoError(t, f.s.CancelInvitation(ctx, ada, "d1", inv.ID))
	require.ErrorIs(t, f.s.CancelInvitation(ctx, ada, "d1", inv.ID), domain.ErrNotFound)

	require.Equal(t, []string{domain.AuditRoleChange, domain.AuditRevoke, domain.AuditInviteCancel},
		[]string{f.repo.audit[0].Action, f.repo.audit[1].Action, f.repo.audit[2].Action})
	require.Equal(t, "bob@acme.com", f.repo.audit[1].Target)
	require.Equal(t, "new@acme.com", f.repo.audit[2].Target)
	require.Equal(t, [2]string{domain.TargetInvitation, inv.ID}, [2]string{f.repo.audit[2].TargetType, f.repo.audit[2].TargetID})
}

func TestTransfer(t *testing.T) {
	f := newSharingFixture()
	ctx := context.Background()
	f.docs.grant("d1", bob.ID, domain.RoleViewer)

	var ve *domain.ValidationError
	require.ErrorAs(t, f.s.Transfer(ctx, ada, "d1", ada.ID), &ve, "already the owner")
	require.ErrorAs(t, f.s.Transfer(ctx, ada, "d1", "u-other-tenant"), &ve, "not a member")

	require.NoError(t, f.s.Transfer(ctx, ada, "d1", bob.ID))
	require.Equal(t, bob.ID, f.docs.docs["d1"].OwnerID)
	require.Equal(t, domain.RoleEditor, f.docs.grants["d1"][ada.ID])
	_, hasGrant := f.docs.grants["d1"][bob.ID]
	require.False(t, hasGrant)
	require.Equal(t, domain.AuditTransfer, f.repo.audit[0].Action)

	require.ErrorIs(t, f.s.Transfer(ctx, ada, "d1", carol.ID), domain.ErrForbidden, "ada is only an editor now")
}

func TestGetAndAudit(t *testing.T) {
	f := newSharingFixture()
	ctx := context.Background()
	_, err := f.s.Share(ctx, ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleViewer})
	require.NoError(t, err)

	sh, err := f.s.Get(ctx, ada, "d1")
	require.NoError(t, err)
	require.Len(t, sh.Grants, 1)

	es, err := f.s.DocumentAudit(ctx, ada, "d1")
	require.NoError(t, err)
	require.Len(t, es, 1)

	es, err = f.s.TenantAudit(ctx, carol)
	require.NoError(t, err)
	require.Len(t, es, 1)
	_, err = f.s.TenantAudit(ctx, ada)
	require.ErrorIs(t, err, domain.ErrForbidden, "admins only")
}

func TestShareEmailWithoutAppURLHasNoLink(t *testing.T) {
	f := newSharingFixture()
	f.s = NewSharing(f.docs, f.repo, f.mail, "")
	_, err := f.s.Share(context.Background(), ShareInput{Actor: ada, DocumentID: "d1", Email: "bob@acme.com", Role: domain.RoleViewer})
	require.NoError(t, err)
	require.NotEmpty(t, f.mail.html)
	require.NotContains(t, f.mail.html, "href")
}
