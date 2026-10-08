package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsUpdate(t *testing.T) {
	f := newFakeDocs()
	s := NewDocuments(f)
	d, _ := s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "old"})

	archived := domain.DocumentArchived
	got, err := s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "u1", State: &archived})
	require.NoError(t, err)
	require.Equal(t, "old", got.Title, "unset fields unchanged")
	require.Equal(t, domain.DocumentArchived, got.State)

	_, err = s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "intruder", State: &archived})
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = s.Update(context.Background(), UpdateDocumentInput{ID: "missing", UserID: "u1"})
	require.ErrorIs(t, err, domain.ErrNotFound)

	bad := "gone"
	_, err = s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "u1", State: &bad})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}
