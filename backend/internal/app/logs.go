package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// Logs groups the log use cases; one method per file. Access per PRD-0004:
// reads need PermRead, writes PermWriteLogs.
type Logs struct {
	docs   ports.DocumentRepo
	logs   ports.LogRepo
	impact ports.ImpactExtractor
	cache  ports.Cache
	now    func() time.Time
}

// NewLogs wires the use cases.
func NewLogs(docs ports.DocumentRepo, logs ports.LogRepo, impact ports.ImpactExtractor, cache ports.Cache) *Logs {
	return &Logs{docs: docs, logs: logs, impact: impact, cache: cache, now: time.Now}
}

const (
	dashboardTTL  = 60 * time.Second // PRD-0005 NFR-1, ADR-0006
	docVersionTTL = 24 * time.Hour   // outlives every entry keyed on it
)

func docVersionKey(docID string) string { return "doc:" + docID + ":v" }

// touch invalidates the document's cached aggregates (ADR-0006): a fresh
// version makes every older entry unreachable. The cache is Degrading, so a
// failed write costs at most one TTL of staleness, never the request.
func (s *Logs) touch(ctx context.Context, docID string) {
	_ = s.cache.Set(ctx, docVersionKey(docID), []byte(rand.Text()), docVersionTTL)
}

// writable checks PermWriteLogs plus the archive rule: archived documents are read-only.
func (s *Logs) writable(ctx context.Context, docID, userID string) (domain.Document, error) {
	d, err := access(ctx, s.docs, docID, userID, domain.PermWriteLogs)
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
	case err != nil && ctx.Err() != nil:
		return nil // client went away; the save fails on the same ctx anyway
	case err != nil:
		slog.WarnContext(ctx, "impact extraction failed; saving without a statement",
			slog.String("document_id", l.DocumentID), slog.Any("err", err))
		return nil
	}
	return &st
}

// checkDate rejects future dates. The 24h slack covers clients sending noon
// local time for today.
func (s *Logs) checkDate(l domain.Log) error {
	if l.CreatedAt.After(s.now().Add(24 * time.Hour)) {
		return domain.NewValidationError(map[string]string{"created_at": "must not be in the future"})
	}
	return nil
}
