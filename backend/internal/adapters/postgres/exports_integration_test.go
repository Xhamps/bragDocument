//go:build integration

package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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
	doc, err := docs.Create(ctx, domain.Document{TenantID: ta.ID, OwnerID: admin.ID, Title: "2026"}, nil, docCreated(admin.ID))
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
	requested := func(actorID string) domain.AuditEntry {
		return domain.AuditEntry{ActorID: actorID, Source: domain.SourceWeb, Action: domain.AuditExportRequested, DocumentID: doc.ID}
	}
	j, err := exports.Create(ctx, domain.ExportJob{TenantID: ta.ID, DocumentID: doc.ID, RequestedBy: admin.ID, Params: params}, requested(admin.ID))
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

	// A running job whose worker died 6 minutes ago is claimed again.
	owner, err := Connect(context.Background(), ownerURL, 5*time.Second)
	require.NoError(t, err)
	defer owner.Close()
	_, err = owner.Pool.Exec(context.Background(), "UPDATE export_jobs SET started_at = now() - interval '6 minutes'")
	require.NoError(t, err)
	re, err := exports.Claim(context.Background())
	require.NoError(t, err)
	require.Equal(t, j.ID, re.ID)

	require.NoError(t, exports.Finish(ctx, j.ID, ta.ID+"/"+j.ID+".pdf"))
	done, err := exports.Get(ctx, doc.ID, j.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ExportDone, done.Status)
	require.Equal(t, ta.ID+"/"+j.ID+".pdf", done.FileKey)
	// A late outcome on a settled job changes nothing.
	require.ErrorIs(t, exports.Finish(ctx, j.ID, "other.pdf"), domain.ErrNotFound)
	require.ErrorIs(t, exports.Fail(ctx, j.ID, "late"), domain.ErrNotFound)

	// Another user's job: claimed, failed, and kept out of admin's list.
	other := uuid.NewString() // requested_by has no FK
	_, err = exports.Create(ctx, domain.ExportJob{TenantID: ta.ID, DocumentID: doc.ID, RequestedBy: other, Params: params}, requested(other))
	require.ErrorIs(t, err, domain.ErrNotFound, "the audit entry names no user")
	_, err = exports.Claim(context.Background())
	require.ErrorIs(t, err, domain.ErrNotFound, "and the job rolled back with it")
	jf, err := exports.Create(ctx, domain.ExportJob{TenantID: ta.ID, DocumentID: doc.ID, RequestedBy: other, Params: params}, requested(admin.ID))
	require.NoError(t, err)
	claimed, err := exports.Claim(context.Background())
	require.NoError(t, err)
	require.Equal(t, jf.ID, claimed.ID)
	require.NoError(t, exports.Fail(ctx, jf.ID, "renderer down"))
	failed, err := exports.Get(ctx, doc.ID, jf.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ExportFailed, failed.Status)
	require.Equal(t, "renderer down", failed.Error)
	list, err = exports.List(ctx, doc.ID, admin.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, j.ID, list[0].ID)

	// Expiry: force it into the past through the owner connection, then clean up.
	_, err = owner.Pool.Exec(context.Background(), "UPDATE export_jobs SET expires_at = now() - interval '1 second'")
	require.NoError(t, err)
	expired, err := exports.Expired(context.Background())
	require.NoError(t, err)
	require.Len(t, expired, 2)
	keys := map[string]string{}
	for _, e := range expired {
		keys[e.ID] = e.FileKey
	}
	require.Equal(t, done.FileKey, keys[j.ID])
	for id := range keys {
		require.NoError(t, exports.Delete(context.Background(), id))
	}
	_, err = exports.Get(ctx, doc.ID, j.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	require.Equal(t, []string{domain.AuditDocumentCreated, domain.AuditExportRequested, domain.AuditExportRequested},
		auditActions(t, db, ta.ID), "one entry per queued job")
}
