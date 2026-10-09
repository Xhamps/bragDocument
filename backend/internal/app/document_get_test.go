package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsGetMarksSharedSeen(t *testing.T) {
	f := newFakeDocs()
	f.docs["d1"] = domain.Document{ID: "d1", OwnerID: "u1", State: domain.DocumentActive}
	f.grant("d1", "u2", domain.RoleViewer)
	s := NewDocuments(f)
	ctx := context.Background()

	d, err := s.Get(ctx, "d1", "u1")
	require.NoError(t, err)
	require.Equal(t, domain.RoleOwner, d.Role)
	require.Empty(t, f.seen, "owners have no badge")

	d, err = s.Get(ctx, "d1", "u2")
	require.NoError(t, err)
	require.Equal(t, domain.RoleViewer, d.Role)
	require.Equal(t, []string{"d1/u2"}, f.seen)

	_, err = s.Get(ctx, "d1", "u2")
	require.NoError(t, err)
	require.Len(t, f.seen, 1, "seen once")

	_, err = s.Get(ctx, "d1", "u3")
	require.ErrorIs(t, err, domain.ErrNotFound)
}
