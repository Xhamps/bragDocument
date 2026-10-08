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
	other := errors.New("boom")
	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{"nil", nil, nil},
		{"no rows", pgx.ErrNoRows, domain.ErrNotFound},
		{"net error", &net.OpError{Op: "dial", Err: errors.New("refused")}, domain.ErrUnavailable},
		{"deadline passes through", context.DeadlineExceeded, context.DeadlineExceeded},
		{"canceled passes through", context.Canceled, context.Canceled},
		{"unknown passes through", other, other},
		{"unique violation", &pgconn.PgError{Code: "23505"}, domain.ErrConflict},
		{"fk violation", &pgconn.PgError{Code: "23503"}, domain.ErrConflict},
		{"rls violation", &pgconn.PgError{Code: "42501"}, domain.ErrForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := wrap(tc.err)
			if tc.want == nil {
				require.NoError(t, got)
				return
			}
			require.ErrorIs(t, got, tc.want)
		})
	}
	require.NotErrorIs(t, wrap(context.DeadlineExceeded), domain.ErrUnavailable, "timeouts are not retryable")
}
