package app

import (
	"context"
	"fmt"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// CreateExportInput is one "Generate" click. TenantName goes on the cover.
type CreateExportInput struct {
	DocumentID, UserID, TenantName string
	Filter                         domain.LogFilter
	Settings                       domain.ReportSettings
}

// Create checks the size (FR-7), saves the settings for next time, and queues the job.
// Viewers can export but not change the document's shared goals and mapping.
func (s *Exports) Create(ctx context.Context, in CreateExportInput) (domain.ExportJob, error) {
	d, err := access(ctx, s.docs, in.DocumentID, in.UserID, domain.PermRead)
	if err != nil {
		return domain.ExportJob{}, err
	}
	f := in.Filter
	f.HideExamples, f.Page, f.PerPage = true, 1, 1
	if err := f.Validate(); err != nil {
		return domain.ExportJob{}, err
	}
	if err := in.Settings.Validate(); err != nil {
		return domain.ExportJob{}, err
	}
	page, err := s.logs.List(ctx, d.ID, f)
	if err != nil {
		return domain.ExportJob{}, err
	}
	switch {
	case page.Total == 0:
		return domain.ExportJob{}, domain.NewValidationError(map[string]string{"filters": "no logs match these filters"})
	case page.Total > domain.MaxReportLogs:
		return domain.ExportJob{}, domain.NewValidationError(map[string]string{"filters": fmt.Sprintf(
			"%d logs match; narrow the filters to at most %d", page.Total, domain.MaxReportLogs)})
	}
	if domain.Can(d.Role, domain.PermWriteLogs) {
		if err := s.jobs.SaveSettings(ctx, d.TenantID, d.ID, in.Settings); err != nil {
			return domain.ExportJob{}, err
		}
	}
	return s.jobs.Create(ctx, domain.ExportJob{TenantID: d.TenantID, DocumentID: d.ID, RequestedBy: in.UserID,
		Params: domain.ExportParams{Filter: f, Settings: in.Settings, TenantName: in.TenantName}})
}
