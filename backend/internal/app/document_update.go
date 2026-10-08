package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// UpdateDocumentInput: nil fields are left unchanged.
type UpdateDocumentInput struct {
	ID          string
	UserID      string
	Title       *string
	Description *string
	State       *string
}

// Update renames, re-describes, archives, or unarchives a document the caller owns.
func (s *Documents) Update(ctx context.Context, in UpdateDocumentInput) (domain.Document, error) {
	d, err := s.owned(ctx, in.ID, in.UserID)
	if err != nil {
		return domain.Document{}, err
	}
	if in.Title != nil {
		d.Title = *in.Title
	}
	if in.Description != nil {
		d.Description = *in.Description
	}
	if in.State != nil {
		d.State = *in.State
	}
	if err := d.Validate(); err != nil {
		return domain.Document{}, err
	}
	return s.docs.Update(ctx, d)
}

// owned loads a document and checks ownership. RLS already hides other
// tenants' documents (404); a same-tenant non-owner gets 403.
func (s *Documents) owned(ctx context.Context, id, userID string) (domain.Document, error) {
	d, err := s.docs.Get(ctx, id)
	if err != nil {
		return domain.Document{}, err
	}
	if d.OwnerID != userID {
		return domain.Document{}, domain.ErrForbidden
	}
	return d, nil
}
