package postgres

import (
	"errors"
	"net"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

func TestWrap(t *testing.T) {
	require.NoError(t, wrap(nil))
	require.ErrorIs(t, wrap(pgx.ErrNoRows), domain.ErrNotFound)
	require.ErrorIs(t, wrap(&net.OpError{Op: "dial", Err: errors.New("refused")}), domain.ErrUnavailable)

	other := errors.New("boom")
	require.Equal(t, other, wrap(other))
}
