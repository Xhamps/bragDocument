package app

import (
	"context"
	"slices"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// UpdateLogInput: nil fields are unchanged; Tags and Links replace the whole set.
type UpdateLogInput struct {
	ID          string
	DocumentID  string
	UserID      string
	Name        *string
	Description *string
	Impact      *string
	Status      *string
	Tags        *[]string
	Links       *[]domain.Link
	CreatedAt   *time.Time
}

// Update edits a log. The impact statement is re-extracted only when the name
// or description changed, which also retries a statement that was never checked.
func (s *Logs) Update(ctx context.Context, in UpdateLogInput) (domain.Log, error) {
	if _, err := s.writable(ctx, in.DocumentID, in.UserID); err != nil {
		return domain.Log{}, err
	}
	l, err := s.logs.Get(ctx, in.DocumentID, in.ID)
	if err != nil {
		return domain.Log{}, err
	}
	old := l
	old.Tags, old.Links = slices.Clone(l.Tags), slices.Clone(l.Links) // Validate normalizes l's in place
	if in.Name != nil {
		l.Name = *in.Name
	}
	if in.Description != nil {
		l.Description = *in.Description
	}
	if in.Impact != nil {
		l.Impact = *in.Impact
	}
	if in.Status != nil {
		l.Status = *in.Status
	}
	if in.Tags != nil {
		l.Tags = *in.Tags
	}
	if in.Links != nil {
		l.Links = *in.Links
	}
	if in.CreatedAt != nil {
		l.CreatedAt = *in.CreatedAt
	}
	if err := s.checkDate(l); err != nil {
		return domain.Log{}, err
	}
	l.IsExample = false // an edited example is the user's log now
	l.UpdatedBy = in.UserID
	if err := l.Validate(); err != nil {
		return domain.Log{}, err
	}
	action, fields := domain.LogChange(old, l)
	if action == "" && !old.IsExample {
		return old, nil // nothing changed: no write, no entry
	}
	if action == "" {
		action = domain.AuditLogEdited // saving an example unchanged still makes it the user's log
	}
	if l.Name != old.Name || l.Description != old.Description {
		l.ImpactStatement = s.extract(ctx, l)
	}
	a := entry(ctx, in.UserID, action, in.DocumentID)
	a.TargetType, a.TargetID, a.Target, a.ChangedFields = domain.TargetLog, l.ID, l.Name, fields
	out, err := s.logs.Update(ctx, l, a)
	if err == nil {
		s.touch(ctx, in.DocumentID)
	}
	return out, err
}
