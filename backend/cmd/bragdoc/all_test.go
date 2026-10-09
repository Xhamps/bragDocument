package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunServicesFirstErrorStopsTheRest(t *testing.T) {
	boom := errors.New("boom")
	waitStop := func(ctx context.Context) error { <-ctx.Done(); return nil }

	err := runServices(context.Background(),
		service{"api", waitStop},
		service{"bot", func(context.Context) error { return boom }},
		service{"worker", waitStop},
	)

	require.ErrorIs(t, err, boom)
	require.EqualError(t, err, "bot: boom")
}

func TestRunServicesStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	waitStop := func(ctx context.Context) error { <-ctx.Done(); return nil }

	require.NoError(t, runServices(ctx, service{"api", waitStop}, service{"bot", waitStop}))
}
