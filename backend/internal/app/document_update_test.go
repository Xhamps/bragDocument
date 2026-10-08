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
	d, err := s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "old"})
	require.NoError(t, err)

	title, desc := " new ", "about"
	got, err := s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "u1", Title: &title, Description: &desc})
	require.NoError(t, err)
	require.Equal(t, "new", got.Title)
	require.Equal(t, "about", got.Description)
	require.Equal(t, domain.DocumentActive, got.State, "unset fields unchanged")

	archived := domain.DocumentArchived
	got, err = s.Update(context.Background(), UpdateDocumentInput{ID: d.ID, UserID: "u1", State: &archived})
	require.NoError(t, err)
	require.Equal(t, "new", got.Title, "unset fields unchanged")
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
