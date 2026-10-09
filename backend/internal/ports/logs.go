package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// LogRepo stores logs of the tenant in the context. Every method that takes a
// documentID only touches logs of that document.
type LogRepo interface {
	List(ctx context.Context, documentID string, f domain.LogFilter) (domain.LogPage, error)
	// Get returns domain.ErrNotFound when the log is not in the document.
	Get(ctx context.Context, documentID, id string) (domain.Log, error)
	// Create, Update, Delete, and DeleteExamples write a in the same
	// transaction (PRD-0009 FR-1). Create sets a.TargetID to the new log's id.
	Create(ctx context.Context, l domain.Log, a domain.AuditEntry) (domain.Log, error)
	// Update replaces every field, including the tag and link sets.
	Update(ctx context.Context, l domain.Log, a domain.AuditEntry) (domain.Log, error)
	Delete(ctx context.Context, documentID, id string, a domain.AuditEntry) error
	DeleteExamples(ctx context.Context, documentID string, a domain.AuditEntry) error
	// ListTags returns the tenant's tag vocabulary, sorted.
	ListTags(ctx context.Context) ([]string, error)
	// Dashboard returns sparse aggregates of the document's non-example logs
	// for p; the caller runs Normalize.
	Dashboard(ctx context.Context, documentID string, p domain.Period) (domain.Dashboard, error)
}

// ImpactExtractor finds the impact stated in a log's text (PRD-0007).
// It returns "" when the text states none.
type ImpactExtractor interface {
	Extract(ctx context.Context, name, description string) (string, error)
}
