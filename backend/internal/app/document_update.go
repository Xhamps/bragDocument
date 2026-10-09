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
	d, err := access(ctx, s.docs, in.ID, in.UserID, domain.PermManage)
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
	out, err := s.docs.Update(ctx, d)
	out.Role, out.OwnerName = d.Role, d.OwnerName
	return out, err
}
