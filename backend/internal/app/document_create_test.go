package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDocumentsCreate(t *testing.T) {
	s := NewDocuments(newFakeDocs())
	d, err := s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: "  2026 "})
	require.NoError(t, err)
	require.Equal(t, "2026", d.Title)
	require.Equal(t, domain.DocumentActive, d.State)

	_, err = s.Create(context.Background(), CreateDocumentInput{TenantID: "t1", OwnerID: "u1", Title: ""})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
}
