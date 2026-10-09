package app

import "context"

// Tags returns the tenant's tag vocabulary for autocomplete (PRD-0002 FR-9).
func (s *Logs) Tags(ctx context.Context) ([]string, error) { return s.logs.ListTags(ctx) }
