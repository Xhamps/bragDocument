package telemetry

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoggerAttachesContextIDs(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger("info", "json", &buf)

	ctx := WithRequestID(context.Background(), "req-1")
	ctx = WithTenantID(ctx, "tenant-9")
	log.InfoContext(ctx, "hello")

	require.Contains(t, buf.String(), `"request_id":"req-1"`)
	require.Contains(t, buf.String(), `"tenant_id":"tenant-9"`)
}

func TestLoggerWithoutContextIDs(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger("info", "json", &buf)

	log.Info("plain")

	require.NotContains(t, buf.String(), "request_id")
}

func TestLoggerLevel(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger("warn", "text", &buf)

	log.Info("hidden")
	log.Warn("shown")

	require.NotContains(t, buf.String(), "hidden")
	require.Contains(t, buf.String(), "shown")
}

func TestRequestIDRoundTrip(t *testing.T) {
	require.Equal(t, "", RequestID(context.Background()))
	require.Equal(t, "abc", RequestID(WithRequestID(context.Background(), "abc")))
}
