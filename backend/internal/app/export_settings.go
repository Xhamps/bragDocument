package app

import (
	"context"
	"errors"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// Settings prefills the export dialog: the last export's goals and mapping, or the defaults.
func (s *Exports) Settings(ctx context.Context, docID, userID string) (domain.ReportSettings, error) {
	if _, err := access(ctx, s.docs, docID, userID, domain.PermRead); err != nil {
		return domain.ReportSettings{}, err
	}
	st, err := s.jobs.Settings(ctx, docID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.DefaultReportSettings(), nil
	}
	return st, err
}
