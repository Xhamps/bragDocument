package app

import (
	"context"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// CreateLogInput comes from the handler. Status defaults to done and
// CreatedAt to now; a past CreatedAt back-dates the log.
type CreateLogInput struct {
	DocumentID  string
	UserID      string
	Name        string
	Description string
	Impact      string
	Status      string
	Tags        []string
	Links       []domain.Link
	CreatedAt   *time.Time
}

// Create validates, extracts the impact statement, and stores a log.
func (s *Logs) Create(ctx context.Context, in CreateLogInput) (domain.Log, error) {
	d, err := s.writable(ctx, in.DocumentID, in.UserID)
	if err != nil {
		return domain.Log{}, err
	}
	l := domain.Log{TenantID: d.TenantID, DocumentID: d.ID, Name: in.Name, Description: in.Description,
		Impact: in.Impact, Status: in.Status, Tags: in.Tags, Links: in.Links,
		CreatedAt: s.now(), CreatedBy: in.UserID, UpdatedBy: in.UserID}
	if l.Status == "" {
		l.Status = domain.StatusDone
	}
	if in.CreatedAt != nil {
		l.CreatedAt = *in.CreatedAt
	}
	if err := l.Validate(); err != nil {
		return domain.Log{}, err
	}
	l.ImpactStatement = s.extract(ctx, l)
	return s.logs.Create(ctx, l)
}
