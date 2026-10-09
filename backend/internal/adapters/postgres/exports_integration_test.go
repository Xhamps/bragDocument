//go:build integration

package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestExports(t *testing.T) {
	ownerURL := startPostgres(t)
	require.NoError(t, Migrate(ownerURL))
	db, err := Connect(context.Background(), appRoleURL(t, ownerURL), 5*time.Second)
	require.NoError(t, err)
	defer db.Close()

	users, docs, exports := NewUserRepo(db), NewDocumentRepo(db), NewExportRepo(db)
	admin, ta := provisionTenant(t, users, "A", "a@example.com")
	_, tb := provisionTenant(t, users, "B", "b@example.com")
	ctx := telemetry.WithTenantID(context.Background(), ta.ID)
	ctxB := telemetry.WithTenantID(context.Background(), tb.ID)
	doc, err := docs.Create(ctx, domain.Document{TenantID: ta.ID, OwnerID: admin.ID, Title: "2026"}, nil)
	require.NoError(t, err)

	// Settings round trip.
	_, err = exports.Settings(ctx, doc.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	s := domain.ReportSettings{GoalsThisYear: "ship", SectionMap: map[string]string{"x": "Projects"}}
	require.NoError(t, exports.SaveSettings(ctx, ta.ID, doc.ID, s))
	require.NoError(t, exports.SaveSettings(ctx, ta.ID, doc.ID, s)) // upsert
	got, err := exports.Settings(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, s, got)

	// Create, get, list; tenant B sees nothing.
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	params := domain.ExportParams{Filter: domain.LogFilter{From: &from, Impacts: []string{"high"}}, Settings: s, TenantName: "A"}
	j, err := exports.Create(ctx, domain.ExportJob{TenantID: ta.ID, DocumentID: doc.ID, RequestedBy: admin.ID, Params: params})
	require.NoError(t, err)
	require.Equal(t, domain.ExportQueued, j.Status)
	require.WithinDuration(t, time.Now().Add(24*time.Hour), j.ExpiresAt, time.Minute)
	j2, err := exports.Get(ctx, doc.ID, j.ID)
	require.NoError(t, err)
	require.Equal(t, params.Filter.Impacts, j2.Params.Filter.Impacts)
	require.True(t, from.Equal(*j2.Params.Filter.From))
	list, err := exports.List(ctx, doc.ID, admin.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	_, err = exports.Get(ctxB, doc.ID, j.ID)
	require.ErrorIs(t, err, domain.ErrNotFound, "RLS")

	// Two concurrent claims take the job once.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			_, err := exports.Claim(context.Background())
			results <- err
		})
	}
	wg.Wait()
	close(results)
	var ok, empty int
	for err := range results {
		if err == nil {
			ok++
		} else {
			require.ErrorIs(t, err, domain.ErrNotFound)
			empty++
		}
	}
	require.Equal(t, 1, ok)
	require.Equal(t, 1, empty)

	require.NoError(t, exports.Finish(ctx, j.ID, ta.ID+"/"+j.ID+".pdf"))
	done, err := exports.Get(ctx, doc.ID, j.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ExportDone, done.Status)
	require.Equal(t, ta.ID+"/"+j.ID+".pdf", done.FileKey)

	// Expiry: force it into the past through the owner connection, then clean up.
	owner, err := Connect(context.Background(), ownerURL, 5*time.Second)
	require.NoError(t, err)
	defer owner.Close()
	_, err = owner.Pool.Exec(context.Background(), "UPDATE export_jobs SET expires_at = now() - interval '1 second'")
	require.NoError(t, err)
	expired, err := exports.Expired(context.Background())
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, done.FileKey, expired[0].FileKey)
	require.NoError(t, exports.Delete(context.Background(), j.ID))
	_, err = exports.Get(ctx, doc.ID, j.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}
