package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func validLog() Log {
	return Log{Name: " Shipped X ", Impact: "high", Status: "done"}
}

func TestLogValidate(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*Log)
		field string // "" means valid
	}{
		{"valid", func(*Log) {}, ""},
		{"empty name", func(l *Log) { l.Name = "  " }, "name"},
		{"long name", func(l *Log) { l.Name = strings.Repeat("a", 121) }, "name"},
		{"long description", func(l *Log) { l.Description = strings.Repeat("a", 20001) }, "description"},
		{"bad impact", func(l *Log) { l.Impact = "huge" }, "impact"},
		{"bad status", func(l *Log) { l.Status = "doing" }, "status"},
		{"long tag", func(l *Log) { l.Tags = []string{strings.Repeat("a", 51)} }, "tags"},
		{"too many tags", func(l *Log) {
			for i := range 21 {
				l.Tags = append(l.Tags, strings.Repeat("t", i+1))
			}
		}, "tags"},
		{"javascript link", func(l *Log) { l.Links = []Link{{URL: "javascript:alert(1)"}} }, "links"},
		{"relative link", func(l *Log) { l.Links = []Link{{URL: "/pr/1"}} }, "links"},
		{"long label", func(l *Log) { l.Links = []Link{{URL: "https://a.com", Label: strings.Repeat("a", 101)}} }, "links"},
		{"too many links", func(l *Log) { l.Links = make([]Link, 21) }, "links"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := validLog()
			c.edit(&l)
			err := l.Validate()
			if c.field == "" {
				require.NoError(t, err)
				return
			}
			var ve *ValidationError
			require.ErrorAs(t, err, &ve)
			require.Contains(t, ve.Fields, c.field)
		})
	}
}

func TestLogValidateNormalizes(t *testing.T) {
	l := validLog()
	l.Tags = []string{" Project ", "project", "", "Mentorship"}
	l.Links = []Link{{URL: " https://WWW.GitHub.com/x/pull/1 ", Label: " PR "}}
	require.NoError(t, l.Validate())
	require.Equal(t, "Shipped X", l.Name)
	require.Equal(t, []string{"project", "mentorship"}, l.Tags)
	require.Equal(t, Link{URL: "https://WWW.GitHub.com/x/pull/1", Label: "PR", Host: "github.com"}, l.Links[0])
}

func TestLogFilterValidate(t *testing.T) {
	f := LogFilter{Tags: []string{" A "}, Domain: " WWW.GitHub.com ", Query: "  mig "}
	require.NoError(t, f.Validate())
	require.Equal(t, "created_at", f.Sort)
	require.True(t, f.Desc)
	require.Equal(t, 1, f.Page)
	require.Equal(t, 50, f.PerPage)
	require.Equal(t, []string{"a"}, f.Tags)
	require.Equal(t, "github.com", f.Domain)
	require.Equal(t, "mig", f.Query)

	f = LogFilter{PerPage: 1000}
	require.NoError(t, f.Validate())
	require.Equal(t, 100, f.PerPage)

	for _, bad := range []LogFilter{
		{Sort: "title"},
		{Statuses: []string{"doing"}},
		{Impacts: []string{"huge"}},
		{From: ptr(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)), To: ptr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))},
	} {
		var ve *ValidationError
		require.ErrorAs(t, bad.Validate(), &ve, "%+v", bad)
	}
}

func TestExampleLogsAreValid(t *testing.T) {
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	ex := ExampleLogs(now, "u1")
	require.Len(t, ex, 3)
	for _, l := range ex {
		require.NoError(t, l.Validate())
		require.True(t, l.IsExample)
		require.NotNil(t, l.ImpactStatement)
		require.NotEmpty(t, *l.ImpactStatement)
		require.Equal(t, "u1", l.CreatedBy)
		require.True(t, l.CreatedAt.Before(now))
	}
}

func ptr[T any](v T) *T { return &v }
