package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // "" means rejected
	}{
		{"normalized", "  Ada@Example.COM ", "ada@example.com"},
		{"not an email", "not-an-email", ""},
		{"display name", "Ada <ada@example.com>", ""},
		{"comment", "ada@example.com (note)", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeEmail(tc.in)
			if tc.want != "" {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
				return
			}
			var ve *ValidationError
			require.ErrorAs(t, err, &ve)
			require.Contains(t, ve.Fields, "email")
		})
	}
}
