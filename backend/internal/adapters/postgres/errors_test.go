package postgres

import (
	"context"
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

	got := wrap(context.DeadlineExceeded)
	require.ErrorIs(t, got, context.DeadlineExceeded)
	require.NotErrorIs(t, got, domain.ErrUnavailable)
	require.Equal(t, context.Canceled, wrap(context.Canceled))

	other := errors.New("boom")
	require.Equal(t, other, wrap(other))
}
