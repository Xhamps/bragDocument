package app

import (
	"context"
	"strconv"
	"time"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/ports"
)

// Exports is the PDF report (PRD-0006): request, status, download, and the
// worker's run and cleanup. Everything needs PermRead (FR-6).
type Exports struct {
	docs  ports.DocumentRepo
	logs  ports.LogRepo
	jobs  ports.ExportRepo
	pdf   ports.ReportRenderer
	files ports.FileStore
	cache ports.Cache // Degrading: progress is cosmetic
	// scope puts the tenant in the context (telemetry.WithTenantID); injected
	// because app may not import telemetry.
	scope func(ctx context.Context, tenantID string) context.Context
	now   func() time.Time
}

// NewExports wires the use cases.
func NewExports(docs ports.DocumentRepo, logs ports.LogRepo, jobs ports.ExportRepo, pdf ports.ReportRenderer,
	files ports.FileStore, cache ports.Cache, scope func(context.Context, string) context.Context) *Exports {
	return &Exports{docs: docs, logs: logs, jobs: jobs, pdf: pdf, files: files, cache: cache, scope: scope, now: time.Now}
}

const progressTTL = time.Hour

func progressKey(jobID string) string { return "export:" + jobID + ":progress" }

func (s *Exports) setProgress(ctx context.Context, jobID string, pct int) {
	_ = s.cache.Set(ctx, progressKey(jobID), []byte(strconv.Itoa(pct)), progressTTL)
}

func (s *Exports) progress(ctx context.Context, j domain.ExportJob) int {
	switch j.Status {
	case domain.ExportDone:
		return 100
	case domain.ExportRunning:
		if b, ok, _ := s.cache.Get(ctx, progressKey(j.ID)); ok {
			n, _ := strconv.Atoi(string(b))
			return n
		}
	}
	return 0
}
