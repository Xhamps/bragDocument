package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestAuditReadPermissions(t *testing.T) {
	admin := domain.User{ID: "adm", Role: domain.RoleAdmin}
	owner := domain.User{ID: "u1", Role: domain.RoleMember}
	editor := domain.User{ID: "u2", Role: domain.RoleMember}
	viewer := domain.User{ID: "u3", Role: domain.RoleMember}
	outsider := domain.User{ID: "u4", Role: domain.RoleMember}
	docs := newFakeDocs()
	docs.docs["d1"] = domain.Document{ID: "d1", OwnerID: "u1"}
	docs.grant("d1", "u2", domain.RoleEditor)
	docs.grant("d1", "u3", domain.RoleViewer)
	ctx := context.Background()

	cases := []struct {
		name  string
		actor domain.User
		doc   string
		err   error  // nil, domain.ErrForbidden, domain.ErrNotFound
		owner string // OwnerID the repo must receive
	}{
		{"admin all", admin, "", nil, ""},
		{"admin any document, even without a grant", admin, "d1", nil, ""},
		{"owner all of theirs", owner, "", nil, "u1"},
		{"owner one of theirs", owner, "d1", nil, "u1"},
		{"editor page: owns nothing", editor, "", domain.ErrForbidden, ""},
		{"editor document", editor, "d1", domain.ErrForbidden, ""},
		{"viewer document", viewer, "d1", domain.ErrForbidden, ""},
		{"outsider document", outsider, "d1", domain.ErrNotFound, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := &fakeAudit{}
			_, err := NewAudit(docs, repo).List(ctx, c.actor, domain.AuditFilter{DocumentID: c.doc})
			if c.err == nil {
				require.NoError(t, err)
				require.Equal(t, c.owner, repo.f.OwnerID)
				require.Equal(t, c.doc, repo.f.DocumentID)
				require.Equal(t, 50, repo.f.Limit, "validated")
				return
			}
			require.ErrorIs(t, err, c.err)
		})
	}
}

func TestAuditListRejectsUnknownAction(t *testing.T) {
	docs := newFakeDocs()
	_, err := NewAudit(docs, &fakeAudit{}).List(context.Background(), domain.User{ID: "adm", Role: domain.RoleAdmin}, domain.AuditFilter{Action: "nope"})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}

func TestAuditFiltersScopedLikeList(t *testing.T) {
	docs := newFakeDocs()
	docs.docs["d1"] = domain.Document{ID: "d1", OwnerID: "u1"}
	repo := &fakeAudit{}
	_, _, err := NewAudit(docs, repo).Filters(context.Background(), domain.User{ID: "u1", Role: domain.RoleMember})
	require.NoError(t, err)
	require.Equal(t, "u1", repo.owner)
	_, _, err = NewAudit(docs, repo).Filters(context.Background(), domain.User{ID: "u9", Role: domain.RoleMember})
	require.ErrorIs(t, err, domain.ErrForbidden)
}
