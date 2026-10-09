package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xhamps/bragdocument/backend/internal/domain"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

func TestSourceFromService(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, domain.SourceWeb, source(ctx))
	require.Equal(t, domain.SourceWeb, source(telemetry.WithService(ctx, "api")))
	require.Equal(t, domain.SourceTelegram, source(telemetry.WithService(ctx, "bot")))
	require.Equal(t, domain.SourceSystem, source(telemetry.WithService(ctx, "worker")))
}
