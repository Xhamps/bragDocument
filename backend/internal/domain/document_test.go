package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentValidate(t *testing.T) {
	cases := []struct {
		name  string
		doc   Document
		field string // "" means valid
	}{
		{"valid", Document{Title: "2026", State: DocumentActive}, ""},
		{"empty title", Document{Title: "   ", State: DocumentActive}, "title"},
		{"long title", Document{Title: strings.Repeat("x", 201), State: DocumentActive}, "title"},
		{"long description", Document{Title: "t", Description: strings.Repeat("x", 2001), State: DocumentActive}, "description"},
		{"bad state", Document{Title: "t", State: "deleted"}, "state"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.doc.Validate()
			if tc.field == "" {
				require.NoError(t, err)
				return
			}
			var ve *ValidationError
			require.ErrorAs(t, err, &ve)
			require.Contains(t, ve.Fields, tc.field)
		})
	}
}
