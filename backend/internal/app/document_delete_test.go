package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsDelete(t *testing.T) {
	f := newFakeDocs()
	s := NewDocuments(f)
	d, err := s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "mine"})
	require.NoError(t, err)

	require.ErrorIs(t, s.Delete(context.Background(), d.ID, "u2"), domain.ErrNotFound)
	f.grant(d.ID, "u2", domain.RoleEditor)
	require.ErrorIs(t, s.Delete(context.Background(), d.ID, "u2"), domain.ErrForbidden)
	require.NoError(t, s.Delete(context.Background(), d.ID, "u1"))
	require.Len(t, f.audit, 2, "created, then deleted; denied deletes write nothing")
	require.Equal(t, domain.AuditDocumentDeleted, f.audit[1].Action)
	require.Equal(t, d.ID, f.audit[1].DocumentID)
	require.Equal(t, "u1", f.audit[1].ActorID)
	require.ErrorIs(t, s.Delete(context.Background(), d.ID, "u1"), domain.ErrNotFound)
}
