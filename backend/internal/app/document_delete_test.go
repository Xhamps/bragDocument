package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsDelete(t *testing.T) {
	s := NewDocuments(newFakeDocs())
	d, err := s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "mine"})
	require.NoError(t, err)

	require.ErrorIs(t, s.Delete(context.Background(), d.ID, "u2"), domain.ErrForbidden)
	require.NoError(t, s.Delete(context.Background(), d.ID, "u1"))
	require.ErrorIs(t, s.Delete(context.Background(), d.ID, "u1"), domain.ErrNotFound)
}
