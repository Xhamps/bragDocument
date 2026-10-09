package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLogMessage(t *testing.T) {
	cases := []struct {
		name, in string
		want     LogDraft
	}{
		{"plain", "Shipped SSO", LogDraft{Name: "Shipped SSO", Impact: "medium", Tags: []string{}, Links: []Link{}}},
		{"all markers", "Shipped SSO #auth !high https://github.com/x/pr/1\nCut login tickets by 40%",
			LogDraft{Name: "Shipped SSO", Description: "Cut login tickets by 40%", Impact: "high",
				Tags: []string{"auth"}, Links: []Link{{URL: "https://github.com/x/pr/1"}}}},
		{"last impact wins, case-insensitive", "Fix !LOW bug !Critical", LogDraft{Name: "Fix bug", Impact: "critical", Tags: []string{}, Links: []Link{}}},
		{"tags in description kept verbatim", "Fix\nfixed the #auth bug !low",
			LogDraft{Name: "Fix", Description: "fixed the #auth bug", Impact: "low", Tags: []string{"auth"}, Links: []Link{}}},
		{"markdown heading is not a tag", "Notes\n# Heading\n##x", LogDraft{Name: "Notes", Description: "# Heading\n##x", Impact: "medium", Tags: []string{}, Links: []Link{}}},
		{"url fragment is not a tag; trailing punctuation trimmed; dedupe",
			"Doc https://a.io/p#sec. https://a.io/p#sec", LogDraft{Name: "Doc", Impact: "medium", Tags: []string{}, Links: []Link{{URL: "https://a.io/p#sec"}}}},
		{"hyphen tags, dedupe", "X #on-call #on-call #Perf", LogDraft{Name: "X", Impact: "medium", Tags: []string{"on-call", "Perf"}, Links: []Link{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseLogMessage(c.in)
			require.NoError(t, err)
			require.Equal(t, c.want, got)
		})
	}
}

func TestParseLogMessageEmptyName(t *testing.T) {
	_, err := ParseLogMessage("#auth !high https://x.io\nbody")
	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	require.Contains(t, ve.Fields, "name")
}
