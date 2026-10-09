package app

import (
	"context"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Get returns one of the caller's jobs with its progress. Another user's job is not found.
func (s *Exports) Get(ctx context.Context, docID, jobID, userID string) (domain.ExportJob, error) {
	if _, err := access(ctx, s.docs, docID, userID, domain.PermRead); err != nil {
		return domain.ExportJob{}, err
	}
	j, err := s.jobs.Get(ctx, docID, jobID)
	if err != nil {
		return domain.ExportJob{}, err
	}
	if j.RequestedBy != userID {
		return domain.ExportJob{}, domain.ErrNotFound
	}
	j.Progress = s.progress(ctx, j)
	return j, nil
}

// List is the dialog's history: the caller's live jobs, newest first.
func (s *Exports) List(ctx context.Context, docID, userID string) ([]domain.ExportJob, error) {
	if _, err := access(ctx, s.docs, docID, userID, domain.PermRead); err != nil {
		return nil, err
	}
	jobs, err := s.jobs.List(ctx, docID, userID)
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		jobs[i].Progress = s.progress(ctx, jobs[i])
	}
	return jobs, nil
}

// Open returns the decrypted PDF while it is downloadable (FR-5). Access is
// checked again, so a revoked reader cannot download an old report.
func (s *Exports) Open(ctx context.Context, docID, jobID, userID string) ([]byte, error) {
	j, err := s.Get(ctx, docID, jobID, userID)
	if err != nil {
		return nil, err
	}
	if !j.Downloadable(s.now()) {
		return nil, domain.ErrNotFound
	}
	return s.files.Get(ctx, j.FileKey)
}
