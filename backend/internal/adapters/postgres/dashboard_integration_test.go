//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestDashboard(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, logs := NewUserRepo(db), NewDocumentRepo(db), NewLogRepo(db)
	admin, ta := provisionTenant(t, users, "A", "a@example.com")
	_, tb := provisionTenant(t, users, "B", "b@example.com")
	ctx := telemetry.WithTenantID(context.Background(), ta.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tb.ID)

	examples := domain.ExampleLogs(time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC), admin.ID)
	doc, err := docs.Create(ctx, domain.Document{TenantID: ta.ID, OwnerID: admin.ID, Title: "2026"}, examples, docCreated(admin.ID))
	require.NoError(t, err)

	mk := func(name, impact, status string, at time.Time, tags ...string) {
		l := domain.Log{TenantID: ta.ID, DocumentID: doc.ID, Name: name, Impact: impact, Status: status,
			Tags: tags, Links: []domain.Link{}, CreatedAt: at, CreatedBy: admin.ID, UpdatedBy: admin.ID}
		require.NoError(t, l.Validate())
		_, err := logs.Create(ctx, l, logEntry(admin.ID, doc.ID, domain.AuditLogCreated))
		require.NoError(t, err)
	}
	at := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 12, 0, 0, 0, time.UTC) }
	mk("a", "high", "done", at(1, 15), "project", "mentorship")
	mk("b", "critical", "in_progress", at(3, 2), "project")
	mk("c", "low", "idea", at(3, 31), "learning")
	mk("d", "medium", "done", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), "project") // before the period
	mk("e", "high", "in_progress", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))       // the exclusive end

	p := domain.Period{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)}
	d, err := logs.Dashboard(ctx, doc.ID, p)
	require.NoError(t, err)
	require.Equal(t, 5, d.Total, "all-time, examples excluded")
	require.Equal(t, 3, d.InPeriod)
	require.Equal(t, 2, d.HighImpact)
	require.Equal(t, 1, d.InProgress)

	d.Normalize()
	b := func(k string, n int) domain.Bucket { return domain.Bucket{Key: k, Count: n} }
	require.Equal(t, []domain.Bucket{b("2026-01", 1), b("2026-02", 0), b("2026-03", 2)}, d.Months, "the example in February does not count")
	require.Equal(t, []domain.Bucket{b("project", 2), b("learning", 1), b("mentorship", 1)}, d.Tags)
	require.Equal(t, []domain.Bucket{b("idea", 1), b("in_progress", 1), b("done", 1), b("dropped", 0)}, d.Statuses)
	require.Equal(t, []domain.Bucket{b("low", 1), b("medium", 0), b("high", 1), b("critical", 1)}, d.Impacts)
	require.Equal(t, b("mentorship", 1), d.Coverage[2])

	// FR-6: every slice equals the list's total for the same filter.
	count := func(f domain.LogFilter) int {
		f.From, f.To, f.HideExamples = &p.From, &p.To, true
		require.NoError(t, f.Validate())
		page, err := logs.List(ctx, doc.ID, f)
		require.NoError(t, err)
		return page.Total
	}
	require.Equal(t, d.InPeriod, count(domain.LogFilter{}))
	require.Equal(t, d.Tags[0].Count, count(domain.LogFilter{Tags: []string{"project"}}))
	require.Equal(t, d.HighImpact, count(domain.LogFilter{Impacts: []string{"high", "critical"}}))
	require.Equal(t, d.InProgress, count(domain.LogFilter{Statuses: []string{"in_progress"}}))
	marFrom, marTo := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), p.To
	mar := domain.LogFilter{From: &marFrom, To: &marTo, HideExamples: true}
	require.NoError(t, mar.Validate())
	marPage, err := logs.List(ctx, doc.ID, mar)
	require.NoError(t, err)
	require.Equal(t, d.Months[2].Count, marPage.Total, "the March bar links to March's list")
	all := domain.LogFilter{}
	require.NoError(t, all.Validate())
	page, err := logs.List(ctx, doc.ID, all)
	require.NoError(t, err)
	require.Equal(t, 5+len(examples), page.Total, "without examples=false the list still shows examples")

	// Other tenants see nothing.
	d, err = logs.Dashboard(ctxB, doc.ID, p)
	require.NoError(t, err)
	require.Zero(t, d.Total)
}

// TestDashboardTiming reports the dashboard latency at 10k logs (NFR-1). The
// bound is generous: NFR-1's 500 ms p95 includes the cache.
func TestDashboardTiming(t *testing.T) {
	ctx, doc, logs := seedTenThousand(t)
	p, err := domain.NewPeriod(nil, nil, time.Now())
	require.NoError(t, err)
	start := time.Now()
	d, err := logs.Dashboard(ctx, doc.ID, p)
	elapsed := time.Since(start)
	require.NoError(t, err)
	require.Equal(t, 10000, d.Total)
	require.Positive(t, d.InPeriod)
	t.Logf("dashboard over %d logs (%d in period) in %s", d.Total, d.InPeriod, elapsed)
	require.Less(t, elapsed, 2*time.Second)
}
