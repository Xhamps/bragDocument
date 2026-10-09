package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

type logsFixture struct {
	s      *Logs
	docs   *fakeDocs
	logs   *fakeLogs
	impact *fakeImpact
}

func newLogsFixture() logsFixture {
	f := logsFixture{docs: newFakeDocs(), logs: newFakeLogs(), impact: &fakeImpact{statement: "Cut p95 by 4x"}}
	f.docs.docs["d1"] = domain.Document{ID: "d1", TenantID: "t1", OwnerID: "u1", State: domain.DocumentActive}
	f.docs.docs["d2"] = domain.Document{ID: "d2", TenantID: "t1", OwnerID: "u1", State: domain.DocumentArchived}
	f.docs.docs["d3"] = domain.Document{ID: "d3", TenantID: "t1", OwnerID: "u1", State: domain.DocumentActive}
	f.s = NewLogs(f.docs, f.logs, f.impact)
	f.s.now = func() time.Time { return time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC) }
	return f
}

func createIn(doc string) CreateLogInput {
	return CreateLogInput{DocumentID: doc, UserID: "u1", Name: "Shipped X", Description: "p95 4x faster", Impact: "high"}
}

func TestLogsCreate(t *testing.T) {
	f := newLogsFixture()
	l, err := f.s.Create(context.Background(), createIn("d1"))
	require.NoError(t, err)
	require.Equal(t, "t1", l.TenantID)
	require.Equal(t, domain.StatusDone, l.Status, "default status")
	require.Equal(t, "u1", l.CreatedBy)
	require.Equal(t, "u1", l.UpdatedBy)
	require.Equal(t, f.s.now(), l.CreatedAt)
	require.Equal(t, "Cut p95 by 4x", *l.ImpactStatement)

	back := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	in := createIn("d1")
	in.CreatedAt = &back
	l, err = f.s.Create(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, back, l.CreatedAt, "back-dated")
}

func TestLogsCreateRules(t *testing.T) {
	f := newLogsFixture()
	ctx := context.Background()

	_, err := f.s.Create(ctx, createIn("nope"))
	require.ErrorIs(t, err, domain.ErrNotFound)

	in := createIn("d1")
	in.UserID = "u2"
	_, err = f.s.Create(ctx, in)
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = f.s.Create(ctx, createIn("d2"))
	require.ErrorIs(t, err, domain.ErrConflict, "archived is read-only")

	in = createIn("d1")
	in.Impact = "huge"
	_, err = f.s.Create(ctx, in)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	require.Zero(t, f.impact.calls, "invalid input never reaches the LLM")
}

func TestLogsCreateExtraction(t *testing.T) {
	ctx := context.Background()

	f := newLogsFixture()
	f.impact.statement = ""
	l, err := f.s.Create(ctx, createIn("d1"))
	require.NoError(t, err)
	require.NotNil(t, l.ImpactStatement)
	require.Equal(t, "", *l.ImpactStatement, "checked, none found")

	for _, e := range []error{errors.New("boom"), domain.ErrUnavailable} {
		f = newLogsFixture()
		f.impact.err = e
		l, err = f.s.Create(ctx, createIn("d1"))
		require.NoError(t, err, "extraction failure never fails the save")
		require.Nil(t, l.ImpactStatement, "not checked")
	}
}

func TestLogsUpdate(t *testing.T) {
	ctx := context.Background()
	f := newLogsFixture()
	l, err := f.s.Create(ctx, createIn("d1"))
	require.NoError(t, err)
	f.impact.calls = 0

	status := "dropped"
	got, err := f.s.Update(ctx, UpdateLogInput{ID: l.ID, DocumentID: "d1", UserID: "u1", Status: &status})
	require.NoError(t, err)
	require.Equal(t, "dropped", got.Status)
	require.Equal(t, "Cut p95 by 4x", *got.ImpactStatement)
	require.Zero(t, f.impact.calls, "text unchanged: no extraction")

	same := " Shipped X "
	_, err = f.s.Update(ctx, UpdateLogInput{ID: l.ID, DocumentID: "d1", UserID: "u1", Name: &same})
	require.NoError(t, err)
	require.Zero(t, f.impact.calls, "whitespace-only change is no change")

	desc := "now 5x faster"
	f.impact.statement = "5x faster"
	tags := []string{"Project"}
	_, err = f.s.Update(ctx, UpdateLogInput{ID: l.ID, DocumentID: "d1", UserID: "u2", Description: &desc, Tags: &tags})
	require.ErrorIs(t, err, domain.ErrForbidden)
	got, err = f.s.Update(ctx, UpdateLogInput{ID: l.ID, DocumentID: "d1", UserID: "u1", Description: &desc, Tags: &tags})
	require.NoError(t, err)
	require.Equal(t, 1, f.impact.calls)
	require.Equal(t, "5x faster", *got.ImpactStatement)
	require.Equal(t, []string{"project"}, got.Tags)

	_, err = f.s.Update(ctx, UpdateLogInput{ID: "missing", DocumentID: "d1", UserID: "u1"})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestLogsUpdateClearsExampleFlag(t *testing.T) {
	ctx := context.Background()
	f := newLogsFixture()
	f.logs.logs["ex"] = domain.Log{ID: "ex", DocumentID: "d1", Name: "Example", Impact: "low", Status: "done", IsExample: true}
	status := "idea"
	got, err := f.s.Update(ctx, UpdateLogInput{ID: "ex", DocumentID: "d1", UserID: "u1", Status: &status})
	require.NoError(t, err)
	require.False(t, got.IsExample, "an edited example is the user's log now")
}

func TestLogsList(t *testing.T) {
	ctx := context.Background()
	f := newLogsFixture()
	_, err := f.s.List(ctx, "d1", "u1", domain.LogFilter{Sort: "title"})
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)

	_, err = f.s.List(ctx, "d1", "u2", domain.LogFilter{})
	require.ErrorIs(t, err, domain.ErrForbidden)

	_, err = f.s.List(ctx, "d2", "u1", domain.LogFilter{})
	require.NoError(t, err, "archived documents are readable")
	require.Equal(t, 50, f.logs.filter.PerPage, "defaults applied before the repo")
}

