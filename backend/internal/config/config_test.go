package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, ":8080", cfg.HTTPAddr)
	require.Equal(t, "json", cfg.LogFormat)
	require.Equal(t, 5*time.Second, cfg.DBTimeout)
	require.Equal(t, 200*time.Millisecond, cfg.CacheTimeout)
	require.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	require.Error(t, err)
}

func TestLoadRejectsBadLogFormat(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("LOG_FORMAT", "xml")

	_, err := Load()
	require.ErrorContains(t, err, "LOG_FORMAT")
}
