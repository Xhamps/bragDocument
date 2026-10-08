package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrateURL(t *testing.T) {
	require.Equal(t, "pgx5://u:p@h:5432/db?sslmode=disable", migrateURL("postgres://u:p@h:5432/db?sslmode=disable"))
	require.Equal(t, "pgx5://u:p@h:5432/db", migrateURL("postgresql://u:p@h:5432/db"))
	require.Equal(t, "pgx5://u:p@h:5432/db", migrateURL("pgx5://u:p@h:5432/db"))
}