func TestLogsDelete(t *testing.T) {
	ctx := context.Background()
	f := newLogsFixture()
	l, err := f.s.Create(ctx, createIn("d1"))
	require.NoError(t, err)
	require.ErrorIs(t, f.s.Delete(ctx, "d1", l.ID, "u2"), domain.ErrForbidden)
	require.NoError(t, f.s.Delete(ctx, "d1", l.ID, "u1"))
	require.ErrorIs(t, f.s.Delete(ctx, "d1", l.ID, "u1"), domain.ErrNotFound)

	require.NoError(t, f.s.DeleteExamples(ctx, "d1", "u1"))
	require.Equal(t, 1, f.logs.examples)
	require.ErrorIs(t, f.s.DeleteExamples(ctx, "d2", "u1"), domain.ErrConflict)
}

func TestLogsTags(t *testing.T) {
	tags, err := newLogsFixture().s.Tags(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"project"}, tags)
}

func TestLogsFutureDate(t *testing.T) {
	ctx := context.Background()
	f := newLogsFixture()
	ok := f.s.now().Add(23 * time.Hour)
	in := createIn("d1")
	in.CreatedAt = &ok
	l, err := f.s.Create(ctx, in)
	require.NoError(t, err, "within the 24h slack")

	future := f.s.now().Add(25 * time.Hour)
	f.impact.calls = 0
	in.CreatedAt = &future
	_, err = f.s.Create(ctx, in)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	require.Zero(t, f.impact.calls, "rejected before extraction")

	_, err = f.s.Update(ctx, UpdateLogInput{ID: l.ID, DocumentID: "d1", UserID: "u1", CreatedAt: &future})
	require.ErrorAs(t, err, &ve)
}

func TestLogsExtractCancelled(t *testing.T) {
	f := newLogsFixture()
	f.impact.err = errors.New("context canceled")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	l, err := f.s.Create(ctx, createIn("d1"))
	require.NoError(t, err)
	require.Nil(t, l.ImpactStatement)
}

func TestLogsUpdateMore(t *testing.T) {
	ctx := context.Background()
	f := newLogsFixture()
	f.logs.logs["a"] = domain.Log{ID: "a", DocumentID: "d2", Name: "A", Impact: "low", Status: "done"}
	name := "renamed"
	_, err := f.s.Update(ctx, UpdateLogInput{ID: "a", DocumentID: "d2", UserID: "u1", Name: &name})
	require.ErrorIs(t, err, domain.ErrConflict, "archived is read-only")

	l, err := f.s.Create(ctx, createIn("d1"))
	require.NoError(t, err)
	f.impact.calls = 0
	got, err := f.s.Update(ctx, UpdateLogInput{ID: l.ID, DocumentID: "d1", UserID: "u1", Name: &name})
	require.NoError(t, err)
	require.Equal(t, 1, f.impact.calls, "name change re-extracts")

	f.impact.err = errors.New("boom")
	desc := "different"
	got, err = f.s.Update(ctx, UpdateLogInput{ID: got.ID, DocumentID: "d1", UserID: "u1", Description: &desc})
	require.NoError(t, err)
	require.Nil(t, got.ImpactStatement, "failed re-extraction: not checked")

	_, err = f.s.Update(ctx, UpdateLogInput{ID: l.ID, DocumentID: "d3", UserID: "u1", Name: &name})
	require.ErrorIs(t, err, domain.ErrNotFound, "log of another document")
	require.ErrorIs(t, f.s.Delete(ctx, "d3", l.ID, "u1"), domain.ErrNotFound, "log of another document")

	require.ErrorIs(t, f.s.DeleteExamples(ctx, "d1", "u2"), domain.ErrForbidden)
}
