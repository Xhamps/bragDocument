package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  Ada@Example.COM ")
	require.NoError(t, err)
	require.Equal(t, "ada@example.com", got)

	_, err = NormalizeEmail("not-an-email")
	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	require.Contains(t, ve.Fields, "email")
}
