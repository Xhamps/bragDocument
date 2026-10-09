package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsCreate(t *testing.T) {
	docs := newFakeDocs()
	s := NewDocuments(docs)
	d, err := s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "  2026 "})
	require.NoError(t, err)
	require.Equal(t, "2026", d.Title)
	require.Equal(t, domain.DocumentActive, d.State)
	require.Len(t, docs.examples, 3)
	for _, ex := range docs.examples {
		require.True(t, ex.IsExample)
		require.Equal(t, "u1", ex.CreatedBy)
	}
	require.Equal(t, "github.com", docs.examples[0].Links[0].Host, "examples validated: hosts derived")

	_, err = s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: ""})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}
