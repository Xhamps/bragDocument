package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// Logs groups the log use cases; one method per file. Owner-only until
// PRD-0004 adds grants.
type Logs struct {
	docs   ports.DocumentRepo
	logs   ports.LogRepo
	impact ports.ImpactExtractor
	now    func() time.Time
}

// NewLogs wires the use cases.
func NewLogs(docs ports.DocumentRepo, logs ports.LogRepo, impact ports.ImpactExtractor) *Logs {
	return &Logs{docs: docs, logs: logs, impact: impact, now: time.Now}
}

// writable is ownedDocument plus the archive rule: archived documents are read-only.
func (s *Logs) writable(ctx context.Context, docID, userID string) (domain.Document, error) {
	d, err := ownedDocument(ctx, s.docs, docID, userID)
	if err != nil {
		return domain.Document{}, err
	}
	if d.State == domain.DocumentArchived {
		return domain.Document{}, fmt.Errorf("%w: document is archived", domain.ErrConflict)
	}
	return d, nil
}

// extract returns the impact statement, or nil when it could not be checked.
// It never fails the save (PRD-0007 FR-6).
func (s *Logs) extract(ctx context.Context, l domain.Log) *string {
	st, err := s.impact.Extract(ctx, l.Name, l.Description)
	switch {
	case errors.Is(err, domain.ErrUnavailable):
		return nil // extraction disabled; logged once at startup
	case err != nil:
		slog.WarnContext(ctx, "impact extraction failed; saving without a statement", slog.Any("err", err))
		return nil
	}
	return &st
}
