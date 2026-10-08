package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentsList(t *testing.T) {
	s := NewDocuments(newFakeDocs())
	for _, in := range []CreateDocumentInput{
		{TenantID: "t1", OwnerID: "u1", Title: "a"},
		{TenantID: "t1", OwnerID: "u1", Title: "b"},
		{TenantID: "t1", OwnerID: "u2", Title: "c"},
	} {
		_, err := s.Create(context.Background(), in)
		require.NoError(t, err)
	}

	out, err := s.List(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, out.Owned, 2)
	require.NotNil(t, out.Shared)
	require.Empty(t, out.Shared)
}
