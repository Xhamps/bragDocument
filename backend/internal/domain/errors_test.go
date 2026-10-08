package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSentinelErrorsWrap(t *testing.T) {
	err := fmt.Errorf("document 42: %w", ErrNotFound)
	require.ErrorIs(t, err, ErrNotFound)
	require.NotErrorIs(t, err, ErrForbidden)
}

func TestValidationErrorCarriesFields(t *testing.T) {
	err := NewValidationError(map[string]string{"name": "required"})
	wrapped := fmt.Errorf("create log: %w", err)

	var ve *ValidationError
	require.True(t, errors.As(wrapped, &ve))
	require.Equal(t, "required", ve.Fields["name"])
	require.Equal(t, "validation failed", ve.Error())
}
