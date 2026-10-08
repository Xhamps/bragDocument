package postgres

import (
	"errors"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/xhamps/bragdocument/backend/internal/domain"
)

// wrap maps driver errors to domain errors so the HTTP mapper and use cases
// never import pgx. Repositories call it on every query error.
func wrap(err error) error {
	var netErr net.Error
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return domain.ErrNotFound
	case pgconn.SafeToRetry(err), errors.As(err, &netErr):
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}
	return err
}
