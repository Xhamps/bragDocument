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

func TestLoadTrimsAppURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("APP_URL", "https://brag.example.com/")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://brag.example.com", cfg.AppURL)
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

func TestLoadOwnerURLFallsBackToDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://app:app@localhost:5432/db")
	t.Setenv("DATABASE_OWNER_URL", "")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, cfg.DatabaseURL, cfg.DatabaseOwnerURL)
}

func TestLoadRejectsNonPositiveLLMTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("LLM_TIMEOUT", "0s")

	_, err := Load()
	require.ErrorContains(t, err, "LLM_TIMEOUT must be positive")
}

func TestExportKeyBytes(t *testing.T) {
	k, err := Config{ExportKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}.ExportKeyBytes()
	require.NoError(t, err)
	require.Len(t, k, 32)
	for _, bad := range []string{"", "AAAA", "not base64!"} {
		_, err := Config{ExportKey: bad}.ExportKeyBytes()
		require.ErrorContains(t, err, "EXPORT_KEY", bad)
	}
}

func TestLoadRejectsNonPositiveGotenbergTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("GOTENBERG_TIMEOUT", "0s")

	_, err := Load()
	require.ErrorContains(t, err, "GOTENBERG_TIMEOUT must be positive")
}
