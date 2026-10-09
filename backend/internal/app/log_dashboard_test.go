package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestDashboardCachesUntilAWrite(t *testing.T) {
	f := newLogsFixture()
	ctx := context.Background()

	d, err := f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	require.Len(t, d.Months, 13, "normalized: the default 12-month window ending mid-month overlaps 13 months")
	cached, err := f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, d, cached, "a hit equals the fresh result")
	require.Equal(t, 1, f.logs.dashCalls, "second read is a hit")

	_, err = f.s.Create(ctx, createIn("d1"))
	require.NoError(t, err)
	d, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 2, f.logs.dashCalls, "a write invalidates")
	require.Equal(t, 1, d.Total)

	_, err = f.s.Dashboard(ctx, "d3", "u1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 3, f.logs.dashCalls, "other documents keep their own entries")
}

func TestDashboardInvalidatedByEveryWrite(t *testing.T) {
	ctx := context.Background()
	writes := map[string]func(f logsFixture, logID string) error{
		"update": func(f logsFixture, id string) error {
			n := "renamed"
			_, err := f.s.Update(ctx, UpdateLogInput{ID: id, DocumentID: "d1", UserID: "u1", Name: &n})
			return err
		},
		"delete":          func(f logsFixture, id string) error { return f.s.Delete(ctx, "d1", id, "u1") },
		"delete examples": func(f logsFixture, _ string) error { return f.s.DeleteExamples(ctx, "d1", "u1") },
	}
	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			f := newLogsFixture()
			l, err := f.s.Create(ctx, createIn("d1"))
			require.NoError(t, err)
			_, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
			require.NoError(t, err)
			require.NoError(t, write(f, l.ID))
			_, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
			require.NoError(t, err)
			require.Equal(t, 2, f.logs.dashCalls)
		})
	}
}

func TestDashboardWithoutCache(t *testing.T) {
	f := newLogsFixture()
	f.cache.err = errors.New("redis down")
	for range 2 {
		_, err := f.s.Dashboard(context.Background(), "d1", "u1", nil, nil)
		require.NoError(t, err)
	}
	require.Equal(t, 2, f.logs.dashCalls, "every read goes to Postgres")
}

func TestDashboardCorruptEntryIsAMiss(t *testing.T) {
	f := newLogsFixture()
	ctx := context.Background()
	_, err := f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	for k := range f.cache.data {
		if strings.HasPrefix(k, "dash:") {
			f.cache.data[k] = []byte("{")
		}
	}
	_, err = f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err)
	require.Equal(t, 2, f.logs.dashCalls)
}

func TestDashboardRules(t *testing.T) {
	f := newLogsFixture()
	ctx := context.Background()
	_, err := f.s.Dashboard(ctx, "d1", "u1", nil, nil)
	require.NoError(t, err, "warm the cache")
	_, err = f.s.Dashboard(ctx, "d1", "u9", nil, nil)
	require.ErrorIs(t, err, domain.ErrNotFound, "no grant, even with a warm cache")
	_, err = f.s.Dashboard(ctx, "d1", "u2", nil, nil)
	require.NoError(t, err, "viewers read the dashboard (FR-5)")

	from, to := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	_, err = f.s.Dashboard(ctx, "d1", "u1", &from, &to)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	require.Equal(t, 1, f.logs.dashCalls, "one read, then hits: access and validation run before any read")
}
