package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsList(t *testing.T) {
	f := newFakeDocs()
	s := NewDocuments(f)
	for _, in := range []CreateDocumentInput{
		{TenantID: "t1", OwnerID: "u1", Title: "a"},
		{TenantID: "t1", OwnerID: "u1", Title: "b"},
		{TenantID: "t1", OwnerID: "u2", Title: "c"},
	} {
		_, err := s.Create(context.Background(), in)
		require.NoError(t, err)
	}
	f.grant("d3", "u1", domain.RoleViewer) // "c", owned by u2

	out, err := s.List(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, out.Owned, 2)
	require.Equal(t, domain.RoleOwner, out.Owned[0].Role)
	require.Len(t, out.Shared, 1)
	require.Equal(t, domain.RoleViewer, out.Shared[0].Role)
}
