//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestLogRepo(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, logs := NewUserRepo(db), NewDocumentRepo(db), NewLogRepo(db)
	admin, tenantA := provisionTenant(t, users, "A", "a@example.com")
	adminB, tenantB := provisionTenant(t, users, "B", "b@example.com")
	ctx := telemetry.WithTenantID(context.Background(), tenantA.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tenantB.ID)

	now := time.Now().UTC().Truncate(time.Microsecond)
	examples := domain.ExampleLogs(now, admin.ID)
	for i := range examples {
		require.NoError(t, examples[i].Validate())
	}
	doc, err := docs.Create(ctx, domain.Document{TenantID: tenantA.ID, OwnerID: admin.ID, Title: "2026"}, examples, docCreated(admin.ID))
	require.NoError(t, err)

	page, err := logs.List(ctx, doc.ID, domain.LogFilter{Sort: "created_at", Desc: true, Page: 1, PerPage: 50})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.True(t, page.Items[0].IsExample)
	require.NotEmpty(t, page.Items[0].Tags)

	// Document counters exclude examples.
	list, err := docs.ListByOwner(ctx, admin.ID)
	require.NoError(t, err)
	require.Equal(t, 0, list[0].LogCount)
	require.Nil(t, list[0].LastLogAt)

	require.NoError(t, logs.DeleteExamples(ctx, doc.ID, logEntry(admin.ID, doc.ID, domain.AuditLogDeleted)))

	mk := func(name, desc, impact, status string, daysAgo int, tags []string, links []domain.Link) domain.Log {
		l := domain.Log{TenantID: tenantA.ID, DocumentID: doc.ID, Name: name, Description: desc, Impact: impact,
			Status: status, Tags: tags, Links: links, CreatedAt: now.AddDate(0, 0, -daysAgo), CreatedBy: admin.ID, UpdatedBy: admin.ID}
		require.NoError(t, l.Validate())
		out, err := logs.Create(ctx, l, logEntry(admin.ID, doc.ID, domain.AuditLogCreated))
		require.NoError(t, err)
		return out
	}
	none := ""
	a := mk("Database migration", "moved 100% of tables", "high", "done", 30, []string{"project"},
		[]domain.Link{{URL: "https://github.com/x/y/pull/1"}, {URL: "https://docs.example.com/a", Label: "doc"}})
	b := mk("Mentoring", "weekly pairing", "medium", "in_progress", 10, []string{"mentorship"}, nil)
	c := mk("Idea: cache", "", "critical", "idea", 1, nil, []domain.Link{{URL: "https://gist.github.com/z"}})

	type q = domain.LogFilter
	names := func(f domain.LogFilter) []string {
		t.Helper()
		require.NoError(t, f.Validate())
		p, err := logs.List(ctx, doc.ID, f)
		require.NoError(t, err)
		out := []string{}
		for _, l := range p.Items {
			out = append(out, l.Name)
		}
		return out
	}
	require.Equal(t, []string{c.Name, b.Name, a.Name}, names(q{}))
	require.Equal(t, []string{a.Name}, names(q{Query: "MIGRATION"}))
	require.Equal(t, []string{a.Name}, names(q{Query: "100%"}), "% is literal")
	require.Empty(t, names(q{Query: "_"}), "_ is literal")
	require.Equal(t, []string{b.Name, a.Name}, names(q{Tags: []string{"mentorship", "project"}}))
	require.Equal(t, []string{c.Name, a.Name}, names(q{Statuses: []string{"idea", "done"}}))
	require.Equal(t, []string{c.Name, b.Name}, names(q{Impacts: []string{"critical", "medium"}}))
	require.Equal(t, []string{c.Name, a.Name}, names(q{Domain: "github.com"}), "subdomains match")
	require.Equal(t, []string{a.Name}, names(q{Domain: "docs.example.com"}))
	from, to := now.AddDate(0, 0, -15), now.AddDate(0, 0, -5)
	require.Equal(t, []string{b.Name}, names(q{From: &from, To: &to}))
	require.Equal(t, []string{c.Name, a.Name, b.Name}, names(q{Sort: "impact", Desc: true}))
	require.Equal(t, []string{c.Name, b.Name, a.Name}, names(q{Sort: "status"}))
	require.Equal(t, []string{a.Name, c.Name, b.Name}, names(q{Sort: "name"}))

	// Paging: total rides on every page, and past the end.
	f := q{PerPage: 2, Page: 2}
	require.NoError(t, f.Validate())
	p, err := logs.List(ctx, doc.ID, f)
	require.NoError(t, err)
	require.Equal(t, 3, p.Total)
	require.Len(t, p.Items, 1)
	f.Page = 9
	p, err = logs.List(ctx, doc.ID, f)
	require.NoError(t, err)
	require.Equal(t, 3, p.Total)
	require.Empty(t, p.Items)

	// Update replaces tags and links and stores the statement.
	b.Tags, b.Links, b.ImpactStatement = []string{"learning"}, []domain.Link{{URL: "https://a.io", Host: "a.io"}}, &none
	got, err := logs.Update(ctx, b, logEntry(admin.ID, doc.ID, domain.AuditLogEdited))
	require.NoError(t, err)
	got, err = logs.Get(ctx, doc.ID, got.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"learning"}, got.Tags)
	require.Len(t, got.Links, 1)
	require.Equal(t, "", *got.ImpactStatement)
	tags, err := logs.ListTags(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"collaboration", "company-building", "documentation", "learning", "mentorship", "project"}, tags)

	// Counters now count real logs.
	list, err = docs.ListByOwner(ctx, admin.ID)
	require.NoError(t, err)
	require.Equal(t, 3, list[0].LogCount)
	require.NotNil(t, list[0].LastLogAt)

	// Tenant isolation on every table.
	pB, err := logs.List(ctxB, doc.ID, q{Page: 1, PerPage: 50, Sort: "created_at"})
	require.NoError(t, err)
	require.Zero(t, pB.Total)
	_, err = logs.Get(ctxB, doc.ID, a.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.ErrorIs(t, logs.Delete(ctxB, doc.ID, a.ID, logEntry(adminB.ID, doc.ID, domain.AuditLogDeleted)), domain.ErrNotFound)
	tagsB, err := logs.ListTags(ctxB)
	require.NoError(t, err)
	require.Empty(t, tagsB)
	aB := a
	aB.Name, aB.UpdatedBy = "hijacked", adminB.ID
	_, err = logs.Update(ctxB, aB, logEntry(adminB.ID, doc.ID, domain.AuditLogEdited))
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.ErrorIs(t, logs.DeleteExamples(ctxB, doc.ID, logEntry(adminB.ID, doc.ID, domain.AuditLogDeleted)), domain.ErrNotFound,
		"the entry names a document B cannot see")
	pA, err := logs.List(ctx, doc.ID, q{Page: 1, PerPage: 50, Sort: "created_at"})
	require.NoError(t, err)
	require.Equal(t, 3, pA.Total)
	// The composite FK (document_id, tenant_id) finds no B document with A's
	// id (FK checks bypass RLS), so the insert is a 23503 -> ErrConflict.
	_, err = logs.Create(ctxB, domain.Log{TenantID: tenantB.ID, DocumentID: doc.ID, Name: "x", Impact: "low",
		Status: "done", CreatedAt: now, CreatedBy: adminB.ID, UpdatedBy: adminB.ID}, logEntry(adminB.ID, doc.ID, domain.AuditLogCreated))
	require.ErrorIs(t, err, domain.ErrConflict)

	// Update of a missing log.
	missing := b
	missing.ID = uuid.NewString()
	_, err = logs.Update(ctx, missing, logEntry(admin.ID, doc.ID, domain.AuditLogEdited))
	require.ErrorIs(t, err, domain.ErrNotFound)

	// Delete and cascade.
	require.NoError(t, logs.Delete(ctx, doc.ID, c.ID, logEntry(admin.ID, doc.ID, domain.AuditLogDeleted)))
	require.ErrorIs(t, logs.Delete(ctx, doc.ID, c.ID, logEntry(admin.ID, doc.ID, domain.AuditLogDeleted)), domain.ErrNotFound)
	require.NoError(t, docs.Delete(ctx, doc.ID, docDeleted(admin.ID, doc.ID)))
	_, err = logs.Get(ctx, doc.ID, a.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

// TestLogListTiming reports (does not assert) the list latency at 10k logs (NFR-1).
func TestLogListTiming(t *testing.T) {
	ctx, doc, logs := seedTenThousand(t)
	for _, f := range []domain.LogFilter{
		{},
		{Query: "migrations", Impacts: []string{"high", "critical"}},
		{Tags: []string{"project"}, Sort: "impact", Desc: true, Page: 50},
	} {
		require.NoError(t, f.Validate())
		start := time.Now()
		p, err := logs.List(ctx, doc.ID, f)
		require.NoError(t, err)
		t.Logf("filter %+v: total=%d in %s", f, p.Total, time.Since(start))
	}
}

// seedTenThousand creates one document with 10,000 logs spread hourly back
// from now, each tagged project or learning.
func seedTenThousand(t *testing.T) (context.Context, domain.Document, *LogRepo) {
	t.Helper()
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 30*time.Second)
	require.NoError(t, err)
	t.Cleanup(db.Close)
	users, docs, logs := NewUserRepo(db), NewDocumentRepo(db), NewLogRepo(db)
	admin, tn := provisionTenant(t, users, "A", "a@example.com")
	ctx := telemetry.WithTenantID(context.Background(), tn.ID)
	doc, err := docs.Create(ctx, domain.Document{TenantID: tn.ID, OwnerID: admin.ID, Title: "big"}, nil, docCreated(admin.ID))
	require.NoError(t, err)

	owner, err := Connect(context.Background(), ownerURL, 60*time.Second)
	require.NoError(t, err)
	defer owner.Close()
	_, err = owner.Pool.Exec(context.Background(), `
		INSERT INTO tags (tenant_id, name) VALUES ($1, 'project'), ($1, 'learning');
		`, tn.ID)
	require.NoError(t, err)
	_, err = owner.Pool.Exec(context.Background(), `
		INSERT INTO logs (tenant_id, document_id, name, description, impact, status, created_at, created_by, updated_by)
		SELECT $1, $2, 'log ' || i, 'description number ' || i || ' about migrations and things',
		       (ARRAY['low','medium','high','critical'])[1 + i % 4], (ARRAY['idea','in_progress','done','dropped'])[1 + i % 4],
		       now() - (i || ' hours')::interval, $3, $3
		FROM generate_series(1, 10000) i`, tn.ID, doc.ID, admin.ID)
	require.NoError(t, err)
	_, err = owner.Pool.Exec(context.Background(), `
		INSERT INTO log_tags (tenant_id, log_id, tag_name)
		SELECT tenant_id, id, CASE WHEN random() < 0.5 THEN 'project' ELSE 'learning' END FROM logs;
		ANALYZE logs; ANALYZE log_tags;`)
	require.NoError(t, err)
	return ctx, doc, logs
}
