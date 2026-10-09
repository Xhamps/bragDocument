package ports

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// ExportRepo stores export jobs and report settings (PRD-0006).
type ExportRepo interface {
	Create(ctx context.Context, j domain.ExportJob) (domain.ExportJob, error)
	// Get returns domain.ErrNotFound when the job is not on the document.
	Get(ctx context.Context, documentID, id string) (domain.ExportJob, error)
	// List returns the user's unexpired jobs on the document, newest first, at most 10.
	List(ctx context.Context, documentID, userID string) ([]domain.ExportJob, error)
	// Claim marks the oldest runnable job running, across tenants. domain.ErrNotFound: queue empty.
	Claim(ctx context.Context) (domain.ExportJob, error)
	Finish(ctx context.Context, id, fileKey string) error
	Fail(ctx context.Context, id, reason string) error
	// Expired and Delete look across tenants (worker cleanup).
	Expired(ctx context.Context) ([]domain.ExportJob, error)
	Delete(ctx context.Context, id string) error
	// Settings returns domain.ErrNotFound before the first export.
	Settings(ctx context.Context, documentID string) (domain.ReportSettings, error)
	SaveSettings(ctx context.Context, tenantID, documentID string, s domain.ReportSettings) error
}

// ReportRenderer turns a report into PDF bytes (ADR-0010). An unreachable
// renderer returns domain.ErrUnavailable.
type ReportRenderer interface {
	Render(ctx context.Context, r domain.Report) ([]byte, error)
}

// FileStore keeps export files, encrypted at rest (PRD-0006 NFR-2).
// Get returns domain.ErrNotFound for a missing file; Delete is idempotent.
type FileStore interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
