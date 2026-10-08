//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres/sqlcgen"
)

func startPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("brag"),
		tcpostgres.WithUsername("brag"),
		tcpostgres.WithPassword("brag"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	url, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	return url
}

func TestMigrateAndTenantScopedTx(t *testing.T) {
	url := startPostgres(t)
	ctx := context.Background()

	require.NoError(t, Migrate(url))
	require.NoError(t, Migrate(url), "second run is a no-op")

	db, err := Connect(ctx, url, 5*time.Second)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Ping(ctx))

	err = db.WithTenant(ctx, "tenant-a", func(ctx context.Context, tx pgx.Tx) error {
		var tenant string
		if err := tx.QueryRow(ctx, "SELECT current_setting('app.tenant_id', true)").Scan(&tenant); err != nil {
			return err
		}
		require.Equal(t, "tenant-a", tenant)

		v, err := sqlcgen.New(tx).GetMeta(ctx, "schema")
		require.NoError(t, err)
		require.Equal(t, "1", v)
		return nil
	})
	require.NoError(t, err)

	// Outside the transaction the setting is gone.
	var after string
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT current_setting('app.tenant_id', true)").Scan(&after))
	require.Equal(t, "", after)
}

func TestConnectFailsFastWhenDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err := Connect(ctx, "postgres://u:p@127.0.0.1:1/db?sslmode=disable", time.Second)
	require.Error(t, err)
}
