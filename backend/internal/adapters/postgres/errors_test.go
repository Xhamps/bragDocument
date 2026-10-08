package postgres

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{"unique violation", &pgconn.PgError{Code: "23505"}, domain.ErrConflict},
		{"fk violation", &pgconn.PgError{Code: "23503"}, domain.ErrConflict},
	} {
		t.Run(tc.name, func(t *testing.T) { require.ErrorIs(t, wrap(tc.err), tc.want) })
	}
}
