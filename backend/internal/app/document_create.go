package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// CreateDocumentInput comes from the handler; owner and tenant from the principal.
type CreateDocumentInput struct {
	TenantID    string
	OwnerID     string
	Title       string
	Description string
}

// Create validates and stores a new active document.
func (s *Documents) Create(ctx context.Context, in CreateDocumentInput) (domain.Document, error) {
	d := domain.Document{TenantID: in.TenantID, OwnerID: in.OwnerID, Title: in.Title, Description: in.Description, State: domain.DocumentActive}
	if err := d.Validate(); err != nil {
		return domain.Document{}, err
	}
	return s.docs.Create(ctx, d)
}
