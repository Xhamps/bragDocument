package domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAuditFilterValidate(t *testing.T) {
	f := AuditFilter{}
	require.NoError(t, f.Validate())
	require.Equal(t, 50, f.Limit, "default page size")

	f = AuditFilter{Limit: 500}
	require.NoError(t, f.Validate())
	require.Equal(t, 100, f.Limit, "clamped")

	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, -1)
	err := (&AuditFilter{Action: "log.exploded", From: &from, To: &to}).Validate()
	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	require.Contains(t, ve.Fields, "action")
	require.Contains(t, ve.Fields, "to")
}

func TestDocumentChangeAction(t *testing.T) {
	old := Document{Title: "a", Description: "x", State: DocumentActive}
	cases := []struct {
		name   string
		mod    func(*Document)
		action string
		fields []string
	}{
		{"rename", func(d *Document) { d.Title = "b" }, AuditDocumentRenamed, []string{"title"}},
		{"describe", func(d *Document) { d.Description = "y" }, AuditDocumentEdited, []string{"description"}},
		{"archive beats rename", func(d *Document) { d.Title, d.State = "b", DocumentArchived }, AuditDocumentArchived, []string{"title", "state"}},
		{"nothing", func(*Document) {}, "", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := old
			c.mod(&d)
			action, fields := DocumentChange(old, d)
			require.Equal(t, c.action, action)
			require.Equal(t, c.fields, fields)
		})
	}
	a, _ := DocumentChange(Document{State: DocumentArchived}, Document{State: DocumentActive})
	require.Equal(t, AuditDocumentUnarchived, a)
}

func TestLogChangeAction(t *testing.T) {
	old := Log{Name: "n", Status: StatusDone, Tags: []string{"a"}}
	l := old
	l.Status = "in_progress"
	a, f := LogChange(old, l)
	require.Equal(t, AuditLogStatusChanged, a)
	require.Equal(t, []string{"status"}, f)

	l.Name, l.Tags = "m", []string{"b"}
	a, f = LogChange(old, l)
	require.Equal(t, AuditLogEdited, a)
	require.Equal(t, []string{"name", "status", "tags"}, f)

	a, f = LogChange(old, old)
	require.Empty(t, a)
	require.Nil(t, f)

	reordered := Log{Name: "n", Status: StatusDone, Tags: []string{"b", "a"}}
	a, f = LogChange(Log{Name: "n", Status: StatusDone, Tags: []string{"a", "b"}}, reordered)
	require.Empty(t, a, "tag order is not a change")
	require.Nil(t, f)
	require.Equal(t, []string{"b", "a"}, reordered.Tags, "inputs not mutated")
}

func TestSourceOf(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, SourceWeb, SourceOf(ctx))
	require.Equal(t, SourceTelegram, SourceOf(WithSource(ctx, SourceTelegram)))
	require.Equal(t, SourceSystem, SourceOf(WithSource(ctx, SourceSystem)))
}
