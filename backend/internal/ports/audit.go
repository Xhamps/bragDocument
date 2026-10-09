package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// AuditRepo reads audit entries of the tenant in the context (PRD-0009).
// Writes go through each resource's repository, in the action's transaction.
type AuditRepo interface {
	// List returns one page; f must have passed Validate.
	List(ctx context.Context, f domain.AuditFilter) (domain.AuditPage, error)
	// Filters returns the distinct actors and documents of the entries visible
	// with ownerID ("" = every entry), for the pickers.
	Filters(ctx context.Context, ownerID string) ([]domain.AuditActor, []domain.AuditDocument, error)
}
