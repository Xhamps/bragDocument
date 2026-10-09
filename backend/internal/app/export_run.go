package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

const (
	reportPageSize = 100             // the list's maximum page
	jobTimeout     = 4 * time.Minute // below the 5-minute reclaim in ClaimExportJob
)

// RunNext claims and runs one job. It returns false when the queue is empty.
// A job that fails is marked failed; only repository errors come back.
func (s *Exports) RunNext(ctx context.Context) (bool, error) {
	j, err := s.jobs.Claim(ctx)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	ctx = s.scope(ctx, j.TenantID)
	runCtx, cancel := context.WithTimeout(ctx, jobTimeout)
	key, err := s.run(runCtx, j)
	cancel()
	if err != nil {
		slog.WarnContext(ctx, "export failed", slog.String("job_id", j.ID), slog.Any("err", err))
		return true, superseded(ctx, j.ID, s.jobs.Fail(ctx, j.ID, failReason(err)))
	}
	slog.InfoContext(ctx, "export done", slog.String("job_id", j.ID))
	return true, superseded(ctx, j.ID, s.jobs.Finish(ctx, j.ID, key))
}

// superseded swallows ErrNotFound from Finish/Fail: the job was reclaimed by
// another worker (it is no longer running), whose result wins.
func superseded(ctx context.Context, jobID string, err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		slog.InfoContext(ctx, "export superseded", slog.String("job_id", jobID))
		return nil
	}
	return err
}

func (s *Exports) run(ctx context.Context, j domain.ExportJob) (string, error) {
	// FR-6: the requester's access as of now, not as of the request.
	d, err := access(ctx, s.docs, j.DocumentID, j.RequestedBy, domain.PermRead)
	if err != nil {
		return "", err
	}
	f := j.Params.Filter
	f.Sort, f.Desc, f.PerPage = "created_at", false, reportPageSize
	var logs []domain.Log
	for f.Page = 1; len(logs) < domain.MaxReportLogs; f.Page++ {
		page, err := s.logs.List(ctx, d.ID, f)
		if err != nil {
			return "", err
		}
		logs = append(logs, page.Items...)
		s.setProgress(ctx, j.ID, 80*len(logs)/max(page.Total, 1))
		if len(page.Items) < reportPageSize {
			break
		}
	}
	r := domain.NewReport(logs, j.Params.Settings)
	r.Title, r.Author, r.Tenant, r.From, r.To, r.GeneratedAt = d.Title, d.OwnerName, j.Params.TenantName, f.From, f.To, s.now()
	pdf, err := s.pdf.Render(ctx, r)
	if err != nil {
		return "", err
	}
	s.setProgress(ctx, j.ID, 95)
	key := j.TenantID + "/" + j.ID + ".pdf"
	return key, s.files.Put(ctx, key, pdf)
}

// failReason is what the user sees in the dialog.
func failReason(err error) string {
	var ae *domain.AccessError
	switch {
	case errors.Is(err, domain.ErrNotFound), errors.As(err, &ae):
		return "You no longer have access to this document."
	case errors.Is(err, domain.ErrUnavailable):
		return "The PDF service is unavailable. Try again in a minute."
	}
	return "Generating the report failed. Try again."
}

// Cleanup deletes expired jobs and their files (FR-5, NFR-2). The file goes
// first: a crash in between leaves a row the next tick retries, never an orphan file.
func (s *Exports) Cleanup(ctx context.Context) error {
	jobs, err := s.jobs.Expired(ctx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.FileKey != "" {
			if err := s.files.Delete(ctx, j.FileKey); err != nil {
				return err
			}
		}
		if err := s.jobs.Delete(ctx, j.ID); err != nil {
			return err
		}
	}
	return nil
}
