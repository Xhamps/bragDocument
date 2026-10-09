package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// access loads a document with the caller's role and checks one permission
// (PRD-0004, ADR-0011). Every document and log use case calls it. No role →
// domain.ErrNotFound, so people without a grant cannot tell the document
// exists (FR-7); a role without the permission → *domain.AccessError (403).
func access(ctx context.Context, docs ports.DocumentRepo, docID, userID string, p domain.Permission) (domain.Document, error) {
	d, err := docs.GetForUser(ctx, docID, userID)
	if err != nil {
		return domain.Document{}, err
	}
	if !domain.Can(d.Role, p) {
		return domain.Document{}, &domain.AccessError{Role: d.Role, Perm: p}
	}
	return d, nil
}
